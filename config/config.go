package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Botastic BotasticConfig `yaml:"botastic"`
}

type BotasticConfig struct {
	AppId     string
	AppSecret string
	Host      string
	Debug     bool
}

func defaultConfig() *Config {
	return &Config{}
}

func (c Config) validate() error {
	return nil
}

func Init(fp string) (*Config, error) {
	c := defaultConfig()

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
