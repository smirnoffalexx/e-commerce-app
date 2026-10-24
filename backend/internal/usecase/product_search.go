package usecase

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"e-commerce-backend/internal/domain"
)

type ListProductsQuery struct {
	Query      string
	CategoryID *uuid.UUID
	MinPrice   *decimal.Decimal
	MaxPrice   *decimal.Decimal
	InStock    bool
	Sort       domain.ProductSort
	PageRequest
}

func (q ListProductsQuery) toFilter() (domain.ProductFilter, PageRequest, error) {
	var verr domain.ValidationError

	page, err := q.normalize()
	collect(&verr, err)

	text := strings.Join(strings.Fields(q.Query), " ")
	switch {
	case utf8.RuneCountInString(text) > domain.MaxSearchQueryLen:
		verr.Add("q", domain.CodeTooLong, "q must be at most 200 characters")
	case strings.ContainsFunc(text, func(r rune) bool { return r < 0x20 || r == 0x7f }):
		verr.Add("q", domain.CodeInvalidFormat, "q must not contain control characters")
	}

	sort := q.Sort
	if sort == "" {
		sort = domain.SortNewest
		if text != "" {
			sort = domain.SortRelevance
		}
	}
	switch {
	case !sort.Valid():
		verr.Add("sort", domain.CodeInvalidFormat, "sort must be one of relevance, newest, price_asc, price_desc, name_asc")
	case sort == domain.SortRelevance && text == "":
		verr.Add("sort", domain.CodeInvalidFormat, "sort=relevance requires q")
	}

	if q.MinPrice != nil && q.MinPrice.IsNegative() {
		verr.Add("min_price", domain.CodeNegative, "min_price must not be negative")
	}
	if q.MaxPrice != nil && q.MaxPrice.IsNegative() {
		verr.Add("max_price", domain.CodeNegative, "max_price must not be negative")
	}
	if q.MinPrice != nil && q.MaxPrice != nil && q.MinPrice.GreaterThan(*q.MaxPrice) {
		verr.Add("min_price", domain.CodeInvalidFormat, "min_price must not be greater than max_price")
	}

	if err := verr.Err(); err != nil {
		return domain.ProductFilter{}, PageRequest{}, err
	}

	return domain.ProductFilter{
		Query:      text,
		CategoryID: q.CategoryID,
		MinPrice:   q.MinPrice,
		MaxPrice:   q.MaxPrice,
		InStock:    q.InStock,
		Sort:       sort,
		Limit:      page.PageSize,
		Offset:     page.offset(),
	}, page, nil
}

func collect(dst *domain.ValidationError, err error) {
	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		dst.Fields = append(dst.Fields, verr.Fields...)
	}
}
