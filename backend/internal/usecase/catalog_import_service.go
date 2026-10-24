package usecase

import (
	"context"
	"errors"
	"io"
	"sort"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

var (
	errRollbackDryRun = errors.New("dry run rollback")
	errRollbackAtomic = errors.New("atomic import rollback")
	placeholderID     = uuid.Must(uuid.Parse("00000000-0000-7000-8000-000000000000"))
)

type CatalogImportService struct {
	parser     CatalogParser
	tx         Transactor
	imports    CatalogImportRepository
	categories CategoryRepository
	products   ProductRepository
}

func NewCatalogImportService(parser CatalogParser, tx Transactor, imports CatalogImportRepository, categories CategoryRepository, products ProductRepository) *CatalogImportService {
	return &CatalogImportService{parser: parser, tx: tx, imports: imports, categories: categories, products: products}
}

type pendingRow struct {
	row   ImportRow
	input domain.ProductInput
	slug  string
}

func (s *CatalogImportService) Import(ctx context.Context, r io.Reader, opts ImportOptions) (ImportReport, error) {
	if opts.Mode == "" {
		opts.Mode = ImportModeUpsert
	}
	if !opts.Mode.Valid() {
		verr := &domain.ValidationError{}
		verr.Add("mode", domain.CodeInvalidFormat, "mode must be one of upsert, skip, fail")
		return ImportReport{}, verr
	}

	parsed, err := s.parser.Parse(r)
	if err != nil {
		return ImportReport{}, err
	}

	report := ImportReport{
		Options:   opts,
		TotalRows: parsed.DataRows,
		BlankRows: parsed.BlankRows,
		Errors:    parsed.Errors,
	}

	rows, duplicates := dedupeBySKU(parsed.Rows)
	report.Duplicates = duplicates

	pending := make([]pendingRow, 0, len(rows))
	for _, row := range rows {
		p, rowErrs := validateRow(row)
		if len(rowErrs) > 0 {
			report.Errors = append(report.Errors, rowErrs...)
			continue
		}
		pending = append(pending, p)
	}

	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.imports.LockCatalogImport(ctx); err != nil {
			return err
		}
		if err := s.apply(ctx, pending, opts, &report); err != nil {
			return err
		}
		switch {
		case opts.Atomic && len(report.Errors) > 0:
			return errRollbackAtomic
		case opts.DryRun:
			return errRollbackDryRun
		}
		return nil
	})
	switch {
	case err == nil:
		report.Committed = true
	case errors.Is(err, errRollbackDryRun), errors.Is(err, errRollbackAtomic):
	default:
		return ImportReport{}, err
	}

	sort.SliceStable(report.Errors, func(i, j int) bool { return report.Errors[i].Line < report.Errors[j].Line })
	report.Failed = countFailedLines(report.Errors)
	return report, nil
}

func (s *CatalogImportService) apply(ctx context.Context, pending []pendingRow, opts ImportOptions, report *ImportReport) error {
	categories, err := s.resolveCategories(ctx, pending, opts, report)
	if err != nil {
		return err
	}

	skus := make([]string, 0, len(pending))
	for i := range pending {
		skus = append(skus, pending[i].input.SKU)
	}
	existing, err := s.imports.ProductsBySKUs(ctx, skus)
	if err != nil {
		return err
	}

	toCreate := make([]domain.Product, 0, len(pending))
	for i := range pending {
		p := &pending[i]
		category, ok := categories[p.slug]
		if !ok {
			continue
		}
		p.input.CategoryID = category.ID

		current, found := existing[p.input.SKU]
		switch {
		case !found:
			product, err := domain.NewProduct(p.input)
			if err != nil {
				report.Errors = append(report.Errors, rowErrorsFrom(p.row, err)...)
				continue
			}
			toCreate = append(toCreate, product)
		case opts.Mode == ImportModeSkip:
			report.Skipped++
		case opts.Mode == ImportModeFail:
			report.Errors = append(report.Errors, ImportRowError{
				Line: p.row.Line, SKU: p.input.SKU, Field: "sku", Code: "sku_taken", Message: "product with this SKU already exists",
			})
		default:
			if err := current.Update(p.input); err != nil {
				report.Errors = append(report.Errors, rowErrorsFrom(p.row, err)...)
				continue
			}
			if err := s.products.Update(ctx, &current); err != nil {
				return err
			}
			report.Updated++
		}
	}

	if err := s.imports.CreateProducts(ctx, toCreate); err != nil {
		return err
	}
	report.Created = len(toCreate)
	return nil
}

