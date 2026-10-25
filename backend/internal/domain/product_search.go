package domain

import (
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const MaxSearchQueryLen = 200

type ProductSort string

const (
	SortRelevance ProductSort = "relevance"
	SortNewest    ProductSort = "newest"
	SortPriceAsc  ProductSort = "price_asc"
	SortPriceDesc ProductSort = "price_desc"
	SortNameAsc   ProductSort = "name_asc"
)

func (s ProductSort) Valid() bool {
	switch s {
	case SortRelevance, SortNewest, SortPriceAsc, SortPriceDesc, SortNameAsc:
		return true
	default:
		return false
	}
}

type ProductFilter struct {
	Query      string
	CategoryID *uuid.UUID
	MinPrice   *decimal.Decimal
	MaxPrice   *decimal.Decimal
	InStock    bool
	Sort       ProductSort
	Limit      int
	Offset     int
}

type ListProductsQuery struct {
	Query      string
	CategoryID *uuid.UUID
	MinPrice   *decimal.Decimal
	MaxPrice   *decimal.Decimal
	InStock    bool
	Sort       ProductSort
	PageRequest
}

func (q ListProductsQuery) Filter() (ProductFilter, PageRequest, error) {
	var verr ValidationError

	page, err := q.Normalize()
	verr.Merge(err)

	text := strings.Join(strings.Fields(q.Query), " ")
	switch {
	case utf8.RuneCountInString(text) > MaxSearchQueryLen:
		verr.Add("q", CodeTooLong, "q must be at most 200 characters")
	case strings.ContainsFunc(text, func(r rune) bool { return r < 0x20 || r == 0x7f }):
		verr.Add("q", CodeInvalidFormat, "q must not contain control characters")
	}

	sort := q.Sort
	if sort == "" {
		sort = SortNewest
		if text != "" {
			sort = SortRelevance
		}
	}
	switch {
	case !sort.Valid():
		verr.Add("sort", CodeInvalidFormat, "sort must be one of relevance, newest, price_asc, price_desc, name_asc")
	case sort == SortRelevance && text == "":
		verr.Add("sort", CodeInvalidFormat, "sort=relevance requires q")
	}

	if q.MinPrice != nil && q.MinPrice.IsNegative() {
		verr.Add("min_price", CodeNegative, "min_price must not be negative")
	}
	if q.MaxPrice != nil && q.MaxPrice.IsNegative() {
		verr.Add("max_price", CodeNegative, "max_price must not be negative")
	}
	if q.MinPrice != nil && q.MaxPrice != nil && q.MinPrice.GreaterThan(*q.MaxPrice) {
		verr.Add("min_price", CodeInvalidFormat, "min_price must not be greater than max_price")
	}

	if err := verr.Err(); err != nil {
		return ProductFilter{}, PageRequest{}, err
	}

	return ProductFilter{
		Query:      text,
		CategoryID: q.CategoryID,
		MinPrice:   q.MinPrice,
		MaxPrice:   q.MaxPrice,
		InStock:    q.InStock,
		Sort:       sort,
		Limit:      page.PageSize,
		Offset:     page.Offset(),
	}, page, nil
}
