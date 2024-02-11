package playback

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"time"

	youtube "jnelle/discord-music-bot/adapter/youtube_dlp"
	"jnelle/discord-music-bot/utils"

	"github.com/ClintonCollins/dca"
	"github.com/bwmarrin/discordgo"
)

var (
	ErrCauseStop              = errors.New("playback stopped")
	ErrCauseTimeout           = errors.New("playback timed out")
	ErrCauseSkip              = errors.New("playback skipped")
	ErrSkipUnavailable        = errors.New("queue is empty")
	ErrSkipNotPossible        = errors.New("nothing to skip")
	ErrPlayerIsAlreadyRunning = errors.New("player is already running")
	ErrPlaybackIsNotRunning   = errors.New("playback service isn't running")
)

type Player struct {
	vc *discordgo.VoiceConnection

	skipFunc context.CancelCauseFunc

	logger *slog.Logger
	queue  []*youtube.Video

	queuePosition int
	mu            sync.RWMutex

	running           bool
	youtubeRepository youtube.YouTubeService
	wg                *sync.WaitGroup
}

func NewPlayer(vc *discordgo.VoiceConnection, youtubeRepository youtube.YouTubeService, wg *sync.WaitGroup) *Player {
	return &Player{
		vc:            vc,
		queue:         make([]*youtube.Video, 0),
		queuePosition: -1,
		logger: slog.With("player.go",
			slog.Group("player", slog.String("guildID", vc.GuildID), slog.String("channelID", vc.ChannelID))),
		youtubeRepository: youtubeRepository,
		wg:                wg,
	}

}

func (s *Player) EnqueueVideo(video *youtube.Video) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !utils.ArrayContains[*youtube.Video](s.queue, video) {
		s.queue = append(s.queue, video)
	}

	return nil
}

func (s *Player) getNextVideo() *youtube.Video {
	s.mu.Lock()
	defer s.mu.Unlock()

	video := s.queue[s.queuePosition]

	return video
}

func (s *Player) nextVideo() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queuePosition++
	return s.queuePosition < len(s.queue)
}

func (s *Player) waitForVideos(ctx context.Context) {
	for {
		if s.Count() > 0 {
			return
		}

		t := time.After(time.Second)
		select {
		case <-ctx.Done():
			return
		case <-t:
		}
	}
}

func (s *Player) Skip(cnt int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.skipFunc == nil {
		return ErrSkipNotPossible
	}

	s.skipFunc(ErrCauseSkip)
	s.skipFunc = nil

	s.queuePosition += (cnt - 1)

	return nil
}

func (s *Player) Queue() []*youtube.Video {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.queue[s.queuePosition:]
}

func (s *Player) Run(ctx context.Context) error {
	if s.IsRunning() {
		return ErrPlayerIsAlreadyRunning
	}

	s.setRunning(true)
	defer s.setRunning(false)
	s.wg.Done()
	s.waitForVideos(ctx)

	for s.nextVideo() {
		video := s.getNextVideo()

		s.mu.Lock()
		err := s.vc.Speaking(true)
		s.mu.Unlock()
		if err != nil {
			return err
		}

		skipCtx, skipFunc := context.WithCancelCause(ctx)

		s.mu.Lock()
		s.skipFunc = skipFunc
		s.mu.Unlock()

		s.logger.Info("player", "guild", s.vc.GuildID, "video", video.Title)
		err = s.playAudioFromURL(skipCtx, video.URL, s.vc)
		if err != nil && !errors.Is(err, ErrCauseSkip) {
			return err
		}

		s.mu.Lock()
		err = s.vc.Speaking(false)
		s.mu.Unlock()
		if err != nil {
			return err
		}
		s.logger.Info("player", "guild", s.vc.GuildID, "video", video.Title)
	}

	s.logger.Info("queue is empty", "guild", s.vc.GuildID)
	return nil
}

func (s *Player) setRunning(val bool) {
	s.mu.Lock()
	s.running = val
	s.mu.Unlock()
}

func (s *Player) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Player) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.vc.Disconnect()
}

func (s *Player) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.queue)
}

func (s *Player) ChannelID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vc.ChannelID
}

func (s *Player) playAudioFromURL(ctx context.Context, url string, vc *discordgo.VoiceConnection) error {
	ytdlp := s.youtubeRepository.PlayVideo(ctx, url)
	stdout, err := ytdlp.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := ytdlp.StderrPipe()
	if err != nil {
		return err
	}

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 128
	options.Channels = 2
	options.Application = dca.AudioApplicationLowDelay
	options.VolumeFloat = 1.0
	options.VBR = true
	options.Threads = 0
	options.BufferedFrames = 100
	options.PacketLoss = 0
	options.FrameDuration = 20

	session, err := dca.EncodeMem(stdout, options)
	if err != nil {
		return err
	}
	defer session.Cleanup()

	done := make(chan error)
	dca.NewStream(session, vc, done)

	err = ytdlp.Start()
	if err != nil {
		return err
	}

	defer func(ytCmd *exec.Cmd, ytStdout io.ReadCloser, lg *slog.Logger) {
		err := ytCmd.Wait()
		if err != nil {
			s.logger.Error("player", slog.String("error", err.Error()))
		}
		_, _ = io.Copy(io.Discard, ytStdout)
		_ = ytStdout.Close()
	}(ytdlp, stdout, s.logger)

	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			s.logger.Info("player", slog.String("ytdlp stderr", sc.Text()))
		}

		if err := sc.Err(); err != nil {
			s.logger.Error("player", slog.String("ytdlp stderr reader error", err.Error()))
		}
	}()

	select {
	case <-ctx.Done():
		if err := session.Stop(); err != nil {
			s.logger.Error("failed to stop encoding session", slog.String("error", err.Error()))
			return err
		}

		if err := ytdlp.Process.Kill(); err != nil {
			s.logger.Error("failed to kill yt-dlp process", slog.String("error", err.Error()))
			return err
		}

		return context.Cause(ctx)
	case err := <-done:
		if err != nil {
			if err == io.EOF {
				s.logger.Info("player", slog.String("message", "playback finished"))
				return nil
			}

			errBuf, _ := io.ReadAll(stderr)
			s.logger.Error("player", slog.String("error", "error occured while playing audio"), slog.String("ffmpeg messages", session.FFMPEGMessages()), slog.String("ytdlp", string(errBuf)))
			_, _ = io.Copy(io.Discard, stderr)
			_ = stderr.Close()

			return err
		}
	}

	return err
}

func (s *Player) EnqueuePlaylist(videos []*youtube.Video) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return ErrPlaybackIsNotRunning
	}

	s.queue = append(s.queue, videos...)

	return nil
}
