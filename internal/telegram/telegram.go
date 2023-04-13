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

var _ service.Adaptor = (*Bot)(nil)

type (
	messageKey struct{}
	doneKey    struct{}
)

type Bot struct {
	cfg    config.TelegramConfig
	client *tgbotapi.BotAPI
}

func Init(cfg config.TelegramConfig) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, err
	}
	bot.Debug = cfg.Debug

	return &Bot{
		cfg:    cfg,
		client: bot,
	}, nil
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
				if !strings.HasPrefix(update.Message.Text, prefix) {
					continue
				}
			}
			update.Message.Text = strings.TrimSpace(strings.TrimPrefix(update.Message.Text, prefix))

			messageCtx := context.WithValue(ctx, messageKey{}, update.Message)
			msgChan <- &service.Message{
				Context:      messageCtx,
				Content:      update.Message.Text,
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

func (b *Bot) GetResultChan(ctx context.Context) chan<- *service.Result {
	resultChan := make(chan *service.Result)
	go func() {
		for {
			select {
			case r := <-resultChan:
				if r.Err != nil && r.IgnoreIfError {
					continue
				}
				text := ""
				if r.Err != nil {
					text = r.Err.Error()
				} else {
					text = r.ConvTurn.Response
				}
				msg := r.Message.Context.Value(messageKey{}).(*tgbotapi.Message)
				reply := tgbotapi.NewMessage(msg.Chat.ID, text)
				reply.ReplyToMessageID = msg.MessageID
				if _, err := b.client.Send(reply); err != nil {
					fmt.Printf("send reply failed: %v\n", err)
				}
			case <-ctx.Done():
				close(resultChan)
				return
			}
		}
	}()

	return resultChan
}
