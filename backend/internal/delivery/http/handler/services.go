package handler

import (
	"context"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

type ProductService interface {
	Create(ctx context.Context, in domain.ProductInput) (domain.Product, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Product, error)
	List(ctx context.Context, q usecase.ListProductsQuery) (usecase.Page[domain.Product], error)
	Update(ctx context.Context, id uuid.UUID, version int, in domain.ProductInput) (domain.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type CategoryService interface {
	Create(ctx context.Context, name string) (domain.Category, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Category, error)
	List(ctx context.Context) ([]domain.Category, error)
	Rename(ctx context.Context, id uuid.UUID, name string) (domain.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

var (
	_ ProductService  = (*usecase.ProductService)(nil)
	_ CategoryService = (*usecase.CategoryService)(nil)
)
