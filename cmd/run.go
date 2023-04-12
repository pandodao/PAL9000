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

		startHandler := func(h *service.Handler, name string, adaptorCfg config.AdaptorConfig) error {
			fmt.Printf("Starting adaptor, name: %s, driver: %s\n", name, adaptorCfg.Driver)
			return h.Start(ctx)
		}

		g := errgroup.Group{}
		for _, name := range cfg.Adaptors.Enabled {
			name := name
			adaptor := cfg.Adaptors.Items[name]
			switch adaptor.Driver {
			case "mixin":
				g.Go(func() error {
					b, err := mixin.Init(ctx, *adaptor.Mixin)
					if err != nil {
						return err
					}

					h := service.NewHandler(getGeneralConfig(cfg.General, adaptor.Mixin.GeneralConfig), store.NewMemoryStore(), b)
					return startHandler(h, name, adaptor)
				})
			case "telegram":
				g.Go(func() error {
					b, err := telegram.Init(*adaptor.Telegram)
					if err != nil {
						return err
					}

					h := service.NewHandler(getGeneralConfig(cfg.General, adaptor.Telegram.GeneralConfig), store.NewMemoryStore(), b)
					return startHandler(h, name, adaptor)
				})
			case "discord":
				g.Go(func() error {
					b := discord.New(*adaptor.Discord)
					h := service.NewHandler(getGeneralConfig(cfg.General, adaptor.Discord.GeneralConfig), store.NewMemoryStore(), b)
					return startHandler(h, name, adaptor)
				})
			case "wechat":
				g.Go(func() error {
					b := wechat.New(*adaptor.WeChat)
					h := service.NewHandler(getGeneralConfig(cfg.General, adaptor.WeChat.GeneralConfig), store.NewMemoryStore(), b)
					return startHandler(h, name, adaptor)
				})
			}
		}

		return g.Wait()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func getGeneralConfig(defaultCfg, overrideCfg config.GeneralConfig) config.GeneralConfig {
	cfg := defaultCfg
	if overrideCfg.Bot != nil {
		cfg.Bot = overrideCfg.Bot
	}
	if overrideCfg.Botastic != nil {
		cfg.Botastic = overrideCfg.Botastic
	}

	return cfg
}
