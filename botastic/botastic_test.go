package botastic

import (
	"os"
	"testing"

	"github.com/pandodao/PAL9000/config"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	client *Client
}

func TestSuite(t *testing.T) {
	client := New(config.BotasticConfig{
		AppId:     os.Getenv("BOTASTIC_APP_ID"),
		AppSecret: os.Getenv("BOTASTIC_APP_SECRET"),
		Host:      os.Getenv("BOTASTIC_HOST"),
		Debug:     true,
	})
	if client.cfg.AppId == "" || client.cfg.AppSecret == "" || client.cfg.Host == "" {
		t.SkipNow()
	}

	suite.Run(t, &Suite{client: client})
}

