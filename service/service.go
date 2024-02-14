package service

import (
	"context"
	"jnelle/discord-music-bot/adapter"
	"jnelle/discord-music-bot/adapter/azure"
	"jnelle/discord-music-bot/app"
	db "jnelle/discord-music-bot/internal/azure"
	"jnelle/discord-music-bot/internal/config"
	"jnelle/discord-music-bot/internal/discord/bot"
	"log/slog"
	"os"
)

func New(cfg config.Config, ctx context.Context) (*app.Application, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	bot, err := bot.New(cfg.GetDiscordBotToken())
	if err != nil {
		slog.Error("[service.go]", "error while creating session: %v", err)
		return nil, err
	}
	azClient := db.NewAzureClient()
	err = azClient.NewAzCosmos(cfg.AzureClientID, cfg.AzureCosmosURL)
	if err != nil {
		slog.Error("[service.go]", "error while opening session: %v", err)
		return nil, err
	}

	containerClient, _ := azClient.CreateContainer(ctx)
	cosmosDB := azure.NewCosmosDB(containerClient)
	azClient.NewAzBlobStorage(cfg.GetAzureBlobStorageConnectionString())
	storage := azure.NewStorageRepository(azClient.GetAzBlobClient())
	adapter := adapter.New(cacheDir, cfg.GetProxy(), cosmosDB, storage)
	app := app.New(adapter.YouTube, bot, adapter)

	err = bot.OpenConnection()
	if err != nil {
		slog.Error("[service.go]", "error while opening session: %v", err)
		return nil, err
	}

	err = app.SetupCommands()
	if err != nil {
		slog.Error("[service.go]", "error while creating commands: %v", err)
		return nil, err
	}

	app.Wg.Wait()

	return app, nil
}
