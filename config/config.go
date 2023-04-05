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
	Mixin    MixinConfig    `yaml:"mixin"`
	Telegram TelegramConfig `yaml:"telegram"`
}

type MixinConfig struct {
	GeneralConfig `yaml:",inline"`

	Enabled  bool   `yaml:"enabled"`
	Keystore string `yaml:"keystore"` // base64 encoded keystore (json format)
}

type TelegramConfig struct {
	GeneralConfig `yaml:",inline"`

	Enabled bool   `yaml:"enabled"`
	Debug   bool   `yaml:"debug"`
	Token   string `yaml:"token"`
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
