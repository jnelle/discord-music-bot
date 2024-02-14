package adapter

import (
	youtubedlp "jnelle/discord-music-bot/adapter/youtube_dlp"
	"jnelle/discord-music-bot/common"
)

type Adapter struct {
	YouTube *youtubedlp.YouTubeRepository
	DB      common.DBService
	Storage common.StorageService
}

func New(cacheDir, proxy string, db common.DBService, storage common.StorageService) *Adapter {
	return &Adapter{
		YouTube: youtubedlp.New(cacheDir, proxy),
		DB:      db,
		Storage: storage,
	}
}
