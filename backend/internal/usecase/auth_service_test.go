package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
	"e-commerce-backend/internal/usecase/mocks"
)

var fixedNow = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

type authDeps struct {
	users   *mocks.UserRepository
	hasher  *mocks.PasswordHasher
	access  *mocks.AccessTokenIssuer
	refresh *mocks.RefreshTokenStore
	svc     *usecase.AuthService
}

func newAuth(t *testing.T) *authDeps {
	t.Helper()
	d := &authDeps{
		users:   mocks.NewUserRepository(t),
		hasher:  mocks.NewPasswordHasher(t),
		access:  mocks.NewAccessTokenIssuer(t),
		refresh: mocks.NewRefreshTokenStore(t),
	}
	d.svc = usecase.NewAuthService(d.users, d.hasher, d.access, d.refresh).WithClock(func() time.Time { return fixedNow })
	return d
}

func (d *authDeps) expectSession(userID uuid.UUID) {
	d.access.EXPECT().Issue(mock.MatchedBy(func(u *domain.User) bool { return u.ID == userID })).
		Return("access-token", fixedNow.Add(15*time.Minute), nil)
	d.refresh.EXPECT().Issue(mock.Anything, userID, usecase.RefreshTokenTTL).Return("refresh-token", nil)
}

func user(t *testing.T, role domain.Role) domain.User {
	t.Helper()
	u, err := domain.NewUser("jane@example.com", "stored-hash", role)
	require.NoError(t, err)
	return u
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("creates USER with hashed password", func(t *testing.T) {
		d := newAuth(t)
		d.hasher.EXPECT().Hash("correct-horse").Return("bcrypt-hash", nil)
		d.users.EXPECT().Create(ctx, mock.MatchedBy(func(u *domain.User) bool {
			return u.Email == "jane@example.com" && u.Role == domain.RoleUser && u.PasswordHash == "bcrypt-hash"
		})).Return(nil)

		u, err := d.svc.Register(ctx, " Jane@Example.com ", "correct-horse")
		require.NoError(t, err)
		assert.Equal(t, domain.RoleUser, u.Role)
	})

	t.Run("reports email and password errors together without hashing", func(t *testing.T) {
		d := newAuth(t)

		_, err := d.svc.Register(ctx, "not-an-email", "short")

		var verr *domain.ValidationError
		require.ErrorAs(t, err, &verr)
		assert.Len(t, verr.Fields, 2)
	})

	t.Run("duplicate email", func(t *testing.T) {
		d := newAuth(t)
		d.hasher.EXPECT().Hash(mock.Anything).Return("h", nil)
		d.users.EXPECT().Create(ctx, mock.Anything).Return(domain.ErrEmailTaken)

		_, err := d.svc.Register(ctx, "jane@example.com", "correct-horse")
		assert.ErrorIs(t, err, domain.ErrEmailTaken)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("issues session", func(t *testing.T) {
		d := newAuth(t)
		u := user(t, domain.RoleUser)
		d.users.EXPECT().GetByEmail(ctx, "jane@example.com").Return(u, nil)
		d.hasher.EXPECT().Compare("stored-hash", "correct-horse").Return(true)
		d.expectSession(u.ID)

		s, err := d.svc.Login(ctx, "jane@example.com", "correct-horse")
		require.NoError(t, err)
		assert.Equal(t, "access-token", s.AccessToken)
		assert.Equal(t, "refresh-token", s.RefreshToken)
		assert.Equal(t, fixedNow.Add(usecase.RefreshTokenTTL), s.RefreshTokenExpiresAt)
	})

	t.Run("unknown email still runs password hashing and returns generic error", func(t *testing.T) {
		d := newAuth(t)
		d.users.EXPECT().GetByEmail(ctx, "ghost@example.com").Return(domain.User{}, domain.ErrUserNotFound)
		d.hasher.EXPECT().Compare("", "whatever-pass").Return(false).Once()

		_, err := d.svc.Login(ctx, "ghost@example.com", "whatever-pass")
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("wrong password and inactive account look the same", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			active   bool
			pwdMatch bool
		}{
			{"wrong password", true, false},
			{"inactive account", false, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				d := newAuth(t)
				u := user(t, domain.RoleAdmin)
				u.Active = tc.active
				d.users.EXPECT().GetByEmail(ctx, mock.Anything).Return(u, nil)
				d.hasher.EXPECT().Compare(mock.Anything, mock.Anything).Return(tc.pwdMatch)

				_, err := d.svc.Login(ctx, u.Email, "correct-horse")
				assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
			})
		}
	})

	t.Run("temporary password", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			expiresAt time.Time
			wantErr   error
		}{
			{"valid temporary password requires change", fixedNow.Add(time.Hour), nil},
			{"expired temporary password is rejected", fixedNow.Add(-time.Second), domain.ErrTempPasswordExpired},
		} {
			t.Run(tc.name, func(t *testing.T) {
				d := newAuth(t)
				u := user(t, domain.RoleAdmin)
				u.SetTemporaryPassword("stored-hash", tc.expiresAt)
				d.users.EXPECT().GetByEmail(ctx, mock.Anything).Return(u, nil)
				d.hasher.EXPECT().Compare("stored-hash", mock.Anything).Return(true)
				if tc.wantErr == nil {
					d.expectSession(u.ID)
				}

				s, err := d.svc.Login(ctx, u.Email, "Temp-Pass-1234")
				if tc.wantErr != nil {
					assert.ErrorIs(t, err, tc.wantErr)
					return
				}
				require.NoError(t, err)
				assert.True(t, s.User.MustChangePassword)
			})
		}
	})
}

