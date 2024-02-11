package adapter

import youtubedlp "jnelle/discord-music-bot/adapter/youtube_dlp"

type Adapter struct {
	YouTube *youtubedlp.YouTubeRepository
}

func New(cacheDir, proxy string) *Adapter {
	return &Adapter{
		YouTube: youtubedlp.New(cacheDir, proxy),
	}
}
