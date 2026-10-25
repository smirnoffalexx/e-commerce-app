package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
	"e-commerce-backend/internal/usecase/mocks"
)

type orderDeps struct {
	tx       *mocks.Transactor
	orders   *mocks.OrderRepository
	users    *mocks.UserRepository
	payments *mocks.PaymentProvider
	locks    *mocks.CheckoutLocker
	svc      *usecase.OrderService
	customer domain.User
	product  domain.Product
	created  domain.Order
}

func newOrders(t *testing.T) *orderDeps {
	t.Helper()
	d := &orderDeps{
		tx:       mocks.NewTransactor(t),
		orders:   mocks.NewOrderRepository(t),
		users:    mocks.NewUserRepository(t),
		payments: mocks.NewPaymentProvider(t),
		locks:    mocks.NewCheckoutLocker(t),
		customer: user(t, domain.RoleUser),
		product: domain.Product{
			ID:    uuid.Must(uuid.NewV7()),
			SKU:   "RS-001",
			Name:  "Running Shoes",
			Price: decimal.RequireFromString("89.99"),
			Stock: 5,
		},
	}
	d.svc = usecase.NewOrderService(d.tx, d.orders, d.users, d.payments, d.locks).WithClock(func() time.Time { return fixedNow })
	d.tx.EXPECT().WithinTransaction(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Maybe()
	return d
}

func (d *orderDeps) input(quantity int, price string) domain.CheckoutInput {
	return domain.CheckoutInput{
		IdempotencyKey: "checkout-1",
		Lines:          []domain.CheckoutLine{{ProductID: d.product.ID, Quantity: quantity, ExpectedPrice: decimal.RequireFromString(price)}},
		Card:           domain.Card{Number: "4242424242424242", ExpMonth: 12, ExpYear: 2030, CVC: "123"},
	}
}

func (d *orderDeps) expectNoPreviousOrder() {
	d.orders.EXPECT().GetByIdempotencyKey(mock.Anything, d.customer.ID, "checkout-1").Return(domain.Order{}, domain.ErrOrderNotFound).Once()
}

func (d *orderDeps) expectLock() {
	d.locks.EXPECT().Acquire(mock.Anything, mock.Anything, usecase.CheckoutLockTTL).Return("token", true, nil)
	d.locks.EXPECT().Release(mock.Anything, mock.Anything, "token").Return(nil)
}

func (d *orderDeps) expectReservation(onReserve ...func()) {
	d.users.EXPECT().GetByID(mock.Anything, d.customer.ID).Return(d.customer, nil)
	d.orders.EXPECT().LockProducts(mock.Anything, []uuid.UUID{d.product.ID}).
		Return(map[uuid.UUID]domain.Product{d.product.ID: d.product}, nil)
	d.orders.EXPECT().ReserveStock(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, []domain.OrderItem) error {
			for _, fn := range onReserve {
				fn()
			}
			return nil
		})
	d.orders.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, o *domain.Order) error {
			d.created = *o
			return nil
		})
}

func (d *orderDeps) expectFinalize() {
	d.orders.EXPECT().LockByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, uuid.UUID) (domain.Order, error) { return d.created, nil })
	d.orders.EXPECT().UpdateStatus(mock.Anything, mock.Anything).Return(nil)
}

func (d *orderDeps) expectCommit() *[]domain.OrderItem {
	var committed []domain.OrderItem
	d.orders.EXPECT().CommitStock(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, items []domain.OrderItem) error {
			committed = items
			return nil
		})
	return &committed
}

func TestOrderService_Checkout_Paid(t *testing.T) {
	d := newOrders(t)
	d.expectNoPreviousOrder()
	d.expectLock()
	d.expectReservation()
	d.payments.EXPECT().Charge(mock.Anything, mock.MatchedBy(func(r domain.PaymentRequest) bool {
		return r.Amount.Equal(decimal.RequireFromString("179.98")) && r.IdempotencyKey == d.created.ID.String()
	})).Return(domain.PaymentResult{Approved: true, PaymentID: "pay_1"}, nil)
	d.expectFinalize()
	committed := d.expectCommit()

	res, err := d.svc.Checkout(context.Background(), d.customer.ID, d.input(2, "89.99"))
	require.NoError(t, err)

	assert.False(t, res.Replayed)
	assert.Equal(t, domain.OrderPaid, res.Order.Status)
	assert.Equal(t, "pay_1", res.Order.PaymentID)
	assert.Equal(t, fixedNow, *res.Order.PaidAt)
	assert.Equal(t, d.customer.Email, res.Order.CustomerEmail)
	require.Len(t, res.Order.Items, 1)
	assert.Equal(t, "RS-001", res.Order.Items[0].SKU)
	require.Len(t, *committed, 1)
	assert.Equal(t, d.product.ID, (*committed)[0].ProductID)
	assert.Equal(t, 2, (*committed)[0].Quantity)
}

