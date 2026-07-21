package cmd

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/timofeevav/gophkeeper/internal/client/api"
	"github.com/timofeevav/gophkeeper/internal/client/crypto"
	"golang.org/x/term"
)

// stdinReader — единственный буферизованный читатель stdin для всего пакета.
// Несколько независимых bufio.NewReader(os.Stdin) конкурируют за одни и те же
// байты, и один из них теряет данные из своего буфера.
var stdinReader = bufio.NewReader(os.Stdin)

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Управление секретами",
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "Показать список секретов",
	RunE: func(cmd *cobra.Command, args []string) error {
		secretType, _ := cmd.Flags().GetString("type")
		resp, err := client.ListSecrets(secretType)
		if err != nil {
			return fmt.Errorf("list secrets: %w", err)
		}

		if len(resp.Secrets) == 0 {
			fmt.Println("Нет сохранённых секретов")
			return nil
		}

		fmt.Printf("%-36s  %-16s  %s\n", "ID", "Тип", "Название")
		fmt.Println(strings.Repeat("-", 72))
		for _, s := range resp.Secrets {
			fmt.Printf("%-36s  %-16s  %s\n", s.ID, s.Type, s.Name)
		}
		return nil
	},
}

var secretGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Получить и расшифровать секрет по ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		secret, err := client.GetSecret(args[0])
		if err != nil {
			return fmt.Errorf("get secret: %w", err)
		}

		masterPassword, err := readSecureInput("Мастер-пароль: ")
		if err != nil {
			return err
		}

		key := crypto.DeriveKey(masterPassword, cfg.UserID)

		encData, err := base64.StdEncoding.DecodeString(secret.Data)
		if err != nil {
			return fmt.Errorf("decode data: %w", err)
		}

		plaintext, err := crypto.Decrypt(key, encData)
		if err != nil {
			return fmt.Errorf("decrypt secret (неверный мастер-пароль?): %w", err)
		}

		fmt.Printf("ID:       %s\n", secret.ID)
		fmt.Printf("Тип:      %s\n", secret.Type)
		fmt.Printf("Название: %s\n", secret.Name)
		fmt.Printf("Мета:     %s\n", secret.Meta)
		fmt.Printf("Данные:   %s\n", string(plaintext))
		return nil
	},
}

var secretAddCmd = &cobra.Command{
	Use:   "add [type]",
	Short: "Добавить новый секрет (типы: login_password, text, binary, card)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var secretType string
		if len(args) > 0 {
			secretType = args[0]
		} else {
			fmt.Print("Тип (login_password, text, binary, card): ")
			secretType = readLine()
		}

		fmt.Print("Название: ")
		name := readLine()

		fmt.Print("Метаданные (опционально): ")
		meta := readLine()

		payload, err := readPayload(secretType)
		if err != nil {
			return err
		}

		masterPassword, err := readSecureInput("Мастер-пароль: ")
		if err != nil {
			return err
		}

		key := crypto.DeriveKey(masterPassword, cfg.UserID)
		encrypted, err := crypto.Encrypt(key, payload)
		if err != nil {
			return fmt.Errorf("encrypt secret: %w", err)
		}

		result, err := client.CreateSecret(api.CreateSecretRequest{
			Type: secretType,
			Name: name,
			Data: base64.StdEncoding.EncodeToString(encrypted),
			Meta: meta,
		})
		if err != nil {
			return fmt.Errorf("create secret: %w", err)
		}

		fmt.Printf("Секрет создан: %s\n", result.ID)
		return nil
	},
}

var secretDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Удалить секрет",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.DeleteSecret(args[0]); err != nil {
			return fmt.Errorf("delete secret: %w", err)
		}
		fmt.Printf("Секрет %s удалён\n", args[0])
		return nil
	},
}

// readLine читает строку из stdinReader; EOF без данных возвращает пустую строку.
func readLine() string {
	line, err := stdinReader.ReadString('\n')
	if err != nil && !isEOF(err) {
		return ""
	}
	return cleanInput(line)
}

// readSecureInput читает строку без эха (для паролей).
// Когда stdin не является терминалом (тест/pipe), читает через stdinReader.
func readSecureInput(prompt string) (string, error) {
	fmt.Print(prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		return string(pwd), nil
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil && !isEOF(err) {
		return "", fmt.Errorf("read input: %w", err)
	}
	return cleanInput(line), nil
}

func isEOF(err error) bool {
	return err == io.EOF
}

// cleanInput обрезает пробелы и UTF-8 BOM, который PowerShell добавляет при pipe.
func cleanInput(s string) string {
	s = strings.TrimSpace(s)
	return strings.TrimPrefix(s, "\xef\xbb\xbf")
}

func readPayload(secretType string) ([]byte, error) {
	switch secretType {
	case "login_password":
		fmt.Print("Логин: ")
		login := readLine()
		password, err := readSecureInput("Пароль: ")
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]string{
			"login":    login,
			"password": password,
		})

	case "text":
		fmt.Print("Текст: ")
		text := readLine()
		return json.Marshal(map[string]string{"text": text})

	case "card":
		fmt.Print("Номер карты: ")
		number := readLine()
		fmt.Print("Держатель: ")
		holder := readLine()
		fmt.Print("Срок (MM/YY): ")
		expiry := readLine()
		cvv, err := readSecureInput("CVV: ")
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]string{
			"number":      number,
			"holder":      holder,
			"expiry_date": expiry,
			"cvv":         cvv,
		})

	case "binary":
		fmt.Print("Путь к файлу: ")
		path := readLine()
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
		return json.Marshal(map[string]interface{}{
			"data":     base64.StdEncoding.EncodeToString(data),
			"filename": path,
		})

	default:
		return nil, fmt.Errorf("неизвестный тип: %s", secretType)
	}
}

func init() {
	secretListCmd.Flags().String("type", "", "фильтр по типу (login_password, text, binary, card)")
	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretGetCmd)
	secretCmd.AddCommand(secretAddCmd)
	secretCmd.AddCommand(secretDeleteCmd)
	rootCmd.AddCommand(secretCmd)
}
