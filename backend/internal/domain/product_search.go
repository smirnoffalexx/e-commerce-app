package domain

import (
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
