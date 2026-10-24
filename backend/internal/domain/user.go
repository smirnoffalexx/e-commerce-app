package domain

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxEmailLen      = 254
	MinPasswordLen   = 10
	MaxPasswordBytes = 72
)

var (
	ErrUserNotFound            = NewError(ErrNotFound, "user_not_found", "user not found")
	ErrAdminNotFound           = NewError(ErrNotFound, "admin_not_found", "admin not found")
	ErrEmailTaken              = NewError(ErrConflict, "email_taken", "user with this email already exists")
	ErrInvalidCredentials      = NewError(ErrUnauthorized, "invalid_credentials", "invalid email or password")
	ErrInvalidToken            = NewError(ErrUnauthorized, "invalid_token", "token is invalid or expired")
	ErrTempPasswordExpired     = NewError(ErrUnauthorized, "temp_password_expired", "temporary password has expired, ask an administrator for a new one")
	ErrPasswordChangeRequired  = NewError(ErrForbidden, "password_change_required", "password must be changed before using the API")
	ErrInsufficientPermissions = NewError(ErrForbidden, "insufficient_permissions", "you do not have permission to perform this action")
)

type User struct {
	ID                    uuid.UUID
	Email                 string
	PasswordHash          string
	Role                  Role
	Active                bool
	MustChangePassword    bool
	TempPasswordExpiresAt *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func NewUser(email, passwordHash string, role Role) (User, error) {
	email = NormalizeEmail(email)

	var verr ValidationError
	validateEmail(&verr, email)
	if !role.Valid() {
		verr.Add("role", CodeInvalidFormat, "role is not supported")
	}
	if err := verr.Err(); err != nil {
		return User{}, err
	}

	return User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		Active:       true,
	}, nil
}

func (u *User) TempPasswordExpired(now time.Time) bool {
	return u.MustChangePassword && u.TempPasswordExpiresAt != nil && !now.Before(*u.TempPasswordExpiresAt)
}

func (u *User) SetPassword(hash string) {
	u.PasswordHash = hash
	u.MustChangePassword = false
	u.TempPasswordExpiresAt = nil
}

func (u *User) SetTemporaryPassword(hash string, expiresAt time.Time) {
	u.PasswordHash = hash
	u.MustChangePassword = true
	u.TempPasswordExpiresAt = &expiresAt
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateEmail(email string) error {
	var verr ValidationError
	validateEmail(&verr, NormalizeEmail(email))
	return verr.Err()
}

func validateEmail(verr *ValidationError, email string) {
	switch {
	case email == "":
		verr.Add("email", CodeRequired, "email is required")
	case len(email) > MaxEmailLen:
		verr.Add("email", CodeTooLong, "email must be at most 254 characters")
	default:
		addr, err := mail.ParseAddress(email)
		if err != nil || addr.Address != email || !strings.Contains(email[strings.LastIndexByte(email, '@')+1:], ".") {
			verr.Add("email", CodeInvalidFormat, "email must be a valid address")
		}
	}
}

func ValidatePassword(field, password, email string) error {
	var verr ValidationError
	switch {
	case password == "":
		verr.Add(field, CodeRequired, field+" is required")
	case len([]rune(password)) < MinPasswordLen:
		verr.Add(field, "too_short", field+" must be at least 10 characters")
	case len(password) > MaxPasswordBytes:
		verr.Add(field, CodeTooLong, field+" must be at most 72 bytes")
	case strings.TrimSpace(password) == "":
		verr.Add(field, CodeInvalidFormat, field+" must not be blank")
	case email != "" && strings.EqualFold(password, NormalizeEmail(email)):
		verr.Add(field, "same_as_email", field+" must not be the same as email")
	}
	return verr.Err()
}

type Principal struct {
	UserID             uuid.UUID
	Role               Role
	MustChangePassword bool
}
