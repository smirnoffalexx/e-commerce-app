package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
	"e-commerce-backend/internal/usecase/mocks"
)

func dec(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}

func TestProductService_Search_BuildsFilter(t *testing.T) {
	ctx := context.Background()
	categoryID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name  string
		query usecase.ListProductsQuery
		want  domain.ProductFilter
	}{
		{
			name:  "no query defaults to newest",
			query: usecase.ListProductsQuery{},
			want:  domain.ProductFilter{Sort: domain.SortNewest, Limit: 20},
		},
		{
			name:  "query defaults to relevance and collapses whitespace",
			query: usecase.ListProductsQuery{Query: "  bluetooth \t  speaker "},
			want:  domain.ProductFilter{Query: "bluetooth speaker", Sort: domain.SortRelevance, Limit: 20},
		},
		{
			name:  "whitespace-only query is no query",
			query: usecase.ListProductsQuery{Query: "   "},
			want:  domain.ProductFilter{Sort: domain.SortNewest, Limit: 20},
		},
		{
			name: "explicit sort and all filters",
			query: usecase.ListProductsQuery{
				Query:       "shoes",
				CategoryID:  &categoryID,
				MinPrice:    dec("10"),
				MaxPrice:    dec("100"),
				InStock:     true,
				Sort:        domain.SortPriceAsc,
				PageRequest: usecase.PageRequest{Page: 2, PageSize: 10},
			},
			want: domain.ProductFilter{
				Query: "shoes", CategoryID: &categoryID, MinPrice: dec("10"), MaxPrice: dec("100"),
				InStock: true, Sort: domain.SortPriceAsc, Limit: 10, Offset: 10,
			},
		},
		{
			name:  "equal price bounds are allowed",
			query: usecase.ListProductsQuery{MinPrice: dec("19.99"), MaxPrice: dec("19.99")},
			want:  domain.ProductFilter{MinPrice: dec("19.99"), MaxPrice: dec("19.99"), Sort: domain.SortNewest, Limit: 20},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewProductRepository(t)
			repo.EXPECT().List(ctx, tc.want).Return([]domain.Product{}, 0, nil)

			_, err := usecase.NewProductService(repo).List(ctx, tc.query)
			require.NoError(t, err)
		})
	}
}

func TestProductService_Search_RejectsInvalidParams(t *testing.T) {
	cases := []struct {
		name   string
		query  usecase.ListProductsQuery
		fields []string
	}{
		{"query too long", usecase.ListProductsQuery{Query: strings.Repeat("a", domain.MaxSearchQueryLen+1)}, []string{"q"}},
		{"control characters", usecase.ListProductsQuery{Query: "shoes\x00"}, []string{"q"}},
		{"unknown sort", usecase.ListProductsQuery{Sort: "popularity"}, []string{"sort"}},
		{"relevance without query", usecase.ListProductsQuery{Sort: domain.SortRelevance}, []string{"sort"}},
		{"negative prices", usecase.ListProductsQuery{MinPrice: dec("-1"), MaxPrice: dec("-2")}, []string{"min_price", "max_price", "min_price"}},
		{"inverted price range", usecase.ListProductsQuery{MinPrice: dec("50"), MaxPrice: dec("10")}, []string{"min_price"}},
		{
			"paging and search errors are reported together",
			usecase.ListProductsQuery{Sort: "x", PageRequest: usecase.PageRequest{PageSize: 1000}},
			[]string{"page_size", "sort"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewProductRepository(t)

			_, err := usecase.NewProductService(repo).List(context.Background(), tc.query)

			var verr *domain.ValidationError
			require.ErrorAs(t, err, &verr)
			fields := make([]string, 0, len(verr.Fields))
			for _, f := range verr.Fields {
				fields = append(fields, f.Field)
			}
			assert.Equal(t, tc.fields, fields)
		})
	}
}