func TestOrderService_Checkout_PaymentChargedWithDeadlineEvenIfClientLeft(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	d := newOrders(t)
	d.expectNoPreviousOrder()
	d.expectLock()
	d.expectReservation(cancel)
	d.payments.EXPECT().Charge(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, _ domain.PaymentRequest) (domain.PaymentResult, error) {
			_, hasDeadline := ctx.Deadline()
			assert.True(t, hasDeadline)
			assert.NoError(t, ctx.Err(), "client disconnect must not abort a reserved order")
			return domain.PaymentResult{Approved: true, PaymentID: "pay_1"}, nil
		})
	d.expectFinalize()
	d.expectCommit()

	res, err := d.svc.Checkout(ctx, d.customer.ID, d.input(1, "89.99"))
	require.NoError(t, err)
	assert.Equal(t, domain.OrderPaid, res.Order.Status)
}

func TestOrderService_Checkout_FailedPaymentReleasesStock(t *testing.T) {
	cases := []struct {
		name     string
		result   domain.PaymentResult
		err      error
		wantCode string
	}{
		{"declined", domain.PaymentResult{DeclineCode: domain.FailureCardDeclined}, nil, domain.FailureCardDeclined},
		{"declined without reason", domain.PaymentResult{}, nil, domain.FailureCardDeclined},
		{"insufficient funds", domain.PaymentResult{DeclineCode: "insufficient_funds"}, nil, "insufficient_funds"},
		{"timeout", domain.PaymentResult{}, usecase.ErrPaymentTimeout, domain.FailurePaymentTimeout},
		{"deadline", domain.PaymentResult{}, context.DeadlineExceeded, domain.FailurePaymentTimeout},
		{"provider error", domain.PaymentResult{}, errors.New("connection reset"), domain.FailurePaymentError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newOrders(t)
			d.expectNoPreviousOrder()
			d.expectLock()
			d.expectReservation()
			d.payments.EXPECT().Charge(mock.Anything, mock.Anything).Return(tc.result, tc.err)
			d.expectFinalize()
			var released []domain.OrderItem
			d.orders.EXPECT().ReleaseStock(mock.Anything, mock.Anything).
				RunAndReturn(func(_ context.Context, items []domain.OrderItem) error {
					released = items
					return nil
				})

			res, err := d.svc.Checkout(context.Background(), d.customer.ID, d.input(2, "89.99"))
			require.NoError(t, err)

			assert.Equal(t, domain.OrderFailed, res.Order.Status)
			assert.Equal(t, tc.wantCode, res.Order.FailureCode)
			assert.Empty(t, res.Order.PaymentID)
			require.Len(t, released, 1)
			assert.Equal(t, d.product.ID, released[0].ProductID)
			assert.Equal(t, 2, released[0].Quantity)
		})
	}
}

func TestOrderService_Checkout_CartConflictsStopBeforePayment(t *testing.T) {
	cases := []struct {
		name     string
		quantity int
		price    string
		stock    int
		wantCode string
	}{
		{"price changed", 1, "79.99", 5, "price_changed"},
		{"not enough stock", 6, "89.99", 5, "items_unavailable"},
		{"sold out", 1, "89.99", 0, "items_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newOrders(t)
			d.product.Stock = tc.stock
			d.expectNoPreviousOrder()
			d.expectLock()
			d.users.EXPECT().GetByID(mock.Anything, d.customer.ID).Return(d.customer, nil)
			d.orders.EXPECT().LockProducts(mock.Anything, mock.Anything).
				Return(map[uuid.UUID]domain.Product{d.product.ID: d.product}, nil)

			_, err := d.svc.Checkout(context.Background(), d.customer.ID, d.input(tc.quantity, tc.price))

			var cerr *domain.CheckoutConflictError
			require.ErrorAs(t, err, &cerr)
			assert.Equal(t, tc.wantCode, cerr.Code())
		})
	}
}

func TestOrderService_Checkout_DeletedProductIsUnavailable(t *testing.T) {
	d := newOrders(t)
	d.expectNoPreviousOrder()
	d.expectLock()
	d.users.EXPECT().GetByID(mock.Anything, d.customer.ID).Return(d.customer, nil)
	d.orders.EXPECT().LockProducts(mock.Anything, mock.Anything).Return(map[uuid.UUID]domain.Product{}, nil)

	_, err := d.svc.Checkout(context.Background(), d.customer.ID, d.input(1, "89.99"))

	var cerr *domain.CheckoutConflictError
	require.ErrorAs(t, err, &cerr)
	assert.Equal(t, domain.IssueProductUnavailable, cerr.Issues[0].Code)
}

