package domain

import "github.com/shopspring/decimal"

type PaymentRequest struct {
	IdempotencyKey string
	Amount         decimal.Decimal
	Card           Card
}

type PaymentResult struct {
	Approved    bool
	PaymentID   string
	DeclineCode string
}
