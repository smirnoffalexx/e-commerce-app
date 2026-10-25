package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	return conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		category, err := lockActiveCategory(tx, p.CategoryID)
		if err != nil {
			return err
		}

		m := productToModel(p)
		if err := tx.Omit(clause.Associations).Create(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return domain.ErrSKUTaken
			}
			return fmt.Errorf("create product: %w", err)
		}

		m.Category = &category
		*p = productToDomain(&m)
		return nil
	})
}

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	var m productModel
	err := r.withCategory(conn(ctx, r.db)).Take(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Product{}, domain.ErrProductNotFound
		}
		return domain.Product{}, fmt.Errorf("get product: %w", err)
	}
	return productToDomain(&m), nil
}

func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	return conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		category, err := lockActiveCategory(tx, p.CategoryID)
		if err != nil {
			return err
		}

		res := tx.Model(&productModel{}).
			Where("id = ? AND version = ? AND reserved <= ?", p.ID, p.Version, p.Stock).
			Updates(map[string]any{
				"sku":         p.SKU,
				"name":        p.Name,
				"description": p.Description,
				"category_id": p.CategoryID,
				"price":       p.Price,
				"stock":       p.Stock,
				"weight_kg":   p.WeightKg,
				"version":     gorm.Expr("version + 1"),
				"updated_at":  gorm.Expr("now()"),
			})
		if res.Error != nil {
			if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
				return domain.ErrSKUTaken
			}
			return fmt.Errorf("update product: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return r.rejectedUpdate(tx, p)
		}

		var m productModel
		if err := tx.Take(&m, "id = ?", p.ID).Error; err != nil {
			return fmt.Errorf("reload product: %w", err)
		}
		m.Category = &category
		*p = productToDomain(&m)
		return nil
	})
}

func (r *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := conn(ctx, r.db).Delete(&productModel{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("delete product: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *ProductRepository) withCategory(q *gorm.DB) *gorm.DB {
	return q.Preload("Category", func(db *gorm.DB) *gorm.DB { return db.Unscoped() })
}

func (r *ProductRepository) rejectedUpdate(tx *gorm.DB, p *domain.Product) error {
	var current productModel
	err := tx.Select("version", "reserved").Take(&current, "id = ?", p.ID).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return domain.ErrProductNotFound
	case err != nil:
		return fmt.Errorf("check product state: %w", err)
	case current.Version == p.Version && current.Reserved > p.Stock:
		return domain.ErrStockBelowReserved
	default:
		return domain.ErrProductVersionStale
	}
}

func lockActiveCategory(tx *gorm.DB, id uuid.UUID) (categoryModel, error) {
	var m categoryModel
	err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthShare}).Take(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return categoryModel{}, domain.ErrCategoryNotFound
		}
		return categoryModel{}, fmt.Errorf("lock category: %w", err)
	}
	return m, nil
}

var _ usecase.ProductRepository = (*ProductRepository)(nil)
