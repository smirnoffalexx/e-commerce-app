package dto

import (
	"time"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

type CategoryRequest struct {
	Name string `json:"name" example:"Food & Beverage"`
}

type CategoryResponse struct {
	ID        uuid.UUID `json:"id" swaggertype:"string" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	Name      string    `json:"name" example:"Food & Beverage"`
	Slug      string    `json:"slug" example:"food-and-beverage"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CategoryListResponse struct {
	Items []CategoryResponse `json:"items"`
}

func NewCategoryResponse(c *domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:        c.ID,
		Name:      c.Name,
		Slug:      c.Slug,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func NewCategoryListResponse(categories []domain.Category) CategoryListResponse {
	items := make([]CategoryResponse, 0, len(categories))
	for i := range categories {
		items = append(items, NewCategoryResponse(&categories[i]))
	}
	return CategoryListResponse{Items: items}
}
