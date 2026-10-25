package domain

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	MaxSKULen         = 64
	MaxNameLen        = 200
	MaxDescriptionLen = 5000
	MaxStock          = math.MaxInt32
	PriceDecimals     = 2
	WeightDecimals    = 3
)

var (
	MaxPrice  = decimal.RequireFromString("9999999999.99")
	MaxWeight = decimal.RequireFromString("9999999.999")
)

var (
	ErrProductNotFound     = NewError(ErrNotFound, "product_not_found", "product not found")
	ErrSKUTaken            = NewError(ErrConflict, "sku_taken", "product with this SKU already exists")
	ErrProductVersionStale = NewError(ErrConflict, "version_conflict", "product was modified by another request, reload and try again")
	ErrStockBelowReserved  = NewError(ErrConflict, "stock_below_reserved", "stock cannot be lower than the quantity reserved by pending orders")
)

type Product struct {
	ID          uuid.UUID
	SKU         string
	Name        string
	Description string
	CategoryID  uuid.UUID
	Category    *Category
	Price       decimal.Decimal
	Stock       int
	Reserved    int
	WeightKg    decimal.Decimal
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProductInput struct {
	SKU         string
	Name        string
	Description string
	CategoryID  uuid.UUID
	Price       decimal.Decimal
	Stock       int
	WeightKg    decimal.Decimal
}

func NewProduct(in ProductInput) (Product, error) {
	p := Product{ID: uuid.Must(uuid.NewV7()), Version: 1}
	if err := p.apply(in); err != nil {
		return Product{}, err
	}
	return p, nil
}

func (p *Product) Update(in ProductInput) error {
	return p.apply(in)
}

func (p *Product) Available() int {
	return p.Stock - p.Reserved
}

func (p *Product) InStock() bool {
	return p.Available() > 0
}

func (p *Product) apply(in ProductInput) error {
	in = in.normalize()
	if err := in.Validate(); err != nil {
		return err
	}
	if in.Stock < p.Reserved {
		return ErrStockBelowReserved
	}

	p.SKU = in.SKU
	p.Name = in.Name
	p.Description = in.Description
	p.CategoryID = in.CategoryID
	p.Price = in.Price
	p.Stock = in.Stock
	p.WeightKg = in.WeightKg
	return nil
}

func (in ProductInput) normalize() ProductInput {
	in.SKU = NormalizeSKU(in.SKU)
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	return in
}

func (in ProductInput) Validate() error {
	in = in.normalize()
	var verr ValidationError

	switch {
	case in.SKU == "":
		verr.Add("sku", CodeRequired, "sku is required")
	case len(in.SKU) > MaxSKULen:
		verr.Add("sku", CodeTooLong, "sku must be at most 64 characters")
	case !isValidSKU(in.SKU):
		verr.Add("sku", CodeInvalidFormat, "sku may contain only latin letters, digits and inner dashes")
	}

	switch {
	case in.Name == "":
		verr.Add("name", CodeRequired, "name is required")
	case utf8.RuneCountInString(in.Name) > MaxNameLen:
		verr.Add("name", CodeTooLong, "name must be at most 200 characters")
	case hasControlChars(in.Name, false):
		verr.Add("name", CodeInvalidFormat, "name must not contain control characters")
	}

	switch {
	case utf8.RuneCountInString(in.Description) > MaxDescriptionLen:
		verr.Add("description", CodeTooLong, "description must be at most 5000 characters")
	case hasControlChars(in.Description, true):
		verr.Add("description", CodeInvalidFormat, "description must not contain control characters")
	}

	if in.CategoryID == uuid.Nil {
		verr.Add("category_id", CodeRequired, "category is required")
	}

	validateDecimal(&verr, "price", in.Price, PriceDecimals, MaxPrice)
	validateDecimal(&verr, "weight_kg", in.WeightKg, WeightDecimals, MaxWeight)

	switch {
	case in.Stock < 0:
		verr.Add("stock", CodeNegative, "stock must not be negative")
	case in.Stock > MaxStock:
		verr.Add("stock", CodeTooLarge, "stock is too large")
	}

	return verr.Err()
}

func NormalizeSKU(sku string) string {
	return strings.ToUpper(strings.TrimSpace(sku))
}

func isValidSKU(sku string) bool {
	if sku[0] == '-' || sku[len(sku)-1] == '-' {
		return false
	}
	for i := 0; i < len(sku); i++ {
		c := sku[i]
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}

func validateDecimal(verr *ValidationError, field string, v decimal.Decimal, places int32, maxValue decimal.Decimal) {
	switch {
	case v.IsNegative():
		verr.Add(field, CodeNegative, field+" must not be negative")
	case v.GreaterThan(maxValue):
		verr.Add(field, CodeTooLarge, field+" must be at most "+maxValue.String())
	case !v.Equal(v.Round(places)):
		verr.Add(field, CodeTooManyDecimals, field+" must have at most "+decimal.NewFromInt32(places).String()+" decimal places")
	}
}
