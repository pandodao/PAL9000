package config

import (
	"fmt"
	"io/ioutil"
	"plugin"

	"gopkg.in/yaml.v3"
)

type Plugin interface {
	PluginName() string
}

type Config struct {
	General  GeneralConfig   `yaml:"general"`
	Adapters *AdaptersConfig `yaml:"adapters"`
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

type PluginItem struct {
	parsed bool

	IgnoreIfError             bool   `yaml:"ignore_if_error"`
	AllowedToTerminatePlugins bool   `yaml:"allowed_to_terminate_plugins"`
	AllowedToTerminateRequest bool   `yaml:"allowed_to_terminate_request"`
	Path                      string `yaml:"path"`
	Plugin                    Plugin `yaml:"-"`
}

type GeneralOptionsConfig struct {
	IgnoreIfError      bool `yaml:"ignore_if_error"`
	IgnoreTurnsIfError bool `yaml:"ignore_turns_if_error"`
}

type GeneralPluginsConfig struct {
	Items []*PluginItem `yaml:"items"`
}

type GeneralConfig struct {
	Options  *GeneralOptionsConfig `yaml:"options,omitempty"`
	Plugins  *GeneralPluginsConfig `yaml:"plugins,omitempty"`
	Bot      *BotConfig            `yaml:"bot,omitempty"`
	Botastic *BotasticConfig       `yaml:"botastic,omitempty"`
}

type AdaptersConfig struct {
	Enabled []string                  `yaml:"enabled"`
	Items   map[string]*AdapterConfig `yaml:"items"`
}

type AdapterConfig struct {
	Driver          string          `yaml:"driver"`
	OverrideGeneral GeneralConfig   `yaml:"override_general,omitempty"`
	Mixin           *MixinConfig    `yaml:"mixin,omitempty"`
	Telegram        *TelegramConfig `yaml:"telegram,omitempty"`
	Discord         *DiscordConfig  `yaml:"discord,omitempty"`
	WeChat          *WeChatConfig   `yaml:"wechat,omitempty"`
}

type WeChatConfig struct {
	Address string `yaml:"address"`
	Path    string `yaml:"path"`
	Token   string `yaml:"token"`
}

type MixinConfig struct {
	Keystore  string   `yaml:"keystore"` // base64 encoded keystore (json format)
	Whitelist []string `yaml:"whitelist"`
}

type TelegramConfig struct {
	Debug     bool     `yaml:"debug"`
	Token     string   `yaml:"token"`
	Whitelist []string `yaml:"whitelist"`
}

type DiscordConfig struct {
	Token     string   `yaml:"token"`
	Whitelist []string `yaml:"whitelist"`
}

func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			Options: &GeneralOptionsConfig{
				IgnoreIfError: true,
			},
			Plugins:  &GeneralPluginsConfig{},
			Botastic: &BotasticConfig{},
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
			Options: &GeneralOptionsConfig{
				IgnoreIfError:      true,
				IgnoreTurnsIfError: true,
			},
			Plugins: &GeneralPluginsConfig{
				Items: []*PluginItem{
					{
						IgnoreIfError:             true,
						AllowedToTerminatePlugins: true,
						AllowedToTerminateRequest: true,
						Path:                      "plugins/echo.so",
					},
				},
			},
		},
		Adapters: &AdaptersConfig{
			Enabled: []string{"test_mixin", "test_telegram", "test_discord", "test_wechat"},
			Items: map[string]*AdapterConfig{
				"test_mixin": {
					Driver: "mixin",
					Mixin: &MixinConfig{
						Keystore:  "base64 encoded keystore",
						Whitelist: []string{"7000104111", "a8d4e38e-9317-4529-8ca9-4289d4668111"},
					},
				},
				"test_telegram": {
					Driver: "telegram",
					OverrideGeneral: GeneralConfig{
						Bot: &BotConfig{
							BotID: 2,
							Lang:  "zh",
						},
						Botastic: &BotasticConfig{
							AppId: "cab1582e-9c30-4d1e-9246-a5c80f74f8f9",
							Host:  "https://botastic-api.pando.im",
						},
					},
					Telegram: &TelegramConfig{
						Debug:     true,
						Token:     "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZ",
						Whitelist: []string{"-10540154212", "xx"},
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

func (c *Config) validate() error {
	for _, name := range c.Adapters.Enabled {
		if _, ok := c.Adapters.Items[name]; !ok {
			return fmt.Errorf("adapter not found: %s", name)
		}
	}
	for name, adapter := range c.Adapters.Items {
		switch adapter.Driver {
		case "mixin":
			if adapter.Mixin == nil {
				return fmt.Errorf("config not found, name: %s, driver: %s", name, adapter.Driver)
			}
		case "telegram":
			if adapter.Telegram == nil {
				return fmt.Errorf("config not found, name: %s, driver: %s", name, adapter.Driver)
			}
		case "discord":
			if adapter.Discord == nil {
				return fmt.Errorf("config not found, name: %s, driver: %s", name, adapter.Driver)
			}
		case "wechat":
			if adapter.WeChat == nil {
				return fmt.Errorf("config not found, name: %s, driver: %s", name, adapter.Driver)
			}
		default:
			return fmt.Errorf("invalid driver, name: %s, driver: %s", name, adapter.Driver)
		}
	}

	for _, name := range c.Adapters.Enabled {
		adapter, ok := c.Adapters.Items[name]
		if !ok {
			return fmt.Errorf("adapter not found: %s", name)
		}

		if adapter.OverrideGeneral.Botastic == nil {
			adapter.OverrideGeneral.Botastic = c.General.Botastic
		}
		if adapter.OverrideGeneral.Bot == nil {
			adapter.OverrideGeneral.Bot = c.General.Bot
		}
		if adapter.OverrideGeneral.Options == nil {
			adapter.OverrideGeneral.Options = c.General.Options
		}
		if adapter.OverrideGeneral.Plugins == nil {
			adapter.OverrideGeneral.Plugins = c.General.Plugins
		}

		if adapter.OverrideGeneral.Plugins != nil {
			for _, p := range adapter.OverrideGeneral.Plugins.Items {
				if p.parsed {
					continue
				}
				plu, err := plugin.Open(p.Path)
				if err != nil {
					return fmt.Errorf("plugin open error, path: %s, error: %s", p.Path, err)
				}
				ins, err := plu.Lookup("PluginInstance")
				if err != nil {
					return fmt.Errorf("plugin lookup error, path: %s, error: %s", p.Path, err)
				}
				ok := false
				p.Plugin, ok = ins.(Plugin)
				if !ok {
					return fmt.Errorf("plugin type error, path: %s", p.Path)
				}

				p.parsed = true
			}
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
