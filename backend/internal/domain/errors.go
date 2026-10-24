package domain

import (
	"errors"
	"strings"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation failed")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type Error struct {
	kind    error
	code    string
	message string
}

func NewError(kind error, code, message string) *Error {
	return &Error{kind: kind, code: code, message: message}
}

func (e *Error) Error() string { return e.message }
func (e *Error) Unwrap() error { return e.kind }
func (e *Error) Code() string  { return e.code }

const (
	CodeRequired        = "required"
	CodeTooLong         = "too_long"
	CodeInvalidFormat   = "invalid_format"
	CodeNegative        = "negative"
	CodeTooLarge        = "too_large"
	CodeTooManyDecimals = "too_many_decimals"
)

type FieldError struct {
	Field   string
	Code    string
	Message string
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, f.Field+": "+f.Message)
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

func (e *ValidationError) Unwrap() error { return ErrValidation }

func (e *ValidationError) Add(field, code, message string) {
	e.Fields = append(e.Fields, FieldError{Field: field, Code: code, Message: message})
}

func (e *ValidationError) Err() error {
	if len(e.Fields) == 0 {
		return nil
	}
	return e
}
