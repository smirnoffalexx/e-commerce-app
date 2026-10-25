package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	MaxOrderLines         = 50
	MaxLineQuantity       = 100
	MaxIdempotencyKeyLen  = 255
	FailureCardDeclined   = "card_declined"
	FailurePaymentTimeout = "payment_timeout"
	FailurePaymentError   = "payment_error"
)

var (
	ErrOrderNotFound        = NewError(ErrNotFound, "order_not_found", "order not found")
	ErrOrderNotPending      = NewError(ErrConflict, "order_not_pending", "order is already finalized")
	ErrCheckoutInProgress   = NewError(ErrConflict, "checkout_in_progress", "a checkout with this Idempotency-Key is still being processed, retry later")
	ErrIdempotencyKeyReused = NewError(ErrValidation, "idempotency_key_reused", "Idempotency-Key was already used for a different request")
	ErrIdempotencyKeyTaken  = NewError(ErrConflict, "idempotency_key_taken", "order with this Idempotency-Key already exists")
	ErrAccountInactive      = NewError(ErrForbidden, "account_inactive", "account is deactivated")
)

type OrderStatus string

const (
	OrderPending OrderStatus = "PENDING"
	OrderPaid    OrderStatus = "PAID"
	OrderFailed  OrderStatus = "FAILED"
)

func (s OrderStatus) Valid() bool {
	return s == OrderPending || s == OrderPaid || s == OrderFailed
}

type Order struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	CustomerEmail  string
	Status         OrderStatus
	Items          []OrderItem
	Total          decimal.Decimal
	IdempotencyKey string
	RequestHash    string
	CardLast4      string
	PaymentID      string
	FailureCode    string
	PaidAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OrderItem struct {
	ProductID uuid.UUID
	SKU       string
	Name      string
	UnitPrice decimal.Decimal
	Quantity  int
	LineTotal decimal.Decimal
}

func NewOrder(customer *User, in *CheckoutInput, products map[uuid.UUID]Product) Order {
	o := Order{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         customer.ID,
		CustomerEmail:  customer.Email,
		Status:         OrderPending,
		Items:          make([]OrderItem, 0, len(in.Lines)),
		Total:          decimal.Zero,
		IdempotencyKey: in.IdempotencyKey,
		RequestHash:    in.Fingerprint(),
		CardLast4:      in.Card.Last4(),
	}
	for _, line := range in.Lines {
		p := products[line.ProductID]
		item := OrderItem{
			ProductID: p.ID,
			SKU:       p.SKU,
			Name:      p.Name,
			UnitPrice: p.Price,
			Quantity:  line.Quantity,
			LineTotal: p.Price.Mul(decimal.NewFromInt(int64(line.Quantity))),
		}
		o.Items = append(o.Items, item)
		o.Total = o.Total.Add(item.LineTotal)
	}
	return o
}

func (o *Order) MarkPaid(paymentID string, at time.Time) error {
	if o.Status != OrderPending {
		return ErrOrderNotPending
	}
	o.Status = OrderPaid
	o.PaymentID = paymentID
	o.PaidAt = &at
	return nil
}

func (o *Order) MarkFailed(code string) error {
	if o.Status != OrderPending {
		return ErrOrderNotPending
	}
	o.Status = OrderFailed
	o.FailureCode = code
	return nil
}

type CheckoutLine struct {
	ProductID     uuid.UUID
	Quantity      int
	ExpectedPrice decimal.Decimal
}

type CheckoutInput struct {
	IdempotencyKey string
	Lines          []CheckoutLine
	Card           Card
}

func (in *CheckoutInput) Validate(now time.Time) error {
	var verr ValidationError

	key := in.IdempotencyKey
	switch {
	case key == "":
		verr.Add("idempotency_key", CodeRequired, "Idempotency-Key header is required")
	case len(key) > MaxIdempotencyKeyLen:
		verr.Add("idempotency_key", CodeTooLong, "Idempotency-Key must be at most 255 characters")
	case !isPrintableASCII(key):
		verr.Add("idempotency_key", CodeInvalidFormat, "Idempotency-Key may contain only printable ASCII characters")
	}

	switch {
	case len(in.Lines) == 0:
		verr.Add("items", CodeRequired, "order must contain at least one item")
	case len(in.Lines) > MaxOrderLines:
		verr.Add("items", CodeTooLarge, "order may contain at most 50 different products")
	}

	seen := make(map[uuid.UUID]bool, len(in.Lines))
	for i, line := range in.Lines {
		field := "items[" + strconv.Itoa(i) + "]."
		switch {
		case line.ProductID == uuid.Nil:
			verr.Add(field+"product_id", CodeRequired, "product_id is required")
		case seen[line.ProductID]:
			verr.Add(field+"product_id", "duplicate", "each product may appear only once, combine the quantities")
		}
		seen[line.ProductID] = true

		if line.Quantity < 1 || line.Quantity > MaxLineQuantity {
			verr.Add(field+"quantity", CodeInvalidFormat, "quantity must be between 1 and 100")
		}
		validateDecimal(&verr, field+"unit_price", line.ExpectedPrice, PriceDecimals, MaxPrice)
	}

	in.Card.validate(&verr, now)
	return verr.Err()
}