func (s *CatalogImportService) resolveCategories(ctx context.Context, pending []pendingRow, opts ImportOptions, report *ImportReport) (map[string]domain.Category, error) {
	names := make(map[string]string)
	for i := range pending {
		if _, ok := names[pending[i].slug]; !ok {
			names[pending[i].slug] = pending[i].row.Category
		}
	}
	slugs := make([]string, 0, len(names))
	for slug := range names {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	found, err := s.imports.CategoriesBySlugs(ctx, slugs)
	if err != nil {
		return nil, err
	}

	for _, slug := range slugs {
		if _, ok := found[slug]; ok || !opts.CreateMissingCategories {
			continue
		}
		c, err := domain.NewCategory(names[slug])
		if err != nil {
			continue
		}
		if err := s.categories.Create(ctx, &c); err != nil {
			return nil, err
		}
		found[slug] = c
		report.CreatedCategories = append(report.CreatedCategories, c.Name)
	}

	for i := range pending {
		if _, ok := found[pending[i].slug]; !ok {
			report.Errors = append(report.Errors, ImportRowError{
				Line: pending[i].row.Line, SKU: pending[i].input.SKU, Field: "category", Code: "category_not_found",
				Message: "category " + quote(pending[i].row.Category) + " does not exist",
			})
		}
	}
	return found, nil
}

func dedupeBySKU(rows []ImportRow) ([]ImportRow, []ImportDuplicate) {
	lastIndex := make(map[string]int, len(rows))
	lines := make(map[string][]int)
	for i, row := range rows {
		sku := domain.NormalizeSKU(row.SKU)
		if sku == "" {
			continue
		}
		lastIndex[sku] = i
		lines[sku] = append(lines[sku], row.Line)
	}

	kept := make([]ImportRow, 0, len(rows))
	var duplicates []ImportDuplicate
	for i, row := range rows {
		sku := domain.NormalizeSKU(row.SKU)
		if sku != "" && lastIndex[sku] != i {
			continue
		}
		kept = append(kept, row)
		if len(lines[sku]) > 1 {
			duplicates = append(duplicates, ImportDuplicate{SKU: sku, Lines: lines[sku], AppliedLine: row.Line})
		}
	}
	sort.Slice(duplicates, func(i, j int) bool { return duplicates[i].Lines[0] < duplicates[j].Lines[0] })
	return kept, duplicates
}

func validateRow(row ImportRow) (pendingRow, []ImportRowError) {
	input := domain.ProductInput{
		SKU:         row.SKU,
		Name:        row.Name,
		Description: row.Description,
		CategoryID:  placeholderID,
		Price:       row.Price,
		Stock:       row.Stock,
		WeightKg:    row.WeightKg,
	}

	var errs []ImportRowError
	if err := input.Validate(); err != nil {
		errs = rowErrorsFrom(row, err)
	}
	slug := domain.Slugify(row.Category)
	if slug == "" {
		errs = append(errs, ImportRowError{Line: row.Line, SKU: row.SKU, Field: "category", Code: domain.CodeRequired, Message: "category is required"})
	}

	input.SKU = domain.NormalizeSKU(input.SKU)
	return pendingRow{row: row, input: input, slug: slug}, errs
}

func rowErrorsFrom(row ImportRow, err error) []ImportRowError {
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		return []ImportRowError{{Line: row.Line, SKU: row.SKU, Code: "invalid_row", Message: err.Error()}}
	}
	out := make([]ImportRowError, 0, len(verr.Fields))
	for _, f := range verr.Fields {
		out = append(out, ImportRowError{Line: row.Line, SKU: row.SKU, Field: f.Field, Code: f.Code, Message: f.Message})
	}
	return out
}

func countFailedLines(errs []ImportRowError) int {
	lines := make(map[int]struct{}, len(errs))
	for _, e := range errs {
		lines[e.Line] = struct{}{}
	}
	return len(lines)
}

func quote(s string) string {
	return `"` + s + `"`
}
