/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"os"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/pandodao/PAL9000/botastic"
	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/internal/mixinbot"
	"github.com/pandodao/PAL9000/service"
	"github.com/pandodao/PAL9000/store"
	"github.com/spf13/cobra"
)

var (
	keystorePath string
)

// mixinCmd represents the mixin command
var mixinCmd = &cobra.Command{
	Use:   "mixin",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Init(cfgFile)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(keystorePath)
		if err != nil {
			return err
		}

		var keystore mixin.Keystore
		if err := json.Unmarshal(data, &keystore); err != nil {
			return err
		}

		client, err := mixin.NewFromKeystore(&keystore)
		if err != nil {
			return err
		}

		h := service.NewHandler(botastic.New(cfg.Botastic), store.NewMemoryStore())

		b := mixinbot.New(client, h)
		ctx := cmd.Context()
		if err := b.SetUserMe(ctx); err != nil {
			return err
		}

		b.Run(ctx)
		return nil
	},
}

func init() {
	mixinCmd.PersistentFlags().StringVar(&keystorePath, "keystore", "keystore.json", "keystore file (default is keystore.json)")
	rootCmd.AddCommand(mixinCmd)
}
