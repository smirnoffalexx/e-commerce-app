package postgres

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

const (
	catalogImportLockKey = 7_301_001
	lookupChunkSize      = 1000
	insertBatchSize      = 500
)

type CatalogImportRepository struct {
	db *gorm.DB
}

func NewCatalogImportRepository(db *gorm.DB) *CatalogImportRepository {
	return &CatalogImportRepository{db: db}
}

func (r *CatalogImportRepository) LockCatalogImport(ctx context.Context) error {
	if err := conn(ctx, r.db).Exec("SELECT pg_advisory_xact_lock(?)", catalogImportLockKey).Error; err != nil {
		return fmt.Errorf("lock catalog import: %w", err)
	}
	return nil
}

func (r *CatalogImportRepository) CategoriesBySlugs(ctx context.Context, slugs []string) (map[string]domain.Category, error) {
	result := make(map[string]domain.Category, len(slugs))
	for _, chunk := range chunks(slugs) {
		var models []categoryModel
		err := conn(ctx, r.db).
			Clauses(clause.Locking{Strength: clause.LockingStrengthShare}).
			Where("slug IN ?", chunk).
			Find(&models).Error
		if err != nil {
			return nil, fmt.Errorf("lock categories: %w", err)
		}
		for i := range models {
			result[models[i].Slug] = categoryToDomain(&models[i])
		}
	}
	return result, nil
}

func (r *CatalogImportRepository) ProductsBySKUs(ctx context.Context, skus []string) (map[string]domain.Product, error) {
	result := make(map[string]domain.Product, len(skus))
	for _, chunk := range chunks(skus) {
		var models []productModel
		err := conn(ctx, r.db).
			Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
			Where("sku IN ?", chunk).
			Find(&models).Error
		if err != nil {
			return nil, fmt.Errorf("lock products: %w", err)
		}
		for i := range models {
			result[models[i].SKU] = productToDomain(&models[i])
		}
	}
	return result, nil
}

func (r *CatalogImportRepository) CreateProducts(ctx context.Context, products []domain.Product) error {
	if len(products) == 0 {
		return nil
	}
	models := make([]productModel, 0, len(products))
	for i := range products {
		models = append(models, productToModel(&products[i]))
	}
	err := conn(ctx, r.db).Omit(clause.Associations).CreateInBatches(&models, insertBatchSize).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrSKUTaken
	}
	if err != nil {
		return fmt.Errorf("insert products: %w", err)
	}
	return nil
}

func chunks(values []string) [][]string {
	var out [][]string
	for start := 0; start < len(values); start += lookupChunkSize {
		out = append(out, values[start:min(start+lookupChunkSize, len(values))])
	}
	return out
}

var _ usecase.CatalogImportRepository = (*CatalogImportRepository)(nil)
