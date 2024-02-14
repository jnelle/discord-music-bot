package config

import (
	"github.com/caarlos0/env/v10"
)

type Config struct {
	DiscordBotToken                  string `env:"DISCORD_BOT_TOKEN,required"`
	Proxy                            string `env:"HTTP_PROXY"`
	AzureClientID                    string `env:"AZURE_CLIENT_ID,required"`
	AzureCosmosURL                   string `env:"AZURE_COSMOS_URL,required"`
	AzureBlobStorageConnectionString string `env:"AZURE_BLOB_STORAGE_CONNECTION_STRING,required"`
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

func (c *Config) GetAzureClientID() string {
	return c.AzureClientID
}

func (c *Config) GetAzureCosmosURL() string {
	return c.AzureCosmosURL
}

func (c *Config) GetAzureBlobStorageConnectionString() string {
	return c.AzureBlobStorageConnectionString
}