func TestOrderService_Checkout_InvalidInputTouchesNothing(t *testing.T) {
	d := newOrders(t)
	in := d.input(0, "89.99")

	_, err := d.svc.Checkout(context.Background(), d.customer.ID, in)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestOrderService_Checkout_InactiveAccount(t *testing.T) {
	d := newOrders(t)
	d.customer.Active = false
	d.expectNoPreviousOrder()
	d.expectLock()
	d.users.EXPECT().GetByID(mock.Anything, d.customer.ID).Return(d.customer, nil)

	_, err := d.svc.Checkout(context.Background(), d.customer.ID, d.input(1, "89.99"))
	assert.ErrorIs(t, err, domain.ErrAccountInactive)
}

func TestOrderService_Checkout_Idempotency(t *testing.T) {
	ctx := context.Background()

	existing := func(d *orderDeps, status domain.OrderStatus, in domain.CheckoutInput) domain.Order {
		return domain.Order{ID: uuid.Must(uuid.NewV7()), UserID: d.customer.ID, Status: status, IdempotencyKey: in.IdempotencyKey, RequestHash: in.Fingerprint()}
	}

	t.Run("same request returns the stored result without charging again", func(t *testing.T) {
		for _, status := range []domain.OrderStatus{domain.OrderPaid, domain.OrderFailed} {
			d := newOrders(t)
			in := d.input(1, "89.99")
			stored := existing(d, status, in)
			d.orders.EXPECT().GetByIdempotencyKey(ctx, d.customer.ID, "checkout-1").Return(stored, nil)

			res, err := d.svc.Checkout(ctx, d.customer.ID, in)
			require.NoError(t, err)
			assert.True(t, res.Replayed)
			assert.Equal(t, stored.ID, res.Order.ID)
		}
	})

	t.Run("same key with a different body is rejected", func(t *testing.T) {
		d := newOrders(t)
		stored := existing(d, domain.OrderPaid, d.input(1, "89.99"))
		d.orders.EXPECT().GetByIdempotencyKey(ctx, d.customer.ID, "checkout-1").Return(stored, nil)

		_, err := d.svc.Checkout(ctx, d.customer.ID, d.input(2, "89.99"))
		assert.ErrorIs(t, err, domain.ErrIdempotencyKeyReused)
	})

	t.Run("pending order means the first request is still running", func(t *testing.T) {
		d := newOrders(t)
		in := d.input(1, "89.99")
		d.orders.EXPECT().GetByIdempotencyKey(ctx, d.customer.ID, "checkout-1").Return(existing(d, domain.OrderPending, in), nil)

		_, err := d.svc.Checkout(ctx, d.customer.ID, in)
		assert.ErrorIs(t, err, domain.ErrCheckoutInProgress)
	})

	t.Run("concurrent request holding the lock", func(t *testing.T) {
		d := newOrders(t)
		d.expectNoPreviousOrder()
		d.locks.EXPECT().Acquire(mock.Anything, mock.Anything, mock.Anything).Return("", false, nil)

		_, err := d.svc.Checkout(ctx, d.customer.ID, d.input(1, "89.99"))
		assert.ErrorIs(t, err, domain.ErrCheckoutInProgress)
	})

	t.Run("redis outage falls back to the unique index", func(t *testing.T) {
		d := newOrders(t)
		in := d.input(1, "89.99")
		d.expectNoPreviousOrder()
		d.locks.EXPECT().Acquire(mock.Anything, mock.Anything, mock.Anything).Return("", false, errors.New("redis down"))
		d.users.EXPECT().GetByID(mock.Anything, d.customer.ID).Return(d.customer, nil)
		d.orders.EXPECT().LockProducts(mock.Anything, mock.Anything).Return(map[uuid.UUID]domain.Product{d.product.ID: d.product}, nil)
		d.orders.EXPECT().ReserveStock(mock.Anything, mock.Anything).Return(nil)
		d.orders.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.ErrIdempotencyKeyTaken)
		stored := existing(d, domain.OrderPaid, in)
		d.orders.EXPECT().GetByIdempotencyKey(mock.Anything, d.customer.ID, "checkout-1").Return(stored, nil).Once()

		res, err := d.svc.Checkout(ctx, d.customer.ID, in)
		require.NoError(t, err)
		assert.True(t, res.Replayed)
		assert.Equal(t, stored.ID, res.Order.ID)
	})

	t.Run("lock is keyed by user and key", func(t *testing.T) {
		keys := map[string]bool{}
		for _, userID := range []uuid.UUID{uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())} {
			d := newOrders(t)
			d.orders.EXPECT().GetByIdempotencyKey(ctx, userID, "checkout-1").Return(domain.Order{}, domain.ErrOrderNotFound)
			d.locks.EXPECT().Acquire(mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(_ context.Context, key string, _ time.Duration) (string, bool, error) {
					keys[key] = true
					return "", false, nil
				})
			_, _ = d.svc.Checkout(ctx, userID, d.input(1, "89.99"))
		}
		assert.Len(t, keys, 2)
	})
}

