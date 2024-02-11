package play

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/bwmarrin/discordgo"
)

func autocompleteResponse(choices []*discordgo.ApplicationCommandOptionChoice) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	}
}

func (c *Command) handlePlayAutocomplete(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData()
	queryString := opt.Options[0].StringValue()
	log := c.logger.With("[play.go]", slog.Group("player/autocomplete", slog.String("query", queryString)))

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, 5)
	defer func() {
		if err := session.InteractionRespond(intr.Interaction, autocompleteResponse(choices)); err != nil {
			log.Error("failed to respond", "error", err)
		}
		log.Info("choices collected", "count", len(choices))
	}()

	if len(queryString) < 3 {
		log.Info("search string is less than 3")
		return
	}

	if _, err := url.ParseRequestURI(queryString); err == nil {
		log.Info("skipping autocomplete")
		return
	}

	log.Info("searching for videos")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	data, err := c.youTubeRepository.SearchYoutube(ctx, queryString)
	if err != nil {
		log.Error("error getting youtube data", "err", err)
		return
	}

	for i := range data {
		ytData := data[i]
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%s %s", ytData.Title, ytData.DurationString),
			Value: ytData.OriginalURL,
		})
	}
}
