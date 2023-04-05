package cmd

import (
	"os"

	"github.com/pandodao/PAL9000/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "PAL9000",
	Short: "PAL9000 is a tool to connect to botastic APIs",
	Long:  `With pal9000, you can easily deploy your own bot application using botastic`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.yaml", "config file (default is config.yaml)")
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
