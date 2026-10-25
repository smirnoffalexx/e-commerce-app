package usecase_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/csvimport"
	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
	"e-commerce-backend/internal/usecase/mocks"
)

const csvHeader = "name,sku,description,category,price,stock,weight_kg\n"

type importDeps struct {
	tx         *mocks.Transactor
	repo       *mocks.CatalogImportRepository
	categories *mocks.CategoryRepository
	products   *mocks.ProductRepository
	svc        *usecase.CatalogImportService
	created    []domain.Product
}

func newImport(t *testing.T) *importDeps {
	t.Helper()
	d := &importDeps{
		tx:         mocks.NewTransactor(t),
		repo:       mocks.NewCatalogImportRepository(t),
		categories: mocks.NewCategoryRepository(t),
		products:   mocks.NewProductRepository(t),
	}
	d.svc = usecase.NewCatalogImportService(csvimport.NewParser(), d.tx, d.repo, d.categories, d.products)
	return d
}

func (d *importDeps) expectTx() {
	d.tx.EXPECT().WithinTransaction(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	d.repo.EXPECT().LockCatalogImport(mock.Anything).Return(nil)
}

func (d *importDeps) expectCatalog(categories []string, existing ...domain.Product) {
	found := map[string]domain.Category{}
	for _, name := range categories {
		c, _ := domain.NewCategory(name)
		found[c.Slug] = c
	}
	d.repo.EXPECT().CategoriesBySlugs(mock.Anything, mock.Anything).Return(found, nil)

	bySKU := map[string]domain.Product{}
	for i := range existing {
		bySKU[existing[i].SKU] = existing[i]
	}
	d.repo.EXPECT().ProductsBySKUs(mock.Anything, mock.Anything).Return(bySKU, nil)
	d.repo.EXPECT().CreateProducts(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, ps []domain.Product) error {
			d.created = append(d.created, ps...)
			return nil
		}).Maybe()
}

func codes(errs []domain.ImportRowError) map[int][]string {
	out := map[int][]string{}
	for _, e := range errs {
		out[e.Line] = append(out[e.Line], e.Field+":"+e.Code)
	}
	return out
}

func existingProduct2(t *testing.T, sku string) domain.Product {
	t.Helper()
	c, _ := domain.NewCategory("Misc")
	p, err := domain.NewProduct(domain.ProductInput{
		SKU: sku, Name: "Old", CategoryID: c.ID, Price: decimal.RequireFromString("1"), Stock: 1, WeightKg: decimal.RequireFromString("1"),
	})
	require.NoError(t, err)
	return p
}

func TestCatalogImport_ExampleFile(t *testing.T) {
	d := newImport(t)
	d.expectTx()
	d.expectCatalog(nil)
	d.categories.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	f, err := os.Open(filepath.Join("..", "..", "..", "example_data.csv"))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	report, err := d.svc.Import(context.Background(), f, domain.ImportOptions{CreateMissingCategories: true})
	require.NoError(t, err)

	assert.True(t, report.Committed)
	assert.Equal(t, domain.ImportModeUpsert, report.Options.Mode, "upsert is the default mode")
	assert.Equal(t, 95, report.TotalRows)
	assert.Equal(t, 2, report.BlankRows)
	assert.Equal(t, 85, report.Created)
	assert.Equal(t, 7, report.Failed)
	assert.Equal(t, map[int][]string{
		4:  {"price:invalid_format"},
		7:  {"price:invalid_format"},
		16: {"stock:negative"},
		25: {"name:required"},
		41: {"name:required"},
		50: {"weight_kg:required"},
		52: {"category:required"},
	}, codes(report.Errors))
	assert.Equal(t, []domain.ImportDuplicate{
		{SKU: "RS-001", Lines: []int{2, 36}, AppliedLine: 36},
		{SKU: "BS-021", Lines: []int{11, 56, 89}, AppliedLine: 89},
	}, report.Duplicates)
	assert.Len(t, report.CreatedCategories, 17)
	assert.NotContains(t, report.CreatedCategories, "Misc", "categories are created only for rows that are otherwise valid")

	var bs021 domain.Product
	for _, p := range d.created {
		if p.SKU == "BS-021" {
			bs021 = p
		}
	}
	assert.True(t, bs021.Price.Equal(decimal.RequireFromString("59.99")), "last duplicate row wins")
}

func TestCatalogImport_Modes(t *testing.T) {
	ctx := context.Background()
	file := csvHeader + "New,NEW-1,,Misc,1,1,1\nUpdated,old-1,,Misc,9.99,5,1\n"

	t.Run("upsert updates existing", func(t *testing.T) {
		d := newImport(t)
		d.expectTx()
		d.expectCatalog([]string{"Misc"}, existingProduct2(t, "OLD-1"))
		d.products.EXPECT().Update(mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return p.SKU == "OLD-1" && p.Name == "Updated" && p.Stock == 5
		})).Return(nil)

		report, err := d.svc.Import(ctx, strings.NewReader(file), domain.ImportOptions{Mode: domain.ImportModeUpsert})
		require.NoError(t, err)
		assert.Equal(t, 1, report.Created)
		assert.Equal(t, 1, report.Updated)
	})

	t.Run("skip leaves existing untouched", func(t *testing.T) {
		d := newImport(t)
		d.expectTx()
		d.expectCatalog([]string{"Misc"}, existingProduct2(t, "OLD-1"))

		report, err := d.svc.Import(ctx, strings.NewReader(file), domain.ImportOptions{Mode: domain.ImportModeSkip})
		require.NoError(t, err)
		assert.Equal(t, 1, report.Created)
		assert.Equal(t, 1, report.Skipped)
		assert.Empty(t, report.Errors)
	})

	t.Run("fail reports existing as row errors", func(t *testing.T) {
		d := newImport(t)
		d.expectTx()
		d.expectCatalog([]string{"Misc"}, existingProduct2(t, "OLD-1"))

		report, err := d.svc.Import(ctx, strings.NewReader(file), domain.ImportOptions{Mode: domain.ImportModeFail})
		require.NoError(t, err)
		assert.True(t, report.Committed)
		assert.Equal(t, 1, report.Created)
		assert.Equal(t, map[int][]string{3: {"sku:sku_taken"}}, codes(report.Errors))
	})

	t.Run("unknown mode is rejected before parsing", func(t *testing.T) {
		d := newImport(t)
		_, err := d.svc.Import(ctx, strings.NewReader(file), domain.ImportOptions{Mode: "merge"})
		assert.ErrorIs(t, err, domain.ErrValidation)
	})
}

