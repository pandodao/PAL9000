package service

import (
	"context"

	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/store"
	"github.com/pandodao/botastic-go"
)

type Adaptor interface {
	GetMessageChan(ctx context.Context) <-chan *Message
	GetResultChan(ctx context.Context) chan<- *Result
}

type Handler struct {
	cfg     config.GeneralConfig
	client  *botastic.Client
	store   store.Store
	adaptor Adaptor
}

type Message struct {
	Context context.Context
	BotID   uint64
	Lang    string

	UserIdentity string
	ConvKey      string
	Content      string
}

type Result struct {
	Message  *Message
	ConvTurn *botastic.ConvTurn
	Err      error
}

func NewHandler(cfg config.GeneralConfig, store store.Store, adaptor Adaptor) *Handler {
	client := botastic.New(cfg.Botastic.AppId, "", botastic.WithDebug(cfg.Botastic.Debug), botastic.WithHost(cfg.Botastic.Host))
	return &Handler{
		cfg:     cfg,
		client:  client,
		store:   store,
		adaptor: adaptor,
	}
}

func (h *Handler) Start(ctx context.Context) error {
	msgChan := h.adaptor.GetMessageChan(ctx)
	resultChan := h.adaptor.GetResultChan(ctx)

	for {
		select {
		case msg := <-msgChan:
			if msg.BotID == 0 {
				msg.BotID = h.cfg.Bot.BotID
			}
			if msg.Lang == "" {
				msg.Lang = h.cfg.Bot.Lang
			}

			turn, err := h.handleMessage(ctx, msg)
			resultChan <- &Result{
				Message:  msg,
				ConvTurn: turn,
				Err:      err,
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (h *Handler) handleMessage(ctx context.Context, m *Message) (*botastic.ConvTurn, error) {
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

	convTurn, err := h.client.PostToConversation(ctx, botastic.PostToConversationPayloadRequest{
		ConversationID: conv.ID,
		Content:        m.Content,
		Category:       "plain-text",
	})
	if err != nil {
		return nil, err
	}

	turn, err := h.client.GetHandledConvTurn(ctx, conv.ID, convTurn.ID)
	if err != nil {
		// TODO: retry
		return nil, err
	}

	return turn, nil
}
