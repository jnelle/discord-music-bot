package config

import (
	"github.com/caarlos0/env/v10"
)

type Config struct {
	DiscordBotToken string `env:"DISCORD_BOT_TOKEN,required"`
	Proxy           string `env:"HTTP_PROXY"`
}

func New() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) GetDiscordBotToken() string {
	return c.DiscordBotToken
}

func (c *Config) GetProxy() string {
	return c.Proxy
}
