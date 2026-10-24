//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
)

func seedSearchCatalog(t *testing.T) (electronics, sports domain.Category) {
	t.Helper()
	resetDB(t)
	categories := NewCategoryRepository(testDB)
	products := NewProductRepository(testDB)
	electronics = mustCategory(t, categories, "Electronics")
	sports = mustCategory(t, categories, "Sports")

	items := []struct {
		sku, name, description string
		category               domain.Category
		price                  string
		stock                  int
	}{
		{"BS-021", "Bluetooth Speaker", "Portable waterproof speaker, 10W, 12hr battery", electronics, "59.99", 110},
		{"BS-099", "Bluetooth Speaker", "Mini speaker, 5W output, clip-on design", electronics, "24.99", 0},
		{"SS-022", "Shower Speaker", "Bluetooth shower speaker, suction cup mount", electronics, "15.99", 280},
		{"RS-001", "Running Shoes", "Lightweight running shoes for daily training", sports, "89.99", 150},
		{"RS-050", "Running Shoes", "Budget running shoes for beginners", sports, "49.99", 200},
		{"CT-005", "Camping Tent", "4-person dome tent, waterproof 3000mm rating", sports, "199.99", 25},
	}
	for i := range items {
		it := &items[i]
		p, err := domain.NewProduct(domain.ProductInput{
			SKU: it.sku, Name: it.name, Description: it.description, CategoryID: it.category.ID,
			Price: decimal.RequireFromString(it.price), Stock: it.stock, WeightKg: decimal.RequireFromString("1"),
		})
		require.NoError(t, err)
		require.NoError(t, products.Create(context.Background(), &p))
	}
	return electronics, sports
}

func skus(products []domain.Product) []string {
	out := make([]string, 0, len(products))
	for i := range products {
		out = append(out, products[i].SKU)
	}
	return out
}

func TestProductSearch(t *testing.T) {
	electronics, _ := seedSearchCatalog(t)
	repo := NewProductRepository(testDB)
	ctx := context.Background()

	search := func(f domain.ProductFilter) ([]string, int64) {
		t.Helper()
		if f.Limit == 0 {
			f.Limit = 20
		}
		if f.Sort == "" {
			f.Sort = domain.SortRelevance
		}
		items, total, err := repo.List(ctx, f)
		require.NoError(t, err)
		return skus(items), total
	}

	t.Run("full text with stemming", func(t *testing.T) {
		got, _ := search(domain.ProductFilter{Query: "running shoe"})
		assert.ElementsMatch(t, []string{"RS-001", "RS-050"}, got)
	})

	t.Run("name matches rank above description matches", func(t *testing.T) {
		got, _ := search(domain.ProductFilter{Query: "bluetooth"})
		require.Len(t, got, 3)
		assert.Equal(t, "SS-022", got[2], "only mentions bluetooth in the description")
	})

	t.Run("exclusion and phrase operators", func(t *testing.T) {
		got, _ := search(domain.ProductFilter{Query: "speaker -portable"})
		assert.ElementsMatch(t, []string{"BS-099", "SS-022"}, got)

		got, _ = search(domain.ProductFilter{Query: `"portable waterproof"`})
		assert.Equal(t, []string{"BS-021"}, got)
	})

	t.Run("exact sku ranks first, prefix matches", func(t *testing.T) {
		got, _ := search(domain.ProductFilter{Query: "rs-050"})
		assert.Equal(t, []string{"RS-050"}, got)

		got, _ = search(domain.ProductFilter{Query: "RS-0"})
		assert.ElementsMatch(t, []string{"RS-001", "RS-050"}, got)
	})

	t.Run("typos fall back to trigram similarity per word", func(t *testing.T) {
		got, _ := search(domain.ProductFilter{Query: "blutooth"})
		assert.ElementsMatch(t, []string{"BS-021", "BS-099"}, got)

		got, _ = search(domain.ProductFilter{Query: "waterprof tent"})
		require.NotEmpty(t, got)
		assert.Equal(t, "CT-005", got[0])
	})

	t.Run("like wildcards in the query are literal", func(t *testing.T) {
		for _, q := range []string{"%", "RS_0", `\`} {
			got, total := search(domain.ProductFilter{Query: q})
			assert.Empty(t, got, q)
			assert.Zero(t, total, q)
		}
	})

	t.Run("no match", func(t *testing.T) {
		got, total := search(domain.ProductFilter{Query: "xyzzy"})
		assert.Empty(t, got)
		assert.Zero(t, total)
	})

	t.Run("filters combine with search", func(t *testing.T) {
		got, total := search(domain.ProductFilter{
			Query:      "speaker",
			CategoryID: &electronics.ID,
			MaxPrice:   ptr(decimal.RequireFromString("50")),
			InStock:    true,
			Sort:       domain.SortPriceAsc,
		})
		assert.Equal(t, []string{"SS-022"}, got, "BS-099 is out of stock, BS-021 is too expensive")
		assert.EqualValues(t, 1, total)
	})

	t.Run("sorting without query", func(t *testing.T) {
		got, _ := search(domain.ProductFilter{Sort: domain.SortPriceDesc})
		assert.Equal(t, []string{"CT-005", "RS-001", "BS-021", "RS-050", "BS-099", "SS-022"}, got)

		got, _ = search(domain.ProductFilter{Sort: domain.SortPriceAsc, MinPrice: ptr(decimal.RequireFromString("49.99")), MaxPrice: ptr(decimal.RequireFromString("89.99"))})
		assert.Equal(t, []string{"RS-050", "BS-021", "RS-001"}, got, "price bounds are inclusive")
	})

	t.Run("pagination is stable for equal ranks", func(t *testing.T) {
		first, total := search(domain.ProductFilter{Query: "speaker", Limit: 2})
		second, _ := search(domain.ProductFilter{Query: "speaker", Limit: 2, Offset: 2})
		assert.EqualValues(t, 3, total)
		assert.Len(t, append(first, second...), 3)
		assert.NotContains(t, first, second[0])
	})
}

func ptr[T any](v T) *T { return &v }