func TestCatalogImport_Categories(t *testing.T) {
	ctx := context.Background()
	file := csvHeader + "A,A-1,,Toys,1,1,1\nB,B-1,,toys,1,1,1\nC,C-1,,Misc,1,1,1\n"

	t.Run("missing category is a row error by default", func(t *testing.T) {
		d := newImport(t)
		d.expectTx()
		d.expectCatalog([]string{"Misc"})

		report, err := d.svc.Import(ctx, strings.NewReader(file), domain.ImportOptions{})
		require.NoError(t, err)
		assert.Equal(t, 1, report.Created)
		assert.Equal(t, map[int][]string{2: {"category:category_not_found"}, 3: {"category:category_not_found"}}, codes(report.Errors))
	})

	t.Run("created once for names that differ only by case", func(t *testing.T) {
		d := newImport(t)
		d.expectTx()
		d.expectCatalog([]string{"Misc"})
		d.categories.EXPECT().Create(mock.Anything, mock.MatchedBy(func(c *domain.Category) bool { return c.Slug == "toys" })).Return(nil).Once()

		report, err := d.svc.Import(ctx, strings.NewReader(file), domain.ImportOptions{CreateMissingCategories: true})
		require.NoError(t, err)
		assert.Equal(t, 3, report.Created)
		assert.Equal(t, []string{"Toys"}, report.CreatedCategories)
	})
}

func TestCatalogImport_DryRunAndAtomic(t *testing.T) {
	ctx := context.Background()
	file := csvHeader + "Good,G-1,,Misc,1,1,1\nBad,B-1,,Misc,free,1,1\n"

	rollbackTx := func(d *importDeps) *error {
		var fnErr error
		d.tx.EXPECT().WithinTransaction(mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				fnErr = fn(ctx)
				return fnErr
			})
		d.repo.EXPECT().LockCatalogImport(mock.Anything).Return(nil)
		return &fnErr
	}

	t.Run("dry run reports exact result but rolls back", func(t *testing.T) {
		d := newImport(t)
		fnErr := rollbackTx(d)
		d.expectCatalog([]string{"Misc"})

		report, err := d.svc.Import(ctx, strings.NewReader(file), domain.ImportOptions{DryRun: true})
		require.NoError(t, err)
		assert.False(t, report.Committed)
		assert.Equal(t, 1, report.Created)
		assert.Equal(t, 1, report.Failed)
		assert.Error(t, *fnErr, "transaction must be rolled back")
	})

	t.Run("atomic import with errors saves nothing", func(t *testing.T) {
		d := newImport(t)
		fnErr := rollbackTx(d)
		d.expectCatalog([]string{"Misc"})

		report, err := d.svc.Import(ctx, strings.NewReader(file), domain.ImportOptions{Atomic: true})
		require.NoError(t, err)
		assert.False(t, report.Committed)
		assert.Error(t, *fnErr)
	})

	t.Run("atomic import without errors commits", func(t *testing.T) {
		d := newImport(t)
		fnErr := rollbackTx(d)
		d.expectCatalog([]string{"Misc"})

		report, err := d.svc.Import(ctx, strings.NewReader(csvHeader+"Good,G-1,,Misc,1,1,1\n"), domain.ImportOptions{Atomic: true})
		require.NoError(t, err)
		assert.True(t, report.Committed)
		assert.NoError(t, *fnErr)
	})
}

func TestCatalogImport_Failures(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid header never opens a transaction", func(t *testing.T) {
		d := newImport(t)
		_, err := d.svc.Import(ctx, strings.NewReader("foo,bar\n"), domain.ImportOptions{})
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("database failure aborts the import", func(t *testing.T) {
		d := newImport(t)
		d.expectTx()
		boom := errors.New("connection reset")
		d.repo.EXPECT().CategoriesBySlugs(mock.Anything, mock.Anything).Return(nil, boom)

		_, err := d.svc.Import(ctx, strings.NewReader(csvHeader+"A,A-1,,Misc,1,1,1\n"), domain.ImportOptions{})
		assert.ErrorIs(t, err, boom)
	})

	t.Run("concurrent sku creation is a conflict", func(t *testing.T) {
		d := newImport(t)
		d.expectTx()
		c, _ := domain.NewCategory("Misc")
		d.repo.EXPECT().CategoriesBySlugs(mock.Anything, mock.Anything).Return(map[string]domain.Category{"misc": c}, nil)
		d.repo.EXPECT().ProductsBySKUs(mock.Anything, mock.Anything).Return(map[string]domain.Product{}, nil)
		d.repo.EXPECT().CreateProducts(mock.Anything, mock.Anything).Return(domain.ErrSKUTaken)

		_, err := d.svc.Import(ctx, strings.NewReader(csvHeader+"A,A-1,,Misc,1,1,1\n"), domain.ImportOptions{})
		assert.ErrorIs(t, err, domain.ErrConflict)
	})
}
