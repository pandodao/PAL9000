package mixinbot

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/fox-one/pkg/uuid"
	"github.com/pandodao/PAL9000/botastic"
	pal9000 "github.com/pandodao/PAL9000/service"
)

type Bot struct {
	convMap map[string]*mixin.Conversation
	userMap map[string]*mixin.User

	client  *mixin.Client
	me      *mixin.User
	handler *pal9000.Handler
}

func New(client *mixin.Client, handler *pal9000.Handler) *Bot {
	return &Bot{
		convMap: make(map[string]*mixin.Conversation),
		userMap: make(map[string]*mixin.User),
		client:  client,
		handler: handler,
	}
}

func (b *Bot) SetUserMe(ctx context.Context) error {
	me, err := b.client.UserMe(ctx)
	if err != nil {
		return fmt.Errorf("client.UserMe error: %v", err)
	}
	b.me = me
	return nil
}

func (b *Bot) Run(ctx context.Context) error {
	for {
		if err := b.client.LoopBlaze(ctx, mixin.BlazeListenFunc(b.run)); err != nil {
			log.Printf("mixinClient.LoopBlaze error: %v\n", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
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

	b.handler.HandleWithCallback(ctx, &pal9000.Message{
		BotID:        2,
		UserIdentity: msg.UserID,
		ConvKey:      msg.ConversationID + ":" + msg.UserID,
		Content:      string(data),
		Lang:         "zh",
	}, func(turn *botastic.ConvTurn, err error) error {
		mq := &mixin.MessageRequest{
			ConversationID: msg.ConversationID,
			MessageID:      uuid.Modify(msg.MessageID, "reply"),
			Category:       msg.Category,
			Data:           msg.Data,
		}
		text := ""
		if err != nil {
			text = err.Error()
		} else {
			text = turn.Response
		}
		if isGroup {
			text = fmt.Sprintf(">  %s\n\n%s", content, text)
		}
		mq.Data = base64.StdEncoding.EncodeToString([]byte(text))
		// fmt.Println(content)
		// fmt.Println(text)
		b.client.SendMessage(ctx, mq)
		return nil
	})

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
