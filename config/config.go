package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Bot      BotConfig      `yaml:"bot"`
	Botastic BotasticConfig `yaml:"botastic"`
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
	AppId     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	Host      string `yaml:"host"`
	Debug     bool   `yaml:"debug"`
}

func DefaultConfig() *Config {
	return &Config{
		Bot: BotConfig{
			Lang: "en",
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
