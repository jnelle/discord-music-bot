package app

import (
	"fmt"
	"jnelle/discord-music-bot/app/commands/play"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

type Command interface {
	Setup() error

	GetSignature() []*discordgo.ApplicationCommand
}

func (a *Application) SetupCommands() error {
	botUserID := a.Bot.Session.State.User.ID
	commands := map[string]Command{
		"play": play.NewCommand(a.Bot, a.YTService, &a.Wg, a.Adapter.DB, a.Adapter.Storage),
	}

	for name, cmd := range commands {
		sigs := cmd.GetSignature()
		for _, sig := range sigs {
			regCmd, err := a.Bot.Session.ApplicationCommandCreate(
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
