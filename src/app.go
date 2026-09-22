package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Metal-Bat/sidequest/internal/database"
)

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if cfg.command != "" {
		return runCommand(ctx, pool, cfg.command)
	}
	return serve(ctx, pool, cfg)
}
