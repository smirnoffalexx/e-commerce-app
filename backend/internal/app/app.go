package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"e-commerce-backend/config"
	httpdelivery "e-commerce-backend/internal/delivery/http"
	"e-commerce-backend/internal/delivery/http/handler"
	"e-commerce-backend/internal/repository/postgres"
	"e-commerce-backend/internal/repository/redis"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 15 * time.Second
	readinessTimeout  = 2 * time.Second
	connectTimeout    = 10 * time.Second
)

func Run(ctx context.Context, cfg config.Config) error {
	log := newLogger(cfg)
	slog.SetDefault(log)

	connectCtx, cancelConnect := context.WithTimeout(ctx, connectTimeout)
	defer cancelConnect()

	db, err := postgres.NewDB(connectCtx, cfg.Postgres.DSN(), log)
	if err != nil {
		return err
	}
	defer func() {
		if err := postgres.Close(db); err != nil {
			log.Error("close postgres", slog.Any("error", err))
		}
	}()

	rdb, err := redis.NewClient(connectCtx, cfg.Redis)
	if err != nil {
		return err
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Error("close redis", slog.Any("error", err))
		}
	}()

	router := httpdelivery.NewRouter(httpdelivery.RouterDeps{
		Logger:     log,
		Production: cfg.IsProduction(),
		Health: handler.NewHealthHandler(readinessTimeout,
			postgres.NewHealthChecker(db),
			redis.NewHealthChecker(rdb),
		),
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		BaseContext:       func(_ net.Listener) context.Context { return context.WithoutCancel(ctx) },
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("http server started", slog.String("addr", srv.Addr))
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}

	log.Info("stopped")
	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
