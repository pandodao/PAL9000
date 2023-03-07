package botastic

import (
	"context"
)

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

func (s *Suite) TestSearchIndices() {
	_, err := s.client.SearchIndices(context.Background(), SearchIndicesRequest{
		Keywords: "test",
	})
	s.NoError(err)
}
