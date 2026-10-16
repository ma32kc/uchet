package config

import (
	"testing"
)

func TestValidateBotRequiresSecrets(t *testing.T) {
	cfg := Config{
		DatabaseURL: "postgres://x",
		LLMBaseURL:  "https://api.groq.com/openai/v1",
		LLMModel:    "openai/gpt-oss-120b",
	}
	if err := cfg.ValidateBot(); err == nil {
		t.Fatal("expected error without token/key")
	}

	cfg.TelegramBotToken = "token"
	cfg.LLMAPIKey = "key"
	if err := cfg.ValidateBot(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateAPI(t *testing.T) {
	cfg := Config{}
	if err := cfg.ValidateAPI(); err == nil {
		t.Fatal("expected error")
	}
	cfg.DatabaseURL = "postgres://x"
	if err := cfg.ValidateAPI(); err != nil {
		t.Fatal(err)
	}
}
