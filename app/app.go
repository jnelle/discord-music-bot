package app

import (
	"jnelle/discord-music-bot/adapter"
	youtubedlp "jnelle/discord-music-bot/adapter/youtube_dlp"
	"jnelle/discord-music-bot/internal/discord/bot"
	"sync"
)

type Application struct {
	Wg        sync.WaitGroup
	YTService youtubedlp.YouTubeService
	Bot       *bot.Bot
	Adapter   *adapter.Adapter
}

func New(yt *youtubedlp.YouTubeRepository, bot *bot.Bot, adapter *adapter.Adapter) *Application {
	return &Application{YTService: yt, Bot: bot, Adapter: adapter}
}
