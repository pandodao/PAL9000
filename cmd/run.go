package cmd

import (
	"context"
	"fmt"

	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/internal/discord"
	"github.com/pandodao/PAL9000/internal/mixin"
	"github.com/pandodao/PAL9000/internal/telegram"
	"github.com/pandodao/PAL9000/internal/wechat"
	"github.com/pandodao/PAL9000/service"
	"github.com/pandodao/PAL9000/store"
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
		ctx := cmd.Context()

		startHandler := func(h *service.Handler, name string, adapterCfg *config.AdapterConfig) error {
			fmt.Printf("Starting adapter, name: %s, driver: %s\n", name, adapterCfg.Driver)
			return h.Start(ctx)
		}

		g := errgroup.Group{}
		for _, name := range cfg.Adapters.Enabled {
			name := name
			adapter := cfg.Adapters.Items[name]
			switch adapter.Driver {
			case "mixin":
				g.Go(func() error {
					b, err := mixin.Init(ctx, *adapter.Mixin)
					if err != nil {
						return err
					}

					h := service.NewHandler(adapter.OverrideGeneral, store.NewMemoryStore(), b)
					return startHandler(h, name, adapter)
				})
			case "telegram":
				g.Go(func() error {
					b, err := telegram.Init(*adapter.Telegram)
					if err != nil {
						return err
					}

					h := service.NewHandler(adapter.OverrideGeneral, store.NewMemoryStore(), b)
					return startHandler(h, name, adapter)
				})
			case "discord":
				g.Go(func() error {
					b := discord.New(*adapter.Discord)
					h := service.NewHandler(adapter.OverrideGeneral, store.NewMemoryStore(), b)
					return startHandler(h, name, adapter)
				})
			case "wechat":
				g.Go(func() error {
					b := wechat.New(*adapter.WeChat)
					h := service.NewHandler(adapter.OverrideGeneral, store.NewMemoryStore(), b)
					return startHandler(h, name, adapter)
				})
			}
		}

		return g.Wait()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