func TestAuthService_Refresh(t *testing.T) {
	ctx := context.Background()

	t.Run("rotates token", func(t *testing.T) {
		d := newAuth(t)
		u := user(t, domain.RoleUser)
		d.refresh.EXPECT().Consume(ctx, "old-refresh").Return(u.ID, nil)
		d.users.EXPECT().GetByID(ctx, u.ID).Return(u, nil)
		d.expectSession(u.ID)

		s, err := d.svc.Refresh(ctx, "old-refresh")
		require.NoError(t, err)
		assert.Equal(t, "refresh-token", s.RefreshToken)
	})

	t.Run("reused or unknown token", func(t *testing.T) {
		d := newAuth(t)
		d.refresh.EXPECT().Consume(ctx, "used").Return(uuid.Nil, domain.ErrInvalidToken)

		_, err := d.svc.Refresh(ctx, "used")
		assert.ErrorIs(t, err, domain.ErrInvalidToken)
	})

	t.Run("deactivated user loses all sessions", func(t *testing.T) {
		d := newAuth(t)
		u := user(t, domain.RoleAdmin)
		u.Active = false
		d.refresh.EXPECT().Consume(ctx, "r").Return(u.ID, nil)
		d.users.EXPECT().GetByID(ctx, u.ID).Return(u, nil)
		d.refresh.EXPECT().RevokeAll(ctx, u.ID).Return(nil)

		_, err := d.svc.Refresh(ctx, "r")
		assert.ErrorIs(t, err, domain.ErrInvalidToken)
	})

	t.Run("deleted user", func(t *testing.T) {
		d := newAuth(t)
		id := uuid.Must(uuid.NewV7())
		d.refresh.EXPECT().Consume(ctx, "r").Return(id, nil)
		d.users.EXPECT().GetByID(ctx, id).Return(domain.User{}, domain.ErrUserNotFound)

		_, err := d.svc.Refresh(ctx, "r")
		assert.ErrorIs(t, err, domain.ErrInvalidToken)
	})
}

