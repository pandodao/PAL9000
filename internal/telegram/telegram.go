package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/service"
)

var _ service.Adapter = (*Bot)(nil)

type (
	messageKey struct{}
)

type Bot struct {
	name   string
	cfg    config.TelegramConfig
	client *tgbotapi.BotAPI
}

func Init(name string, cfg config.TelegramConfig) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, err
	}
	bot.Debug = cfg.Debug

	return &Bot{
		name:   name,
		cfg:    cfg,
		client: bot,
	}, nil
}

func (b *Bot) GetName() string {
	return b.name
}

func (b *Bot) GetMessageChan(ctx context.Context) <-chan *service.Message {
	msgChan := make(chan *service.Message)
	go func() {
		u := tgbotapi.NewUpdate(0)
		updates := b.client.GetUpdatesChan(u)
		for update := range updates {
			if update.Message == nil || update.Message.Chat == nil || update.Message.Text == "" {
				continue
			}

			allowed := len(b.cfg.Whitelist) == 0
			for _, id := range b.cfg.Whitelist {
				if strconv.FormatInt(update.Message.Chat.ID, 10) == id || strconv.FormatInt(update.Message.From.ID, 10) == id {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}

			prefix := "@" + b.client.Self.UserName
			if update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup() {
				if update.Message.ReplyToMessage == nil || update.Message.ReplyToMessage.From.ID != b.client.Self.ID {
					if !strings.HasPrefix(update.Message.Text, prefix) {
						continue
					}
				}
			}
			replyContent := ""
			if update.Message.ReplyToMessage != nil {
				replyContent = update.Message.ReplyToMessage.Text
			}

			content := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, prefix))
			messageCtx := context.WithValue(ctx, messageKey{}, update.Message)
			msgChan <- &service.Message{
				ReplyContent: replyContent,
				Context:      messageCtx,
				Content:      content,
				UserIdentity: strconv.FormatInt(update.Message.From.ID, 10),
				ConvKey:      strconv.FormatInt(update.Message.Chat.ID, 10),
			}
		}
		select {
		case <-ctx.Done():
			b.client.StopReceivingUpdates()
			close(msgChan)
			return
		}
	}()

	return msgChan
}

func (b *Bot) HandleResult(req *service.Message, r *service.Result) {
	if r.Err != nil && r.IgnoreIfError {
		return
	}
	text := ""
	if r.Err != nil {
		text = r.Err.Error()
	} else {
		text = r.ConvTurn.Response
	}
	msg := req.Context.Value(messageKey{}).(*tgbotapi.Message)
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	// reply.ReplyToMessageID = msg.MessageID
	if _, err := b.client.Send(reply); err != nil {
		fmt.Printf("send reply failed: %v\n", err)
	}
}
