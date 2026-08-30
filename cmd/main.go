package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"tldr/internal/config"
	"tldr/internal/web"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	initLogger(cfg.Server.LogLevel)

	container := web.NewContainer(cfg)
	router := web.NewRouter(container.Handler)

	addr := ":" + cfg.Server.Port
	slog.Info("server starting", slog.String("addr", addr))
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func initLogger(levelStr string) {
	var level slog.Level
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}
