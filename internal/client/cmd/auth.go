package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	clientConfig "github.com/timofeevav/gophkeeper/internal/client/config"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Зарегистрировать нового пользователя",
	RunE: func(cmd *cobra.Command, args []string) error {
		login, password, err := credentials(cmd)
		if err != nil {
			return err
		}

		result, err := client.Register(login, password)
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}

		cfg.Token = result.Token
		cfg.UserID = result.UserID
		if err := clientConfig.Save(cfg, cfgFile); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Зарегистрирован и выполнен вход: %s\n", login)
		return nil
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Аутентифицироваться на сервере",
	RunE: func(cmd *cobra.Command, args []string) error {
		login, password, err := credentials(cmd)
		if err != nil {
			return err
		}

		result, err := client.Login(login, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		cfg.Token = result.Token
		cfg.UserID = result.UserID
		if err := clientConfig.Save(cfg, cfgFile); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Выполнен вход: %s\n", login)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Удалить токен сессии",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.Token = ""
		cfg.UserID = ""
		if err := clientConfig.Save(cfg, cfgFile); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Println("Сессия завершена")
		return nil
	},
}

// credentials возвращает логин (из флага --login или промпта) и пароль.
// Пароль вводится только интерактивно без эха: переданный флагом пароль
// остаётся в истории шелла и виден в выводе ps.
func credentials(cmd *cobra.Command) (login, password string, err error) {
	login, _ = cmd.Flags().GetString("login")
	if login == "" {
		fmt.Print("Логин: ")
		login = readLine()
	}
	password, err = readSecureInput("Пароль: ")
	return login, password, err
}

func init() {
	registerCmd.Flags().String("login", "", "логин пользователя")
	rootCmd.AddCommand(registerCmd)

	loginCmd.Flags().String("login", "", "логин пользователя")
	rootCmd.AddCommand(loginCmd)

	rootCmd.AddCommand(logoutCmd)
}
