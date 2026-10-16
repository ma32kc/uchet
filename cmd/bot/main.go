package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/k1ra/uchet/internal/agent"
	"github.com/k1ra/uchet/internal/config"
	"github.com/k1ra/uchet/internal/service"
	"github.com/k1ra/uchet/internal/store"
	"github.com/k1ra/uchet/internal/telegram"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	if err := cfg.ValidateBot(); err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	repos := store.NewRepositories(pool)
	svc := service.NewExpenseService(repos, cfg.Location)
	llm := agent.NewLLMClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMTimeout)
	ag := agent.New(llm, svc, cfg.Location, cfg.MaxToolIters)

	bot, err := telegram.New(cfg.TelegramBotToken, ag, svc)
	if err != nil {
		slog.Error("telegram", "err", err)
		os.Exit(1)
	}

	if err := bot.Run(ctx); err != nil {
		slog.Error("bot stopped", "err", err)
		os.Exit(1)
	}
}
