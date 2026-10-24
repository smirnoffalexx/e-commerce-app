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
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}
