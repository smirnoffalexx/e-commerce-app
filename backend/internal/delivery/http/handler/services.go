package handler

import (
	"context"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

type ProductService interface {
	Create(ctx context.Context, in domain.ProductInput) (domain.Product, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Product, error)
	List(ctx context.Context, q usecase.ListProductsQuery) (usecase.Page[domain.Product], error)
	Update(ctx context.Context, id uuid.UUID, version int, in domain.ProductInput) (domain.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type CategoryService interface {
	Create(ctx context.Context, name string) (domain.Category, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Category, error)
	List(ctx context.Context) ([]domain.Category, error)
	Rename(ctx context.Context, id uuid.UUID, name string) (domain.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type AuthService interface {
	Register(ctx context.Context, email, password string) (domain.User, error)
	Login(ctx context.Context, email, password string) (usecase.Session, error)
	Refresh(ctx context.Context, refreshToken string) (usecase.Session, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID uuid.UUID) (domain.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, current, next string) (usecase.Session, error)
}

type AdminService interface {
	Create(ctx context.Context, email string) (usecase.AdminCredentials, error)
	List(ctx context.Context, req usecase.PageRequest) (usecase.Page[domain.User], error)
	ResetPassword(ctx context.Context, id uuid.UUID) (usecase.AdminCredentials, error)
	SetActive(ctx context.Context, id uuid.UUID, active bool) (domain.User, error)
}

var (
	_ AdminService    = (*usecase.AdminService)(nil)
	_ ProductService  = (*usecase.ProductService)(nil)
	_ CategoryService = (*usecase.CategoryService)(nil)
	_ AuthService     = (*usecase.AuthService)(nil)
)
