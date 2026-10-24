package csvimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

const header = "name,sku,description,category,price,stock,weight_kg\n"

func parse(t *testing.T, content string) (usecase.ParsedCatalog, error) {
	t.Helper()
	return NewParser().Parse(strings.NewReader(content))
}

func errorsByLine(errs []usecase.ImportRowError) map[int][]string {
	out := map[int][]string{}
	for _, e := range errs {
		out[e.Line] = append(out[e.Line], e.Field+":"+e.Code)
	}
	return out
}

func TestParse_ExampleFile(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "..", "example_data.csv"))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	parsed, err := NewParser().Parse(f)
	require.NoError(t, err)

	assert.Equal(t, 95, parsed.DataRows)
	assert.Equal(t, 2, parsed.BlankRows, "lines 62-63 are ,,,,,,")
	assert.Len(t, parsed.Rows, 92)
	assert.Equal(t, map[int][]string{
		4:  {"price:invalid_format"},
		7:  {"price:invalid_format"},
		50: {"weight_kg:required"},
	}, errorsByLine(parsed.Errors))

	byLine := map[int]usecase.ImportRow{}
	for _, r := range parsed.Rows {
		byLine[r.Line] = r
	}
	assert.Equal(t, "<script>alert('xss')</script>", byLine[20].Name, "kept as plain text")
	assert.Equal(t, "Robert'); DROP TABLE products;--", byLine[29].Name)
	assert.Equal(t, "Comma, In Product Name", byLine[53].Name)
	assert.Equal(t, `Quote "Inside" Name`, byLine[59].Name)
	assert.Equal(t, "Stainless steel, 1L — keeps drinks cold for 24hrs™", byLine[31].Description)
	assert.Equal(t, -5, byLine[16].Stock, "negative stock is rejected later by domain validation")
	assert.Equal(t, "", byLine[25].Name)
	assert.True(t, byLine[47].Price.IsZero())
}

func TestParse_Header(t *testing.T) {
	t.Run("case, whitespace, order and BOM are tolerated", func(t *testing.T) {
		parsed, err := parse(t, "\xEF\xBB\xBF SKU ,Name,PRICE,stock,weight_kg,category,description\r\nRS-001,Shoes,1.00,1,0.5,Footwear,\r\n")
		require.NoError(t, err)
		require.Len(t, parsed.Rows, 1)
		assert.Equal(t, "RS-001", parsed.Rows[0].SKU)
		assert.Equal(t, "Shoes", parsed.Rows[0].Name)
		assert.Equal(t, 2, parsed.Rows[0].Line)
	})

	cases := map[string]struct {
		content string
		code    string
	}{
		"empty file":      {"", domain.CodeRequired},
		"missing columns": {"name,sku,price\n", "missing_columns"},
		"unknown columns": {strings.TrimSuffix(header, "\n") + ",colour\n", "unknown_columns"},
		"duplicate":       {strings.TrimSuffix(header, "\n") + ",sku\n", "duplicate_columns"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := parse(t, tc.content)
			var verr *domain.ValidationError
			require.ErrorAs(t, err, &verr)
			assert.Equal(t, "file", verr.Fields[0].Field)
			assert.Equal(t, tc.code, verr.Fields[0].Code)
		})
	}

	t.Run("header only", func(t *testing.T) {
		parsed, err := parse(t, header)
		require.NoError(t, err)
		assert.Zero(t, parsed.DataRows)
	})
}

func TestParse_Values(t *testing.T) {
	cases := []struct {
		name   string
		row    string
		errors []string
	}{
		{"valid", "Shoes,RS-1,desc,Footwear,89.99,150,0.35", nil},
		{"currency symbol", "Shoes,RS-1,,Footwear,$29.99,1,1", []string{"price:invalid_format"}},
		{"word instead of price", "Shoes,RS-1,,Footwear,free,1,1", []string{"price:invalid_format"}},
		{"thousands separator", "Shoes,RS-1,,Footwear,\"1,299.00\",1,1", []string{"price:invalid_format"}},
		{"scientific notation", "Shoes,RS-1,,Footwear,1e3,1,1", []string{"price:invalid_format"}},
		{"empty price", "Shoes,RS-1,,Footwear,,1,1", []string{"price:required"}},
		{"fractional stock", "Shoes,RS-1,,Footwear,1,1.5,1", []string{"stock:invalid_format"}},
		{"empty stock", "Shoes,RS-1,,Footwear,1,,1", []string{"stock:required"}},
		{"empty weight", "Shoes,RS-1,,Footwear,1,1,", []string{"weight_kg:required"}},
		{"all numbers bad at once", "Shoes,RS-1,,Footwear,x,y,z", []string{"price:invalid_format", "weight_kg:invalid_format", "stock:invalid_format"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := parse(t, header+tc.row+"\n")
			require.NoError(t, err)
			if tc.errors == nil {
				require.Len(t, parsed.Rows, 1)
				assert.True(t, parsed.Rows[0].Price.Equal(decimal.RequireFromString("89.99")))
				return
			}
			assert.Empty(t, parsed.Rows)
			assert.ElementsMatch(t, tc.errors, errorsByLine(parsed.Errors)[2])
		})
	}
}

func TestParse_StructuralErrorsDoNotStopTheFile(t *testing.T) {
	content := header +
		"Too,Few,Columns\n" +
		"\"Multi\nline\",ML-1,\"desc\nmore\",Misc,1,1,1\n" +
		"Bad \"quote,BQ-1,,Misc,1,1,1\n" +
		"After,AF-1,,Misc,1,1,1\n"

	parsed, err := parse(t, content)
	require.NoError(t, err)

	assert.Equal(t, map[int][]string{2: {":column_count"}, 6: {":malformed_row"}}, errorsByLine(parsed.Errors))
	require.Len(t, parsed.Rows, 2)
	assert.Equal(t, 3, parsed.Rows[0].Line, "multi-line record is reported at its first line")
	assert.Equal(t, "Multi\nline", parsed.Rows[0].Name)
	assert.Equal(t, 7, parsed.Rows[1].Line)
}

func TestParse_RowLimit(t *testing.T) {
	var b strings.Builder
	b.WriteString(header)
	for range MaxRows + 1 {
		b.WriteString("P,SKU,,Misc,1,1,1\n")
	}

	_, err := parse(t, b.String())
	var verr *domain.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, domain.CodeTooLarge, verr.Fields[0].Code)
}
