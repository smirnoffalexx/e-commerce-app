package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"e-commerce-backend/internal/domain"
)

type ProductRequest struct {
	SKU         string   `json:"sku" example:"RS-001"`
	Name        string   `json:"name" example:"Running Shoes"`
	Description string   `json:"description" example:"Lightweight running shoes for daily training"`
	CategoryID  string   `json:"category_id" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	Price       *Decimal `json:"price" swaggertype:"string" example:"89.99"`
	Stock       *int     `json:"stock" example:"150"`
	WeightKg    *Decimal `json:"weight_kg" swaggertype:"string" example:"0.35"`
}

type UpdateProductRequest struct {
	ProductRequest
	Version *int `json:"version" example:"1"`
}

func (r *ProductRequest) ToInput() (domain.ProductInput, error) {
	f := newFieldCollector()
	in := r.toInput(f)
	return in, f.err()
}

func (r *UpdateProductRequest) ToInput() (domain.ProductInput, int, error) {
	f := newFieldCollector()
	version := f.int("version", r.Version)
	in := r.toInput(f)
	return in, version, f.err()
}

func (r *ProductRequest) toInput(f *fieldCollector) domain.ProductInput {
	in := domain.ProductInput{
		SKU:         r.SKU,
		Name:        r.Name,
		Description: r.Description,
		CategoryID:  f.uuid("category_id", r.CategoryID),
		Price:       f.decimal("price", r.Price),
		Stock:       f.int("stock", r.Stock),
		WeightKg:    f.decimal("weight_kg", r.WeightKg),
	}
	f.merge(in.Validate())
	return in
}

type ProductResponse struct {
	ID          uuid.UUID         `json:"id" swaggertype:"string" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	SKU         string            `json:"sku" example:"RS-001"`
	Name        string            `json:"name" example:"Running Shoes"`
	Description string            `json:"description" example:"Lightweight running shoes for daily training"`
	Category    *CategoryResponse `json:"category,omitempty"`
	CategoryID  uuid.UUID         `json:"category_id" swaggertype:"string" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	Price       decimal.Decimal   `json:"price" swaggertype:"string" example:"89.99"`
	Stock       int               `json:"stock" example:"150"`
	InStock     bool              `json:"in_stock" example:"true"`
	WeightKg    decimal.Decimal   `json:"weight_kg" swaggertype:"string" example:"0.35"`
	Version     int               `json:"version" example:"1"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ProductListResponse struct {
	Items      []ProductResponse `json:"items"`
	Page       int               `json:"page" example:"1"`
	PageSize   int               `json:"page_size" example:"20"`
	Total      int64             `json:"total" example:"97"`
	TotalPages int               `json:"total_pages" example:"5"`
}

func NewProductResponse(p *domain.Product) ProductResponse {
	resp := ProductResponse{
		ID:          p.ID,
		SKU:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		CategoryID:  p.CategoryID,
		Price:       p.Price.Round(domain.PriceDecimals),
		Stock:       p.Stock,
		InStock:     p.InStock(),
		WeightKg:    p.WeightKg,
		Version:     p.Version,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if p.Category != nil {
		c := NewCategoryResponse(p.Category)
		resp.Category = &c
	}
	return resp
}

func NewProductListResponse(page *domain.Page[domain.Product]) ProductListResponse {
	items := make([]ProductResponse, 0, len(page.Items))
	for i := range page.Items {
		items = append(items, NewProductResponse(&page.Items[i]))
	}
	return ProductListResponse{
		Items:      items,
		Page:       page.Page,
		PageSize:   page.PageSize,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
	}
}
