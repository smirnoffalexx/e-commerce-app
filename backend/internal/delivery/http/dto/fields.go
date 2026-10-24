package dto

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"e-commerce-backend/internal/domain"
)

type fieldCollector struct {
	verr     domain.ValidationError
	reported map[string]bool
}

func newFieldCollector() *fieldCollector {
	return &fieldCollector{reported: map[string]bool{}}
}

func (f *fieldCollector) add(field, code, message string) {
	f.reported[field] = true
	f.verr.Add(field, code, message)
}

func (f *fieldCollector) uuid(field, raw string) uuid.UUID {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		f.add(field, domain.CodeInvalidFormat, field+" must be a valid UUID")
		return uuid.Nil
	}
	return id
}

func (f *fieldCollector) decimal(field string, v *Decimal) decimal.Decimal {
	switch {
	case v == nil:
		f.add(field, domain.CodeRequired, field+" is required")
	case !v.valid:
		f.add(field, domain.CodeInvalidFormat, field+" must be a decimal number")
	default:
		return v.value
	}
	return decimal.Zero
}

func (f *fieldCollector) int(field string, v *int) int {
	if v == nil {
		f.add(field, domain.CodeRequired, field+" is required")
		return 0
	}
	return *v
}

func (f *fieldCollector) merge(err error) {
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		return
	}
	for _, fe := range verr.Fields {
		if !f.reported[fe.Field] {
			f.add(fe.Field, fe.Code, fe.Message)
		}
	}
}

func (f *fieldCollector) err() error {
	return f.verr.Err()
}
