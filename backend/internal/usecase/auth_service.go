package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

const RefreshTokenTTL = 7 * 24 * time.Hour

type AuthService struct {
	users   UserRepository
	hasher  PasswordHasher
	access  AccessTokenIssuer
	refresh RefreshTokenStore
	now     func() time.Time
}

func NewAuthService(users UserRepository, hasher PasswordHasher, access AccessTokenIssuer, refresh RefreshTokenStore) *AuthService {
	return &AuthService{users: users, hasher: hasher, access: access, refresh: refresh, now: time.Now}
}

func (s *AuthService) WithClock(now func() time.Time) *AuthService {
	s.now = now
	return s
}

func (s *AuthService) Register(ctx context.Context, email, password string) (domain.User, error) {
	if err := validateCredentials(email, password); err != nil {
		return domain.User{}, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, err
	}
	u, err := domain.NewUser(email, hash, domain.RoleUser)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.users.Create(ctx, &u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (domain.Session, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrUserNotFound) {
		s.hasher.Compare("", password)
		return domain.Session{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return domain.Session{}, err
	}

	if !s.hasher.Compare(u.PasswordHash, password) || !u.Active {
		return domain.Session{}, domain.ErrInvalidCredentials
	}
	if u.TempPasswordExpired(s.now()) {
		return domain.Session{}, domain.ErrTempPasswordExpired
	}
	return s.newSession(ctx, &u)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (domain.Session, error) {
	userID, err := s.refresh.Consume(ctx, refreshToken)
	if err != nil {
		return domain.Session{}, err
	}

	u, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.Session{}, domain.ErrInvalidToken
	}
	if err != nil {
		return domain.Session{}, err
	}
	if !u.Active {
		if err := s.refresh.RevokeAll(ctx, u.ID); err != nil {
			return domain.Session{}, err
		}
		return domain.Session{}, domain.ErrInvalidToken
	}
	if u.TempPasswordExpired(s.now()) {
		return domain.Session{}, domain.ErrTempPasswordExpired
	}
	return s.newSession(ctx, &u)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.refresh.Revoke(ctx, refreshToken)
}

func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, current, next string) (domain.Session, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return domain.Session{}, err
	}

	var verr domain.ValidationError
	if !s.hasher.Compare(u.PasswordHash, current) {
		verr.Add("current_password", "incorrect", "current password is incorrect")
		return domain.Session{}, &verr
	}
	if err := domain.ValidatePassword("new_password", next, u.Email); err != nil {
		return domain.Session{}, err
	}
	if next == current {
		verr.Add("new_password", "same_as_current", "new password must differ from the current one")
		return domain.Session{}, &verr
	}

	hash, err := s.hasher.Hash(next)
	if err != nil {
		return domain.Session{}, err
	}
	u.SetPassword(hash)
	if err := s.users.Update(ctx, &u); err != nil {
		return domain.Session{}, err
	}
	if err := s.refresh.RevokeAll(ctx, u.ID); err != nil {
		return domain.Session{}, err
	}
	return s.newSession(ctx, &u)
}

func (s *AuthService) EnsureSuperAdmin(ctx context.Context, email, password string) error {
	if email == "" {
		return nil
	}

	existing, err := s.users.GetByEmail(ctx, email)
	switch {
	case err == nil:
		if existing.Role != domain.RoleSuperAdmin {
			return fmt.Errorf("superadmin email %q belongs to a user with role %s", existing.Email, existing.Role)
		}
		return nil
	case !errors.Is(err, domain.ErrUserNotFound):
		return err
	}

	if err := validateCredentials(email, password); err != nil {
		return fmt.Errorf("invalid superadmin credentials: %w", err)
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}
	u, err := domain.NewUser(email, hash, domain.RoleSuperAdmin)
	if err != nil {
		return err
	}
	if err := s.users.Create(ctx, &u); err != nil && !errors.Is(err, domain.ErrEmailTaken) {
		return err
	}
	return nil
}

func (s *AuthService) newSession(ctx context.Context, u *domain.User) (domain.Session, error) {
	access, accessExp, err := s.access.Issue(u)
	if err != nil {
		return domain.Session{}, err
	}
	refresh, err := s.refresh.Issue(ctx, u.ID, RefreshTokenTTL)
	if err != nil {
		return domain.Session{}, err
	}
	return domain.Session{
		AccessToken:           access,
		AccessTokenExpiresAt:  accessExp,
		RefreshToken:          refresh,
		RefreshTokenExpiresAt: s.now().Add(RefreshTokenTTL),
		User:                  *u,
	}, nil
}

func validateCredentials(email, password string) error {
	var verr domain.ValidationError
	for _, err := range []error{domain.ValidateEmail(email), domain.ValidatePassword("password", password, email)} {
		var fe *domain.ValidationError
		if errors.As(err, &fe) {
			verr.Fields = append(verr.Fields, fe.Fields...)
		}
	}
	return verr.Err()
}
