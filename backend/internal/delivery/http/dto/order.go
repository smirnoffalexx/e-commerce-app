package dto

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"e-commerce-backend/internal/domain"
)

type CheckoutRequest struct {
	Items   []CheckoutItemRequest `json:"items"`
	Payment CardRequest           `json:"payment"`
}

type CheckoutItemRequest struct {
	ProductID string   `json:"product_id" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	Quantity  *int     `json:"quantity" example:"2"`
	UnitPrice *Decimal `json:"unit_price" swaggertype:"string" example:"89.99"`
}

type CardRequest struct {
	CardNumber string `json:"card_number" example:"4242 4242 4242 4242"`
	ExpMonth   *int   `json:"exp_month" example:"12"`
	ExpYear    *int   `json:"exp_year" example:"2030"`
	CVC        string `json:"cvc" example:"123"`
}

func (r *CheckoutRequest) ToInput(idempotencyKey string, now time.Time) (domain.CheckoutInput, error) {
	f := newFieldCollector()
	in := domain.CheckoutInput{
		IdempotencyKey: idempotencyKey,
		Lines:          make([]domain.CheckoutLine, 0, len(r.Items)),
		Card: domain.Card{
			Number:   r.Payment.CardNumber,
			ExpMonth: f.int("payment.exp_month", r.Payment.ExpMonth),
			ExpYear:  f.int("payment.exp_year", r.Payment.ExpYear),
			CVC:      r.Payment.CVC,
		},
	}
	for i, item := range r.Items {
		field := "items[" + strconv.Itoa(i) + "]."
		in.Lines = append(in.Lines, domain.CheckoutLine{
			ProductID:     f.uuid(field+"product_id", item.ProductID),
			Quantity:      f.int(field+"quantity", item.Quantity),
			ExpectedPrice: f.decimal(field+"unit_price", item.UnitPrice),
		})
	}
	f.merge(in.Validate(now))
	return in, f.err()
}

type OrderItemResponse struct {
	ProductID uuid.UUID       `json:"product_id" swaggertype:"string" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	SKU       string          `json:"sku" example:"RS-001"`
	Name      string          `json:"name" example:"Running Shoes"`
	UnitPrice decimal.Decimal `json:"unit_price" swaggertype:"string" example:"89.99"`
	Quantity  int             `json:"quantity" example:"2"`
	LineTotal decimal.Decimal `json:"line_total" swaggertype:"string" example:"179.98"`
}

type OrderResponse struct {
	ID            uuid.UUID           `json:"id" swaggertype:"string" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	UserID        uuid.UUID           `json:"user_id" swaggertype:"string" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	CustomerEmail string              `json:"customer_email" example:"jane@example.com"`
	Status        string              `json:"status" enums:"PENDING,PAID,FAILED" example:"PAID"`
	Items         []OrderItemResponse `json:"items"`
	Total         decimal.Decimal     `json:"total" swaggertype:"string" example:"179.98"`
	CardLast4     string              `json:"card_last4" example:"4242"`
	PaymentID     string              `json:"payment_id,omitempty" example:"fake_0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	FailureCode   string              `json:"failure_code,omitempty" example:"card_declined"`
	PaidAt        *time.Time          `json:"paid_at,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

type OrderListResponse struct {
	Items      []OrderResponse `json:"items"`
	Page       int             `json:"page" example:"1"`
	PageSize   int             `json:"page_size" example:"20"`
	Total      int64           `json:"total" example:"12"`
	TotalPages int             `json:"total_pages" example:"1"`
}

func NewOrderResponse(o *domain.Order) OrderResponse {
	items := make([]OrderItemResponse, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderItemResponse{
			ProductID: it.ProductID,
			SKU:       it.SKU,
			Name:      it.Name,
			UnitPrice: it.UnitPrice.Round(domain.PriceDecimals),
			Quantity:  it.Quantity,
			LineTotal: it.LineTotal.Round(domain.PriceDecimals),
		})
	}
	return OrderResponse{
		ID:            o.ID,
		UserID:        o.UserID,
		CustomerEmail: o.CustomerEmail,
		Status:        string(o.Status),
		Items:         items,
		Total:         o.Total.Round(domain.PriceDecimals),
		CardLast4:     o.CardLast4,
		PaymentID:     o.PaymentID,
		FailureCode:   o.FailureCode,
		PaidAt:        o.PaidAt,
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
	}
}

func NewOrderListResponse(page *domain.Page[domain.Order]) OrderListResponse {
	items := make([]OrderResponse, 0, len(page.Items))
	for i := range page.Items {
		items = append(items, NewOrderResponse(&page.Items[i]))
	}
	return OrderListResponse{
		Items:      items,
		Page:       page.Page,
		PageSize:   page.PageSize,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
	}
}
