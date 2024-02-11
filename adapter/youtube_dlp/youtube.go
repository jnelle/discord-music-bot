package youtubedlp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

type YouTubeRepository struct {
	cacheDir string
	proxy    string
}

type YouTubeService interface {
	SearchYoutube(ctx context.Context, query string) ([]*Song, error)
	GetYoutubeData(ctx context.Context, videoURL string) (*Song, error)
	GetPlaylistInfo(ctx context.Context, url string, shuffle bool) ([]*Song, error)
	PlayVideo(ctx context.Context, url string) *exec.Cmd
}

var (
	ErrNoSongsFoundInPlaylist = errors.New("no songs found in playlist")
	ErrParseYouTubeResult     = errors.New("failed parsing youtube result: ")
	ErrKillProcess            = errors.New("failed to kill process")
)

func New(cacheDir, proxy string) *YouTubeRepository {
	return &YouTubeRepository{
		cacheDir: cacheDir,
		proxy:    proxy,
	}
}

func (y *YouTubeRepository) SearchYoutube(ctx context.Context, query string) ([]*Song, error) {
	ytdlCtx, ytdlCtxCancel := context.WithTimeout(ctx, time.Minute*1)
	defer ytdlCtxCancel()

	ytdlp := exec.CommandContext(ytdlCtx,
		"yt-dlp",
		"--proxy", y.proxy,
		"ytsearch5:"+query,
		"--dump-json",
		"--flat-playlist",
		"--lazy-playlist",
		"--ies", "youtube:search",
		"--cache-dir", y.cacheDir,
	)

	stdout, err := ytdlp.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := ytdlp.Start(); err != nil {
		return nil, err
	}

	res := []*Song{}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			if err := ytdlp.Process.Kill(); err != nil {
				slog.Error("[youtube.go]", slog.String("error", "failed to kill process "+err.Error()))
			}
			slog.Info("[youtube.go]", slog.String("message", "SearchYoutube canceled via context for query: "+query))
			return nil, ctx.Err()
		default:
			result := &Song{}
			err := json.Unmarshal(scanner.Bytes(), &result)
			if err != nil {
				return nil, errors.Join(ErrParseYouTubeResult, errors.New(scanner.Text()))
			}

			res = append(res, result)
		}

	}

	slog.Info("[youtube.go]", slog.String("SearchYoutube finished query", query))

	if err := ytdlp.Wait(); err != nil {
		slog.Error("[youtube.go]", "SearchYoutube error on wait", "error", err)
		return nil, err
	}

	return res, nil
}

func (y *YouTubeRepository) GetYoutubeData(ctx context.Context, videoURL string) (*Song, error) {
	ytdlCtx, ytdlCtxCancel := context.WithTimeout(ctx, time.Minute*1)
	defer ytdlCtxCancel()

	log := slog.With(slog.Group("[youtube.go]", "song_data", slog.String("query", videoURL)))
	ytdlp := exec.CommandContext(ytdlCtx,
		"yt-dlp",
		"--proxy", y.proxy,
		videoURL,
		"-f", "ba",
		"--dump-json",
		"--no-playlist",
		"--no-progress",
		"--cache-dir", y.cacheDir,
	)

	output, err := ytdlp.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := ytdlp.Start(); err != nil {
		return nil, err
	}

	result := &Song{}
	stdout, err := io.ReadAll(output)
	if err != nil {
		log.Error("error unmarshalling song")
		return nil, err
	}

	err = json.Unmarshal(stdout, &result)
	if err != nil {
		log.Error("error unmarshalling song", slog.String("error", string(stdout)))
		return nil, err
	}

	// Handle context cancellation
	select {
	case <-ctx.Done():
		if err := ytdlp.Process.Kill(); err != nil {
			slog.Error("[youtube.go]", "failed to kill process", slog.String("error", err.Error()))
		}
		slog.Info("[youtube.go]", "GetYoutubeData canceled via context", slog.String("videoUrl", videoURL))
		return nil, ctx.Err()
	default:
		if err := ytdlp.Wait(); err != nil {
			return nil, err
		}
	}
	return result, nil

}

func (y *YouTubeRepository) GetPlaylistInfo(ctx context.Context, url string, shuffle bool) ([]*Song, error) {
	ytdlCtx, ytdlCtxCancel := context.WithTimeout(ctx, time.Minute*5)
	defer ytdlCtxCancel()
	ytdlpArgs := []string{
		"--dump-json",
		"--flat-playlist",
		"--no-progress",
		"--no-warnings",
		"--default-search",
		"ytsearch",
		"--no-call-home",
		"--skip-download",
		"--proxy", y.proxy,
		"--cache-dir", y.cacheDir,
	}
	if shuffle {
		ytdlpArgs = append(ytdlpArgs, "--playlist-random")
	}
	ytdlpArgs = append(ytdlpArgs, url)

	cmd := exec.CommandContext(ytdlCtx, "yt-dlp", ytdlpArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("[youtube.go]", slog.String("error", "error getting playlist info "+err.Error()))
		return nil, err
	}
	var playListSongs []*Song
	songs := bytes.Split(output, []byte("\n"))
	for _, song := range songs {
		s := song
		if len(s) == 0 {
			continue
		}
		// Ignore hidden videos
		if strings.Contains(string(s), "unavailable video is hidden") {
			continue
		}
		songInfo := &Song{}
		errUnmarshal := json.Unmarshal(s, songInfo)
		if errUnmarshal != nil {
			slog.Error("[youtube.go]", slog.String("error", "error unmarshalling song "+errUnmarshal.Error()))
			continue
		}
		if (songInfo.Duration == 0 && !songInfo.IsLive) || songInfo.Title == "[Deleted video]" || songInfo.Title == "[Private video]" {
			slog.Info("[youtube.go]", slog.String("song_title", songInfo.Title),
				slog.String("song_urls", songInfo.Urls),
				slog.String("song_duration", fmt.Sprint(songInfo.Duration)),
				slog.String("song_extractor", songInfo.Extractor),
				slog.String("song_webpage_url", songInfo.WebpageURL),
				slog.String("message", "Skipping invalid playlist song"),
			)
			continue
		}
		slog.Debug("[youtube.go]", slog.String("message", "Found song in playlist: "+songInfo.Title))
		playListSongs = append(playListSongs, songInfo)
	}
	if len(playListSongs) == 0 {
		slog.Debug("[youtube.go]", slog.String("message", "No songs found in playlist"))
		return nil, ErrNoSongsFoundInPlaylist
	}
	return playListSongs, nil
}

func (y *YouTubeRepository) PlayVideo(ctx context.Context, url string) *exec.Cmd {
	return exec.CommandContext(ctx,
		"yt-dlp",
		"--format", "ba",
		url,
		"--cache-dir", y.cacheDir,
		"--proxy", y.proxy,
		"--quiet",
		"--no-warnings",
		"--no-progress",
		"-o", "-",
	)
}
