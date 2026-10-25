package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var checkoutNow = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func validCard() Card {
	return Card{Number: "4242 4242 4242 4242", ExpMonth: 12, ExpYear: 2030, CVC: "123"}
}

func validCheckout(lines ...CheckoutLine) CheckoutInput {
	if len(lines) == 0 {
		lines = []CheckoutLine{{ProductID: uuid.Must(uuid.NewV7()), Quantity: 2, ExpectedPrice: decimal.RequireFromString("89.99")}}
	}
	return CheckoutInput{IdempotencyKey: "key-1", Lines: lines, Card: validCard()}
}

func fieldCodes(t *testing.T, err error) map[string]string {
	t.Helper()
	var verr *ValidationError
	require.True(t, errors.As(err, &verr), "expected validation error, got %v", err)
	out := map[string]string{}
	for _, f := range verr.Fields {
		out[f.Field] = f.Code
	}
	return out
}

func TestCheckoutInput_Validate(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	line := func(q int, price string) CheckoutLine {
		return CheckoutLine{ProductID: uuid.Must(uuid.NewV7()), Quantity: q, ExpectedPrice: decimal.RequireFromString(price)}
	}
	tooMany := make([]CheckoutLine, MaxOrderLines+1)
	for i := range tooMany {
		tooMany[i] = line(1, "1")
	}
	maxed := make([]CheckoutLine, MaxOrderLines)
	for i := range maxed {
		maxed[i] = line(MaxLineQuantity, "1")
	}

	cases := []struct {
		name   string
		modify func(in *CheckoutInput)
		want   map[string]string
	}{
		{"valid", func(*CheckoutInput) {}, nil},
		{"limits are inclusive", func(in *CheckoutInput) { in.Lines = maxed }, nil},
		{"empty cart", func(in *CheckoutInput) { in.Lines = nil }, map[string]string{"items": CodeRequired}},
		{"too many lines", func(in *CheckoutInput) { in.Lines = tooMany }, map[string]string{"items": CodeTooLarge}},
		{"zero quantity", func(in *CheckoutInput) { in.Lines = []CheckoutLine{line(0, "1")} }, map[string]string{"items[0].quantity": CodeInvalidFormat}},
		{"quantity above limit", func(in *CheckoutInput) { in.Lines = []CheckoutLine{line(101, "1")} }, map[string]string{"items[0].quantity": CodeInvalidFormat}},
		{"negative price", func(in *CheckoutInput) { in.Lines = []CheckoutLine{line(1, "-1")} }, map[string]string{"items[0].unit_price": CodeNegative}},
		{"price with fractions of a cent", func(in *CheckoutInput) { in.Lines = []CheckoutLine{line(1, "1.001")} }, map[string]string{"items[0].unit_price": CodeTooManyDecimals}},
		{"missing product", func(in *CheckoutInput) { in.Lines = []CheckoutLine{{Quantity: 1}} }, map[string]string{"items[0].product_id": CodeRequired}},
		{
			"duplicate product",
			func(in *CheckoutInput) {
				in.Lines = []CheckoutLine{{ProductID: id, Quantity: 1}, {ProductID: id, Quantity: 2}}
			},
			map[string]string{"items[1].product_id": "duplicate"},
		},
		{"missing key", func(in *CheckoutInput) { in.IdempotencyKey = "" }, map[string]string{"idempotency_key": CodeRequired}},
		{"non-ascii key", func(in *CheckoutInput) { in.IdempotencyKey = "ключ" }, map[string]string{"idempotency_key": CodeInvalidFormat}},
		{"long key", func(in *CheckoutInput) { in.IdempotencyKey = string(make([]byte, 256)) }, map[string]string{"idempotency_key": CodeTooLong}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validCheckout()
			tc.modify(&in)
			err := in.Validate(checkoutNow)
			if tc.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.Equal(t, tc.want, fieldCodes(t, err))
		})
	}
}

