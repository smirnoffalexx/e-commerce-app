package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

type userModel struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email                 string
	PasswordHash          string
	Role                  string
	IsActive              bool
	MustChangePassword    bool
	TempPasswordExpiresAt *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (userModel) TableName() string { return "users" }

func userToModel(u *domain.User) userModel {
	return userModel{
		ID:                    u.ID,
		Email:                 u.Email,
		PasswordHash:          u.PasswordHash,
		Role:                  string(u.Role),
		IsActive:              u.Active,
		MustChangePassword:    u.MustChangePassword,
		TempPasswordExpiresAt: u.TempPasswordExpiresAt,
		CreatedAt:             u.CreatedAt,
		UpdatedAt:             u.UpdatedAt,
	}
}

func userToDomain(m *userModel) domain.User {
	u := domain.User{
		ID:                 m.ID,
		Email:              m.Email,
		PasswordHash:       m.PasswordHash,
		Role:               domain.Role(m.Role),
		Active:             m.IsActive,
		MustChangePassword: m.MustChangePassword,
		CreatedAt:          m.CreatedAt.UTC(),
		UpdatedAt:          m.UpdatedAt.UTC(),
	}
	if m.TempPasswordExpiresAt != nil {
		t := m.TempPasswordExpiresAt.UTC()
		u.TempPasswordExpiresAt = &t
	}
	return u
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	m := userToModel(u)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("create user: %w", err)
	}
	*u = userToDomain(&m)
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return r.get(ctx, "id = ?", id)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.get(ctx, "email = ?", domain.NormalizeEmail(email))
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	res := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("id = ?", u.ID).
		Updates(map[string]any{
			"password_hash":            u.PasswordHash,
			"role":                     string(u.Role),
			"is_active":                u.Active,
			"must_change_password":     u.MustChangePassword,
			"temp_password_expires_at": u.TempPasswordExpiresAt,
			"updated_at":               gorm.Expr("now()"),
		})
	if res.Error != nil {
		return fmt.Errorf("update user: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) get(ctx context.Context, query string, arg any) (domain.User, error) {
	var m userModel
	if err := r.db.WithContext(ctx).Take(&m, query, arg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return userToDomain(&m), nil
}

var _ usecase.UserRepository = (*UserRepository)(nil)
