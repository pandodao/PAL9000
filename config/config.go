package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	General  GeneralConfig  `yaml:"general"`
	Adapters AdaptersConfig `yaml:"adapters"`
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

type GeneralOptionsConfig struct {
	IgnoreIfError bool `yaml:"ignore_if_error"`
}

type GeneralConfig struct {
	Options  *GeneralOptionsConfig `yaml:"options,omitempty"`
	Bot      *BotConfig            `yaml:"bot,omitempty"`
	Botastic *BotasticConfig       `yaml:"botastic,omitempty"`
}

type AdaptersConfig struct {
	Enabled []string                 `yaml:"enabled"`
	Items   map[string]AdapterConfig `yaml:"items"`
}

type AdapterConfig struct {
	Driver   string          `yaml:"driver"`
	Mixin    *MixinConfig    `yaml:"mixin,omitempty"`
	Telegram *TelegramConfig `yaml:"telegram,omitempty"`
	Discord  *DiscordConfig  `yaml:"discord,omitempty"`
	WeChat   *WeChatConfig   `yaml:"wechat,omitempty"`
}

type WeChatConfig struct {
	GeneralConfig `yaml:",inline"`

	Address string `yaml:"address"`
	Path    string `yaml:"path"`
	Token   string `yaml:"token"`
}

type MixinConfig struct {
	GeneralConfig `yaml:",inline"`

	Keystore  string   `yaml:"keystore"` // base64 encoded keystore (json format)
	Whitelist []string `yaml:"whitelist"`
}

type TelegramConfig struct {
	GeneralConfig `yaml:",inline"`

	Debug     bool     `yaml:"debug"`
	Token     string   `yaml:"token"`
	Whitelist []string `yaml:"whitelist"`
}

type DiscordConfig struct {
	GeneralConfig `yaml:",inline"`

	Token     string   `yaml:"token"`
	Whitelist []string `yaml:"whitelist"`
}

func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			Options: &GeneralOptionsConfig{
				IgnoreIfError: true,
			},
			Bot: &BotConfig{
				Lang: "en",
			},
		},
	}
}

func ExampleConfig() *Config {
	return &Config{
		General: GeneralConfig{
			Bot: &BotConfig{
				BotID: 1,
				Lang:  "en",
			},
			Botastic: &BotasticConfig{
				AppId: "cab1582e-9c30-4d1e-9246-a5c80f74f8f9",
				Host:  "https://botastic-api.pando.im",
				Debug: true,
			},
		},
		Adapters: AdaptersConfig{
			Enabled: []string{"test_mixin", "test_telegram", "test_discord", "test_wechat"},
			Items: map[string]AdapterConfig{
				"test_mixin": {
					Driver: "mixin",
					Mixin: &MixinConfig{
						Keystore:  "base64 encoded keystore",
						Whitelist: []string{"7000104111", "a8d4e38e-9317-4529-8ca9-4289d4668111"},
					},
				},
				"test_telegram": {
					Driver: "telegram",
					Telegram: &TelegramConfig{
						Debug:     true,
						Token:     "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZ",
						Whitelist: []string{"-10540154212", "xx"},
						GeneralConfig: GeneralConfig{
							Bot: &BotConfig{
								BotID: 2,
								Lang:  "zh",
							},
							Botastic: &BotasticConfig{
								AppId: "cab1582e-9c30-4d1e-9246-a5c80f74f8f9",
								Host:  "https://botastic-api.pando.im",
							},
						},
					},
				},
				"test_discord": {
					Driver: "discord",
					Discord: &DiscordConfig{
						Token:     "1234567890",
						Whitelist: []string{"1093104389113266186"},
					},
				},
				"test_wechat": {
					Driver: "wechat",
					WeChat: &WeChatConfig{
						Address: ":8080",
						Path:    "/wechat",
						Token:   "123456",
					},
				},
			},
		},
	}
}

func (c Config) validate() error {
	for _, name := range c.Adapters.Enabled {
		if _, ok := c.Adapters.Items[name]; !ok {
			return fmt.Errorf("adapter not found: %s", name)
		}
	}
	for name, c := range c.Adapters.Items {
		switch c.Driver {
		case "mixin":
			if c.Mixin == nil {
				return fmt.Errorf("config not found, name: %s, driver: %s", name, c.Driver)
			}
		case "telegram":
			if c.Telegram == nil {
				return fmt.Errorf("config not found, name: %s, driver: %s", name, c.Driver)
			}
		case "discord":
			if c.Discord == nil {
				return fmt.Errorf("config not found, name: %s, driver: %s", name, c.Driver)
			}
		case "wechat":
			if c.WeChat == nil {
				return fmt.Errorf("config not found, name: %s, driver: %s", name, c.Driver)
			}
		default:
			return fmt.Errorf("invalid driver, name: %s, driver: %s", name, c.Driver)
		}
	}
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
