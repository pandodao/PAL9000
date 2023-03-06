package botastic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/pandodao/PAL9000/config"
)

type Client struct {
	cfg config.BotasticConfig
}

func New(cfg config.BotasticConfig) *Client {
	cfg.Host = strings.TrimRight(cfg.Host, "/") + "/api"
	return &Client{
		cfg: cfg,
	}
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
