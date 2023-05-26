package service

import (
	"context"
	"fmt"

	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/store"
	"github.com/pandodao/botastic-go"
	"github.com/sirupsen/logrus"
)

type Adapter interface {
	GetMessageChan(ctx context.Context) <-chan *Message
	GetResultChan(ctx context.Context) chan<- *Result
}

type Handler struct {
	cfg     config.GeneralConfig
	client  *botastic.Client
	store   store.Store
	adapter Adapter
	logger  *logrus.Entry
}

type Message struct {
	Context context.Context `json:"-"`
	BotID   uint64          `json:"bot_id"`
	Lang    string          `json:"lang"`

	UserIdentity string `json:"user_identity"`
	ConvKey      string `json:"conv_key"`
	Content      string `json:"content"`
}

type ConvTurn struct {
	*botastic.ConvTurn
	IsPluginCustomResponse bool
	ResponseModified       bool
}

type Result struct {
	Message *Message
	Turns   []*ConvTurn
	Err     error
	Options config.GeneralOptionsConfig
}

func NewHandler(cfg config.GeneralConfig, store store.Store, adapter Adapter) *Handler {
	client := botastic.New(cfg.Botastic.AppId, "", botastic.WithDebug(cfg.Botastic.Debug), botastic.WithHost(cfg.Botastic.Host))
	return &Handler{
		cfg:     cfg,
		client:  client,
		store:   store,
		adapter: adapter,
		logger:  logrus.WithField("adapter", fmt.Sprintf("%T", adapter)).WithField("component", "service"),
	}
}

func (h *Handler) Start(ctx context.Context) error {
	msgChan := h.adapter.GetMessageChan(ctx)
	resultChan := h.adapter.GetResultChan(ctx)

	for {
		select {
		case msg := <-msgChan:
			h.logger.WithField("msg", msg).Info("received message")
			if msg.BotID == 0 {
				msg.BotID = h.cfg.Bot.BotID
			}
			if msg.Lang == "" {
				msg.Lang = h.cfg.Bot.Lang
			}

			turns, err := h.handleMessage(ctx, msg)
			resultChan <- &Result{
				Turns:   turns,
				Message: msg,
				Err:     err,
				Options: *h.cfg.Options,
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (h *Handler) handleMessage(ctx context.Context, m *Message) ([]*ConvTurn, error) {
	conv, err := h.store.GetConversationByKey(m.ConvKey)
	if err != nil {
		return nil, err
	}

	if conv == nil {
		conv, err = h.client.CreateConversation(ctx, botastic.CreateConversationRequest{
			BotID:        m.BotID,
			UserIdentity: m.UserIdentity,
			Lang:         m.Lang,
		})
		if err != nil {
			return nil, err
		}

		if err := h.store.SetConversation(m.ConvKey, conv); err != nil {
			return nil, err
		}
	}

	pbr, err := h.handlePluginExecuteBefore(ctx, *m)
	if err != nil {
		return nil, err
	}

	turns := []*ConvTurn{}
	if pbr != nil {
		if pbr.ModifiedRequest != "" {
			m.Content = pbr.ModifiedRequest
		}

		for _, r := range pbr.CustomResponse {
			turns = append(turns, &ConvTurn{
				IsPluginCustomResponse: true,
				ConvTurn: &botastic.ConvTurn{
					Status:   2,
					Response: r,
				},
			})
		}
	}

	if pbr.TerminateRequest {
		return turns, nil
	}

	convTurn, err := h.client.PostToConversation(ctx, botastic.PostToConversationPayloadRequest{
		ConversationID: conv.ID,
		Content:        m.Content,
		Category:       "plain-text",
	})
	if err != nil {
		return turns, err
	}

	turn, err := h.client.GetConvTurn(ctx, conv.ID, convTurn.ID, true)
	if err != nil {
		// TODO: retry
		return turns, err
	}
	if turn.Status != 2 {
		return turns, fmt.Errorf("unexpected status: %d", turn.Status)
	}

	par, err := h.handlePluginExecuteAfter(ctx, turn)
	if err != nil {
		return turns, err
	}

	responseModified := false
	if par != nil {
		if par.ModifiedResponse != "" {
			responseModified = true
			turn.Response = par.ModifiedResponse
		}
	}

	turns = append(turns, &ConvTurn{ConvTurn: turn, ResponseModified: responseModified})
	return turns, nil
}
