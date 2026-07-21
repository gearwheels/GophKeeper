// Package config содержит конфигурацию CLI-клиента.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config хранит настройки клиента.
type Config struct {
	ServerAddress string `mapstructure:"server"`
	Insecure      bool   `mapstructure:"insecure"`
	Token         string `mapstructure:"token"`
	UserID        string `mapstructure:"user_id"`
}

// DefaultConfigPath возвращает путь к файлу конфигурации по умолчанию.
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gophkeeper", "config.yaml")
}

// Load загружает конфигурацию из файла и переменных окружения.
func Load(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(filepath.Join(home, ".gophkeeper"))
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetDefault("server", "https://localhost:8080")

	_ = viper.ReadInConfig()
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal client config: %w", err)
	}
	return &cfg, nil
}

// Save сохраняет конфигурацию в файл.
func Save(cfg *Config, cfgFile string) error {
	if cfgFile == "" {
		cfgFile = DefaultConfigPath()
	}
	if err := os.MkdirAll(filepath.Dir(cfgFile), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	viper.Set("server", cfg.ServerAddress)
	viper.Set("insecure", cfg.Insecure)
	viper.Set("token", cfg.Token)
	viper.Set("user_id", cfg.UserID)
	if err := viper.WriteConfigAs(cfgFile); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
