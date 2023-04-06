package cmd

import (
	"github.com/pandodao/PAL9000/internal/wechat"
	"github.com/pandodao/PAL9000/service"
	"github.com/pandodao/PAL9000/store"
	"github.com/spf13/cobra"
)

// wechatCmd represents the wechat command
var wechatCmd = &cobra.Command{
	Use:   "wechat",
	Short: "Start a wechat bot service",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg, err := getOrInitConfig(ctx)
		if err != nil {
			return err
		}

		b := wechat.New(cfg.Adaptors.WeChat)
		h := service.NewHandler(getGeneralConfig(cfg.General, cfg.Adaptors.WeChat.GeneralConfig), store.NewMemoryStore(), b)
		return h.Start(ctx)
	},
}

func init() {
	rootCmd.AddCommand(wechatCmd)
}
