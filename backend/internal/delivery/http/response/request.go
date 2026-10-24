package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

const MaxJSONBodyBytes = 1 << 20

type RequestError struct {
	Status int
	Code   string
	Detail string
	Fields []domain.FieldError
}

func (e *RequestError) Error() string { return e.Detail }

func badRequest(code, detail string, fields ...domain.FieldError) *RequestError {
	return &RequestError{Status: http.StatusBadRequest, Code: code, Detail: detail, Fields: fields}
}

func BindJSON(c *gin.Context, dst any) error {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &RequestError{
			Status: http.StatusUnsupportedMediaType,
			Code:   "unsupported_media_type",
			Detail: "Content-Type must be application/json",
		}
	}

	dec := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, MaxJSONBodyBytes))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return badRequest("malformed_json", "request body must contain a single JSON object")
	}
	return nil
}

func decodeError(err error) *RequestError {
	var (
		syntaxErr *json.SyntaxError
		typeErr   *json.UnmarshalTypeError
		sizeErr   *http.MaxBytesError
	)

	switch {
	case errors.Is(err, io.EOF):
		return badRequest("empty_body", "request body must not be empty")
	case errors.Is(err, io.ErrUnexpectedEOF):
		return badRequest("malformed_json", "request body contains malformed JSON")
	case errors.As(err, &syntaxErr):
		return badRequest("malformed_json", fmt.Sprintf("request body contains malformed JSON at position %d", syntaxErr.Offset))
	case errors.As(err, &typeErr):
		field := typeErr.Field
		if field == "" {
			return badRequest("invalid_type", "request body must be a JSON object")
		}
		return badRequest("invalid_type", "request body contains a field of invalid type",
			domain.FieldError{Field: field, Code: "invalid_type", Message: field + " must be of type " + typeErr.Type.String()})
	case errors.As(err, &sizeErr):
		return &RequestError{
			Status: http.StatusRequestEntityTooLarge,
			Code:   "body_too_large",
			Detail: fmt.Sprintf("request body must not exceed %d bytes", sizeErr.Limit),
		}
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), `"`)
		return badRequest("unknown_field", "request body contains an unknown field",
			domain.FieldError{Field: field, Code: "unknown_field", Message: field + " is not a known field"})
	default:
		return badRequest("malformed_json", "request body could not be parsed")
	}
}

func PathUUID(c *gin.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		return uuid.Nil, badRequest("invalid_path_param", name+" must be a valid UUID",
			domain.FieldError{Field: name, Code: domain.CodeInvalidFormat, Message: name + " must be a valid UUID"})
	}
	return id, nil
}

type QueryParser struct {
	c      *gin.Context
	fields []domain.FieldError
}

func NewQueryParser(c *gin.Context) *QueryParser {
	return &QueryParser{c: c}
}

func (q *QueryParser) Int(name string) int {
	raw := strings.TrimSpace(q.c.Query(name))
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		q.fields = append(q.fields, domain.FieldError{Field: name, Code: domain.CodeInvalidFormat, Message: name + " must be an integer"})
		return 0
	}
	return v
}

func (q *QueryParser) UUID(name string) *uuid.UUID {
	raw := strings.TrimSpace(q.c.Query(name))
	if raw == "" {
		return nil
	}
	v, err := uuid.Parse(raw)
	if err != nil {
		q.fields = append(q.fields, domain.FieldError{Field: name, Code: domain.CodeInvalidFormat, Message: name + " must be a valid UUID"})
		return nil
	}
	return &v
}

func (q *QueryParser) Err() error {
	if len(q.fields) == 0 {
		return nil
	}
	return badRequest("invalid_query_param", "query contains invalid parameters", q.fields...)
}
