package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInput() ProductInput {
	return ProductInput{
		SKU:         "RS-001",
		Name:        "Running Shoes",
		Description: "Lightweight running shoes for daily training",
		CategoryID:  uuid.Must(uuid.NewV7()),
		Price:       decimal.RequireFromString("89.99"),
		Stock:       150,
		WeightKg:    decimal.RequireFromString("0.35"),
	}
}

func TestNewProduct_Valid(t *testing.T) {
	in := validInput()
	in.SKU = "  rs-001 "
	in.Name = "  Running Shoes\t"

	p, err := NewProduct(in)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, p.ID)
	assert.Equal(t, 1, p.Version)
	assert.Equal(t, "RS-001", p.SKU)
	assert.Equal(t, "Running Shoes", p.Name)
	assert.True(t, p.Price.Equal(decimal.RequireFromString("89.99")))
	assert.True(t, p.InStock())
}

func TestNewProduct_AllowsEdgeValues(t *testing.T) {
	cases := map[string]func(*ProductInput){
		"zero price":                 func(in *ProductInput) { in.Price = decimal.Zero },
		"zero stock":                 func(in *ProductInput) { in.Stock = 0 },
		"zero weight":                func(in *ProductInput) { in.WeightKg = decimal.Zero },
		"empty description":          func(in *ProductInput) { in.Description = "" },
		"max price":                  func(in *ProductInput) { in.Price = MaxPrice },
		"trailing zeros in price":    func(in *ProductInput) { in.Price = decimal.RequireFromString("19.900") },
		"html is kept as plain text": func(in *ProductInput) { in.Name = "<script>alert('xss')</script>" },
		"sql is kept as plain text":  func(in *ProductInput) { in.Name = "Robert'); DROP TABLE products;--" },
		"comma and quotes in name":   func(in *ProductInput) { in.Name = `Quote "Inside", Name` },
		"unicode in description":     func(in *ProductInput) { in.Description = "1L — keeps drinks cold for 24hrs™" },
		"multiline description":      func(in *ProductInput) { in.Description = "line one\nline two" },
		"name at max length":         func(in *ProductInput) { in.Name = strings.Repeat("я", MaxNameLen) },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := validInput()
			mutate(&in)

			_, err := NewProduct(in)
			assert.NoError(t, err)
		})
	}
}

func TestNewProduct_Invalid(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ProductInput)
		field  string
		code   string
	}{
		{"empty name", func(in *ProductInput) { in.Name = "" }, "name", CodeRequired},
		{"whitespace-only name", func(in *ProductInput) { in.Name = "     " }, "name", CodeRequired},
		{"name too long", func(in *ProductInput) { in.Name = strings.Repeat("a", MaxNameLen+1) }, "name", CodeTooLong},
		{"control char in name", func(in *ProductInput) { in.Name = "Shoes\x00" }, "name", CodeInvalidFormat},
		{"newline in name", func(in *ProductInput) { in.Name = "Shoes\nforged" }, "name", CodeInvalidFormat},
		{"empty sku", func(in *ProductInput) { in.SKU = " " }, "sku", CodeRequired},
		{"sku too long", func(in *ProductInput) { in.SKU = strings.Repeat("A", MaxSKULen+1) }, "sku", CodeTooLong},
		{"sku with space", func(in *ProductInput) { in.SKU = "RS 001" }, "sku", CodeInvalidFormat},
		{"sku with leading dash", func(in *ProductInput) { in.SKU = "-RS" }, "sku", CodeInvalidFormat},
		{"sku with trailing dash", func(in *ProductInput) { in.SKU = "RS-" }, "sku", CodeInvalidFormat},
		{"sku with sql", func(in *ProductInput) { in.SKU = "SQL-001';--" }, "sku", CodeInvalidFormat},
		{"description too long", func(in *ProductInput) { in.Description = strings.Repeat("a", MaxDescriptionLen+1) }, "description", CodeTooLong},
		{"control char in description", func(in *ProductInput) { in.Description = "bad\x07bell" }, "description", CodeInvalidFormat},
		{"missing category", func(in *ProductInput) { in.CategoryID = uuid.Nil }, "category_id", CodeRequired},
		{"negative price", func(in *ProductInput) { in.Price = decimal.RequireFromString("-0.01") }, "price", CodeNegative},
		{"price too large", func(in *ProductInput) { in.Price = MaxPrice.Add(decimal.RequireFromString("0.01")) }, "price", CodeTooLarge},
		{"price with 3 decimals", func(in *ProductInput) { in.Price = decimal.RequireFromString("9.999") }, "price", CodeTooManyDecimals},
		{"negative stock", func(in *ProductInput) { in.Stock = -5 }, "stock", CodeNegative},
		{"stock too large", func(in *ProductInput) { in.Stock = MaxStock + 1 }, "stock", CodeTooLarge},
		{"negative weight", func(in *ProductInput) { in.WeightKg = decimal.RequireFromString("-1") }, "weight_kg", CodeNegative},
		{"weight with 4 decimals", func(in *ProductInput) { in.WeightKg = decimal.RequireFromString("0.0001") }, "weight_kg", CodeTooManyDecimals},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput()
			tc.mutate(&in)

			_, err := NewProduct(in)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrValidation)

			var verr *ValidationError
			require.ErrorAs(t, err, &verr)
			require.Len(t, verr.Fields, 1)
			assert.Equal(t, tc.field, verr.Fields[0].Field)
			assert.Equal(t, tc.code, verr.Fields[0].Code)
		})
	}
}