func TestOrderService_Get(t *testing.T) {
	ctx := context.Background()
	owner := uuid.Must(uuid.NewV7())
	order := domain.Order{ID: uuid.Must(uuid.NewV7()), UserID: owner}

	cases := []struct {
		name    string
		viewer  domain.Principal
		wantErr error
	}{
		{"owner", domain.Principal{UserID: owner, Role: domain.RoleUser}, nil},
		{"another customer gets not found", domain.Principal{UserID: uuid.Must(uuid.NewV7()), Role: domain.RoleUser}, domain.ErrOrderNotFound},
		{"admin", domain.Principal{UserID: uuid.Must(uuid.NewV7()), Role: domain.RoleAdmin}, nil},
		{"superadmin", domain.Principal{UserID: uuid.Must(uuid.NewV7()), Role: domain.RoleSuperAdmin}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newOrders(t)
			d.orders.EXPECT().GetByID(ctx, order.ID).Return(order, nil)

			got, err := d.svc.Get(ctx, tc.viewer, order.ID)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, order.ID, got.ID)
		})
	}

	t.Run("unknown role", func(t *testing.T) {
		d := newOrders(t)
		_, err := d.svc.Get(ctx, domain.Principal{Role: "GUEST"}, order.ID)
		assert.ErrorIs(t, err, domain.ErrInsufficientPermissions)
	})
}

func TestOrderService_List(t *testing.T) {
	ctx := context.Background()
	me := uuid.Must(uuid.NewV7())
	other := uuid.Must(uuid.NewV7())

	t.Run("customer is always scoped to own orders", func(t *testing.T) {
		d := newOrders(t)
		d.orders.EXPECT().List(ctx, domain.OrderFilter{UserID: &me, Status: domain.OrderPaid, Limit: 20, Offset: 0}).
			Return([]domain.Order{{ID: uuid.Must(uuid.NewV7())}}, 1, nil)

		page, err := d.svc.List(ctx, domain.Principal{UserID: me, Role: domain.RoleUser}, domain.ListOrdersQuery{Status: domain.OrderPaid})
		require.NoError(t, err)
		assert.Equal(t, int64(1), page.Total)
	})

	t.Run("customer cannot ask for someone else's orders", func(t *testing.T) {
		d := newOrders(t)
		_, err := d.svc.List(ctx, domain.Principal{UserID: me, Role: domain.RoleUser}, domain.ListOrdersQuery{UserID: &other})
		assert.ErrorIs(t, err, domain.ErrInsufficientPermissions)
	})

	t.Run("admin sees all orders and may filter by customer", func(t *testing.T) {
		d := newOrders(t)
		d.orders.EXPECT().List(ctx, domain.OrderFilter{Limit: 10, Offset: 10}).Return(nil, 0, nil)
		d.orders.EXPECT().List(ctx, domain.OrderFilter{UserID: &other, Limit: 20}).Return(nil, 0, nil)

		admin := domain.Principal{UserID: me, Role: domain.RoleAdmin}
		_, err := d.svc.List(ctx, admin, domain.ListOrdersQuery{PageRequest: domain.PageRequest{Page: 2, PageSize: 10}})
		require.NoError(t, err)
		_, err = d.svc.List(ctx, admin, domain.ListOrdersQuery{UserID: &other})
		require.NoError(t, err)
	})

	t.Run("invalid status and paging are reported together", func(t *testing.T) {
		d := newOrders(t)
		_, err := d.svc.List(ctx, domain.Principal{UserID: me, Role: domain.RoleAdmin},
			domain.ListOrdersQuery{Status: "SHIPPED", PageRequest: domain.PageRequest{PageSize: 1000}})

		var verr *domain.ValidationError
		require.ErrorAs(t, err, &verr)
		assert.Len(t, verr.Fields, 2)
	})
}
