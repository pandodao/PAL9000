package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/store"
	"github.com/pandodao/botastic-go"
	"github.com/sirupsen/logrus"
)

var (
	linkRegex = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)
)

type Adapter interface {
	GetName() string
	GetMessageChan(ctx context.Context) <-chan *Message
	HandleResult(message *Message, result *Result)
}

type Handler struct {
	cfg     config.GeneralConfig
	client  *botastic.Client
	store   store.Store
	adapter Adapter
	logger  *logrus.Entry
}

type Message struct {
	Context context.Context
	BotID   uint64
	Lang    string

	UserIdentity string
	ConvKey      string
	Content      string
	ReplyContent string

	DoneChan chan struct{}
}

type Result struct {
	ConvTurn      *botastic.ConvTurn
	Err           error
	IgnoreIfError bool
}

func NewHandler(cfg config.GeneralConfig, store store.Store, adapter Adapter) *Handler {
	client := botastic.New(cfg.Botastic.AppId, "", botastic.WithDebug(cfg.Botastic.Debug), botastic.WithHost(cfg.Botastic.Host))
	return &Handler{
		cfg:     cfg,
		client:  client,
		store:   store,
		adapter: adapter,
		logger:  logrus.WithField("adapter", fmt.Sprintf("%T", adapter)).WithField("component", "service").WithField("adapter_name", adapter.GetName()),
	}
}

func (h *Handler) Start(ctx context.Context) error {
	msgChan := h.adapter.GetMessageChan(ctx)

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

			turn, err := h.handleMessage(ctx, msg)
			h.logger.WithFields(logrus.Fields{
				"turn":       turn,
				"result_err": err,
			}).Info("handled message")
			h.adapter.HandleResult(msg, &Result{
				ConvTurn:      turn,
				IgnoreIfError: h.cfg.Options.IgnoreIfError,
				Err:           err,
			})
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

	content := ""
	if m.ReplyContent != "" {
		content = fmt.Sprintf(`"%s" `, m.ReplyContent)
	}
	content += m.Content

	convTurn, err := h.client.PostToConversation(ctx, botastic.PostToConversationPayloadRequest{
		ConversationID: conv.ID,
		Content:        content,
		Category:       "plain-text",
	})
	if err != nil {
		return nil, err
	}

	turn, err := h.client.GetConvTurn(ctx, conv.ID, convTurn.ID, true)
	if err != nil {
		// TODO: retry
		return nil, err
	}
	if turn.Status != 2 {
		return nil, fmt.Errorf("unexpected status: %d", turn.Status)
	}

	if h.cfg.Options.FormatLinks && turn.Response != "" {
		turn.Response = formatLink(turn.Response)
	}

	return turn, nil
}

func formatLink(str string) string {
	isSpace := func(c byte) bool {
		return c == ' ' || c == '\t' || c == '\n' || c == '\r'
	}
	matches := linkRegex.FindAllStringSubmatchIndex(str, -1)
	var result strings.Builder
	lastIdx := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		result.WriteString(str[lastIdx:start])

		if start > 0 && !isSpace(str[start-1]) {
			result.WriteString(" ")
		}

		result.WriteString(str[start:end])

		if end < len(str) && !isSpace(str[end]) {
			result.WriteString(" ")
		}

		lastIdx = end
	}

	result.WriteString(str[lastIdx:])
	return result.String()
}
