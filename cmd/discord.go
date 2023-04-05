package cmd

import (
	"github.com/pandodao/PAL9000/internal/discord"
	"github.com/pandodao/PAL9000/service"
	"github.com/pandodao/PAL9000/store"
	"github.com/spf13/cobra"
)

// discordCmd represents the discord command
var discordCmd = &cobra.Command{
	Use:   "discord",
	Short: "Start a discord bot service",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg, err := getOrInitConfig(ctx)
		if err != nil {
			return err
		}

		b := discord.New(cfg.Adaptors.Discord)
		h := service.NewHandler(getGeneralConfig(cfg.General, cfg.Adaptors.Discord.GeneralConfig), store.NewMemoryStore(), b)
		return h.Start(ctx)
	},
}

func init() {
	rootCmd.AddCommand(discordCmd)
}
