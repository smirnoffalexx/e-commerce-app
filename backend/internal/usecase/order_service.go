package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"e-commerce-backend/internal/domain"
)

const (
	CheckoutLockTTL = 30 * time.Second
	PaymentTimeout  = 10 * time.Second
)

var ErrPaymentTimeout = errors.New("payment provider timed out")

type PaymentRequest struct {
	IdempotencyKey string
	Amount         decimal.Decimal
	Card           domain.Card
}

type PaymentResult struct {
	Approved    bool
	PaymentID   string
	DeclineCode string
}

type CheckoutResult struct {
	Order    domain.Order
	Replayed bool
}

type ListOrdersQuery struct {
	UserID *uuid.UUID
	Status domain.OrderStatus
	PageRequest
}

type OrderService struct {
	tx       Transactor
	orders   OrderRepository
	users    UserRepository
	payments PaymentProvider
	locks    CheckoutLocker
	now      func() time.Time
}

func NewOrderService(tx Transactor, orders OrderRepository, users UserRepository, payments PaymentProvider, locks CheckoutLocker) *OrderService {
	return &OrderService{tx: tx, orders: orders, users: users, payments: payments, locks: locks, now: time.Now}
}

func (s *OrderService) WithClock(now func() time.Time) *OrderService {
	s.now = now
	return s
}

func (s *OrderService) Checkout(ctx context.Context, userID uuid.UUID, in domain.CheckoutInput) (CheckoutResult, error) {
	if err := in.Validate(s.now()); err != nil {
		return CheckoutResult{}, err
	}
	if res, done, err := s.replay(ctx, userID, &in); done {
		return res, err
	}

	lockKey := checkoutLockKey(userID, in.IdempotencyKey)
	token, acquired, err := s.locks.Acquire(ctx, lockKey, CheckoutLockTTL)
	switch {
	case err != nil:
		slog.WarnContext(ctx, "checkout lock unavailable, relying on database uniqueness", slog.Any("error", err))
	case !acquired:
		return CheckoutResult{}, domain.ErrCheckoutInProgress
	default:
		defer s.release(ctx, lockKey, token)
	}

	customer, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return CheckoutResult{}, err
	}
	if !customer.Active {
		return CheckoutResult{}, domain.ErrAccountInactive
	}

	order, err := s.reserve(ctx, &customer, &in)
	if errors.Is(err, domain.ErrIdempotencyKeyTaken) {
		if res, done, err := s.replay(ctx, userID, &in); done {
			return res, err
		}
		return CheckoutResult{}, domain.ErrCheckoutInProgress
	}
	if err != nil {
		return CheckoutResult{}, err
	}

	order, err = s.pay(context.WithoutCancel(ctx), &order, in.Card)
	if err != nil {
		return CheckoutResult{}, err
	}
	return CheckoutResult{Order: order}, nil
}

