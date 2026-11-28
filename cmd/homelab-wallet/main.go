package main

import (
	"flag"
	"log/slog"
	"os"

	"homelab-wallet/internal/app"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config.json", "path to the application JSON config")
	logLevelName := flag.String("log-level", os.Getenv("HOMELAB_WALLET_LOG_LEVEL"), "log level: debug, info, warn, or error")
	flag.Parse()

	level := getLogLevel(*logLevelName)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
	slog.Info("starting Homelab Wallet", "version", version)

	application, err := app.New()
	if err == nil {
		err = application.Run(*configPath)
	}
	if err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func getLogLevel(name string) slog.Level {
	level := slog.LevelInfo
	if err := level.UnmarshalText([]byte(name)); err != nil {
		return slog.LevelInfo
	}
	return level
}
