package play

import (
	"context"
	"errors"
	"fmt"
	youtube "jnelle/discord-music-bot/adapter/youtube_dlp"
	"jnelle/discord-music-bot/domain/playback"
	"jnelle/discord-music-bot/internal/discord/bot"
	"jnelle/discord-music-bot/internal/discord/embed"
	"jnelle/discord-music-bot/internal/discord/format"
	"jnelle/discord-music-bot/utils"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const interactionSkippedSongResponse string = "Skipped current song."
const interactionNothingToSkip string = "Nothing to skip."

var ErrUserIsNotInGuild = errors.New("user is not in any voice channels")

type Command struct {
	playerStorage     *playback.PlayerStorage
	logger            *slog.Logger
	bot               *bot.Bot
	youTubeRepository youtube.YouTubeService
	wg                *sync.WaitGroup
}

func NewCommand(bot *bot.Bot, YouTubeRepository youtube.YouTubeService, wg *sync.WaitGroup) *Command {
	return &Command{
		playerStorage:     playback.NewManager(),
		logger:            slog.Default(),
		bot:               bot,
		youTubeRepository: YouTubeRepository,
		wg:                wg,
	}
}

func (c *Command) Setup() error {
	c.bot.Session.AddHandler(func(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
		if intr.Type != discordgo.InteractionApplicationCommand && intr.Type != discordgo.InteractionApplicationCommandAutocomplete {
			return
		}
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			opt := intr.ApplicationCommandData()
			switch opt.Name {
			case "play":
				c.handlePlay(sesh, intr)
			case "skip":
				c.handleSkip(sesh, intr)
			case "queue":
				c.handleQueue(sesh, intr)
				// case "playlist":
				// 	c.handlePlayPlaylist(sesh, intr)
			}
		}()

	})

	return nil
}

func (c *Command) GetSignature() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "play",
			Description: "Play a youtube video",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:         "search",
					Description:  "Youtube link or search query",
					Type:         discordgo.ApplicationCommandOptionString,
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "playlist",
			Description: "Play a youtube video playlist",
			Type:        discordgo.ChatApplicationCommand,
		},
		{
			Name:        "stop",
			Description: "Stop audio playback",
			Type:        discordgo.ChatApplicationCommand,
		},
		{
			Name:        "skip",
			Description: "Skip current song",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "amount",
					Description: "Amount of songs to skip",
					Type:        discordgo.ApplicationCommandOptionInteger,
					MinValue:    utils.ToPtr[float64](1.0),
				},
			},
		},
		{
			Name:        "queue",
			Description: "View the current song queue",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "amount",
					Description: "Amount of fields to show. Each field can contain up to 10 songs.",
					Type:        discordgo.ApplicationCommandOptionInteger,
					MinValue:    utils.ToPtr[float64](1.0),
				},
			},
		},
	}
}

var allowedHosts = []string{
	"www.youtube.com",
	"youtube.com",
	"youtu.be",
}

// func (c *Command) handlePlayPlaylist(session *discordgo.Session, intr *discordgo.InteractionCreate) {
// 	opt := intr.ApplicationCommandData()
// 	queryString := opt.Options[0].StringValue()

// 	log := c.logger.With("query", queryString)
// 	url, err := url.ParseRequestURI(queryString)
// 	if err != nil {
// 		log.Error("error parsing url", "err", err)
// 		format.DisplayInteractionError(session, intr, "Error parsing url!")
// 		return
// 	}
// 	if !utils.ArrayContains[string](allowedHosts, url.Host) {
// 		log.Error("error parsing url: incorrect domain " + url.Host)
// 		format.DisplayInteractionError(session, intr, "Domain must be `youtube.com`, `youtu.be` or `www.youtube.com`")
// 		return
// 	}

// 	playlistURL := url.String()
// 	log.Info("requesting playlist data", "url", playlistURL)

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	playlist, err := c.YouTubeRepository.GetPlaylistInfo(ctx, playlistURL, false)
// 	if err != nil {
// 		log.Error("error getting youtube data", "err", err)
// 		format.DisplayInteractionError(session, intr, "Error getting video data from youtube. See the log for details.")
// 		return
// 	}
// 	var player *playback.Player
// 	if ps := c.playerStorage.Get(intr.GuildID); ps != nil {
// 		log.Info("get stored player")
// 		player = ps
// 	}
// }