func TestAuthService_ChangePassword(t *testing.T) {
	ctx := context.Background()

	t.Run("clears temporary flag, revokes sessions and issues new ones", func(t *testing.T) {
		d := newAuth(t)
		u := user(t, domain.RoleAdmin)
		u.SetTemporaryPassword("temp-hash", fixedNow.Add(time.Hour))
		d.users.EXPECT().GetByID(ctx, u.ID).Return(u, nil)
		d.hasher.EXPECT().Compare("temp-hash", "Temp-Pass-1234").Return(true)
		d.hasher.EXPECT().Hash("correct-horse-battery").Return("new-hash", nil)
		d.users.EXPECT().Update(ctx, mock.MatchedBy(func(u *domain.User) bool {
			return u.PasswordHash == "new-hash" && !u.MustChangePassword && u.TempPasswordExpiresAt == nil
		})).Return(nil)
		d.refresh.EXPECT().RevokeAll(ctx, u.ID).Return(nil)
		d.expectSession(u.ID)

		s, err := d.svc.ChangePassword(ctx, u.ID, "Temp-Pass-1234", "correct-horse-battery")
		require.NoError(t, err)
		assert.False(t, s.User.MustChangePassword)
	})

	cases := []struct {
		name      string
		current   string
		next      string
		currentOK bool
		wantField string
		wantCode  string
	}{
		{"wrong current password", "nope", "correct-horse-battery", false, "current_password", "incorrect"},
		{"weak new password", "correct-horse", "short", true, "new_password", "too_short"},
		{"new equals current", "correct-horse", "correct-horse", true, "new_password", "same_as_current"},
		{"new equals email", "correct-horse", "jane@example.com", true, "new_password", "same_as_email"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newAuth(t)
			u := user(t, domain.RoleUser)
			d.users.EXPECT().GetByID(ctx, u.ID).Return(u, nil)
			d.hasher.EXPECT().Compare("stored-hash", tc.current).Return(tc.currentOK)

			_, err := d.svc.ChangePassword(ctx, u.ID, tc.current, tc.next)

			var verr *domain.ValidationError
			require.ErrorAs(t, err, &verr)
			assert.Equal(t, tc.wantField, verr.Fields[0].Field)
			assert.Equal(t, tc.wantCode, verr.Fields[0].Code)
		})
	}
}

func TestAuthService_EnsureSuperAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("skipped without email", func(t *testing.T) {
		require.NoError(t, newAuth(t).svc.EnsureSuperAdmin(ctx, "", ""))
	})

	t.Run("creates once", func(t *testing.T) {
		d := newAuth(t)
		d.users.EXPECT().GetByEmail(ctx, "root@example.com").Return(domain.User{}, domain.ErrUserNotFound)
		d.hasher.EXPECT().Hash("super-secret-pass").Return("h", nil)
		d.users.EXPECT().Create(ctx, mock.MatchedBy(func(u *domain.User) bool { return u.Role == domain.RoleSuperAdmin })).Return(nil)

		require.NoError(t, d.svc.EnsureSuperAdmin(ctx, "root@example.com", "super-secret-pass"))
	})

	t.Run("existing superadmin is left untouched", func(t *testing.T) {
		d := newAuth(t)
		d.users.EXPECT().GetByEmail(ctx, "jane@example.com").Return(user(t, domain.RoleSuperAdmin), nil)

		require.NoError(t, d.svc.EnsureSuperAdmin(ctx, "jane@example.com", "different-password"))
	})

	t.Run("email owned by regular user is an error", func(t *testing.T) {
		d := newAuth(t)
		d.users.EXPECT().GetByEmail(ctx, "jane@example.com").Return(user(t, domain.RoleUser), nil)

		assert.Error(t, d.svc.EnsureSuperAdmin(ctx, "jane@example.com", "super-secret-pass"))
	})

	t.Run("concurrent creation by another instance is fine", func(t *testing.T) {
		d := newAuth(t)
		d.users.EXPECT().GetByEmail(ctx, mock.Anything).Return(domain.User{}, domain.ErrUserNotFound)
		d.hasher.EXPECT().Hash(mock.Anything).Return("h", nil)
		d.users.EXPECT().Create(ctx, mock.Anything).Return(domain.ErrEmailTaken)

		require.NoError(t, d.svc.EnsureSuperAdmin(ctx, "root@example.com", "super-secret-pass"))
	})

	t.Run("weak password is rejected", func(t *testing.T) {
		d := newAuth(t)
		d.users.EXPECT().GetByEmail(ctx, mock.Anything).Return(domain.User{}, domain.ErrUserNotFound)

		err := d.svc.EnsureSuperAdmin(ctx, "root@example.com", "123")
		assert.True(t, errors.Is(err, domain.ErrValidation))
	})
}
