package app

import (
	youtubedlp "jnelle/discord-music-bot/adapter/youtube_dlp"
	"jnelle/discord-music-bot/internal/discord/bot"
	"sync"
)

type Application struct {
	Wg        sync.WaitGroup
	YTService youtubedlp.YouTubeService
	Bot       *bot.Bot
}

func New(yt *youtubedlp.YouTubeRepository, bot *bot.Bot) *Application {
	return &Application{YTService: yt, Bot: bot}
}
