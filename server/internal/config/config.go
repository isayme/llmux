package config

import (
	"log/slog"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ServerConfig server config
type ServerConfig struct {
	// Port listen port
	Port int `json:"port" mapstructure:"port"`

	// MasterKey master key used to access web UI APIs
	MasterKey string `json:"master_key" mapstructure:"master_key"`
}

// ProviderConfig provider config
type ProviderConfig struct {
	// ID provider id
	ID string `json:"id" mapstructure:"id"`

	// Name provider display name, display ID if name is empty
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
	// Name model alias name
	Name string `json:"name" mapstructure:"name"`
	// Provider target provider
	Provider string `json:"provider" mapstructure:"provider"`
	// Model model of target provider
	Model string `json:"model" mapstructure:"model"`
	// Enabled model alias enabled
	Enabled bool `json:"enabled" mapstructure:"enabled"`
}

type Config struct {
	Server ServerConfig `json:"server" mapstructure:"server"`

	Providers map[string]*ProviderConfig `json:"providers" mapstructure:"providers"`

	Aliases map[string]*ModelAlias `json:"aliases" mapstructure:"aliases"`

	APIKeys []*ApiKeyConfig `json:"api_keys" mapstructure:"api_keys"`
}

var globalConfig *Config

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config/")

	viper.SetDefault("server.port", "8080")

	if err := viper.ReadInConfig(); err != nil {
		slog.Info("read config failed", "err", err)
		return err
	}

	if err := readConfig(); err != nil {
		slog.Info("read config failed", "err", err)
		return err
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		slog.Info("config changed")

		if err := readConfig(); err != nil {
			slog.Error("unmarshal config failed", "err", err)
		} else {
			slog.Info("config updated")
		}
	})

	slog.Info("debug config", "config", globalConfig)
	return nil
}

func readConfig() error {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return err
	}

	globalConfig = &config

	for providerId, provider := range globalConfig.Providers {
		if provider.ID == "" {
			provider.ID = providerId
		}
	}

	for aliasId, alias := range globalConfig.Aliases {
		if alias.Name == "" {
			alias.Name = aliasId
		}
	}

	return nil
}

func Get() *Config {
	return globalConfig
}
