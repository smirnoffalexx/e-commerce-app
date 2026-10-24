package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"e-commerce-backend/internal/delivery/http/auth"
	"e-commerce-backend/internal/delivery/http/dto"
	"e-commerce-backend/internal/delivery/http/response"
	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

const (
	headerIdempotencyKey      = "Idempotency-Key"
	headerIdempotentReplayed  = "Idempotent-Replayed"
	defaultPaymentFailureText = "payment could not be completed"
)

var paymentFailureText = map[string]string{
	domain.FailureCardDeclined:   "card was declined",
	"insufficient_funds":         "card has insufficient funds",
	domain.FailurePaymentTimeout: "payment provider did not respond in time, nothing was charged; retry with a new Idempotency-Key",
	domain.FailurePaymentError:   "payment provider failed, nothing was charged; retry with a new Idempotency-Key",
}

type OrderHandler struct {
	orders OrderService
	now    func() time.Time
}

func NewOrderHandler(orders OrderService) *OrderHandler {
	return &OrderHandler{orders: orders, now: time.Now}
}

func (h *OrderHandler) Register(r gin.IRouter, authenticate gin.HandlerFunc) {
	g := r.Group("/orders", authenticate)
	g.POST("", auth.Require(domain.PermOrdersCreate), h.Checkout)

	read := auth.RequireAny(domain.PermOrdersReadOwn, domain.PermOrdersReadAll)
	g.GET("", read, h.List)
	g.GET("/:id", read, h.Get)
}

// Checkout godoc
//
//	@Summary		Place an order and pay
//	@Description	Only USER accounts can buy. The client sends the unit prices it showed to the customer; if any price changed the order is rejected with 409 price_changed and the current prices, nothing is reserved or charged. Stock is reserved atomically (rows locked in id order, UPDATE ... WHERE stock >= quantity), then the fake provider is charged outside the transaction; a declined or failed payment marks the order FAILED and returns the stock.
//	@Description	The Idempotency-Key header is required: repeating a request with the same key returns the original result with Idempotent-Replayed: true, the same key with a different body is rejected with 422, a request still in progress with 409.
//	@Description	Test cards: 4242 4242 4242 4242 succeeds, numbers ending in 0002 are declined, 9995 have insufficient funds, 0119 time out.
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			Idempotency-Key	header		string				true	"Unique key per checkout attempt, up to 255 printable ASCII characters"	example(0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b)
//	@Param			body			body		dto.CheckoutRequest	true	"Cart and card"
//	@Success		201				{object}	dto.OrderResponse	"Order paid"
//	@Failure		400				{object}	response.Problem	"Malformed JSON or missing Idempotency-Key"
//	@Failure		401				{object}	response.Problem	"Missing or invalid token"
//	@Failure		402				{object}	response.Problem	"Payment failed: card_declined, insufficient_funds, payment_timeout or payment_error; order_id points to the FAILED order"
//	@Failure		403				{object}	response.Problem	"Only USER accounts can buy"
//	@Failure		409				{object}	response.Problem	"price_changed or items_unavailable with per-item details, or checkout_in_progress"
//	@Failure		422				{object}	response.Problem	"Invalid cart or card, or idempotency_key_reused"
//	@Failure		500				{object}	response.Problem
//	@Router			/api/v1/orders [post]
func (h *OrderHandler) Checkout(c *gin.Context) {
	key := c.GetHeader(headerIdempotencyKey)
	if key == "" {
		response.Error(c, &response.RequestError{
			Status: http.StatusBadRequest,
			Code:   "idempotency_key_required",
			Detail: "Idempotency-Key header is required",
		})
		return
	}

	var req dto.CheckoutRequest
	if err := response.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	in, err := req.ToInput(key, h.now())
	if err != nil {
		response.Error(c, err)
		return
	}

	principal, _ := auth.PrincipalFrom(c)
	res, err := h.orders.Checkout(c.Request.Context(), principal.UserID, in)
	if err != nil {
		response.Error(c, err)
		return
	}

	if res.Replayed {
		c.Header(headerIdempotentReplayed, "true")
	}
	if res.Order.Status == domain.OrderFailed {
		detail, ok := paymentFailureText[res.Order.FailureCode]
		if !ok {
			detail = defaultPaymentFailureText
		}
		response.Write(c, response.Problem{
			Status:  http.StatusPaymentRequired,
			Code:    res.Order.FailureCode,
			Detail:  detail,
			OrderID: res.Order.ID.String(),
		})
		return
	}

	c.Header("Location", "/api/v1/orders/"+res.Order.ID.String())
	c.JSON(http.StatusCreated, dto.NewOrderResponse(&res.Order))
}

// List godoc
//
//	@Summary		List orders
//	@Description	USER sees only own orders, ADMIN and SUPERADMIN see all orders and may filter by user_id. Newest first.
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id		query		string	false	"Filter by customer (staff only)"	format(uuid)
//	@Param			status		query		string	false	"Filter by status"					Enums(PENDING,PAID,FAILED)
//	@Param			page		query		int		false	"Page number (1..10000)"			default(1)
//	@Param			page_size	query		int		false	"Items per page (1..100)"			default(20)
//	@Success		200			{object}	dto.OrderListResponse
//	@Failure		400			{object}	response.Problem	"Malformed query parameters"
//	@Failure		401			{object}	response.Problem	"Missing or invalid token"
//	@Failure		403			{object}	response.Problem	"USER filtering by another user"
//	@Failure		422			{object}	response.Problem	"Invalid status or paging"
//	@Failure		500			{object}	response.Problem
//	@Router			/api/v1/orders [get]
func (h *OrderHandler) List(c *gin.Context) {
	q := response.NewQueryParser(c)
	query := usecase.ListOrdersQuery{
		UserID:      q.UUID("user_id"),
		Status:      domain.OrderStatus(c.Query("status")),
		PageRequest: usecase.PageRequest{Page: q.Int("page"), PageSize: q.Int("page_size")},
	}
	if err := q.Err(); err != nil {
		response.Error(c, err)
		return
	}

	principal, _ := auth.PrincipalFrom(c)
	page, err := h.orders.List(c.Request.Context(), principal, query)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewOrderListResponse(&page))
}

// Get godoc
//
//	@Summary		Get order
//	@Description	USER can read only own orders; another customer's order is reported as not found.
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Order ID"	format(uuid)
//	@Success		200	{object}	dto.OrderResponse
//	@Failure		400	{object}	response.Problem	"Invalid ID"
//	@Failure		401	{object}	response.Problem	"Missing or invalid token"
//	@Failure		404	{object}	response.Problem	"Order not found"
//	@Failure		500	{object}	response.Problem
//	@Router			/api/v1/orders/{id} [get]
func (h *OrderHandler) Get(c *gin.Context) {
	id, err := response.PathUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	principal, _ := auth.PrincipalFrom(c)
	o, err := h.orders.Get(c.Request.Context(), principal, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewOrderResponse(&o))
}
