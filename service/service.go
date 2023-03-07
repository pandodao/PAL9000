package service

import (
	"context"

	"github.com/pandodao/PAL9000/botastic"
	"github.com/pandodao/PAL9000/store"
)

type Handler struct {
	client *botastic.Client
	store  store.Store
}

type Message struct {
	BotID        uint64
	UserIdentity string
	ConvKey      string
	Content      string
	Lang         string
}

func NewHandler(client *botastic.Client, store store.Store) *Handler {
	return &Handler{
		client: client,
		store:  store,
	}
}

func (h *Handler) HandleWithCallback(ctx context.Context, m *Message, callback func(*botastic.ConvTurn, error) error) error {
	return callback(h.Handle(ctx, m))
}

func (h *Handler) Handle(ctx context.Context, m *Message) (*botastic.ConvTurn, error) {
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
