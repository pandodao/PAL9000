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
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

type Message struct {
	Content string
	UserID  string
}

type (
	messageKey struct{}
	userKey    struct{}
	convKey    struct{}
)

var _ service.Adapter = (*Bot)(nil)

type Bot struct {
	name    string
	convMap map[string]*mixin.Conversation
	userMap map[string]*mixin.User

	client       *mixin.Client
	msgChan      chan *service.Message
	me           *mixin.User
	cfg          config.MixinConfig
	logger       logrus.FieldLogger
	messageCache *cache.Cache
}

func Init(ctx context.Context, name string, cfg config.MixinConfig) (*Bot, error) {
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

	if cfg.MessageCacheExpiration == 0 {
		cfg.MessageCacheExpiration = 60 * 60 * 24
	}

	return &Bot{
		name:         name,
		convMap:      make(map[string]*mixin.Conversation),
		userMap:      make(map[string]*mixin.User),
		client:       client,
		msgChan:      make(chan *service.Message),
		cfg:          cfg,
		me:           me,
		logger:       logrus.WithField("adapter", "mixin").WithField("name", name),
		messageCache: cache.New(time.Duration(cfg.MessageCacheExpiration)*time.Second, 10*time.Minute),
	}, nil
}

func (b *Bot) GetName() string {
	return b.name
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

func (b *Bot) HandleResult(req *service.Message, r *service.Result) {
	defer close(req.DoneChan)

	b.logger.WithField("result", r).Info("get result")
	if r.Err != nil && r.IgnoreIfError {
		b.logger.WithError(r.Err).Error("ignore error")
		return
	}

	msg := req.Context.Value(messageKey{}).(*mixin.MessageView)
	user := req.Context.Value(userKey{}).(*mixin.User)
	conv := req.Context.Value(convKey{}).(*mixin.Conversation)

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

	b.messageCache.Add(mq.MessageID, &Message{
		Content: text,
		UserID:  b.me.UserID,
	}, cache.DefaultExpiration)

	if conv.Category == mixin.ConversationCategoryGroup {
		text = fmt.Sprintf("> @%s %s\n\n%s", user.IdentityNumber, req.Content, text)
	}
	mq.Data = base64.StdEncoding.EncodeToString([]byte(text))
	if err := b.client.SendMessage(req.Context, mq); err != nil {
		b.logger.WithError(err).Error("send message error")
	}
}

func (b *Bot) run(ctx context.Context, msg *mixin.MessageView, userID string) error {
	b.logger.WithField("msg", msg).Info("in run func, get message")

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

	data, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		return nil
	}
	content := string(data)
	prefix := fmt.Sprintf("@%s", b.me.IdentityNumber)

	b.messageCache.Add(msg.MessageID, &Message{
		Content: strings.TrimPrefix(content, prefix),
	}, cache.DefaultExpiration)

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

	conversationKey := msg.ConversationID + ":" + msg.UserID

	var quoteMessage *Message
	if msg.QuoteMessageID != "" {
		if v, ok := b.messageCache.Get(msg.QuoteMessageID); ok {
			quoteMessage = v.(*Message)
		}
	}

	// super group bot
	if strings.HasPrefix(user.IdentityNumber, "700") {
		if quoteMessage == nil || quoteMessage.UserID != b.me.UserID {
			if !strings.HasPrefix(content, prefix) || msg.RepresentativeID == "" {
				return nil
			}
		}
		conversationKey = msg.ConversationID + ":" + msg.RepresentativeID
	}

	replyContent := ""
	if quoteMessage != nil {
		replyContent = quoteMessage.Content
	}
	content = strings.TrimSpace(strings.TrimPrefix(content, prefix))

	ctx = context.WithValue(ctx, messageKey{}, msg)
	ctx = context.WithValue(ctx, userKey{}, user)
	ctx = context.WithValue(ctx, convKey{}, conv)

	doneChan := make(chan struct{})
	b.msgChan <- &service.Message{
		Context:      ctx,
		UserIdentity: msg.UserID,
		ConvKey:      conversationKey,
		ReplyContent: replyContent,
		Content:      content,
		DoneChan:     doneChan,
	}

	<-doneChan
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
