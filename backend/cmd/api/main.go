package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"e-commerce-backend/config"
	"e-commerce-backend/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return app.Run(ctx, cfg)
}
