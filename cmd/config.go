package cmd

import (
	"fmt"

	"github.com/pandodao/PAL9000/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Display default config",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.DefaultConfig())
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
