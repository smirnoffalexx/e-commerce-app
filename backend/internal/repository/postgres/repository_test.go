//go:build integration

package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
)

func mustCategory(t *testing.T, repo *CategoryRepository, name string) domain.Category {
	t.Helper()
	c, err := domain.NewCategory(name)
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), &c))
	return c
}

func mustProduct(t *testing.T, repo *ProductRepository, sku string, categoryID uuid.UUID) domain.Product {
	t.Helper()
	p, err := domain.NewProduct(domain.ProductInput{
		SKU:        sku,
		Name:       "Product " + sku,
		CategoryID: categoryID,
		Price:      decimal.RequireFromString("19.99"),
		Stock:      10,
		WeightKg:   decimal.RequireFromString("0.5"),
	})
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), &p))
	return p
}

func TestCategoryRepository(t *testing.T) {
	ctx := context.Background()
	categories := NewCategoryRepository(testDB)
	products := NewProductRepository(testDB)

	t.Run("slug collision is a conflict", func(t *testing.T) {
		resetDB(t)
		mustCategory(t, categories, "Food & Beverage")

		dup, err := domain.NewCategory("food and beverage")
		require.NoError(t, err)
		assert.ErrorIs(t, categories.Create(ctx, &dup), domain.ErrCategoryExists)
	})

	t.Run("rename into existing slug is a conflict", func(t *testing.T) {
		resetDB(t)
		mustCategory(t, categories, "Games")
		books := mustCategory(t, categories, "Books")

		require.NoError(t, books.Rename("GAMES"))
		assert.ErrorIs(t, categories.Update(ctx, &books), domain.ErrCategoryExists)
	})

	t.Run("cannot delete category with active products", func(t *testing.T) {
		resetDB(t)
		c := mustCategory(t, categories, "Sports")
		p := mustProduct(t, products, "YM-015", c.ID)

		assert.ErrorIs(t, categories.Delete(ctx, c.ID), domain.ErrCategoryInUse)

		require.NoError(t, products.Delete(ctx, p.ID))
		require.NoError(t, categories.Delete(ctx, c.ID), "soft-deleted products must not block deletion")

		_, err := categories.GetByID(ctx, c.ID)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
		assert.ErrorIs(t, categories.Delete(ctx, c.ID), domain.ErrCategoryNotFound)
	})

	t.Run("deleted slug can be reused", func(t *testing.T) {
		resetDB(t)
		c := mustCategory(t, categories, "Misc")
		require.NoError(t, categories.Delete(ctx, c.ID))

		mustCategory(t, categories, "Misc")
	})

	t.Run("get by slugs returns only existing", func(t *testing.T) {
		resetDB(t)
		mustCategory(t, categories, "Electronics")
		mustCategory(t, categories, "Books")

		found, err := categories.GetBySlugs(ctx, []string{"electronics", "books", "unknown"})
		require.NoError(t, err)
		assert.Len(t, found, 2)
		assert.Equal(t, "Electronics", found["electronics"].Name)
	})
}

