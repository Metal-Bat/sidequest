package main

import (
	"errors"
	"os"
)

type config struct {
	databaseURL string
	environment string
	address     string
	command     string
}

func loadConfig() (config, error) {
	cfg := config{
		databaseURL: os.Getenv("DATABASE_URL"),
		environment: os.Getenv("APP_ENV"),
		address:     os.Getenv("APP_ADDR"),
	}
	if cfg.databaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	switch cfg.environment {
	case "", "development", "production", "test":
	default:
		return cfg, errors.New("APP_ENV must be development, test, or production")
	}
	if cfg.address == "" {
		cfg.address = ":8000"
	}
	if len(os.Args) > 1 {
		if len(os.Args) != 2 || (os.Args[1] != "migrate" && os.Args[1] != "create-admin") {
			return cfg, errors.New("usage: sidequest [migrate|create-admin]")
		}
		cfg.command = os.Args[1]
	}
	return cfg, nil
}
