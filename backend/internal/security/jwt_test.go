package security

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-commerce-backend/internal/domain"
)

const testSecret = "test-secret-with-at-least-32-bytes!!"

func TestJWTManager_RoundTrip(t *testing.T) {
	m := NewJWTManager(testSecret, 15*time.Minute)
	u, err := domain.NewUser("admin@example.com", "h", domain.RoleAdmin)
	require.NoError(t, err)
	u.SetTemporaryPassword("h", time.Now().Add(time.Hour))

	token, exp, err := m.Issue(&u)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), exp, time.Second)

	p, err := m.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, u.ID, p.UserID)
	assert.Equal(t, domain.RoleAdmin, p.Role)
	assert.True(t, p.MustChangePassword)
}

func TestJWTManager_RejectsInvalidTokens(t *testing.T) {
	m := NewJWTManager(testSecret, 15*time.Minute)
	u, err := domain.NewUser("jane@example.com", "h", domain.RoleUser)
	require.NoError(t, err)
	valid, _, err := m.Issue(&u)
	require.NoError(t, err)

	expired := NewJWTManager(testSecret, time.Minute)
	expired.now = func() time.Time { return time.Now().Add(-time.Hour) }
	expiredToken, _, err := expired.Issue(&u)
	require.NoError(t, err)

	otherSecret, _, err := NewJWTManager("another-secret-with-at-least-32-bytes", time.Minute).Issue(&u)
	require.NoError(t, err)

	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": u.ID.String(), "role": "SUPERADMIN", "iss": issuer, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	parts := strings.Split(valid, ".")
	escalated, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": u.ID.String(), "role": "SUPERADMIN", "iss": issuer, "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("guessed-secret"))
	require.NoError(t, err)

	cases := map[string]string{
		"garbage":          "not-a-jwt",
		"expired":          expiredToken,
		"other secret":     otherSecret,
		"alg none":         unsigned,
		"tampered payload": parts[0] + "." + strings.Split(escalated, ".")[1] + "." + parts[2],
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := m.Parse(token)
			assert.ErrorIs(t, err, domain.ErrInvalidToken)
		})
	}
}
