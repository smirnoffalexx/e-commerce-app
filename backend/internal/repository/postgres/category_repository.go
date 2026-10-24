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

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	m := categoryToModel(c)
	if err := conn(ctx, r.db).Create(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrCategoryExists
		}
		return fmt.Errorf("create category: %w", err)
	}
	*c = categoryToDomain(&m)
	return nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Category, error) {
	var m categoryModel
	if err := conn(ctx, r.db).Take(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Category{}, domain.ErrCategoryNotFound
		}
		return domain.Category{}, fmt.Errorf("get category: %w", err)
	}
	return categoryToDomain(&m), nil
}

func (r *CategoryRepository) GetBySlugs(ctx context.Context, slugs []string) (map[string]domain.Category, error) {
	result := make(map[string]domain.Category, len(slugs))
	if len(slugs) == 0 {
		return result, nil
	}

	var models []categoryModel
	if err := conn(ctx, r.db).Where("slug IN ?", slugs).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("get categories by slugs: %w", err)
	}
	for i := range models {
		result[models[i].Slug] = categoryToDomain(&models[i])
	}
	return result, nil
}

func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	var models []categoryModel
	if err := conn(ctx, r.db).Order("name ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	categories := make([]domain.Category, 0, len(models))
	for i := range models {
		categories = append(categories, categoryToDomain(&models[i]))
	}
	return categories, nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	res := conn(ctx, r.db).
		Model(&categoryModel{}).
		Where("id = ?", c.ID).
		Updates(map[string]any{"name": c.Name, "slug": c.Slug, "updated_at": gorm.Expr("now()")})
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrCategoryExists
		}
		return fmt.Errorf("update category: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		var m categoryModel
		err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).Take(&m, "id = ?", id).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrCategoryNotFound
			}
			return fmt.Errorf("lock category: %w", err)
		}

		var inUse bool
		err = tx.Raw("SELECT EXISTS (SELECT 1 FROM products WHERE category_id = ? AND deleted_at IS NULL)", id).Scan(&inUse).Error
		if err != nil {
			return fmt.Errorf("check category usage: %w", err)
		}
		if inUse {
			return domain.ErrCategoryInUse
		}

		if err := tx.Delete(&categoryModel{}, "id = ?", id).Error; err != nil {
			return fmt.Errorf("delete category: %w", err)
		}
		return nil
	})
}

var _ usecase.CategoryRepository = (*CategoryRepository)(nil)
