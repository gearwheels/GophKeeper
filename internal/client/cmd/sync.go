package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	clientSync "github.com/timofeevav/gophkeeper/internal/client/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Принудительно синхронизировать данные с сервером",
	RunE: func(cmd *cobra.Command, args []string) error {
		syncer := clientSync.New(client, store)
		result, err := syncer.Run()
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		fmt.Printf("Синхронизация завершена: обновлено %d, конфликтов %d\n",
			result.Updated, result.Conflicts)

		if result.Conflicts > 0 {
			fmt.Println("Конфликты обнаружены. Используйте 'secret get <id>' для просмотра серверной версии.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
