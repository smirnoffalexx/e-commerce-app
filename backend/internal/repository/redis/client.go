package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"e-commerce-backend/config"
)

func NewClient(ctx context.Context, cfg config.Redis) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

type HealthChecker struct {
	client *goredis.Client
}

func NewHealthChecker(client *goredis.Client) *HealthChecker {
	return &HealthChecker{client: client}
}

func (h *HealthChecker) Name() string { return "redis" }

func (h *HealthChecker) Check(ctx context.Context) error {
	return h.client.Ping(ctx).Err()
}
