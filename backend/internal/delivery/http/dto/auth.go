package dto

import (
	"time"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

type CredentialsRequest struct {
	Email    string `json:"email" example:"jane@example.com"`
	Password string `json:"password" example:"correct-horse-battery"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" example:"q3Jx0dR8vV2..."`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" example:"Temp-Pass-1234"`
	NewPassword     string `json:"new_password" example:"correct-horse-battery"`
}

type UserResponse struct {
	ID                 uuid.UUID   `json:"id" swaggertype:"string" format:"uuid" example:"0192f6b8-3c5e-7a51-9d3f-2b1c4e5f6a7b"`
	Email              string      `json:"email" example:"jane@example.com"`
	Role               domain.Role `json:"role" swaggertype:"string" enums:"USER,ADMIN,SUPERADMIN" example:"USER"`
	Active             bool        `json:"active" example:"true"`
	MustChangePassword bool        `json:"must_change_password" example:"false"`
	CreatedAt          time.Time   `json:"created_at"`
}

type SessionResponse struct {
	TokenType              string       `json:"token_type" example:"Bearer"`
	AccessToken            string       `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresIn              int          `json:"expires_in" example:"900"`
	AccessTokenExpiresAt   time.Time    `json:"access_token_expires_at"`
	RefreshToken           string       `json:"refresh_token" example:"q3Jx0dR8vV2..."`
	RefreshTokenExpiresAt  time.Time    `json:"refresh_token_expires_at"`
	PasswordChangeRequired bool         `json:"password_change_required" example:"false"`
	User                   UserResponse `json:"user"`
}

func NewUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:                 u.ID,
		Email:              u.Email,
		Role:               u.Role,
		Active:             u.Active,
		MustChangePassword: u.MustChangePassword,
		CreatedAt:          u.CreatedAt,
	}
}

func NewSessionResponse(s *domain.Session, now time.Time) SessionResponse {
	return SessionResponse{
		TokenType:              "Bearer",
		AccessToken:            s.AccessToken,
		ExpiresIn:              int(s.AccessTokenExpiresAt.Sub(now).Seconds()),
		AccessTokenExpiresAt:   s.AccessTokenExpiresAt.UTC(),
		RefreshToken:           s.RefreshToken,
		RefreshTokenExpiresAt:  s.RefreshTokenExpiresAt.UTC(),
		PasswordChangeRequired: s.User.MustChangePassword,
		User:                   NewUserResponse(&s.User),
	}
}
