package redis

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"e-commerce-backend/internal/usecase"
)

const checkoutLockPrefix = "checkout:lock:"

var releaseLockScript = goredis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
end
return 0
`)

type CheckoutLocker struct {
	client *goredis.Client
}

func NewCheckoutLocker(client *goredis.Client) *CheckoutLocker {
	return &CheckoutLocker{client: client}
}

func (l *CheckoutLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", false, fmt.Errorf("generate lock token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	ok, err := l.client.SetNX(ctx, checkoutLockPrefix+key, token, ttl).Result()
	if err != nil {
		return "", false, fmt.Errorf("acquire checkout lock: %w", err)
	}
	return token, ok, nil
}

func (l *CheckoutLocker) Release(ctx context.Context, key, token string) error {
	if err := releaseLockScript.Run(ctx, l.client, []string{checkoutLockPrefix + key}, token).Err(); err != nil {
		return fmt.Errorf("release checkout lock: %w", err)
	}
	return nil
}

var _ usecase.CheckoutLocker = (*CheckoutLocker)(nil)
