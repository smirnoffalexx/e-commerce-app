package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/delivery/http/dto"
	"e-commerce-backend/internal/domain"
)

const checkoutProductID = "0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"

var validCheckout = `{"items":[{"product_id":"` + checkoutProductID + `","quantity":2,"unit_price":"89.99"}],` +
	`"payment":{"card_number":"4242 4242 4242 4242","exp_month":12,"exp_year":` + strconv.Itoa(time.Now().Year()+2) + `,"cvc":"123"}}`

var customerID = uuid.MustParse("0192f6b8-0000-7000-8000-000000000003")

func (a *testAPI) checkout(body string) *httptest.ResponseRecorder {
	return a.do(http.MethodPost, "/api/v1/orders", body, "Authorization", "Bearer user", "Idempotency-Key", "checkout-1")
}

func TestCheckout_OnlyCustomersCanBuy(t *testing.T) {
	a := newTestAPI(t)
	for _, token := range []string{"admin", "superadmin"} {
		w := a.do(http.MethodPost, "/api/v1/orders", validCheckout, "Authorization", "Bearer "+token, "Idempotency-Key", "k")
		assert.Equal(t, http.StatusForbidden, w.Code, token)
	}
	w := a.do(http.MethodPost, "/api/v1/orders", validCheckout, "Authorization", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCheckout_RejectsBadRequestsWithoutCallingService(t *testing.T) {
	a := newTestAPI(t)

	w := a.do(http.MethodPost, "/api/v1/orders", validCheckout, "Authorization", "Bearer user")
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "idempotency_key_required", decodeProblem(t, w).Code)

	w = a.do(http.MethodPost, "/api/v1/orders",
		`{"items":[{"product_id":"nope","quantity":0}],"payment":{"card_number":"4242424242424241","exp_month":1,"exp_year":2020,"cvc":"1"}}`,
		"Authorization", "Bearer user", "Idempotency-Key", "k")
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, map[string]string{
		"items[0].product_id": domain.CodeInvalidFormat,
		"items[0].quantity":   domain.CodeInvalidFormat,
		"items[0].unit_price": domain.CodeRequired,
		"payment.card_number": domain.CodeInvalidFormat,
		"payment.exp_year":    domain.CodeInvalidFormat,
		"payment.cvc":         domain.CodeInvalidFormat,
	}, fieldCodes(decodeProblem(t, w)))
}

func TestCheckout_PaidOrderIsCreated(t *testing.T) {
	a := newTestAPI(t)
	order := domain.Order{ID: uuid.Must(uuid.NewV7()), UserID: customerID, Status: domain.OrderPaid, Total: decimal.RequireFromString("179.98"), PaymentID: "fake_1"}
	a.orders.EXPECT().Checkout(mock.Anything, customerID, mock.MatchedBy(func(in domain.CheckoutInput) bool {
		return in.IdempotencyKey == "checkout-1" &&
			len(in.Lines) == 1 && in.Lines[0].Quantity == 2 &&
			in.Lines[0].ExpectedPrice.Equal(decimal.RequireFromString("89.99"))
	})).Return(domain.CheckoutResult{Order: order}, nil)

	w := a.checkout(validCheckout)
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "/api/v1/orders/"+order.ID.String(), w.Header().Get("Location"))
	assert.Empty(t, w.Header().Get("Idempotent-Replayed"))

	var body dto.OrderResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "PAID", body.Status)
	assert.Equal(t, "179.98", body.Total.StringFixed(2))
}

func TestCheckout_ReplayIsMarked(t *testing.T) {
	a := newTestAPI(t)
	order := domain.Order{ID: uuid.Must(uuid.NewV7()), Status: domain.OrderPaid}
	a.orders.EXPECT().Checkout(mock.Anything, customerID, mock.Anything).Return(domain.CheckoutResult{Order: order, Replayed: true}, nil)

	w := a.checkout(validCheckout)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "true", w.Header().Get("Idempotent-Replayed"))
}

func TestCheckout_FailedPaymentIs402WithOrderID(t *testing.T) {
	a := newTestAPI(t)
	order := domain.Order{ID: uuid.Must(uuid.NewV7()), Status: domain.OrderFailed, FailureCode: domain.FailureCardDeclined}
	a.orders.EXPECT().Checkout(mock.Anything, customerID, mock.Anything).Return(domain.CheckoutResult{Order: order}, nil)

	w := a.checkout(validCheckout)
	require.Equal(t, http.StatusPaymentRequired, w.Code)
	p := decodeProblem(t, w)
	assert.Equal(t, domain.FailureCardDeclined, p.Code)
	assert.Equal(t, order.ID.String(), p.OrderID)
}

func TestCheckout_PriceChangeReturnsCurrentPrices(t *testing.T) {
	a := newTestAPI(t)
	productID := uuid.MustParse(checkoutProductID)
	a.orders.EXPECT().Checkout(mock.Anything, customerID, mock.Anything).Return(domain.CheckoutResult{}, &domain.CheckoutConflictError{
		Issues: []domain.CheckoutIssue{{
			ProductID:         productID,
			Code:              domain.IssuePriceChanged,
			RequestedQuantity: 2,
			AvailableQuantity: 5,
			ExpectedPrice:     decimal.RequireFromString("89.99"),
			CurrentPrice:      decimal.RequireFromString("94.5"),
		}},
	})

	w := a.checkout(validCheckout)
	require.Equal(t, http.StatusConflict, w.Code)
	p := decodeProblem(t, w)
	assert.Equal(t, "price_changed", p.Code)
	require.Len(t, p.Items, 1)
	assert.Equal(t, checkoutProductID, p.Items[0].ProductID)
	assert.Equal(t, "94.50", p.Items[0].CurrentPrice.StringFixed(2))
	assert.Equal(t, 5, *p.Items[0].AvailableQuantity)
}

func TestCheckout_IdempotencyErrors(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
	}{
		{domain.ErrIdempotencyKeyReused, http.StatusUnprocessableEntity},
		{domain.ErrCheckoutInProgress, http.StatusConflict},
	}
	for _, tc := range cases {
		a := newTestAPI(t)
		a.orders.EXPECT().Checkout(mock.Anything, customerID, mock.Anything).Return(domain.CheckoutResult{}, tc.err)

		w := a.checkout(validCheckout)
		assert.Equal(t, tc.wantStatus, w.Code)
	}
}

func TestOrders_ReadPassesViewerToService(t *testing.T) {
	a := newTestAPI(t)
	other := uuid.Must(uuid.NewV7())
	a.orders.EXPECT().List(mock.Anything, mock.MatchedBy(func(p domain.Principal) bool { return p.Role == domain.RoleAdmin }),
		domain.ListOrdersQuery{UserID: &other, Status: domain.OrderPaid, PageRequest: domain.PageRequest{Page: 2}}).
		Return(domain.Page[domain.Order]{Page: 2, PageSize: 20}, nil)

	w := a.do(http.MethodGet, "/api/v1/orders?user_id="+other.String()+"&status=PAID&page=2", "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	orderID := uuid.Must(uuid.NewV7())
	a.orders.EXPECT().Get(mock.Anything, domain.Principal{UserID: customerID, Role: domain.RoleUser}, orderID).
		Return(domain.Order{}, domain.ErrOrderNotFound)
	w = a.do(http.MethodGet, "/api/v1/orders/"+orderID.String(), "", "Authorization", "Bearer user")
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = a.do(http.MethodGet, "/api/v1/orders?user_id=bad", "", "Authorization", "Bearer user")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
