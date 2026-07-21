// Package cmd содержит CLI-команды клиента GophKeeper.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/timofeevav/gophkeeper/internal/client/api"
	clientConfig "github.com/timofeevav/gophkeeper/internal/client/config"
	"github.com/timofeevav/gophkeeper/internal/client/storage"
)

var (
	cfgFile string
	cfg     *clientConfig.Config
	client  *api.Client
	store   *storage.Storage
)

// rootCmd — корневая команда CLI.
var rootCmd = &cobra.Command{
	Use:   "gophkeeper",
	Short: "GophKeeper — безопасный менеджер паролей",
	Long:  "GophKeeper позволяет надёжно хранить и синхронизировать приватные данные.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "version" {
			return nil
		}

		var err error
		cfg, err = clientConfig.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Флаги командной строки перекрывают значения из конфига
		if server, _ := cmd.Root().PersistentFlags().GetString("server"); server != "" {
			cfg.ServerAddress = server
		}
		if insecure, _ := cmd.Root().PersistentFlags().GetBool("insecure"); insecure {
			cfg.Insecure = true
		}

		client = api.New(cfg.ServerAddress, cfg.Insecure)
		if cfg.Token != "" {
			client.SetToken(cfg.Token)
		}

		if cmd.Name() != "register" && cmd.Name() != "login" {
			home, _ := os.UserHomeDir()
			dbPath := home + "/.gophkeeper/data.db"
			store, err = storage.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open local storage: %w", err)
			}
		}
		return nil
	},
}

// Execute запускает корневую команду.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "путь к файлу конфигурации")
	rootCmd.PersistentFlags().String("server", "", "адрес сервера (например: https://localhost:8080)")
	rootCmd.PersistentFlags().Bool("insecure", false, "отключить проверку TLS-сертификата (для самоподписанных сертификатов)")
}