func TestProductRepository(t *testing.T) {
	ctx := context.Background()
	categories := NewCategoryRepository(testDB)
	products := NewProductRepository(testDB)

	t.Run("create and get with category", func(t *testing.T) {
		resetDB(t)
		c := mustCategory(t, categories, "Footwear")
		p := mustProduct(t, products, "RS-001", c.ID)

		got, err := products.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, "RS-001", got.SKU)
		assert.True(t, got.Price.Equal(decimal.RequireFromString("19.99")))
		assert.Equal(t, 1, got.Version)
		require.NotNil(t, got.Category)
		assert.Equal(t, "Footwear", got.Category.Name)
		assert.False(t, got.CreatedAt.IsZero())
	})

	t.Run("duplicate sku is a conflict", func(t *testing.T) {
		resetDB(t)
		c := mustCategory(t, categories, "Electronics")
		mustProduct(t, products, "BS-021", c.ID)

		dup, err := domain.NewProduct(domain.ProductInput{SKU: "bs-021", Name: "Dup", CategoryID: c.ID})
		require.NoError(t, err)
		assert.ErrorIs(t, products.Create(ctx, &dup), domain.ErrSKUTaken)
	})

	t.Run("sku of deleted product can be reused", func(t *testing.T) {
		resetDB(t)
		c := mustCategory(t, categories, "Electronics")
		p := mustProduct(t, products, "BS-021", c.ID)
		require.NoError(t, products.Delete(ctx, p.ID))

		mustProduct(t, products, "BS-021", c.ID)
	})

	t.Run("unknown or deleted category is rejected", func(t *testing.T) {
		resetDB(t)
		deleted := mustCategory(t, categories, "Old")
		require.NoError(t, categories.Delete(ctx, deleted.ID))

		for _, id := range []uuid.UUID{uuid.Must(uuid.NewV7()), deleted.ID} {
			p, err := domain.NewProduct(domain.ProductInput{SKU: "X-1", Name: "X", CategoryID: id})
			require.NoError(t, err)
			assert.ErrorIs(t, products.Create(ctx, &p), domain.ErrCategoryNotFound)
		}
	})

	t.Run("database constraints back up domain validation", func(t *testing.T) {
		resetDB(t)
		c := mustCategory(t, categories, "Misc")
		p := mustProduct(t, products, "DL-007", c.ID)

		p.Stock = -5
		err := products.Update(ctx, &p)
		require.Error(t, err)
		assert.False(t, errors.Is(err, domain.ErrConflict))
	})

	t.Run("update bumps version and rejects stale version", func(t *testing.T) {
		resetDB(t)
		c := mustCategory(t, categories, "Home & Office")
		p := mustProduct(t, products, "SD-004", c.ID)
		stale := p

		p.Price = decimal.RequireFromString("399.99")
		require.NoError(t, products.Update(ctx, &p))
		assert.Equal(t, 2, p.Version)

		stale.Stock = 1
		assert.ErrorIs(t, products.Update(ctx, &stale), domain.ErrProductVersionStale)

		got, err := products.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.True(t, got.Price.Equal(decimal.RequireFromString("399.99")))
		assert.Equal(t, 10, got.Stock)
	})

	t.Run("update or delete of missing product is not found", func(t *testing.T) {
		resetDB(t)
		c := mustCategory(t, categories, "Misc")
		p := mustProduct(t, products, "WS-001", c.ID)
		require.NoError(t, products.Delete(ctx, p.ID))

		assert.ErrorIs(t, products.Update(ctx, &p), domain.ErrProductNotFound)
		assert.ErrorIs(t, products.Delete(ctx, p.ID), domain.ErrProductNotFound)
		_, err := products.GetByID(ctx, p.ID)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("list paginates and filters by category", func(t *testing.T) {
		resetDB(t)
		sports := mustCategory(t, categories, "Sports")
		books := mustCategory(t, categories, "Books")
		for _, sku := range []string{"S-1", "S-2", "S-3"} {
			mustProduct(t, products, sku, sports.ID)
		}
		mustProduct(t, products, "B-1", books.ID)

		page, total, err := products.List(ctx, ProductFilter{Limit: 2})
		require.NoError(t, err)
		assert.EqualValues(t, 4, total)
		require.Len(t, page, 2)
		assert.Equal(t, "B-1", page[0].SKU, "newest first")

		page, total, err = products.List(ctx, ProductFilter{CategoryID: &sports.ID, Limit: 10, Offset: 1})
		require.NoError(t, err)
		assert.EqualValues(t, 3, total)
		assert.Len(t, page, 2)
	})
}

func TestProductRepository_ConcurrentUpdatesWithSameVersion(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	categories := NewCategoryRepository(testDB)
	products := NewProductRepository(testDB)

	c := mustCategory(t, categories, "Sports")
	base := mustProduct(t, products, "JR-022", c.ID)

	const writers = 10
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		stale     int
	)
	for i := range writers {
		wg.Go(func() {
			p := base
			p.Stock = 100 + i
			err := products.Update(ctx, &p)

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, domain.ErrProductVersionStale):
				stale++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
	wg.Wait()

	assert.Equal(t, 1, succeeded)
	assert.Equal(t, writers-1, stale)

	got, err := products.GetByID(ctx, base.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.Version)
}

func TestCategoryDeleteRacesWithProductCreate(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	categories := NewCategoryRepository(testDB)
	products := NewProductRepository(testDB)

	for i := range 30 {
		c := mustCategory(t, categories, "Race "+uuid.NewString()[:8])
		p, err := domain.NewProduct(domain.ProductInput{SKU: "RACE-" + decimal.NewFromInt(int64(i)).String(), Name: "Race", CategoryID: c.ID})
		require.NoError(t, err)

		var wg sync.WaitGroup
		var deleteErr, createErr error
		wg.Go(func() { deleteErr = categories.Delete(ctx, c.ID) })
		wg.Go(func() { createErr = products.Create(ctx, &p) })
		wg.Wait()

		switch {
		case deleteErr == nil:
			assert.ErrorIs(t, createErr, domain.ErrCategoryNotFound)
		case createErr == nil:
			assert.ErrorIs(t, deleteErr, domain.ErrCategoryInUse)
		default:
			t.Fatalf("both failed: delete=%v create=%v", deleteErr, createErr)
		}
	}

	var orphans int64
	err := testDB.Raw(`SELECT count(*) FROM products p JOIN categories c ON c.id = p.category_id
		WHERE p.deleted_at IS NULL AND c.deleted_at IS NOT NULL`).Scan(&orphans).Error
	require.NoError(t, err)
	assert.Zero(t, orphans, "no active product may reference a deleted category")
}
