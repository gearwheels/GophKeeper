package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/timofeevav/gophkeeper/internal/client/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Показать версию и дату сборки клиента",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version:    %s\n", version.Version)
		fmt.Printf("Build Date: %s\n", version.BuildDate)
		fmt.Printf("Commit:     %s\n", version.Commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
