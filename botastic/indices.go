package botastic

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type CreateIndicesItem struct {
	ObjectID   string `json:"object_id"`
	Category   string `json:"category"`
	Data       string `json:"data"`
	Properties string `json:"properties"`
}

type CreateIndicesRequest struct {
	Items []*CreateIndicesItem `json:"items"`
}

type SearchIndicesRequest struct {
	Keywords string
	N        int
}

type Index struct {
	Data       string  `json:"data"`
	ObjectID   string  `json:"object_id"`
	Category   string  `json:"category"`
	Properties string  `json:"properties"`
	CreatedAt  int64   `json:"created_at"`
	Score      float32 `json:"score"`
}

type SearchIndicesResponse struct {
	Indices []*Index `json:"indices"`
}

type Error struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("code: %d, msg: %s", e.Code, e.Msg)
}

func (c *Client) CreateIndices(ctx context.Context, req CreateIndicesRequest) error {
	return c.request(ctx, http.MethodPost, "/indices", req, nil)
}

func (c *Client) DeleteIndex(ctx context.Context, objectId string) error {
	return c.request(ctx, http.MethodDelete, "/indices/"+objectId, nil, nil)
}

func (c *Client) SearchIndices(ctx context.Context, req SearchIndicesRequest) (*SearchIndicesResponse, error) {
	values := url.Values{}
	values.Add("keywords", req.Keywords)
	if req.N != 0 {
		values.Add("n", strconv.Itoa(req.N))
	}

	result := &SearchIndicesResponse{}
	if err := c.request(ctx, http.MethodGet, "/indices/search?"+values.Encode(), nil, &result.Indices); err != nil {
		return nil, err
	}
	return result, nil
}
