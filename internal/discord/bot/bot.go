package bot

import (
	"log/slog"
	"os"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session *discordgo.Session
}

func New(botToken string) (*Bot, error) {
	discord, err := discordgo.New("Bot " + botToken)
	if err != nil {
		slog.Error("[bot.go]", slog.String("error", err.Error()))
		return nil, err
	}
	slog.Info("[bot.go]", slog.String("message", "Starting bot..."))

	return &Bot{
		Session: discord,
	}, nil
}

func (bot *Bot) OpenConnection() error {
	return bot.Session.Open()
}

func (bot *Bot) CreateCommands(commands []*discordgo.ApplicationCommand) error {
	for _, v := range commands {
		_, err := bot.Session.ApplicationCommandCreate(bot.Session.State.User.ID, "", v)
		if err != nil {
			slog.Error("[bot.go]", "error while creating command", slog.String("cmd", v.Name), slog.String("error", err.Error()))
			return err
		}

		slog.Info("[bot.go]", slog.String("created command", "cmd "+v.Name))
	}

	return nil
}

func (bot *Bot) Shutdown() {
	slog.Info("[bot.go]", slog.String("message", "Shutting down..."))
	registeredCommands, err := bot.Session.ApplicationCommands(bot.Session.State.User.ID, "")
	if err != nil {
		slog.Error("[bot.go]", slog.String("error", err.Error()))
		os.Exit(1)
	}

	for _, v := range registeredCommands {
		err := bot.Session.ApplicationCommandDelete(bot.Session.State.User.ID, "", v.ID)
		if err != nil {
			slog.Error("[bot.go]", "cannot delete command", slog.String("cmd", v.Name), slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

}