func TestNewProduct_ReportsAllInvalidFields(t *testing.T) {
	_, err := NewProduct(ProductInput{
		Price:    decimal.RequireFromString("-1"),
		Stock:    -1,
		WeightKg: decimal.RequireFromString("-1"),
	})

	var verr *ValidationError
	require.ErrorAs(t, err, &verr)

	fields := make([]string, 0, len(verr.Fields))
	for _, f := range verr.Fields {
		fields = append(fields, f.Field)
	}
	assert.ElementsMatch(t, []string{"sku", "name", "category_id", "price", "weight_kg", "stock"}, fields)
}

func TestProduct_Update(t *testing.T) {
	p, err := NewProduct(validInput())
	require.NoError(t, err)
	original := p

	in := validInput()
	in.Price = decimal.RequireFromString("94.99")
	in.Stock = 120
	require.NoError(t, p.Update(in))
	assert.Equal(t, original.ID, p.ID)
	assert.True(t, p.Price.Equal(decimal.RequireFromString("94.99")))
	assert.Equal(t, 120, p.Stock)

	beforeInvalid := p
	in.Stock = -1
	in.Name = ""
	require.ErrorIs(t, p.Update(in), ErrValidation)
	assert.Equal(t, beforeInvalid, p, "invalid update must not partially modify the product")
}

func TestProduct_UpdateKeepsReservedUnits(t *testing.T) {
	p, err := NewProduct(validInput())
	require.NoError(t, err)
	p.Reserved = 30
	assert.Equal(t, 120, p.Available())

	before := p
	in := validInput()
	in.Stock = 29
	require.ErrorIs(t, p.Update(in), ErrStockBelowReserved)
	assert.Equal(t, before, p)

	in.Stock = 30
	require.NoError(t, p.Update(in))
	assert.Zero(t, p.Available())
	assert.False(t, p.InStock())
}

func TestProductErrors_Kinds(t *testing.T) {
	assert.ErrorIs(t, ErrProductNotFound, ErrNotFound)
	assert.ErrorIs(t, ErrSKUTaken, ErrConflict)
	assert.ErrorIs(t, ErrProductVersionStale, ErrConflict)
	assert.ErrorIs(t, ErrStockBelowReserved, ErrConflict)
	assert.False(t, errors.Is(ErrSKUTaken, ErrNotFound))

	var derr *Error
	require.ErrorAs(t, ErrSKUTaken, &derr)
	assert.Equal(t, "sku_taken", derr.Code())
}
