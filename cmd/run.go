package cmd

import (
	"context"

	"github.com/pandodao/PAL9000/config"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type configKey struct{}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run all bots by config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Init(cfgFile)
		if err != nil {
			return err
		}
		cmd.SetContext(context.WithValue(cmd.Context(), configKey{}, cfg))

		g := errgroup.Group{}
		for _, adaptor := range cfg.Adaptors.Enabled {
			switch adaptor {
			case "mixin":
				g.Go(func() error {
					return mixinCmd.RunE(cmd, []string{})
				})
			case "telegram":
				g.Go(func() error {
					return telegramCmd.RunE(cmd, []string{})
				})
			case "discord":
				g.Go(func() error {
					return discordCmd.RunE(cmd, []string{})
				})
			}
		}

		return g.Wait()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
