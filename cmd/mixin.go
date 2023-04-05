package cmd

import (
	"github.com/pandodao/PAL9000/internal/mixin"
	"github.com/pandodao/PAL9000/service"
	"github.com/pandodao/PAL9000/store"
	"github.com/spf13/cobra"
)

// mixinCmd represents the mixin command
var mixinCmd = &cobra.Command{
	Use:   "mixin",
	Short: "Start a mixin bot service",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg, err := getOrInitConfig(ctx)
		if err != nil {
			return err
		}

		b, err := mixin.Init(ctx, cfg.Adaptors.Mixin)
		if err != nil {
			return err
		}

		h := service.NewHandler(getGeneralConfig(cfg.General, cfg.Adaptors.Mixin.GeneralConfig), store.NewMemoryStore(), b)
		return h.Start(ctx)
	},
}

func init() {
	rootCmd.AddCommand(mixinCmd)
}