func TestCard_Validate(t *testing.T) {
	cases := []struct {
		name   string
		modify func(c *Card)
		want   map[string]string
	}{
		{"valid with spaces", func(*Card) {}, nil},
		{"valid with dashes", func(c *Card) { c.Number = "4000-0000-0000-0002" }, nil},
		{"current month is not expired", func(c *Card) { c.ExpMonth, c.ExpYear = 10, 2026 }, nil},
		{"missing number", func(c *Card) { c.Number = " " }, map[string]string{"payment.card_number": CodeRequired}},
		{"letters", func(c *Card) { c.Number = "4242abcd42424242" }, map[string]string{"payment.card_number": CodeInvalidFormat}},
		{"too short", func(c *Card) { c.Number = "42424242" }, map[string]string{"payment.card_number": CodeInvalidFormat}},
		{"luhn mismatch", func(c *Card) { c.Number = "4242424242424241" }, map[string]string{"payment.card_number": CodeInvalidFormat}},
		{"expired last month", func(c *Card) { c.ExpMonth, c.ExpYear = 9, 2026 }, map[string]string{"payment.exp_month": "expired"}},
		{"past year", func(c *Card) { c.ExpYear = 2025 }, map[string]string{"payment.exp_year": CodeInvalidFormat}},
		{"two-digit year", func(c *Card) { c.ExpYear = 30 }, map[string]string{"payment.exp_year": CodeInvalidFormat}},
		{"month 13", func(c *Card) { c.ExpMonth = 13 }, map[string]string{"payment.exp_month": CodeInvalidFormat}},
		{"short cvc", func(c *Card) { c.CVC = "12" }, map[string]string{"payment.cvc": CodeInvalidFormat}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validCheckout()
			tc.modify(&in.Card)
			err := in.Validate(checkoutNow)
			if tc.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.Equal(t, tc.want, fieldCodes(t, err))
		})
	}
}

func TestCard_Last4(t *testing.T) {
	assert.Equal(t, "4242", validCard().Last4())
	assert.Equal(t, "0119", Card{Number: "4000-0000-0000-0119"}.Last4())
}

func TestCheckoutInput_Fingerprint(t *testing.T) {
	a := CheckoutLine{ProductID: uuid.Must(uuid.NewV7()), Quantity: 1, ExpectedPrice: decimal.RequireFromString("10")}
	b := CheckoutLine{ProductID: uuid.Must(uuid.NewV7()), Quantity: 2, ExpectedPrice: decimal.RequireFromString("5.50")}

	base := validCheckout(a, b)
	reordered := validCheckout(b, a)
	reordered.Lines[1].ExpectedPrice = decimal.RequireFromString("10.00")
	assert.Equal(t, base.Fingerprint(), reordered.Fingerprint(), "line order and price formatting do not matter")

	otherQty := validCheckout(a, CheckoutLine{ProductID: b.ProductID, Quantity: 3, ExpectedPrice: b.ExpectedPrice})
	assert.NotEqual(t, base.Fingerprint(), otherQty.Fingerprint())

	otherCard := validCheckout(a, b)
	otherCard.Card.Number = "4000000000000002"
	assert.NotEqual(t, base.Fingerprint(), otherCard.Fingerprint())
}

func product(stock int, price string) Product {
	return Product{ID: uuid.Must(uuid.NewV7()), SKU: "SKU-1", Name: "Item", Price: decimal.RequireFromString(price), Stock: stock}
}

