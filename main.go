package main

import (
	"context"
	"jnelle/discord-music-bot/internal/config"
	"jnelle/discord-music-bot/service"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()
	cfg, err := config.New()
	if err != nil {
		slog.Error("[main.go]", slog.String("error", err.Error()))
		os.Exit(1)
	}

	app, err := service.New(*cfg, ctx)
	if err != nil {
		slog.Error("[main.go]", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer app.Bot.Shutdown()

	stop := make(chan os.Signal, 1)
	go func() {
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		slog.Info("[main.go]", slog.String("message", "Press Ctrl+C to exit"))

	}()
	<-stop
}
