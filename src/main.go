package main

import (
	"log/slog"
	"os"
)

func main() {
	configureLogging()
	if err := run(); err != nil {
		slog.Error("sidequest stopped", "error", err)
		os.Exit(1)
	}
}
