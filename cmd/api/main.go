package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k1ra/uchet/internal/api"
	"github.com/k1ra/uchet/internal/config"
	"github.com/k1ra/uchet/internal/httpserver"
	"github.com/k1ra/uchet/internal/service"
	"github.com/k1ra/uchet/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	if err := cfg.ValidateAPI(); err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	migrationsDir := envOr("MIGRATIONS_DIR", "migrations")
	if abs, err := filepath.Abs(migrationsDir); err == nil {
		migrationsDir = abs
	}
	if err := store.Migrate(cfg.DatabaseURL, migrationsDir); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	repos := store.NewRepositories(pool)
	svc := service.NewExpenseService(repos, cfg.Location)
	h := api.NewHandler(svc)
	router := api.NewRouter(h, &poolPinger{pool: pool})

	if err := httpserver.Run(ctx, cfg.HTTPAddr, router, cfg.ShutdownTimeout); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

type poolPinger struct {
	pool *pgxpool.Pool
}

func (p *poolPinger) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
