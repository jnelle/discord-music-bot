package play

import (
	"fmt"
	youtube "jnelle/discord-music-bot/adapter/youtube_dlp"
	"jnelle/discord-music-bot/internal/discord/embed"
	"jnelle/discord-music-bot/internal/discord/format"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	maxFields          int = 10
	maxLineLen         int = 102
	maxTitleLen        int = 44
	queueEmptyErrorMsg     = "There is nothing in the queue."
	responseErrorMsg       = "Failure responding to interaction. See the log for details."
)

func (c *Command) handleQueue(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
	guildID := intr.GuildID

	var queue []*youtube.Video
	if ps := c.playerStorage.Get(guildID); ps != nil {
		queue = ps.Queue()
	}
	if len(queue) == 0 {
		format.DisplayInteractionError(sesh, intr, "There is nothing in the queue.")
		return
	}

	queueLength := len(queue)

	opt := intr.ApplicationCommandData().Options

	totalNumOfFields := 1
	if len(opt) > 0 {
		totalNumOfFields = int(opt[0].IntValue())
	}
	totalLength := 0

	currentVideo := queue[0]
	embed := embed.NewEmbed().
		SetAuthor("Currently playing").
		SetTitle(currentVideo.Title).
		SetThumbnail(currentVideo.Thumbnail).
		SetUrl(currentVideo.GetShortURL()).
		SetDescription(currentVideo.Length).
		SetTimestamp(time.Now().Format(time.RFC3339))

	fieldStart := 1
	fieldEnd := 10
	if queueLength > 1 {
		embed.AddField("In queue", "")

		var sb strings.Builder

		for i := 0; i < totalNumOfFields && fieldStart < queueLength; i++ {
			if fieldEnd > queueLength {
				fieldEnd = queueLength
			}

			for x, video := range queue[fieldStart:fieldEnd] {
				titleLen := len(video.Title)
				if titleLen > maxTitleLen {
					video.Title = video.Title[:maxTitleLen-3] + "..."
				}
				fmt.Fprintf(&sb, "%d: [%s](%s) - (%s)\n", fieldStart+x+1, video.Title, video.GetShortURL(), video.Length)
			}

			embed.AddField("", sb.String())
			fieldStart = fieldEnd
			fieldEnd += 10

			sb.Reset()

		}
	}

	for _, video := range queue {
		if !strings.Contains(video.Length, ":") {
			break
		}
		durations := strings.Split(video.Length, ":")
		minutes, err := strconv.ParseInt(durations[0], 10, 64)
		if err != nil {
			minutes = 0
		}
		seconds, err := strconv.ParseInt(durations[1], 10, 64)
		if err != nil {
			seconds = 0
		}
		totalLength += int(minutes*60 + seconds)
		slog.Debug("[queue.go]", slog.String("DURATION", fmt.Sprint(durations)))
	}

	duration, err := time.ParseDuration(fmt.Sprintf("%ds", totalLength))
	if err != nil {
		slog.Error("[queue.go]", slog.String("error", err.Error()))
	}
	embed.SetFooter(fmt.Sprintf("Total count: %d Total length: %s", queueLength, duration.String()), "")
	err = sesh.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed.MessageEmbed},
		},
	})
	if err != nil {
		c.logger.Error("failure responding to interaction", "error", err)
		format.DisplayInteractionError(sesh, intr, "Failure responding to interaction. See the log for details.")
	}
}
