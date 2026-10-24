package redis

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"e-commerce-backend/internal/domain"
	"e-commerce-backend/internal/usecase"
)

const (
	refreshKeyPrefix     = "auth:refresh:"
	userSessionKeyPrefix = "auth:user_sessions:"
)

var revokeAllScript = goredis.NewScript(`
local hashes = redis.call('SMEMBERS', KEYS[1])
for _, h in ipairs(hashes) do
	redis.call('DEL', ARGV[1] .. h)
end
redis.call('DEL', KEYS[1])
return #hashes
`)

type RefreshTokenStore struct {
	client *goredis.Client
}

func NewRefreshTokenStore(client *goredis.Client) *RefreshTokenStore {
	return &RefreshTokenStore{client: client}
}

func (s *RefreshTokenStore) Issue(ctx context.Context, userID uuid.UUID, ttl time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := hashToken(token)
	sessions := userSessionKeyPrefix + userID.String()

	_, err := s.client.TxPipelined(ctx, func(p goredis.Pipeliner) error {
		p.Set(ctx, refreshKeyPrefix+hash, userID.String(), ttl)
		p.SAdd(ctx, sessions, hash)
		p.Expire(ctx, sessions, ttl)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}
	return token, nil
}

func (s *RefreshTokenStore) Consume(ctx context.Context, token string) (uuid.UUID, error) {
	if token == "" {
		return uuid.Nil, domain.ErrInvalidToken
	}
	hash := hashToken(token)

	raw, err := s.client.GetDel(ctx, refreshKeyPrefix+hash).Result()
	if errors.Is(err, goredis.Nil) {
		return uuid.Nil, domain.ErrInvalidToken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("consume refresh token: %w", err)
	}

	userID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, domain.ErrInvalidToken
	}
	if err := s.client.SRem(ctx, userSessionKeyPrefix+raw, hash).Err(); err != nil {
		return uuid.Nil, fmt.Errorf("untrack refresh token: %w", err)
	}
	return userID, nil
}

func (s *RefreshTokenStore) Revoke(ctx context.Context, token string) error {
	_, err := s.Consume(ctx, token)
	if errors.Is(err, domain.ErrInvalidToken) {
		return nil
	}
	return err
}

func (s *RefreshTokenStore) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	err := revokeAllScript.Run(ctx, s.client, []string{userSessionKeyPrefix + userID.String()}, refreshKeyPrefix).Err()
	if err != nil {
		return fmt.Errorf("revoke refresh tokens: %w", err)
	}
	return nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

var _ usecase.RefreshTokenStore = (*RefreshTokenStore)(nil)
