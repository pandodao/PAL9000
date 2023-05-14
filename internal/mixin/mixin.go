package mixin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/fox-one/pkg/uuid"
	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/service"
	"github.com/sirupsen/logrus"
)

type (
	messageKey struct{}
	userKey    struct{}
	convKey    struct{}
)

var _ service.Adapter = (*Bot)(nil)

type Bot struct {
	convMap map[string]*mixin.Conversation
	userMap map[string]*mixin.User

	client  *mixin.Client
	msgChan chan *service.Message
	me      *mixin.User
	cfg     config.MixinConfig
	logger  logrus.FieldLogger
}

func Init(ctx context.Context, cfg config.MixinConfig) (*Bot, error) {
	data, err := base64.StdEncoding.DecodeString(cfg.Keystore)
	if err != nil {
		return nil, fmt.Errorf("base64 decode keystore error: %w", err)
	}

	var keystore mixin.Keystore
	if err := json.Unmarshal(data, &keystore); err != nil {
		return nil, fmt.Errorf("json unmarshal keystore error: %w", err)
	}

	client, err := mixin.NewFromKeystore(&keystore)
	if err != nil {
		return nil, fmt.Errorf("mixin.NewFromKeystore error: %w", err)
	}

	me, err := client.UserMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("mixinClient.UserMe error: %w", err)
	}

	return &Bot{
		convMap: make(map[string]*mixin.Conversation),
		userMap: make(map[string]*mixin.User),
		client:  client,
		msgChan: make(chan *service.Message),
		cfg:     cfg,
		me:      me,
		logger:  logrus.WithField("adapter", "mixin"),
	}, nil
}

func (b *Bot) GetMessageChan(ctx context.Context) <-chan *service.Message {
	go func() {
		for {
			b.logger.Info("start to get message")
			if err := b.client.LoopBlaze(ctx, mixin.BlazeListenFunc(b.run)); err != nil {
				b.logger.WithError(err).Error("loop blaze error")
			}

			select {
			case <-ctx.Done():
				b.logger.Info("get message chan done")
				close(b.msgChan)
				return
			case <-time.After(time.Second):
			}
		}
	}()

	return b.msgChan
}

func (b *Bot) GetResultChan(ctx context.Context) chan<- *service.Result {
	resultChan := make(chan *service.Result)
	go func() {
		for {
			select {
			case r := <-resultChan:
				b.logger.WithField("result", r).Info("get result")
				if r.Err != nil && r.IgnoreIfError {
					b.logger.WithError(r.Err).Error("ignore error")
					continue
				}

				msg := r.Message.Context.Value(messageKey{}).(mixin.MessageView)
				user := r.Message.Context.Value(userKey{}).(*mixin.User)
				conv := r.Message.Context.Value(convKey{}).(*mixin.Conversation)

				mq := &mixin.MessageRequest{
					ConversationID: msg.ConversationID,
					MessageID:      uuid.Modify(msg.MessageID, "reply"),
					Category:       msg.Category,
				}

				text := ""
				if r.Err != nil {
					text = r.Err.Error()
				} else {
					text = r.ConvTurn.Response
				}

				if conv.Category == mixin.ConversationCategoryGroup {
					text = fmt.Sprintf("> @%s %s\n\n%s", user.IdentityNumber, r.Message.Content, text)
				}
				mq.Data = base64.StdEncoding.EncodeToString([]byte(text))
				if err := b.client.SendMessage(ctx, mq); err != nil {
					b.logger.WithError(err).Error("send message error")
					go func() {
						time.Sleep(time.Second)
						resultChan <- r
					}()
				}
			case <-ctx.Done():
				b.logger.Info("get result chan done")
				close(resultChan)
				return
			}
		}
	}()

	return resultChan
}

func (b *Bot) run(ctx context.Context, msg *mixin.MessageView, userID string) error {
	if msg.Category != mixin.MessageCategoryPlainText {
		return nil
	}
	if uuid.IsNil(msg.UserID) {
		return nil
	}
	conv, err := b.getConversation(ctx, msg.ConversationID)
	if err != nil {
		log.Println("getConversation error:", err)
		return nil
	}
	user, err := b.getUser(ctx, msg.UserID)
	if err != nil {
		log.Println("getUser error:", err)
		return nil
	}
	if user.IdentityNumber == "0" {
		log.Println("user is not a messenger user, ignored")
		return nil
	}

	allowed := len(b.cfg.Whitelist) == 0
	for _, id := range b.cfg.Whitelist {
		if id == user.IdentityNumber || conv.ConversationID == id {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil
	}

	data, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		return nil
	}

	content := string(data)
	prefix := fmt.Sprintf("@%s", b.me.IdentityNumber)

	conversationKey := msg.ConversationID + ":" + msg.UserID

	// super group bot
	if strings.HasPrefix(user.IdentityNumber, "700") {
		if !strings.HasPrefix(content, prefix) || msg.RepresentativeID == "" {
			return nil
		}
		conversationKey = msg.ConversationID + ":" + msg.RepresentativeID
	}

	content = strings.TrimSpace(strings.TrimPrefix(content, prefix))

	ctx = context.WithValue(ctx, messageKey{}, *msg)
	ctx = context.WithValue(ctx, userKey{}, user)
	ctx = context.WithValue(ctx, convKey{}, conv)

	b.msgChan <- &service.Message{
		Context:      ctx,
		UserIdentity: msg.UserID,
		ConvKey:      conversationKey,
		Content:      content,
	}

	return nil
}

func (b *Bot) getConversation(ctx context.Context, convID string) (*mixin.Conversation, error) {
	if conv, ok := b.convMap[convID]; ok {
		return conv, nil
	}
	conv, err := b.client.ReadConversation(ctx, convID)
	if err != nil {
		return nil, err
	}
	b.convMap[convID] = conv
	return conv, nil
}

func (b *Bot) getUser(ctx context.Context, userID string) (*mixin.User, error) {
	if user, ok := b.userMap[userID]; ok {
		return user, nil
	}
	user, err := b.client.ReadUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	b.userMap[userID] = user
	return user, nil
}