func (s *OrderService) Get(ctx context.Context, viewer domain.Principal, id uuid.UUID) (domain.Order, error) {
	if !viewer.Role.Can(domain.PermOrdersReadAll) && !viewer.Role.Can(domain.PermOrdersReadOwn) {
		return domain.Order{}, domain.ErrInsufficientPermissions
	}
	o, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if !viewer.Role.Can(domain.PermOrdersReadAll) && o.UserID != viewer.UserID {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return o, nil
}

func (s *OrderService) List(ctx context.Context, viewer domain.Principal, q ListOrdersQuery) (Page[domain.Order], error) {
	var verr domain.ValidationError
	page, err := q.normalize()
	collect(&verr, err)
	if q.Status != "" && !q.Status.Valid() {
		verr.Add("status", domain.CodeInvalidFormat, "status must be one of PENDING, PAID, FAILED")
	}
	if err := verr.Err(); err != nil {
		return Page[domain.Order]{}, err
	}

	filter := domain.OrderFilter{UserID: q.UserID, Status: q.Status, Limit: page.PageSize, Offset: page.offset()}
	switch {
	case viewer.Role.Can(domain.PermOrdersReadAll):
	case viewer.Role.Can(domain.PermOrdersReadOwn):
		if q.UserID != nil && *q.UserID != viewer.UserID {
			return Page[domain.Order]{}, domain.ErrInsufficientPermissions
		}
		filter.UserID = &viewer.UserID
	default:
		return Page[domain.Order]{}, domain.ErrInsufficientPermissions
	}

	items, total, err := s.orders.List(ctx, filter)
	if err != nil {
		return Page[domain.Order]{}, err
	}
	return Page[domain.Order]{Items: items, Total: total, Page: page.Page, PageSize: page.PageSize}, nil
}

func (s *OrderService) replay(ctx context.Context, userID uuid.UUID, in *domain.CheckoutInput) (CheckoutResult, bool, error) {
	o, err := s.orders.GetByIdempotencyKey(ctx, userID, in.IdempotencyKey)
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		return CheckoutResult{}, false, nil
	case err != nil:
		return CheckoutResult{}, true, err
	case o.RequestHash != in.Fingerprint():
		return CheckoutResult{}, true, domain.ErrIdempotencyKeyReused
	case o.Status == domain.OrderPending:
		return CheckoutResult{}, true, domain.ErrCheckoutInProgress
	}
	return CheckoutResult{Order: o, Replayed: true}, true, nil
}

func (s *OrderService) reserve(ctx context.Context, customer *domain.User, in *domain.CheckoutInput) (domain.Order, error) {
	var order domain.Order
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		products, err := s.orders.LockProducts(ctx, in.ProductIDs())
		if err != nil {
			return err
		}
		if err := domain.CheckCart(in, products); err != nil {
			return err
		}
		order = domain.NewOrder(customer, in, products)
		if err := s.orders.ReserveStock(ctx, order.Items); err != nil {
			return err
		}
		return s.orders.Create(ctx, &order)
	})
	return order, err
}

func (s *OrderService) pay(ctx context.Context, order *domain.Order, card domain.Card) (domain.Order, error) {
	payCtx, cancel := context.WithTimeout(ctx, PaymentTimeout)
	result, err := s.payments.Charge(payCtx, PaymentRequest{IdempotencyKey: order.ID.String(), Amount: order.Total, Card: card})
	cancel()

	var failure string
	switch {
	case errors.Is(err, ErrPaymentTimeout), errors.Is(err, context.DeadlineExceeded):
		failure = domain.FailurePaymentTimeout
	case err != nil:
		slog.ErrorContext(ctx, "payment provider failed", slog.String("order_id", order.ID.String()), slog.Any("error", err))
		failure = domain.FailurePaymentError
	case !result.Approved && result.DeclineCode != "":
		failure = result.DeclineCode
	case !result.Approved:
		failure = domain.FailureCardDeclined
	}
	return s.finalize(ctx, order.ID, result.PaymentID, failure)
}

func (s *OrderService) finalize(ctx context.Context, id uuid.UUID, paymentID, failure string) (domain.Order, error) {
	var order domain.Order
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		o, err := s.orders.LockByID(ctx, id)
		if err != nil {
			return err
		}
		if failure == "" {
			err = o.MarkPaid(paymentID, s.now().UTC())
		} else {
			err = o.MarkFailed(failure)
		}
		if err != nil {
			return err
		}
		if failure != "" {
			if err := s.orders.ReleaseStock(ctx, o.Items); err != nil {
				return err
			}
		}
		if err := s.orders.UpdateStatus(ctx, &o); err != nil {
			return err
		}
		order = o
		return nil
	})
	return order, err
}

func (s *OrderService) release(ctx context.Context, key, token string) {
	if err := s.locks.Release(context.WithoutCancel(ctx), key, token); err != nil {
		slog.WarnContext(ctx, "release checkout lock", slog.Any("error", err))
	}
}

func checkoutLockKey(userID uuid.UUID, idempotencyKey string) string {
	sum := sha256.Sum256([]byte(idempotencyKey))
	return userID.String() + ":" + hex.EncodeToString(sum[:])
}
