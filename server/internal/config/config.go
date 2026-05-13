package config

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

// ServerConfig server config
type ServerConfig struct {
	// Port listen port
	Port int `json:"port" mapstructure:"port"`
}

// ProviderConfig provider config
type ProviderConfig struct {
	// Name provider name
	Name string `json:"name" mapstructure:"name"`
	// APIKey provider api key
	APIKey string `json:"api_key" mapstructure:"api_key"`

	// BaseURL provider base url
	BaseURL string `json:"base_url" mapstructure:"base_url"`

	// Type provider type
	Type string `json:"type" mapstructure:"type"`

	// Enabled provider enabled
	Enabled bool `json:"enabled" mapstructure:"enabled"`
}

// ApiKeyConfig api key config
type ApiKeyConfig struct {
	// Name api key name
	Name string `json:"name" mapstructure:"name"`
	// Key api key key
	Key string `json:"key" mapstructure:"key"`

	// Enabled api key enabled
	Enabled bool `json:"enabled" mapstructure:"enabled"`
}

// ModelAlias model alias
type ModelAlias struct {
	// Target target model aliased to
	Target string `json:"target" mapstructure:"target"`
	// Enabled model alias enabled
	Enabled bool `json:"enabled" mapstructure:"enabled"`
}

type Config struct {
	Server ServerConfig `json:"server" mapstructure:"server"`

	Providers map[string]ProviderConfig `json:"providers" mapstructure:"providers"`

	Aliases map[string]ModelAlias `json:"aliases" mapstructure:"aliases"`

	ApiKeys []ApiKeyConfig `json:"apiKeys" mapstructure:"apiKeys"`
}

var GlobalConfig *Config

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config/")

	viper.SetDefault("server.port", "8080")

	if err := viper.ReadInConfig(); err != nil {
		slog.Info("read config failed", "err", err)
		os.Exit(-1)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return err
	}

	GlobalConfig = &config

	slog.Info("debug config", "config", GlobalConfig)
	return nil
}
