package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"e-commerce-backend/internal/domain"
)

type categoryModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

func (categoryModel) TableName() string { return "categories" }

type productModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	SKU         string    `gorm:"column:sku"`
	Name        string
	Description string
	CategoryID  uuid.UUID       `gorm:"type:uuid"`
	Category    *categoryModel  `gorm:"foreignKey:CategoryID"`
	Price       decimal.Decimal `gorm:"type:numeric(12,2)"`
	Stock       int
	Reserved    int             `gorm:"->"`
	WeightKg    decimal.Decimal `gorm:"column:weight_kg;type:numeric(10,3)"`
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt
}

func (productModel) TableName() string { return "products" }

func categoryToModel(c *domain.Category) categoryModel {
	return categoryModel{
		ID:        c.ID,
		Name:      c.Name,
		Slug:      c.Slug,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func categoryToDomain(m *categoryModel) domain.Category {
	return domain.Category{
		ID:        m.ID,
		Name:      m.Name,
		Slug:      m.Slug,
		CreatedAt: m.CreatedAt.UTC(),
		UpdatedAt: m.UpdatedAt.UTC(),
	}
}

func productToModel(p *domain.Product) productModel {
	return productModel{
		ID:          p.ID,
		SKU:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		CategoryID:  p.CategoryID,
		Price:       p.Price,
		Stock:       p.Stock,
		WeightKg:    p.WeightKg,
		Version:     p.Version,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func productToDomain(m *productModel) domain.Product {
	p := domain.Product{
		ID:          m.ID,
		SKU:         m.SKU,
		Name:        m.Name,
		Description: m.Description,
		CategoryID:  m.CategoryID,
		Price:       m.Price,
		Stock:       m.Stock,
		Reserved:    m.Reserved,
		WeightKg:    m.WeightKg,
		Version:     m.Version,
		CreatedAt:   m.CreatedAt.UTC(),
		UpdatedAt:   m.UpdatedAt.UTC(),
	}
	if m.Category != nil {
		c := categoryToDomain(m.Category)
		p.Category = &c
	}
	return p
}
