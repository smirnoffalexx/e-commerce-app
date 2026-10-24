package usecase

import (
	"context"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

type CategoryService struct {
	categories CategoryRepository
}

func NewCategoryService(categories CategoryRepository) *CategoryService {
	return &CategoryService{categories: categories}
}

func (s *CategoryService) Create(ctx context.Context, name string) (domain.Category, error) {
	c, err := domain.NewCategory(name)
	if err != nil {
		return domain.Category{}, err
	}
	if err := s.categories.Create(ctx, &c); err != nil {
		return domain.Category{}, err
	}
	return c, nil
}

func (s *CategoryService) Get(ctx context.Context, id uuid.UUID) (domain.Category, error) {
	return s.categories.GetByID(ctx, id)
}

func (s *CategoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.categories.List(ctx)
}

func (s *CategoryService) Rename(ctx context.Context, id uuid.UUID, name string) (domain.Category, error) {
	c, err := s.categories.GetByID(ctx, id)
	if err != nil {
		return domain.Category{}, err
	}
	if err := c.Rename(name); err != nil {
		return domain.Category{}, err
	}
	if err := s.categories.Update(ctx, &c); err != nil {
		return domain.Category{}, err
	}
	return c, nil
}

func (s *CategoryService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.categories.Delete(ctx, id)
}
