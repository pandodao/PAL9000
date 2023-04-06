package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	General  GeneralConfig  `yaml:"general"`
	Adaptors AdaptorsConfig `yaml:"adaptors"`
}

func (s *Config) String() string {
	data, _ := yaml.Marshal(s)
	return string(data)
}

type BotConfig struct {
	BotID uint64 `yaml:"bot_id"`
	Lang  string `yaml:"lang"`
}

type BotasticConfig struct {
	AppId string `yaml:"app_id"`
	Host  string `yaml:"host"`
	Debug bool   `yaml:"debug"`
}

type GeneralConfig struct {
	Bot      *BotConfig      `yaml:"bot"`
	Botastic *BotasticConfig `yaml:"botastic"`
}

type AdaptorsConfig struct {
	Enabled  []string       `yaml:"enabled"`
	Mixin    MixinConfig    `yaml:"mixin"`
	Telegram TelegramConfig `yaml:"telegram"`
	Discord  DiscordConfig  `yaml:"discord"`
	WeChat   WeChatConfig   `yaml:"wechat"`
}

type WeChatConfig struct {
	GeneralConfig `yaml:",inline"`

	Address string `yaml:"address"`
	Path    string `yaml:"path"`
	Token   string `yaml:"token"`
}

type MixinConfig struct {
	GeneralConfig `yaml:",inline"`

	Keystore string `yaml:"keystore"` // base64 encoded keystore (json format)
}

type TelegramConfig struct {
	GeneralConfig `yaml:",inline"`

	Debug bool   `yaml:"debug"`
	Token string `yaml:"token"`
}

type DiscordConfig struct {
	GeneralConfig `yaml:",inline"`

	Token string `yaml:"token"`
}

func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			Bot: &BotConfig{
				Lang: "en",
			},
		},
	}
}

func (c Config) validate() error {
	return nil
}

func Init(fp string) (*Config, error) {
	c := DefaultConfig()

	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadFile error: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal error: %w", err)
	}

	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("validate error: %w", err)
	}

	return c, nil
}
