package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

const TempPasswordTTL = 24 * time.Hour

type AdminService struct {
	users     UserRepository
	hasher    PasswordHasher
	passwords TempPasswordGenerator
	refresh   RefreshTokenStore
	now       func() time.Time
}

func NewAdminService(users UserRepository, hasher PasswordHasher, passwords TempPasswordGenerator, refresh RefreshTokenStore) *AdminService {
	return &AdminService{users: users, hasher: hasher, passwords: passwords, refresh: refresh, now: time.Now}
}

func (s *AdminService) WithClock(now func() time.Time) *AdminService {
	s.now = now
	return s
}

func (s *AdminService) Create(ctx context.Context, email string) (domain.AdminCredentials, error) {
	if err := domain.ValidateEmail(email); err != nil {
		return domain.AdminCredentials{}, err
	}

	u, err := domain.NewUser(email, "", domain.RoleAdmin)
	if err != nil {
		return domain.AdminCredentials{}, err
	}
	creds, err := s.assignTemporaryPassword(&u)
	if err != nil {
		return domain.AdminCredentials{}, err
	}
	if err := s.users.Create(ctx, &u); err != nil {
		return domain.AdminCredentials{}, err
	}
	creds.User = u
	return creds, nil
}

func (s *AdminService) List(ctx context.Context, req domain.PageRequest) (domain.Page[domain.User], error) {
	page, err := req.Normalize()
	if err != nil {
		return domain.Page[domain.User]{}, err
	}
	items, total, err := s.users.ListByRole(ctx, domain.RoleAdmin, page.PageSize, page.Offset())
	if err != nil {
		return domain.Page[domain.User]{}, err
	}
	return domain.Page[domain.User]{Items: items, Total: total, Page: page.Page, PageSize: page.PageSize}, nil
}

func (s *AdminService) ResetPassword(ctx context.Context, id uuid.UUID) (domain.AdminCredentials, error) {
	u, err := s.getAdmin(ctx, id)
	if err != nil {
		return domain.AdminCredentials{}, err
	}

	creds, err := s.assignTemporaryPassword(&u)
	if err != nil {
		return domain.AdminCredentials{}, err
	}
	if err := s.users.Update(ctx, &u); err != nil {
		return domain.AdminCredentials{}, err
	}
	if err := s.refresh.RevokeAll(ctx, u.ID); err != nil {
		return domain.AdminCredentials{}, err
	}
	creds.User = u
	return creds, nil
}

func (s *AdminService) SetActive(ctx context.Context, id uuid.UUID, active bool) (domain.User, error) {
	u, err := s.getAdmin(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if u.Active == active {
		return u, nil
	}

	u.Active = active
	if err := s.users.Update(ctx, &u); err != nil {
		return domain.User{}, err
	}
	if !active {
		if err := s.refresh.RevokeAll(ctx, u.ID); err != nil {
			return domain.User{}, err
		}
	}
	return u, nil
}

func (s *AdminService) getAdmin(ctx context.Context, id uuid.UUID) (domain.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if errors.Is(err, domain.ErrUserNotFound) || (err == nil && u.Role != domain.RoleAdmin) {
		return domain.User{}, domain.ErrAdminNotFound
	}
	return u, err
}

func (s *AdminService) assignTemporaryPassword(u *domain.User) (domain.AdminCredentials, error) {
	password, err := s.passwords.Generate()
	if err != nil {
		return domain.AdminCredentials{}, err
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.AdminCredentials{}, err
	}
	expiresAt := s.now().Add(TempPasswordTTL).UTC()
	u.SetTemporaryPassword(hash, expiresAt)
	return domain.AdminCredentials{TemporaryPassword: password, TemporaryPasswordExpiresAt: expiresAt}, nil
}
