package botastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Config struct {
	AppId     string
	AppSecret string
	Host      string
	Debug     bool
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	cfg.Host = strings.TrimRight(cfg.Host, "/") + "/api"
	return &Client{
		cfg: cfg,
	}
}

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

func (c *Client) request(ctx context.Context, method string, uri string, body, result any) error {
	if c.cfg.Debug {
		log.Println(method, uri)
	}
	var r io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		r = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.Host+uri, r)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-BOTASTIC-APPID", c.cfg.AppId)
	req.Header.Set("X-BOTASTIC-SECRET", c.cfg.AppSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if c.cfg.Debug {
		log.Println(method, uri, string(respData))
	}

	{
		var res struct {
			Data Error `json:"data"`
		}

		if err := json.Unmarshal(respData, &res); err == nil && res.Data.Code != 0 {
			return &res.Data
		}
	}

	var res struct {
		Data any `json:"data"`
	}
	res.Data = result

	return json.Unmarshal(respData, &res)
}
