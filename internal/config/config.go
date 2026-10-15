package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL      string
	HTTPAddr         string
	TelegramBotToken string
	LLMBaseURL       string
	LLMAPIKey        string
	LLMModel         string
	AppTimezone      string
	Location         *time.Location
	PublicWebURL     string
	ShutdownTimeout  time.Duration
	LLMTimeout       time.Duration
	MaxToolIters     int
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:      env("DATABASE_URL", "postgres://uchet:uchet@localhost:5432/uchet?sslmode=disable"),
		HTTPAddr:         env("HTTP_ADDR", ":8080"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		LLMBaseURL:       env("LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMAPIKey:        os.Getenv("LLM_API_KEY"),
		LLMModel:         env("LLM_MODEL", "gpt-4o-mini"),
		AppTimezone:      env("APP_TIMEZONE", "Europe/Moscow"),
		PublicWebURL:     env("PUBLIC_WEB_URL", "http://localhost:3000"),
		ShutdownTimeout:  durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		LLMTimeout:       durationEnv("LLM_TIMEOUT", 60*time.Second),
		MaxToolIters:     intEnv("MAX_TOOL_ITERS", 5),
	}

	loc, err := time.LoadLocation(cfg.AppTimezone)
	if err != nil {
		return Config{}, fmt.Errorf("load timezone %q: %w", cfg.AppTimezone, err)
	}
	cfg.Location = loc

	return cfg, nil
}

func (c Config) ValidateAPI() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	return nil
}

func (c Config) ValidateBot() error {
	if err := c.ValidateAPI(); err != nil {
		return err
	}
	if strings.TrimSpace(c.TelegramBotToken) == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if strings.TrimSpace(c.LLMAPIKey) == "" {
		return fmt.Errorf("LLM_API_KEY is required")
	}
	if strings.TrimSpace(c.LLMBaseURL) == "" {
		return fmt.Errorf("LLM_BASE_URL is required")
	}
	if strings.TrimSpace(c.LLMModel) == "" {
		return fmt.Errorf("LLM_MODEL is required")
	}
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func intEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
