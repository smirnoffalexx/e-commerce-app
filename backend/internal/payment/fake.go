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

func (FakeProvider) Charge(ctx context.Context, req usecase.PaymentRequest) (usecase.PaymentResult, error) {
	if err := ctx.Err(); err != nil {
		return usecase.PaymentResult{}, err
	}

	number := domain.NormalizeCardNumber(req.Card.Number)
	switch {
	case strings.HasSuffix(number, timeoutSuffix):
		return usecase.PaymentResult{}, usecase.ErrPaymentTimeout
	case strings.HasSuffix(number, declinedSuffix):
		return usecase.PaymentResult{DeclineCode: domain.FailureCardDeclined}, nil
	case strings.HasSuffix(number, insufficientFundsSuffix):
		return usecase.PaymentResult{DeclineCode: "insufficient_funds"}, nil
	}
	return usecase.PaymentResult{Approved: true, PaymentID: "fake_" + req.IdempotencyKey}, nil
}

var _ usecase.PaymentProvider = FakeProvider{}