func (c *Command) handlePlay(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	if intr.Type == discordgo.InteractionApplicationCommandAutocomplete {
		c.handlePlayAutocomplete(session, intr)
		return
	}
	opt := intr.ApplicationCommandData()
	queryString := opt.Options[0].StringValue()

	log := c.logger.With("[play.go]", slog.String("query", queryString))

	url, err := url.ParseRequestURI(queryString)
	if err != nil {
		log.Error("error parsing url", "err", err)
		format.DisplayInteractionError(session, intr, "Error parsing url!")
		return
	}

	if !utils.ArrayContains[string](allowedHosts, url.Host) {
		log.Error("error parsing url: incorrect domain")
		format.DisplayInteractionError(session, intr, "Domain must be `youtube.com`, `youtu.be` and etc.")
		return
	}

	videoURL := url.String()

	log.Info("requesting video data", "url", videoURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	data, err := c.youTubeRepository.GetYoutubeData(ctx, videoURL)
	if err != nil {
		log.Error("error getting youtube data", "err", err)
		format.DisplayInteractionError(session, intr, "Error getting video data from youtube. See the log for details.")
		return
	}

	err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Error("failure responding to interaction", "err", err)
		return
	}

	if err := c.isUserAndBotInSameChannel(session, intr.GuildID, intr.Member.User.ID); err != nil {
		switch {
		case errors.Is(err, errUserNotInAnyChannel):
			fallthrough
		case errors.Is(err, errUserNotInBotsChannel):
			format.DisplayInteractionError(session, intr, "You must be in the same voice channel as the bot to use this command.")
			return
		}
	}

	var player *playback.Player
	if ps := c.playerStorage.Get(intr.GuildID); ps != nil {
		log.Info("get stored player")
		player = ps
	} else {
		log.Info("creating new player")

		channelID, err := c.getUserChannelID(session, intr.GuildID, intr.Member.User.ID)
		if err != nil {
			log.Error("failure getting channel id", "err", err)
			format.DisplayInteractionError(session, intr, "You must be in a voice channel to use this command.")
			return
		}

		voice, err := session.ChannelVoiceJoin(intr.GuildID, channelID, false, true)
		if err != nil {
			if voice != nil {
				voice.Close()
			}
			log.Error("failure joining voice channel", "channelId", channelID, "err", err)
			format.DisplayInteractionError(session, intr, "Error joining voice channel.")
			return
		}

		c.wg.Add(1)
		player = c.setupPlayer(session, intr, voice, log)
		if player == nil {
			if voice != nil {
				voice.Close()
			}
			format.DisplayInteractionError(session, intr, "Error starting playback.")
			return
		}
	}

	video := c.toYouTubeModel(videoURL, data.Title, data.Thumbnail, data.DurationString, data.ID)
	if err := player.EnqueueVideo(video); err != nil {
		log.Error("Failed to enqueue video", slog.String("error", err.Error()))
		return
	}
	duration, err := time.ParseDuration(fmt.Sprintf("%vs", data.Duration))
	if err != nil {
		log.Error("Duration doesnt exist", slog.String("error", err.Error()))
	}

	log.Info("added video to player", "video", video.Title)

	embed := embed.NewEmbed().
		SetAuthor("Added to queue").
		SetTitle(video.Title).
		SetUrl(video.GetShortURL()).
		SetThumbnail(video.Thumbnail).
		SetDescription(video.Length).
		SetFooter(fmt.Sprintf("Queue length: %d Queue duration: %s", player.Count(), duration.String()), "").
		MessageEmbed

	_, err = session.FollowupMessageCreate(intr.Interaction, false, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Error("failure creating followup message to interaction", slog.String("err", err.Error()))
		return
	}

}

func (c *Command) setupPlayer(session *discordgo.Session, intr *discordgo.InteractionCreate, voice *discordgo.VoiceConnection, log *slog.Logger) *playback.Player {
	player := playback.NewPlayer(voice, c.youTubeRepository, c.wg)
	if err := c.playerStorage.Add(intr.GuildID, player); err != nil {
		log.Error("error adding a new playback service", "guildId", intr.GuildID, "err", err)
		return nil
	}

	// Run the service
	go func(guildId string) {
		playbackContext, playbackCancel := context.WithCancelCause(context.Background())
		stopHandlerCancel := createStopHandler(session, playbackCancel, guildId)

		// Setup service timeout ticker, in case bot is left alone in a channel
		go func(channelId string) {
			tick := time.NewTicker(time.Minute)
			defer tick.Stop()
			for {
				select {
				case <-playbackContext.Done():
					return
				case <-tick.C:
					if last, err := c.isBotLastInVoiceChannel(session, guildId, channelId); err != nil {
						log.Error("timeout ticker error", "error", err)
						return
					} else if last {
						playbackCancel(playback.ErrCauseTimeout)
						return
					}

				}
			}
		}(player.ChannelID())

		err := player.Run(playbackContext)
		if err != nil && !errors.Is(playbackContext.Err(), context.Canceled) {
			log.Error("playback error has occured", "err", err)
		}
		stopHandlerCancel()

		if err := player.Cleanup(); err != nil {
			log.Error("failure to close player", "err", err)
		}

		log.Info("deleting player", "guildId", guildId)
		if err := c.playerStorage.Delete(guildId); err != nil {
			log.Error("error deleting player", "guildId", guildId, "err", err)
		}
	}(intr.GuildID)

	return player
}

