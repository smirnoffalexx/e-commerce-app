package dto

import (
	"time"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

type CreateAdminRequest struct {
	Email string `json:"email" example:"admin@example.com"`
}

type AdminCredentialsResponse struct {
	User                       UserResponse `json:"user"`
	TemporaryPassword          string       `json:"temporary_password" example:"Kd7mQx2RtLp9WzHn"`
	TemporaryPasswordExpiresAt time.Time    `json:"temporary_password_expires_at"`
}

type UserListResponse struct {
	Items      []UserResponse `json:"items"`
	Page       int            `json:"page" example:"1"`
	PageSize   int            `json:"page_size" example:"20"`
	Total      int64          `json:"total" example:"3"`
	TotalPages int            `json:"total_pages" example:"1"`
}

func NewAdminCredentialsResponse(c *usecase.AdminCredentials) AdminCredentialsResponse {
	return AdminCredentialsResponse{
		User:                       NewUserResponse(&c.User),
		TemporaryPassword:          c.TemporaryPassword,
		TemporaryPasswordExpiresAt: c.TemporaryPasswordExpiresAt,
	}
}

func NewUserListResponse(page *usecase.Page[domain.User]) UserListResponse {
	items := make([]UserResponse, 0, len(page.Items))
	for i := range page.Items {
		items = append(items, NewUserResponse(&page.Items[i]))
	}
	return UserListResponse{
		Items:      items,
		Page:       page.Page,
		PageSize:   page.PageSize,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
	}
}
