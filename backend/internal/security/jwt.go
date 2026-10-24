package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"e-commerce-backend/internal/domain"
)

const issuer = "e-commerce-backend"

type claims struct {
	Role               domain.Role `json:"role"`
	MustChangePassword bool        `json:"pwd_change,omitempty"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (m *JWTManager) Issue(u *domain.User) (string, time.Time, error) {
	now := m.now()
	expiresAt := now.Add(m.ttl)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role:               u.Role,
		MustChangePassword: u.MustChangePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   u.ID.String(),
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

func (m *JWTManager) Parse(token string) (domain.Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return domain.Principal{}, errors.Join(domain.ErrInvalidToken, err)
	}

	id, err := uuid.Parse(c.Subject)
	if err != nil || !c.Role.Valid() {
		return domain.Principal{}, domain.ErrInvalidToken
	}
	return domain.Principal{UserID: id, Role: c.Role, MustChangePassword: c.MustChangePassword}, nil
}
