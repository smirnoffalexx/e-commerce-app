package usecase

import (
	"context"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

type CategoryRepository interface {
	Create(ctx context.Context, c *domain.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Category, error)
	GetBySlugs(ctx context.Context, slugs []string) (map[string]domain.Category, error)
	List(ctx context.Context) ([]domain.Category, error)
	Update(ctx context.Context, c *domain.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
	List(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int64, error)
	Update(ctx context.Context, p *domain.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
}
