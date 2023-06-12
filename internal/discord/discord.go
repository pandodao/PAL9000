package discord

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/service"
)

var _ service.Adapter = (*Bot)(nil)

type messageKey struct{}
type sessionKey struct{}

type Bot struct {
	name string
	cfg  config.DiscordConfig
}

func New(name string, cfg config.DiscordConfig) *Bot {
	return &Bot{
		name: name,
		cfg:  cfg,
	}
}

func (b *Bot) GetName() string {
	return b.name
}

func (b *Bot) GetMessageChan(ctx context.Context) <-chan *service.Message {
	msgChan := make(chan *service.Message)

	dg, _ := discordgo.New("Bot " + b.cfg.Token)
	dg.Identify.Intents = discordgo.IntentGuildMessages | discordgo.IntentDirectMessages | discordgo.IntentMessageContent
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		// only text message
		if m.Type != discordgo.MessageTypeDefault {
			return
		}

		allowed := len(b.cfg.Whitelist) == 0
		for _, id := range b.cfg.Whitelist {
			if id == m.Author.ID || (m.GuildID != "" && id == m.GuildID) {
				allowed = true
				break
			}
		}

		if !allowed {
			return
		}

		prefix := fmt.Sprintf("<@%s>", s.State.User.ID)
		if m.GuildID != "" {
			// return if not mentioned
			if !strings.HasPrefix(m.Content, prefix) {
				return
			}
		}
		m.Content = strings.TrimSpace(strings.TrimPrefix(m.Content, prefix))

		ctx = context.WithValue(ctx, messageKey{}, m)
		ctx = context.WithValue(ctx, sessionKey{}, s)

		msgChan <- &service.Message{
			Context:      ctx,
			UserIdentity: m.Author.ID,
			Content:      m.Content,
			ConvKey:      m.ChannelID,
		}
	})

	go func() {
		if err := dg.Open(); err != nil {
			log.Printf("error opening connection to Discord, %v\n", err)
		}

		select {
		case <-ctx.Done():
			dg.Close()
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
	msg := req.Context.Value(messageKey{}).(*discordgo.MessageCreate)
	s := req.Context.Value(sessionKey{}).(*discordgo.Session)
	if _, err := s.ChannelMessageSend(msg.ChannelID, text); err != nil {
		log.Printf("error sending message to Discord, %v\n", err)
	}
}
