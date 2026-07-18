package main

import (
	"fmt"
	"os"
	"time"
)

const (
	// defaultListenAddr is used when LISTEN_ADDR is not set.
	defaultListenAddr = ":8080"

	// longPollerTimeout is how long the Telegram long-poller waits for updates.
	longPollerTimeout = 10 * time.Second

	// dbQueryTimeout bounds individual database calls made from handlers.
	dbQueryTimeout = 5 * time.Second

	// remindSendDelay is the pause between outgoing messages during a reminder
	// pass, to stay within Telegram's rate limits.
	remindSendDelay = 100 * time.Millisecond

	// httpShutdownTimeout bounds how long we wait for in-flight HTTP requests
	// to drain during graceful shutdown.
	httpShutdownTimeout = 10 * time.Second

	// ackButtonUnique is the callback unique identifier for the "Done" button.
	ackButtonUnique = "ack"
)

// config holds all environment-derived settings for the bot service.
type config struct {
	BotToken   string
	DBURL      string
	ListenAddr string
}

// loadConfig reads configuration from the environment and validates it.
// The process should exit immediately if required values are missing.
func loadConfig() (config, error) {
	cfg := config{
		BotToken:   os.Getenv("BOT_TOKEN"),
		DBURL:      os.Getenv("DB_URL"),
		ListenAddr: os.Getenv("LISTEN_ADDR"),
	}

	if cfg.BotToken == "" {
		return config{}, fmt.Errorf("BOT_TOKEN is required")
	}
	if cfg.DBURL == "" {
		return config{}, fmt.Errorf("DB_URL is required")
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}

	return cfg, nil
}
