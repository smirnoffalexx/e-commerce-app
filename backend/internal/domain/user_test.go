package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser_NormalizesEmail(t *testing.T) {
	u, err := NewUser("  Jane.Doe@Example.COM ", "hash", RoleUser)
	require.NoError(t, err)

	assert.Equal(t, "jane.doe@example.com", u.Email)
	assert.True(t, u.Active)
	assert.False(t, u.MustChangePassword)
}

func TestValidateEmail(t *testing.T) {
	valid := []string{"a@b.co", "jane+shop@example.com", "x@sub.domain.org"}
	for _, email := range valid {
		assert.NoError(t, ValidateEmail(email), email)
	}

	invalid := map[string]string{
		"":                                   CodeRequired,
		"   ":                                CodeRequired,
		"plainaddress":                       CodeInvalidFormat,
		"Jane <jane@example.com>":            CodeInvalidFormat,
		"jane@localhost":                     CodeInvalidFormat,
		"jane@example.com\nBcc: x@evil.com":  CodeInvalidFormat,
		strings.Repeat("a", 250) + "@ex.com": CodeTooLong,
	}
	for email, code := range invalid {
		var verr *ValidationError
		require.ErrorAs(t, ValidateEmail(email), &verr, email)
		assert.Equal(t, code, verr.Fields[0].Code, email)
	}
}

func TestValidatePassword(t *testing.T) {
	assert.NoError(t, ValidatePassword("password", "correct-horse", "jane@example.com"))
	assert.NoError(t, ValidatePassword("password", strings.Repeat("я", 36), ""), "72 bytes of multi-byte runes is allowed")

	cases := map[string]struct {
		password string
		code     string
	}{
		"empty":                {"", CodeRequired},
		"too short":            {"short-pw", "too_short"},
		"short in runes":       {"ёёёёёёёёё", "too_short"},
		"over bcrypt limit":    {strings.Repeat("a", MaxPasswordBytes+1), CodeTooLong},
		"multi-byte over 72 b": {strings.Repeat("я", 37), CodeTooLong},
		"only spaces":          {strings.Repeat(" ", 12), CodeInvalidFormat},
		"same as email":        {"Jane@Example.com", "same_as_email"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var verr *ValidationError
			require.ErrorAs(t, ValidatePassword("password", tc.password, "jane@example.com"), &verr)
			assert.Equal(t, tc.code, verr.Fields[0].Code)
		})
	}
}

func TestUser_TemporaryPassword(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	u, err := NewUser("admin@example.com", "hash", RoleAdmin)
	require.NoError(t, err)

	assert.False(t, u.TempPasswordExpired(now), "regular password never expires")

	u.SetTemporaryPassword("temp-hash", now.Add(24*time.Hour))
	assert.True(t, u.MustChangePassword)
	assert.False(t, u.TempPasswordExpired(now))
	assert.True(t, u.TempPasswordExpired(now.Add(24*time.Hour)), "expires exactly at the deadline")

	u.SetPassword("new-hash")
	assert.False(t, u.MustChangePassword)
	assert.Nil(t, u.TempPasswordExpiresAt)
	assert.False(t, u.TempPasswordExpired(now.Add(48*time.Hour)))
}

func TestRolePermissions(t *testing.T) {
	cases := []struct {
		role    Role
		allowed []Permission
		denied  []Permission
	}{
		{
			role:    RoleUser,
			allowed: []Permission{PermOrdersCreate, PermOrdersReadOwn},
			denied:  []Permission{PermProductsWrite, PermCategoriesWrite, PermProductsImport, PermOrdersReadAll, PermAdminsManage},
		},
		{
			role:    RoleAdmin,
			allowed: []Permission{PermProductsWrite, PermCategoriesWrite, PermProductsImport, PermOrdersReadAll},
			denied:  []Permission{PermAdminsManage, PermOrdersCreate},
		},
		{
			role:    RoleSuperAdmin,
			allowed: []Permission{PermProductsWrite, PermCategoriesWrite, PermProductsImport, PermOrdersReadAll, PermAdminsManage},
			denied:  []Permission{PermOrdersCreate},
		},
		{
			role:   Role("HACKER"),
			denied: []Permission{PermProductsWrite, PermOrdersCreate},
		},
	}

	for _, tc := range cases {
		for _, p := range tc.allowed {
			assert.True(t, tc.role.Can(p), "%s should be able to %s", tc.role, p)
		}
		for _, p := range tc.denied {
			assert.False(t, tc.role.Can(p), "%s should not be able to %s", tc.role, p)
		}
	}
	assert.False(t, Role("HACKER").Valid())
}
