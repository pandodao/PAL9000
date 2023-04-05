package cmd

import (
	"github.com/spf13/cobra"
)

// telegramCmd represents the telegram command
var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Start a telegram bot service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(telegramCmd)
}
