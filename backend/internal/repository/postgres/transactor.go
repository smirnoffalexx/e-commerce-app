package postgres

import (
	"context"

	"gorm.io/gorm"

	"e-commerce-backend/internal/usecase"
)

type txKey struct{}

type Transactor struct {
	db *gorm.DB
}

func NewTransactor(db *gorm.DB) *Transactor {
	return &Transactor{db: db}
}

func (t *Transactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

var _ usecase.Transactor = (*Transactor)(nil)
