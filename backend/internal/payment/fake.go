package payment

import (
	"context"
	"strings"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

const (
	declinedSuffix          = "0002"
	insufficientFundsSuffix = "9995"
	timeoutSuffix           = "0119"
)

type FakeProvider struct{}

func NewFakeProvider() *FakeProvider {
	return &FakeProvider{}
}

func (FakeProvider) Charge(ctx context.Context, req domain.PaymentRequest) (domain.PaymentResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.PaymentResult{}, err
	}

	number := domain.NormalizeCardNumber(req.Card.Number)
	switch {
	case strings.HasSuffix(number, timeoutSuffix):
		return domain.PaymentResult{}, usecase.ErrPaymentTimeout
	case strings.HasSuffix(number, declinedSuffix):
		return domain.PaymentResult{DeclineCode: domain.FailureCardDeclined}, nil
	case strings.HasSuffix(number, insufficientFundsSuffix):
		return domain.PaymentResult{DeclineCode: "insufficient_funds"}, nil
	}
	return domain.PaymentResult{Approved: true, PaymentID: "fake_" + req.IdempotencyKey}, nil
}

var _ usecase.PaymentProvider = FakeProvider{}
