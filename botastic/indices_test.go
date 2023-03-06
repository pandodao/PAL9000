package botastic

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	client *Client
}

func TestSuite(t *testing.T) {
	client := New(Config{
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

func (s *Suite) TestSearchIndices() {
	_, err := s.client.SearchIndices(context.Background(), SearchIndicesRequest{
		Keywords: "test",
	})
	s.NoError(err)
}

func (s *Suite) TestCreateIndices() {
	err := s.client.CreateIndices(context.Background(), CreateIndicesRequest{
		Items: []*CreateIndicesItem{
			{
				ObjectID:   "test-object-id",
				Category:   "plain-text",
				Data:       "test-data",
				Properties: "test-properties",
			},
		},
	})
	s.NoError(err)
}

func (s *Suite) TestDeleteIndex() {
	err := s.client.DeleteIndex(context.Background(), "test-object-id")
	s.NoError(err)
}