func TestCheckCart(t *testing.T) {
	t.Run("everything matches", func(t *testing.T) {
		p := product(5, "10.00")
		in := validCheckout(CheckoutLine{ProductID: p.ID, Quantity: 5, ExpectedPrice: decimal.RequireFromString("10")})
		assert.NoError(t, CheckCart(&in, map[uuid.UUID]Product{p.ID: p}))
	})

	t.Run("price change reports current price", func(t *testing.T) {
		p := product(5, "12.50")
		in := validCheckout(CheckoutLine{ProductID: p.ID, Quantity: 1, ExpectedPrice: decimal.RequireFromString("10.00")})

		err := CheckCart(&in, map[uuid.UUID]Product{p.ID: p})
		var cerr *CheckoutConflictError
		require.ErrorAs(t, err, &cerr)
		assert.ErrorIs(t, err, ErrConflict)
		assert.Equal(t, "price_changed", cerr.Code())
		require.Len(t, cerr.Issues, 1)
		assert.Equal(t, IssuePriceChanged, cerr.Issues[0].Code)
		assert.True(t, cerr.Issues[0].CurrentPrice.Equal(decimal.RequireFromString("12.50")))
	})

	t.Run("all issues are reported together and stock wins over price", func(t *testing.T) {
		soldOut := product(0, "1.00")
		short := product(2, "1.00")
		repriced := product(9, "2.00")
		missing := uuid.Must(uuid.NewV7())
		one := decimal.RequireFromString("1.00")
		in := validCheckout(
			CheckoutLine{ProductID: soldOut.ID, Quantity: 1, ExpectedPrice: one},
			CheckoutLine{ProductID: short.ID, Quantity: 3, ExpectedPrice: decimal.RequireFromString("0.50")},
			CheckoutLine{ProductID: repriced.ID, Quantity: 1, ExpectedPrice: one},
			CheckoutLine{ProductID: missing, Quantity: 1, ExpectedPrice: one},
		)

		err := CheckCart(&in, map[uuid.UUID]Product{soldOut.ID: soldOut, short.ID: short, repriced.ID: repriced})
		var cerr *CheckoutConflictError
		require.ErrorAs(t, err, &cerr)
		assert.Equal(t, "items_unavailable", cerr.Code())

		codes := map[uuid.UUID]CheckoutIssueCode{}
		for _, issue := range cerr.Issues {
			codes[issue.ProductID] = issue.Code
		}
		assert.Equal(t, map[uuid.UUID]CheckoutIssueCode{
			soldOut.ID:  IssueOutOfStock,
			short.ID:    IssueInsufficientStock,
			repriced.ID: IssuePriceChanged,
			missing:     IssueProductUnavailable,
		}, codes)
	})

	t.Run("reserved units are not available", func(t *testing.T) {
		p := product(5, "1.00")
		p.Reserved = 3
		in := validCheckout(CheckoutLine{ProductID: p.ID, Quantity: 3, ExpectedPrice: decimal.RequireFromString("1.00")})

		var cerr *CheckoutConflictError
		require.ErrorAs(t, CheckCart(&in, map[uuid.UUID]Product{p.ID: p}), &cerr)
		require.Len(t, cerr.Issues, 1)
		assert.Equal(t, IssueInsufficientStock, cerr.Issues[0].Code)
		assert.Equal(t, 2, cerr.Issues[0].AvailableQuantity)
	})
}

func TestNewOrder_SnapshotsProducts(t *testing.T) {
	a := product(10, "89.99")
	b := product(10, "0.10")
	customer, err := NewUser("jane@example.com", "hash", RoleUser)
	require.NoError(t, err)
	in := validCheckout(
		CheckoutLine{ProductID: a.ID, Quantity: 2, ExpectedPrice: a.Price},
		CheckoutLine{ProductID: b.ID, Quantity: 3, ExpectedPrice: b.Price},
	)

	o := NewOrder(&customer, &in, map[uuid.UUID]Product{a.ID: a, b.ID: b})

	assert.Equal(t, OrderPending, o.Status)
	assert.Equal(t, customer.ID, o.UserID)
	assert.Equal(t, "jane@example.com", o.CustomerEmail)
	assert.Equal(t, "4242", o.CardLast4)
	assert.Equal(t, in.Fingerprint(), o.RequestHash)
	require.Len(t, o.Items, 2)
	assert.Equal(t, a.SKU, o.Items[0].SKU)
	assert.True(t, o.Items[0].LineTotal.Equal(decimal.RequireFromString("179.98")))
	assert.True(t, o.Items[1].LineTotal.Equal(decimal.RequireFromString("0.30")))
	assert.True(t, o.Total.Equal(decimal.RequireFromString("180.28")))
}

func TestOrder_Transitions(t *testing.T) {
	paid := Order{Status: OrderPending}
	require.NoError(t, paid.MarkPaid("pay_1", checkoutNow))
	assert.Equal(t, OrderPaid, paid.Status)
	assert.Equal(t, "pay_1", paid.PaymentID)
	assert.ErrorIs(t, paid.MarkFailed(FailureCardDeclined), ErrOrderNotPending)
	assert.ErrorIs(t, paid.MarkPaid("pay_2", checkoutNow), ErrOrderNotPending)

	failed := Order{Status: OrderPending}
	require.NoError(t, failed.MarkFailed(FailureCardDeclined))
	assert.Equal(t, OrderFailed, failed.Status)
	assert.ErrorIs(t, failed.MarkPaid("pay_1", checkoutNow), ErrOrderNotPending)
}