func (in *CheckoutInput) ProductIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(in.Lines))
	for _, line := range in.Lines {
		ids = append(ids, line.ProductID)
	}
	return ids
}

func (in *CheckoutInput) Fingerprint() string {
	lines := slices.Clone(in.Lines)
	slices.SortFunc(lines, func(a, b CheckoutLine) int { return strings.Compare(a.ProductID.String(), b.ProductID.String()) })

	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line.ProductID.String())
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(line.Quantity))
		b.WriteByte(':')
		b.WriteString(line.ExpectedPrice.StringFixed(PriceDecimals))
		b.WriteByte('\n')
	}
	b.WriteString(in.Card.Last4())
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

type CheckoutIssueCode string

const (
	IssueProductUnavailable CheckoutIssueCode = "product_unavailable"
	IssueOutOfStock         CheckoutIssueCode = "out_of_stock"
	IssueInsufficientStock  CheckoutIssueCode = "insufficient_stock"
	IssuePriceChanged       CheckoutIssueCode = "price_changed"
)

type CheckoutIssue struct {
	ProductID         uuid.UUID
	Code              CheckoutIssueCode
	RequestedQuantity int
	AvailableQuantity int
	ExpectedPrice     decimal.Decimal
	CurrentPrice      decimal.Decimal
}

type CheckoutConflictError struct {
	Issues []CheckoutIssue
}

func CheckCart(in *CheckoutInput, products map[uuid.UUID]Product) error {
	var issues []CheckoutIssue
	for _, line := range in.Lines {
		issue := CheckoutIssue{ProductID: line.ProductID, RequestedQuantity: line.Quantity, ExpectedPrice: line.ExpectedPrice}
		p, ok := products[line.ProductID]
		if !ok {
			issue.Code = IssueProductUnavailable
			issues = append(issues, issue)
			continue
		}

		issue.AvailableQuantity = p.Available()
		issue.CurrentPrice = p.Price
		switch {
		case !p.InStock():
			issue.Code = IssueOutOfStock
		case p.Available() < line.Quantity:
			issue.Code = IssueInsufficientStock
		case !p.Price.Equal(line.ExpectedPrice):
			issue.Code = IssuePriceChanged
		default:
			continue
		}
		issues = append(issues, issue)
	}
	if len(issues) == 0 {
		return nil
	}
	return &CheckoutConflictError{Issues: issues}
}

func (e *CheckoutConflictError) Error() string {
	switch e.Code() {
	case string(IssuePriceChanged):
		return "prices of some products have changed, review the cart and confirm the new prices"
	default:
		return "some products are no longer available in the requested quantity"
	}
}

func (e *CheckoutConflictError) Unwrap() error { return ErrConflict }

func (e *CheckoutConflictError) Code() string {
	for _, issue := range e.Issues {
		if issue.Code != IssuePriceChanged {
			return "items_unavailable"
		}
	}
	return string(IssuePriceChanged)
}

type OrderFilter struct {
	UserID *uuid.UUID
	Status OrderStatus
	Limit  int
	Offset int
}

type CheckoutResult struct {
	Order    Order
	Replayed bool
}

type ListOrdersQuery struct {
	UserID *uuid.UUID
	Status OrderStatus
	PageRequest
}

func (q ListOrdersQuery) Filter() (OrderFilter, PageRequest, error) {
	var verr ValidationError
	page, err := q.Normalize()
	verr.Merge(err)
	if q.Status != "" && !q.Status.Valid() {
		verr.Add("status", CodeInvalidFormat, "status must be one of PENDING, PAID, FAILED")
	}
	if err := verr.Err(); err != nil {
		return OrderFilter{}, PageRequest{}, err
	}
	return OrderFilter{UserID: q.UserID, Status: q.Status, Limit: page.PageSize, Offset: page.Offset()}, page, nil
}
