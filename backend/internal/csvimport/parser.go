package csvimport

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

const MaxRows = 10000

var (
	columns     = []string{"name", "sku", "description", "category", "price", "stock", "weight_kg"}
	plainNumber = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	utf8BOM     = []byte{0xEF, 0xBB, 0xBF}
)

type Parser struct{}

func NewParser() Parser {
	return Parser{}
}

func (Parser) Parse(r io.Reader) (domain.ParsedCatalog, error) {
	br := bufio.NewReader(r)
	if prefix, err := br.Peek(len(utf8BOM)); err == nil && bytes.Equal(prefix, utf8BOM) {
		_, _ = br.Discard(len(utf8BOM))
	}

	cr := csv.NewReader(br)
	cr.FieldsPerRecord = -1

	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return domain.ParsedCatalog{}, fileError(domain.CodeRequired, "file is empty")
	}
	if err != nil {
		return domain.ParsedCatalog{}, readError(err)
	}
	index, err := headerIndex(header)
	if err != nil {
		return domain.ParsedCatalog{}, err
	}

	var out domain.ParsedCatalog
	for {
		record, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		var parseErr *csv.ParseError
		if errors.As(err, &parseErr) {
			out.DataRows++
			out.Errors = append(out.Errors, domain.ImportRowError{
				Line: parseErr.StartLine, Code: "malformed_row", Message: "row is not valid CSV: " + parseErr.Err.Error(),
			})
			continue
		}
		if err != nil {
			return domain.ParsedCatalog{}, readError(err)
		}
		line, _ := cr.FieldPos(0)

		if isBlank(record) {
			out.BlankRows++
			continue
		}
		out.DataRows++
		if out.DataRows > MaxRows {
			return domain.ParsedCatalog{}, fileError(domain.CodeTooLarge, fmt.Sprintf("file must contain at most %d rows", MaxRows))
		}
		if len(record) != len(columns) {
			out.Errors = append(out.Errors, domain.ImportRowError{
				Line: line, Code: "column_count", Message: fmt.Sprintf("expected %d columns, got %d", len(columns), len(record)),
			})
			continue
		}

		row, rowErrs := toRow(line, record, index)
		if len(rowErrs) > 0 {
			out.Errors = append(out.Errors, rowErrs...)
			continue
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

func headerIndex(header []string) (map[string]int, error) {
	index := make(map[string]int, len(header))
	var unknown, duplicate []string
	for i, h := range header {
		name := strings.ToLower(strings.TrimSpace(h))
		if _, seen := index[name]; seen {
			duplicate = append(duplicate, name)
			continue
		}
		index[name] = i
	}

	known := make(map[string]bool, len(columns))
	var missing []string
	for _, c := range columns {
		known[c] = true
		if _, ok := index[c]; !ok {
			missing = append(missing, c)
		}
	}
	for name := range index {
		if !known[name] {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)

	var verr domain.ValidationError
	if len(missing) > 0 {
		verr.Add("file", "missing_columns", "missing required columns: "+strings.Join(missing, ", "))
	}
	if len(unknown) > 0 {
		verr.Add("file", "unknown_columns", "unknown columns: "+strings.Join(unknown, ", "))
	}
	if len(duplicate) > 0 {
		verr.Add("file", "duplicate_columns", "duplicate columns: "+strings.Join(duplicate, ", "))
	}
	if err := verr.Err(); err != nil {
		return nil, err
	}
	return index, nil
}

func toRow(line int, record []string, index map[string]int) (domain.ImportRow, []domain.ImportRowError) {
	get := func(col string) string { return strings.TrimSpace(record[index[col]]) }

	row := domain.ImportRow{
		Line:        line,
		SKU:         get("sku"),
		Name:        get("name"),
		Description: get("description"),
		Category:    get("category"),
	}

	var errs []domain.ImportRowError
	fail := func(field, code, msg string) {
		errs = append(errs, domain.ImportRowError{Line: line, SKU: row.SKU, Field: field, Code: code, Message: msg})
	}

	row.Price = parseDecimal(get("price"), "price", fail)
	row.WeightKg = parseDecimal(get("weight_kg"), "weight_kg", fail)

	if stock := get("stock"); stock == "" {
		fail("stock", domain.CodeRequired, "stock is required")
	} else if v, err := strconv.Atoi(stock); err != nil {
		fail("stock", domain.CodeInvalidFormat, fmt.Sprintf("stock must be a whole number, got %q", stock))
	} else {
		row.Stock = v
	}

	return row, errs
}

func parseDecimal(raw, field string, fail func(field, code, msg string)) decimal.Decimal {
	if raw == "" {
		fail(field, domain.CodeRequired, field+" is required")
		return decimal.Zero
	}
	if !plainNumber.MatchString(raw) {
		fail(field, domain.CodeInvalidFormat, fmt.Sprintf("%s must be a plain decimal number like 29.99, got %q", field, raw))
		return decimal.Zero
	}
	return decimal.RequireFromString(raw)
}

func isBlank(record []string) bool {
	for _, v := range record {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func fileError(code, msg string) error {
	verr := &domain.ValidationError{}
	verr.Add("file", code, msg)
	return verr
}

func readError(err error) error {
	return fmt.Errorf("read csv: %w", err)
}

var _ usecase.CatalogParser = Parser{}
