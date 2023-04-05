package cmd

import (
	"github.com/pandodao/PAL9000/internal/telegram"
	"github.com/pandodao/PAL9000/service"
	"github.com/pandodao/PAL9000/store"
	"github.com/spf13/cobra"
)

// telegramCmd represents the telegram command
var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Start a telegram bot service",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg, err := getOrInitConfig(ctx)
		if err != nil {
			return err
		}

		b, err := telegram.Init(cfg.Adaptors.Telegram)
		if err != nil {
			return err
		}

		h := service.NewHandler(getGeneralConfig(cfg.General, cfg.Adaptors.Telegram.GeneralConfig), store.NewMemoryStore(), b)
		return h.Start(ctx)
	},
}

func init() {
	rootCmd.AddCommand(telegramCmd)
}
