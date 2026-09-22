package main

import (
	"log/slog"
	"os"
)

func configureLogging() {
	var handler slog.Handler = slog.NewTextHandler(os.Stderr, nil)
	if os.Getenv("APP_ENV") == "production" {
		handler = slog.NewJSONHandler(os.Stderr, nil)
	}
	slog.SetDefault(slog.New(handler).With("service", "sidequest"))
}
