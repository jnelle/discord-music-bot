package app

import (
	"fmt"
	"jnelle/discord-music-bot/app/commands/play"
	"jnelle/discord-music-bot/internal/discord/bot"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

type Command interface {
	Setup() error

	GetSignature() []*discordgo.ApplicationCommand
}

func (a *Application) SetupCommands(bot *bot.Bot) error {
	botUserID := bot.Session.State.User.ID
	commands := map[string]Command{
		"play": play.NewCommand(bot, a.YTService, &a.Wg),
	}

	for name, cmd := range commands {
		sigs := cmd.GetSignature()
		for _, sig := range sigs {
			regCmd, err := bot.Session.ApplicationCommandCreate(
				botUserID, "", sig,
			)
			if err != nil {
				return fmt.Errorf("[commands.go] message=FAILTED_TO_REGISTER %s: %w", sig.Name, err)
			}
			slog.Info("[commands.go]", slog.String("message", "registered "+regCmd.Name))
		}
		if err := cmd.Setup(); err != nil {
			return fmt.Errorf("[commands.go] message=FAILED_TO_SETUP_COMMAND %s: %w", name, err)
		}
	}
	return nil
}
