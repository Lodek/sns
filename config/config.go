package config

import (
	"os"
	"time"
)

type Config struct {
	GRPCAddr          string
	BadgerDir         string
	TelegramToken     string
	TelegramChatID    string
	DiscordWebhookURL string
	WorkerInterval    time.Duration
}

func Load() Config {
	cfg := Config{
		GRPCAddr:          envOr("GRPC_ADDR", ":9090"),
		BadgerDir:         envOr("BADGER_DIR", "./data/badger"),
		TelegramToken:     os.Getenv("TELEGRAM_TOKEN"),
		TelegramChatID:    os.Getenv("TELEGRAM_CHAT_ID"),
		DiscordWebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),
		WorkerInterval:    time.Minute,
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
