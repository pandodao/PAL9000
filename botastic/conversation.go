package botastic

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type CreateConversationRequest struct {
	BotID        uint64 `json:"bot_id"`
	UserIdentity string `json:"user_identity"`
	Lang         string `json:"lang"`
}

type Middleware struct {
	ID      uint64                 `json:"id"`
	Name    string                 `json:"name"`
	Options map[string]interface{} `json:"options"`
}

type Bot struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type App struct {
	ID        uint64     `json:"id"`
	AppID     string     `json:"app_id"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type ConvTurn struct {
	ID             uint64     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	BotID          uint64     `json:"bot_id"`
	AppID          uint64     `json:"app_id"`
	UserIdentity   string     `json:"user_identity"`
	Request        string     `json:"request"`
	RequestToken   int        `json:"request_token"`
	Response       string     `json:"response"`
	ResponseToken  int        `json:"response_token"`
	Status         int        `json:"status"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

type Conversation struct {
	ID           string      `json:"id"`
	Bot          *Bot        `json:"bot"`
	App          *App        `json:"app"`
	UserIdentity string      `json:"user_identity"`
	Lang         string      `json:"lang"`
	History      []*ConvTurn `json:"history"`
	ExpiredAt    time.Time   `json:"expired_at"`
}

type PostToConversationPayloadRequest struct {
	ConversationID string `json:"-"`
	Content        string `json:"content"`
	Category       string `json:"category"`
}

type UpdateConversationRequest struct {
	ConversationID string `json:"-"`
	Lang           string `json:"lang"`
}

func (c *Client) CreateConversation(ctx context.Context, req CreateConversationRequest) (*Conversation, error) {
	conv := &Conversation{}
	if err := c.request(ctx, http.MethodPost, "/conversations", req, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (c *Client) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	conv := &Conversation{}
	if err := c.request(ctx, http.MethodGet, "/conversations/"+conversationID, nil, conv); err != nil {
		return nil, err
	}

	return conv, nil
}

func (c *Client) PostToConversation(ctx context.Context, req PostToConversationPayloadRequest) (*ConvTurn, error) {
	conv := &ConvTurn{}
	if err := c.request(ctx, http.MethodPost, "/conversations/"+req.ConversationID, req, conv); err != nil {
		return nil, err
	}

	return conv, nil
}

func (c *Client) DeleteConversation(ctx context.Context, conversationID string) error {
	return c.request(ctx, http.MethodDelete, "/conversations/"+conversationID, nil, nil)
}

func (c *Client) UpdateConversation(ctx context.Context, req UpdateConversationRequest) error {
	return c.request(ctx, http.MethodPut, "/conversations/"+req.ConversationID, req, nil)
}

func (c *Client) GetHandledConvTurn(ctx context.Context, conversationID string, turnID uint64) (*ConvTurn, error) {
	conv := &ConvTurn{}
	if err := c.request(ctx, http.MethodGet, fmt.Sprintf("/conversations/%s/%d", conversationID, turnID), nil, conv); err != nil {
		return nil, err
	}
	return conv, nil
}