func createStopHandler(sesh *discordgo.Session, cancel context.CancelCauseFunc, guildID string) func() {
	return sesh.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.GuildID != guildID {
			return
		}

		opt := i.ApplicationCommandData()
		if opt.Name != "stop" {
			return
		}

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Stopping playback.",
			},
		})
		if err != nil {
			format.DisplayInteractionError(s, i, "Failure responding to interaction. See the log for details.")
		}

		cancel(playback.ErrCauseStop)
	})
}

func (c *Command) handleSkip(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
	guildID := intr.GuildID
	userID := intr.Member.User.ID

	if err := c.isUserAndBotInSameChannel(sesh, guildID, userID); err != nil {
		switch {
		case errors.Is(err, errUserNotInAnyChannel):
			fallthrough
		case errors.Is(err, errUserNotInBotsChannel):
			format.DisplayInteractionError(sesh, intr, "You must be in the same voice channel as the bot to use this command.")
			return
		case errors.Is(err, errBotIsNotInAnyChannel):
			format.DisplayInteractionError(sesh, intr, interactionNothingToSkip)
			return
		}
	}

	opt := intr.ApplicationCommandData().Options

	skipAmount := int64(1)
	if len(opt) > 0 {
		skipAmount = intr.ApplicationCommandData().Options[0].IntValue()
	}

	if ps := c.playerStorage.Get(guildID); ps != nil {
		err := ps.Skip(int(skipAmount))
		if errors.Is(err, playback.ErrSkipUnavailable) {
			format.DisplayInteractionError(sesh, intr, interactionNothingToSkip)
			return
		}
	} else {
		format.DisplayInteractionError(sesh, intr, interactionNothingToSkip)
		return
	}

	err := sesh.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: interactionSkippedSongResponse,
		},
	})
	if err != nil {
		slog.Error("[command.go]", "failure responding to interaction", slog.String("error", err.Error()))
		format.DisplayInteractionError(sesh, intr, "Failure responding to interaction. See the log for details.")
	}
}

var (
	errBotIsNotInAnyChannel = errors.New("bot isn't in any channels")
	errUserNotInAnyChannel  = errors.New("you must be in a voice channel")
	errUserNotInBotsChannel = errors.New("you must be in the same channel as the bot")
	errFailedGetGuild       = errors.New("failure getting guild: ")
)

func (c *Command) isUserAndBotInSameChannel(sesh *discordgo.Session, guildID string, userID string) error {
	botUserID := sesh.State.User.ID

	botChannelID, err := c.getUserChannelID(sesh, guildID, botUserID)
	if err != nil {
		return errBotIsNotInAnyChannel
	}

	channelID, err := c.getUserChannelID(sesh, guildID, userID)
	if err != nil {
		return errUserNotInAnyChannel
	}

	if channelID != botChannelID {
		return errUserNotInBotsChannel
	}

	return nil
}

func (c *Command) getUserChannelID(sesh *discordgo.Session, guildID string, userID string) (string, error) {
	var channelID string

	g, err := sesh.State.Guild(guildID)
	if err != nil {
		if !errors.Is(err, discordgo.ErrStateNotFound) {
			return channelID, errors.Join(errFailedGetGuild, err)
		}

		g, err = sesh.Guild(guildID)
		if err != nil {
			return channelID, errors.Join(errFailedGetGuild, err)
		}
	}

	c.logger.Info("guild acquired", "guildId", g.ID, "name", g.Name)

	for _, vs := range g.VoiceStates {
		if vs.UserID == userID {
			c.logger.Info("user found in channel", "usr", vs.UserID, "chn", vs.ChannelID)
			channelID = vs.ChannelID
			break
		}
	}
	if len(channelID) == 0 {
		return channelID, ErrUserIsNotInGuild
	}

	return channelID, nil
}

func (c *Command) isBotLastInVoiceChannel(sesh *discordgo.Session, guildID string, channelID string) (bool, error) {
	g, err := sesh.State.Guild(guildID)
	if err != nil {
		return false, fmt.Errorf("failure getting guild: %w", err)
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID != sesh.State.User.ID && vs.ChannelID == channelID {
			return false, nil
		}
	}

	return true, nil
}

func (c *Command) toYouTubeModel(videoURL, title, thumbnail, length, ID string) *youtube.Video {
	return &youtube.Video{
		ID:        ID,
		Title:     title,
		Thumbnail: "https://i.ytimg.com/vi/" + ID + "/maxresdefault.jpg",
		Length:    length,
		URL:       videoURL,
	}
}
