package response

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"e-commerce-backend/internal/delivery/http/middleware"
	"e-commerce-backend/internal/domain"
)

const (
	ContentTypeProblem = "application/problem+json"
	statusClientClosed = 499
)

type FieldProblem struct {
	Field   string `json:"field" example:"price"`
	Code    string `json:"code" example:"negative"`
	Message string `json:"message" example:"price must not be negative"`
}

type ItemProblem struct {
	ProductID         string           `json:"product_id" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	Code              string           `json:"code" example:"price_changed" enums:"product_unavailable,out_of_stock,insufficient_stock,price_changed"`
	RequestedQuantity int              `json:"requested_quantity" example:"2"`
	AvailableQuantity *int             `json:"available_quantity,omitempty" example:"1"`
	ExpectedPrice     *decimal.Decimal `json:"expected_price,omitempty" swaggertype:"string" example:"89.99"`
	CurrentPrice      *decimal.Decimal `json:"current_price,omitempty" swaggertype:"string" example:"94.99"`
}

type Problem struct {
	Type      string         `json:"type" example:"about:blank"`
	Title     string         `json:"title" example:"Validation failed"`
	Status    int            `json:"status" example:"422"`
	Code      string         `json:"code" example:"validation_failed"`
	Detail    string         `json:"detail,omitempty" example:"request contains invalid fields"`
	Instance  string         `json:"instance,omitempty" example:"/api/v1/products"`
	RequestID string         `json:"request_id,omitempty" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	Errors    []FieldProblem `json:"errors,omitempty"`
	Items     []ItemProblem  `json:"items,omitempty"`
	OrderID   string         `json:"order_id,omitempty" format:"uuid"`
}

func Error(c *gin.Context, err error) {
	p := toProblem(err)
	if p.Status >= http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "request failed",
			slog.String("request_id", middleware.GetRequestID(c)),
			slog.String("path", c.Request.URL.Path),
			slog.Any("error", err),
		)
	}
	Write(c, p)
}

func Write(c *gin.Context, p Problem) {
	p.Type = "about:blank"
	p.Instance = c.Request.URL.Path
	p.RequestID = middleware.GetRequestID(c)
	if p.Title == "" {
		p.Title = http.StatusText(p.Status)
	}

	c.Header("Content-Type", ContentTypeProblem)
	c.AbortWithStatusJSON(p.Status, p)
}

func toProblem(err error) Problem {
	var (
		verr   *domain.ValidationError
		reqErr *RequestError
		derr   *domain.Error
		cerr   *domain.CheckoutConflictError
	)

	switch {
	case errors.As(err, &verr):
		return Problem{
			Status: http.StatusUnprocessableEntity,
			Title:  "Validation failed",
			Code:   "validation_failed",
			Detail: "request contains invalid fields",
			Errors: fieldProblems(verr.Fields),
		}
	case errors.As(err, &reqErr):
		return Problem{
			Status: reqErr.Status,
			Code:   reqErr.Code,
			Detail: reqErr.Detail,
			Errors: fieldProblems(reqErr.Fields),
		}
	case errors.As(err, &cerr):
		return Problem{Status: http.StatusConflict, Code: cerr.Code(), Detail: cerr.Error(), Items: itemProblems(cerr.Issues)}
	case errors.As(err, &derr):
		return Problem{Status: statusForKind(err), Code: derr.Code(), Detail: derr.Error()}
	case errors.Is(err, context.DeadlineExceeded):
		return Problem{Status: http.StatusGatewayTimeout, Code: "timeout", Detail: "request took too long"}
	case errors.Is(err, context.Canceled):
		return Problem{Status: statusClientClosed, Title: "Client Closed Request", Code: "canceled"}
	}

	if status := statusForKind(err); status != http.StatusInternalServerError {
		return Problem{Status: status, Code: codeForStatus(status)}
	}
	return Problem{Status: http.StatusInternalServerError, Code: "internal_error", Detail: "unexpected error, please try again later"}
}

func statusForKind(err error) int {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusUnprocessableEntity:
		return "validation_failed"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	default:
		return "error"
	}
}

func fieldProblems(fields []domain.FieldError) []FieldProblem {
	if len(fields) == 0 {
		return nil
	}
	out := make([]FieldProblem, 0, len(fields))
	for _, f := range fields {
		out = append(out, FieldProblem{Field: f.Field, Code: f.Code, Message: f.Message})
	}
	return out
}

func itemProblems(issues []domain.CheckoutIssue) []ItemProblem {
	out := make([]ItemProblem, 0, len(issues))
	for _, issue := range issues {
		p := ItemProblem{
			ProductID:         issue.ProductID.String(),
			Code:              string(issue.Code),
			RequestedQuantity: issue.RequestedQuantity,
		}
		if issue.Code != domain.IssueProductUnavailable {
			available := issue.AvailableQuantity
			expected := issue.ExpectedPrice.Round(domain.PriceDecimals)
			current := issue.CurrentPrice.Round(domain.PriceDecimals)
			p.AvailableQuantity, p.ExpectedPrice, p.CurrentPrice = &available, &expected, &current
		}
		out = append(out, p)
	}
	return out
}
