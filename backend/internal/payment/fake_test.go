package payment_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/payment"
	"e-commerce-backend/internal/usecase"
)

func charge(ctx context.Context, number string) (usecase.PaymentResult, error) {
	return payment.NewFakeProvider().Charge(ctx, usecase.PaymentRequest{
		IdempotencyKey: "order-1",
		Amount:         decimal.RequireFromString("10.00"),
		Card:           domain.Card{Number: number},
	})
}

func TestFakeProvider_IsDeterministicByCardNumber(t *testing.T) {
	ctx := context.Background()

	res, err := charge(ctx, "4242 4242 4242 4242")
	require.NoError(t, err)
	assert.Equal(t, usecase.PaymentResult{Approved: true, PaymentID: "fake_order-1"}, res)

	again, err := charge(ctx, "4242424242424242")
	require.NoError(t, err)
	assert.Equal(t, res.PaymentID, again.PaymentID, "same idempotency key gives the same payment")

	res, err = charge(ctx, "4000 0000 0000 0002")
	require.NoError(t, err)
	assert.Equal(t, usecase.PaymentResult{DeclineCode: domain.FailureCardDeclined}, res)

	res, err = charge(ctx, "4000000000009995")
	require.NoError(t, err)
	assert.Equal(t, usecase.PaymentResult{DeclineCode: "insufficient_funds"}, res)

	_, err = charge(ctx, "4000-0000-0000-0119")
	assert.ErrorIs(t, err, usecase.ErrPaymentTimeout)
}

func TestFakeProvider_RespectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := charge(ctx, "4242424242424242")
	assert.ErrorIs(t, err, context.Canceled)
}
