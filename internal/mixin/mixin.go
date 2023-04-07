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
)

type (
	messageKey struct{}
	userKey    struct{}
	convKey    struct{}
)

var _ service.Adaptor = (*Bot)(nil)

type Bot struct {
	convMap map[string]*mixin.Conversation
	userMap map[string]*mixin.User

	client  *mixin.Client
	msgChan chan *service.Message
	me      *mixin.User
	cfg     config.MixinConfig
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
	}, nil
}

func (b *Bot) Name() string {
	return "mixin_bot"
}

func (b *Bot) GetMessageChan(ctx context.Context) <-chan *service.Message {
	go func() {
		for {
			if err := b.client.LoopBlaze(ctx, mixin.BlazeListenFunc(b.run)); err != nil {
				log.Printf("mixinClient.LoopBlaze error: %v\n", err)
			}

			select {
			case <-ctx.Done():
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
				if err := b.handleResult(ctx, r); err != nil {
					log.Printf("handleResult error: %v\n", err)
				}
			case <-ctx.Done():
				close(resultChan)
				return
			}
		}
	}()

	return resultChan
}

func (b *Bot) handleResult(ctx context.Context, r *service.Result) error {
	msg := r.Message.Context.Value(messageKey{}).(*mixin.MessageView)
	user := r.Message.Context.Value(userKey{}).(*mixin.User)
	conv := r.Message.Context.Value(convKey{}).(*mixin.Conversation)

	mq := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		MessageID:      uuid.Modify(msg.MessageID, "reply"),
		Category:       msg.Category,
		Data:           msg.Data,
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
	return b.client.SendMessage(ctx, mq)
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

	if user.IdentityNumber == "0" || strings.HasPrefix(user.IdentityNumber, "700") {
		log.Println("user is not a messenger user, ignored")
		return nil
	}

	data, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		return nil
	}

	content := string(data)
	isGroup := conv.Category == mixin.ConversationCategoryGroup
	prefix := fmt.Sprintf("@%s", b.me.IdentityNumber)
	if isGroup {
		content = strings.TrimSpace(strings.TrimPrefix(content, prefix))
		content = strings.TrimSpace(content)
	}

	ctx = context.WithValue(ctx, messageKey{}, msg)
	ctx = context.WithValue(ctx, userKey{}, user)
	ctx = context.WithValue(ctx, convKey{}, conv)

	b.msgChan <- &service.Message{
		Context:      ctx,
		UserIdentity: msg.UserID,
		ConvKey:      msg.ConversationID + ":" + msg.UserID,
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
