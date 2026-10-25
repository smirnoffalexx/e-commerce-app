package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
	"e-commerce-backend/internal/usecase/mocks"
)

func productInput() domain.ProductInput {
	return domain.ProductInput{
		SKU:        "rs-001",
		Name:       " Running Shoes ",
		CategoryID: uuid.Must(uuid.NewV7()),
		Price:      decimal.RequireFromString("89.99"),
		Stock:      150,
		WeightKg:   decimal.RequireFromString("0.35"),
	}
}

func existingProduct(t *testing.T, version int) domain.Product {
	t.Helper()
	p, err := domain.NewProduct(productInput())
	require.NoError(t, err)
	p.Version = version
	return p
}

func TestProductService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("normalizes and stores valid product", func(t *testing.T) {
		repo := mocks.NewProductRepository(t)
		repo.EXPECT().Create(ctx, mock.MatchedBy(func(p *domain.Product) bool {
			return p.SKU == "RS-001" && p.Name == "Running Shoes" && p.Version == 1
		})).Return(nil)

		p, err := usecase.NewProductService(repo).Create(ctx, productInput())
		require.NoError(t, err)
		assert.Equal(t, "RS-001", p.SKU)
	})

	t.Run("invalid input never reaches repository", func(t *testing.T) {
		repo := mocks.NewProductRepository(t)
		in := productInput()
		in.Stock = -5

		_, err := usecase.NewProductService(repo).Create(ctx, in)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("propagates repository conflicts", func(t *testing.T) {
		for _, repoErr := range []error{domain.ErrSKUTaken, domain.ErrCategoryNotFound} {
			repo := mocks.NewProductRepository(t)
			repo.EXPECT().Create(ctx, mock.Anything).Return(repoErr)

			_, err := usecase.NewProductService(repo).Create(ctx, productInput())
			assert.ErrorIs(t, err, repoErr)
		}
	})
}

func TestProductService_List(t *testing.T) {
	ctx := context.Background()
	categoryID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name       string
		query      domain.ListProductsQuery
		wantLimit  int
		wantOffset int
		wantPage   int
	}{
		{"defaults", domain.ListProductsQuery{}, domain.DefaultPageSize, 0, 1},
		{"explicit page", domain.ListProductsQuery{PageRequest: domain.PageRequest{Page: 3, PageSize: 10}}, 10, 20, 3},
		{"max page size", domain.ListProductsQuery{PageRequest: domain.PageRequest{PageSize: domain.MaxPageSize}}, domain.MaxPageSize, 0, 1},
		{"with category", domain.ListProductsQuery{CategoryID: &categoryID}, domain.DefaultPageSize, 0, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewProductRepository(t)
			repo.EXPECT().List(ctx, domain.ProductFilter{
				CategoryID: tc.query.CategoryID,
				Limit:      tc.wantLimit,
				Offset:     tc.wantOffset,
				Sort:       domain.SortNewest,
			}).Return([]domain.Product{}, 45, nil)

			page, err := usecase.NewProductService(repo).List(ctx, tc.query)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPage, page.Page)
			assert.Equal(t, tc.wantLimit, page.PageSize)
			assert.EqualValues(t, 45, page.Total)
		})
	}
}

func TestProductService_List_RejectsInvalidPaging(t *testing.T) {
	cases := map[string]domain.PageRequest{
		"negative page":      {Page: -1},
		"page too large":     {Page: domain.MaxPage + 1},
		"negative page size": {PageSize: -10},
		"page size too big":  {PageSize: domain.MaxPageSize + 1},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			repo := mocks.NewProductRepository(t)

			_, err := usecase.NewProductService(repo).List(context.Background(), domain.ListProductsQuery{PageRequest: req})
			assert.ErrorIs(t, err, domain.ErrValidation)
		})
	}
}

func TestPage_TotalPages(t *testing.T) {
	assert.Equal(t, 0, domain.Page[int]{Total: 0, PageSize: 20}.TotalPages())
	assert.Equal(t, 1, domain.Page[int]{Total: 20, PageSize: 20}.TotalPages())
	assert.Equal(t, 3, domain.Page[int]{Total: 41, PageSize: 20}.TotalPages())
}

func TestProductService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("applies changes with expected version", func(t *testing.T) {
		current := existingProduct(t, 2)
		repo := mocks.NewProductRepository(t)
		repo.EXPECT().GetByID(ctx, current.ID).Return(current, nil)
		repo.EXPECT().Update(ctx, mock.MatchedBy(func(p *domain.Product) bool {
			return p.ID == current.ID && p.Version == 2 && p.Stock == 7
		})).Run(func(_ context.Context, p *domain.Product) { p.Version++ }).Return(nil)

		in := productInput()
		in.Stock = 7
		p, err := usecase.NewProductService(repo).Update(ctx, current.ID, 2, in)
		require.NoError(t, err)
		assert.Equal(t, 3, p.Version)
		assert.Equal(t, 7, p.Stock)
	})

	t.Run("missing version is a validation error", func(t *testing.T) {
		repo := mocks.NewProductRepository(t)

		_, err := usecase.NewProductService(repo).Update(ctx, uuid.Must(uuid.NewV7()), 0, productInput())
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("stale version is rejected before writing", func(t *testing.T) {
		current := existingProduct(t, 5)
		repo := mocks.NewProductRepository(t)
		repo.EXPECT().GetByID(ctx, current.ID).Return(current, nil)

		_, err := usecase.NewProductService(repo).Update(ctx, current.ID, 4, productInput())
		assert.ErrorIs(t, err, domain.ErrProductVersionStale)
	})

	t.Run("concurrent change between read and write is reported as stale", func(t *testing.T) {
		current := existingProduct(t, 1)
		repo := mocks.NewProductRepository(t)
		repo.EXPECT().GetByID(ctx, current.ID).Return(current, nil)
		repo.EXPECT().Update(ctx, mock.Anything).Return(domain.ErrProductVersionStale)

		_, err := usecase.NewProductService(repo).Update(ctx, current.ID, 1, productInput())
		assert.ErrorIs(t, err, domain.ErrProductVersionStale)
	})

	t.Run("invalid changes never reach repository", func(t *testing.T) {
		current := existingProduct(t, 1)
		repo := mocks.NewProductRepository(t)
		repo.EXPECT().GetByID(ctx, current.ID).Return(current, nil)

		in := productInput()
		in.Price = decimal.RequireFromString("-1")
		_, err := usecase.NewProductService(repo).Update(ctx, current.ID, 1, in)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("missing product", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		repo := mocks.NewProductRepository(t)
		repo.EXPECT().GetByID(ctx, id).Return(domain.Product{}, domain.ErrProductNotFound)

		_, err := usecase.NewProductService(repo).Update(ctx, id, 1, productInput())
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}
