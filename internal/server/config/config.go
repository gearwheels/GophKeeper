// Package config содержит конфигурацию сервера.
package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config хранит все параметры запуска сервера.
type Config struct {
	ServerAddress string
	DatabaseURI   string
	JWTSecret     string
	TLSCertFile   string
	TLSKeyFile    string
	LogLevel      string
}

// Load читает конфигурацию из переменных окружения и .env-файла.
// Переменные окружения имеют приоритет над .env-файлом.
func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()

	cfg := &Config{
		ServerAddress: getenv("SERVER_ADDRESS", ":8080"),
		DatabaseURI:   getenv("DATABASE_URI", ""),
		JWTSecret:     getenv("JWT_SECRET", ""),
		TLSCertFile:   getenv("TLS_CERT_FILE", ""),
		TLSKeyFile:    getenv("TLS_KEY_FILE", ""),
		LogLevel:      getenv("LOG_LEVEL", "info"),
	}

	if cfg.DatabaseURI == "" {
		return nil, fmt.Errorf("DATABASE_URI is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// getenv возвращает значение переменной окружения, затем значение из .env-файла,
// и если ни одно не задано — fallback. os.LookupEnv позволяет отличить
// незаданную переменную от заданной пустой строкой.
func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	if v := viper.GetString(key); v != "" {
		return v
	}
	return fallback
}
