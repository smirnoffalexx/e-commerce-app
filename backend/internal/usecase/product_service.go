package usecase

import (
	"context"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

type ProductService struct {
	products ProductRepository
}

func NewProductService(products ProductRepository) *ProductService {
	return &ProductService{products: products}
}

func (s *ProductService) Create(ctx context.Context, in domain.ProductInput) (domain.Product, error) {
	p, err := domain.NewProduct(in)
	if err != nil {
		return domain.Product{}, err
	}
	if err := s.products.Create(ctx, &p); err != nil {
		return domain.Product{}, err
	}
	return p, nil
}

func (s *ProductService) Get(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return s.products.GetByID(ctx, id)
}

func (s *ProductService) List(ctx context.Context, q domain.ListProductsQuery) (domain.Page[domain.Product], error) {
	filter, page, err := q.Filter()
	if err != nil {
		return domain.Page[domain.Product]{}, err
	}

	items, total, err := s.products.List(ctx, filter)
	if err != nil {
		return domain.Page[domain.Product]{}, err
	}

	return domain.Page[domain.Product]{Items: items, Total: total, Page: page.Page, PageSize: page.PageSize}, nil
}

func (s *ProductService) Update(ctx context.Context, id uuid.UUID, version int, in domain.ProductInput) (domain.Product, error) {
	if version < 1 {
		verr := &domain.ValidationError{}
		verr.Add("version", domain.CodeRequired, "version is required for updates")
		return domain.Product{}, verr
	}

	p, err := s.products.GetByID(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	if p.Version != version {
		return domain.Product{}, domain.ErrProductVersionStale
	}

	if err := p.Update(in); err != nil {
		return domain.Product{}, err
	}
	if err := s.products.Update(ctx, &p); err != nil {
		return domain.Product{}, err
	}
	return p, nil
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.products.Delete(ctx, id)
}
