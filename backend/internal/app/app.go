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
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 15 * time.Second
	readinessTimeout  = 2 * time.Second
)

func Run(ctx context.Context, cfg config.Config) error {
	log := newLogger(cfg)
	slog.SetDefault(log)

	router := httpdelivery.NewRouter(httpdelivery.RouterDeps{
		Logger:     log,
		Production: cfg.IsProduction(),
		Health:     handler.NewHealthHandler(readinessTimeout),
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
