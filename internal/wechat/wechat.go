package wechat

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/PAL9000/service"
)

type httpRequsetKey struct{}
type httpResponseKey struct{}
type rawMessageKey struct{}

type TextMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        int64    `xml:"MsgId"`
}

type Bot struct {
	name string
	cfg  config.WeChatConfig
}

func New(name string, cfg config.WeChatConfig) *Bot {
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
	go func() {
		server := &http.Server{
			Addr:    b.cfg.Address,
			Handler: http.DefaultServeMux,
		}

		validateSignature := func(signature, timestamp, nonce string) bool {
			params := []string{b.cfg.Token, timestamp, nonce}
			sort.Strings(params)
			combined := strings.Join(params, "")

			hash := sha1.New()
			hash.Write([]byte(combined))
			hashStr := hex.EncodeToString(hash.Sum(nil))

			return hashStr == signature
		}

		http.HandleFunc(b.cfg.Path, func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			signature := r.Form.Get("signature")
			timestamp := r.Form.Get("timestamp")
			nonce := r.Form.Get("nonce")
			echostr := r.Form.Get("echostr")

			if !validateSignature(signature, timestamp, nonce) {
				http.Error(w, "Invalid signature", http.StatusForbidden)
				return
			}

			if r.Method == "GET" {
				w.Write([]byte(echostr))
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			fmt.Println(string(body))

			var receivedMessage TextMessage
			err = xml.Unmarshal(body, &receivedMessage)
			if err != nil {
				http.Error(w, "Failed to parse request body", http.StatusBadRequest)
				return
			}

			ctx = r.Context()
			ctx = context.WithValue(ctx, httpRequsetKey{}, r)
			ctx = context.WithValue(ctx, httpResponseKey{}, w)
			ctx = context.WithValue(ctx, rawMessageKey{}, receivedMessage)
			doneChan := make(chan struct{})
			msgChan <- &service.Message{
				Context:      ctx,
				UserIdentity: receivedMessage.FromUserName,
				ConvKey:      receivedMessage.FromUserName,
				Content:      receivedMessage.Content,
				DoneChan:     doneChan,
			}
			<-doneChan
		})

		go func() {
			fmt.Printf("wechat HTTP server run at: %s\n", b.cfg.Address)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
			}
		}()

		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v\n", err)
		} else {
			log.Println("Server gracefully stopped")
		}
	}()

	return msgChan
}

func (b *Bot) HandleResult(req *service.Message, r *service.Result) {
	defer close(req.DoneChan)

	w := req.Context.Value(httpResponseKey{}).(http.ResponseWriter)
	if r.Err != nil && r.IgnoreIfError {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Write([]byte("<xml></xml>"))
		return
	}
	receivedMessage := req.Context.Value(rawMessageKey{}).(TextMessage)

	text := ""
	if r.Err != nil {
		text = r.Err.Error()
	} else {
		text = r.ConvTurn.Response
	}

	responseMessage := TextMessage{
		ToUserName:   receivedMessage.FromUserName,
		FromUserName: receivedMessage.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      text,
	}

	responseXML, err := xml.MarshalIndent(responseMessage, "", "  ")
	if err != nil {
		http.Error(w, "Failed to create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write(responseXML)
}
