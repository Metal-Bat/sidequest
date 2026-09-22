package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Metal-Bat/sidequest/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

func serve(ctx context.Context, pool *pgxpool.Pool, cfg config) error {
	handler, err := web.New(pool, resources, cfg.environment == "production")
	if err != nil {
		return fmt.Errorf("initialize web handler: %w", err)
	}
	server := newServer(cfg.address, handler)
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	slog.Info("server listening", "address", listener.Addr().String())
	result := make(chan error, 1)
	go func() {
		result <- server.Serve(listener)
	}()

	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		return shutdownServer(server)
	}
}

func newServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
}

func shutdownServer(server *http.Server) error {
	slog.Info("server shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown server: %w", err)
	}
	slog.Info("server stopped")
	return nil
}
