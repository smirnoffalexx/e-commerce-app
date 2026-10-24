package usecase

import (
	"io"

	"github.com/shopspring/decimal"
)

type ImportMode string

const (
	ImportModeUpsert ImportMode = "upsert"
	ImportModeSkip   ImportMode = "skip"
	ImportModeFail   ImportMode = "fail"
)

func (m ImportMode) Valid() bool {
	return m == ImportModeUpsert || m == ImportModeSkip || m == ImportModeFail
}

type ImportRow struct {
	Line        int
	SKU         string
	Name        string
	Description string
	Category    string
	Price       decimal.Decimal
	Stock       int
	WeightKg    decimal.Decimal
}

type ImportRowError struct {
	Line    int
	SKU     string
	Field   string
	Code    string
	Message string
}

type ParsedCatalog struct {
	Rows      []ImportRow
	Errors    []ImportRowError
	DataRows  int
	BlankRows int
}

type CatalogParser interface {
	Parse(r io.Reader) (ParsedCatalog, error)
}

type ImportOptions struct {
	Mode                    ImportMode
	DryRun                  bool
	Atomic                  bool
	CreateMissingCategories bool
}

type ImportDuplicate struct {
	SKU         string
	Lines       []int
	AppliedLine int
}

type ImportReport struct {
	Options           ImportOptions
	Committed         bool
	TotalRows         int
	BlankRows         int
	Created           int
	Updated           int
	Skipped           int
	Failed            int
	CreatedCategories []string
	Duplicates        []ImportDuplicate
	Errors            []ImportRowError
}
