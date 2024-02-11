package service

import (
	"jnelle/discord-music-bot/adapter"
	"jnelle/discord-music-bot/app"
	"jnelle/discord-music-bot/internal/config"
	"jnelle/discord-music-bot/internal/discord/bot"
	"log/slog"
	"os"
)

func New(cfg config.Config) (*app.Application, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	bot, err := bot.New(cfg.GetDiscordBotToken())
	if err != nil {
		slog.Error("error while creating session: %v", err)
		return nil, err
	}
	adapter := adapter.New(cacheDir, cfg.GetProxy())
	app := app.New(adapter.YouTube, bot)

	err = bot.OpenConnection()
	if err != nil {
		slog.Error("error while opening session: %v", err)
		return nil, err
	}

	err = app.SetupCommands(bot)
	if err != nil {
		slog.Error("error while creating commands: %v", err)
		return nil, err
	}

	app.Wg.Wait()

	return app, nil
}
