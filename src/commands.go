package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Metal-Bat/sidequest/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

func runCommand(ctx context.Context, pool *pgxpool.Pool, command string) error {
	slog.Info("command started", "command", command)
	var err error
	switch command {
	case "create-admin":
		err = createAdmin(ctx, pool)
	case "migrate":
		migrationCtx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		err = database.Migrate(migrationCtx, pool, resources)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", command, err)
	}
	slog.Info("command completed", "command", command)
	return nil
}
