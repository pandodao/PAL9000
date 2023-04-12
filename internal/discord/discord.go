package discord

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/service"
)

var _ service.Adaptor = (*Bot)(nil)

type messageKey struct{}
type sessionKey struct{}

type Bot struct {
	cfg config.DiscordConfig
}

func New(cfg config.DiscordConfig) *Bot {
	return &Bot{
		cfg: cfg,
	}
}

func (b *Bot) GetMessageChan(ctx context.Context) <-chan *service.Message {
	msgChan := make(chan *service.Message)

	dg, _ := discordgo.New("Bot " + b.cfg.Token)
	dg.Identify.Intents = discordgo.IntentDirectMessages
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

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

func (b *Bot) GetResultChan(ctx context.Context) chan<- *service.Result {
	resultChan := make(chan *service.Result)
	go func() {
		for {
			select {
			case r := <-resultChan:
				text := ""
				if r.Err != nil {
					text = r.Err.Error()
				} else {
					text = r.ConvTurn.Response
				}
				msg := r.Message.Context.Value(messageKey{}).(*discordgo.MessageCreate)
				s := r.Message.Context.Value(sessionKey{}).(*discordgo.Session)
				if _, err := s.ChannelMessageSend(msg.ChannelID, text); err != nil {
					log.Printf("error sending message to Discord, %v\n", err)
				}
			case <-ctx.Done():
				close(resultChan)
				return
			}
		}
	}()

	return resultChan
}
