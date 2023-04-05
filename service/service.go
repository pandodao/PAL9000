package service

import (
	"context"

	"github.com/pandodao/PAL9000/store"
	"github.com/pandodao/botastic-go"
	"golang.org/x/sync/errgroup"
)

type Adaptor interface {
	GetMessageChan() <-chan *Message
	HandleResult(ctx context.Context, t *botastic.ConvTurn, err error) error
	Start(ctx context.Context) error
}

type Handler struct {
	client  *botastic.Client
	store   store.Store
	adaptor Adaptor
}

type Message struct {
	BotID        uint64
	UserIdentity string
	ConvKey      string
	Content      string
	Lang         string
	Resp         *botastic.ConvTurn
}

func NewHandler(client *botastic.Client, store store.Store, adaptor Adaptor) *Handler {
	return &Handler{
		client:  client,
		store:   store,
		adaptor: adaptor,
	}
}

func (h *Handler) Run(ctx context.Context) error {
	g := errgroup.Group{}

	g.Go(func() error {
		return h.adaptor.Start(ctx)
	})

	g.Go(func() error {
		for {
			select {
			case msg := <-h.adaptor.GetMessageChan():
				turn, err := h.handleMessage(ctx, msg)
				if err := h.adaptor.HandleResult(ctx, turn, err); err != nil {
					return err
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	return g.Wait()
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
