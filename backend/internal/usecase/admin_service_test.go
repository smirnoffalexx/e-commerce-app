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

type adminDeps struct {
	users     *mocks.UserRepository
	hasher    *mocks.PasswordHasher
	passwords *mocks.TempPasswordGenerator
	refresh   *mocks.RefreshTokenStore
	svc       *usecase.AdminService
}

func newAdmins(t *testing.T) *adminDeps {
	t.Helper()
	d := &adminDeps{
		users:     mocks.NewUserRepository(t),
		hasher:    mocks.NewPasswordHasher(t),
		passwords: mocks.NewTempPasswordGenerator(t),
		refresh:   mocks.NewRefreshTokenStore(t),
	}
	d.svc = usecase.NewAdminService(d.users, d.hasher, d.passwords, d.refresh).WithClock(func() time.Time { return fixedNow })
	return d
}

func (d *adminDeps) expectTempPassword() {
	d.passwords.EXPECT().Generate().Return("Kd7mQx2RtLp9WzHn", nil)
	d.hasher.EXPECT().Hash("Kd7mQx2RtLp9WzHn").Return("temp-hash", nil)
}

func TestAdminService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates admin that must change temporary password", func(t *testing.T) {
		d := newAdmins(t)
		d.expectTempPassword()
		d.users.EXPECT().Create(ctx, mock.MatchedBy(func(u *domain.User) bool {
			return u.Role == domain.RoleAdmin &&
				u.Email == "admin@example.com" &&
				u.PasswordHash == "temp-hash" &&
				u.MustChangePassword &&
				u.TempPasswordExpiresAt.Equal(fixedNow.Add(usecase.TempPasswordTTL))
		})).Return(nil)

		creds, err := d.svc.Create(ctx, " Admin@Example.com ")
		require.NoError(t, err)
		assert.Equal(t, "Kd7mQx2RtLp9WzHn", creds.TemporaryPassword)
		assert.Equal(t, fixedNow.Add(24*time.Hour), creds.TemporaryPasswordExpiresAt)
		assert.NotEqual(t, creds.TemporaryPassword, creds.User.PasswordHash, "plain password is never stored")
	})

	t.Run("invalid email is rejected before generating a password", func(t *testing.T) {
		d := newAdmins(t)

		_, err := d.svc.Create(ctx, "not-an-email")
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("existing email is a conflict", func(t *testing.T) {
		d := newAdmins(t)
		d.expectTempPassword()
		d.users.EXPECT().Create(ctx, mock.Anything).Return(domain.ErrEmailTaken)

		_, err := d.svc.Create(ctx, "jane@example.com")
		assert.ErrorIs(t, err, domain.ErrEmailTaken)
	})
}

func TestAdminService_ResetPassword(t *testing.T) {
	ctx := context.Background()

	t.Run("issues new temporary password and revokes sessions", func(t *testing.T) {
		d := newAdmins(t)
		admin := user(t, domain.RoleAdmin)
		d.users.EXPECT().GetByID(ctx, admin.ID).Return(admin, nil)
		d.expectTempPassword()
		d.users.EXPECT().Update(ctx, mock.MatchedBy(func(u *domain.User) bool {
			return u.PasswordHash == "temp-hash" && u.MustChangePassword
		})).Return(nil)
		d.refresh.EXPECT().RevokeAll(ctx, admin.ID).Return(nil)

		creds, err := d.svc.ResetPassword(ctx, admin.ID)
		require.NoError(t, err)
		assert.True(t, creds.User.MustChangePassword)
	})

	t.Run("only ADMIN accounts can be managed", func(t *testing.T) {
		for _, role := range []domain.Role{domain.RoleUser, domain.RoleSuperAdmin} {
			d := newAdmins(t)
			target := user(t, role)
			d.users.EXPECT().GetByID(ctx, target.ID).Return(target, nil)

			_, err := d.svc.ResetPassword(ctx, target.ID)
			assert.ErrorIs(t, err, domain.ErrAdminNotFound, role)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		d := newAdmins(t)
		id := uuid.Must(uuid.NewV7())
		d.users.EXPECT().GetByID(ctx, id).Return(domain.User{}, domain.ErrUserNotFound)

		_, err := d.svc.ResetPassword(ctx, id)
		assert.ErrorIs(t, err, domain.ErrAdminNotFound)
	})

	t.Run("repository failure is not masked as not found", func(t *testing.T) {
		d := newAdmins(t)
		id := uuid.Must(uuid.NewV7())
		boom := errors.New("connection refused")
		d.users.EXPECT().GetByID(ctx, id).Return(domain.User{}, boom)

		_, err := d.svc.ResetPassword(ctx, id)
		assert.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestAdminService_SetActive(t *testing.T) {
	ctx := context.Background()

	t.Run("deactivation revokes sessions", func(t *testing.T) {
		d := newAdmins(t)
		admin := user(t, domain.RoleAdmin)
		d.users.EXPECT().GetByID(ctx, admin.ID).Return(admin, nil)
		d.users.EXPECT().Update(ctx, mock.MatchedBy(func(u *domain.User) bool { return !u.Active })).Return(nil)
		d.refresh.EXPECT().RevokeAll(ctx, admin.ID).Return(nil)

		u, err := d.svc.SetActive(ctx, admin.ID, false)
		require.NoError(t, err)
		assert.False(t, u.Active)
	})

	t.Run("activation does not touch sessions", func(t *testing.T) {
		d := newAdmins(t)
		admin := user(t, domain.RoleAdmin)
		admin.Active = false
		d.users.EXPECT().GetByID(ctx, admin.ID).Return(admin, nil)
		d.users.EXPECT().Update(ctx, mock.Anything).Return(nil)

		u, err := d.svc.SetActive(ctx, admin.ID, true)
		require.NoError(t, err)
		assert.True(t, u.Active)
	})

	t.Run("repeating the same state is a no-op", func(t *testing.T) {
		d := newAdmins(t)
		admin := user(t, domain.RoleAdmin)
		d.users.EXPECT().GetByID(ctx, admin.ID).Return(admin, nil)

		_, err := d.svc.SetActive(ctx, admin.ID, true)
		require.NoError(t, err)
	})

	t.Run("superadmin cannot be deactivated through admin management", func(t *testing.T) {
		d := newAdmins(t)
		root := user(t, domain.RoleSuperAdmin)
		d.users.EXPECT().GetByID(ctx, root.ID).Return(root, nil)

		_, err := d.svc.SetActive(ctx, root.ID, false)
		assert.ErrorIs(t, err, domain.ErrAdminNotFound)
	})
}

func TestAdminService_List(t *testing.T) {
	d := newAdmins(t)
	d.users.EXPECT().ListByRole(mock.Anything, domain.RoleAdmin, 10, 10).Return([]domain.User{}, 11, nil)

	page, err := d.svc.List(context.Background(), usecase.PageRequest{Page: 2, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, page.TotalPages())
}
