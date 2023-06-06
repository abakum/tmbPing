package main

import (
	"context"
	"regexp"
	"strings"

	"github.com/fasthttp/router"
	tg "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/valyala/fasthttp"
	"golang.ngrok.com/ngrok"
	nc "golang.ngrok.com/ngrok/config"
)

func tmtc(update tg.Update) (tc string, m *tg.Message) {
	if update.Message == nil {
		return "", nil
	}
	for _, tm := range []*tg.Message{
		update.Message,
		update.EditedMessage,
		update.ChannelPost,
		update.EditedChannelPost,
	} {
		if tm != nil {
			m = tm
			tc += tm.Text + " "
			tc += tm.Caption + " "
			re := tm.ReplyToMessage
			if re != nil {
				tc += re.Text + " "
				tc += re.Caption + " "
			}
			break
		}
	}
	return
}
func anyWithMatch(pattern *regexp.Regexp) th.Predicate {
	return func(update tg.Update) bool {
		tc, _ := tmtc(update)
		return pattern.MatchString(tc)
	}
}
func AnyCommand() th.Predicate {
	return func(update tg.Update) bool {
		_, ctm := tmtc(update)
		if ctm == nil {
			return false
		}
		return strings.HasPrefix(ctm.Text, "/") || strings.HasPrefix(ctm.Caption, "/")
	}
}
func leftChat() th.Predicate {
	return func(update tg.Update) bool {
		return update.Message != nil &&
			update.Message.LeftChatMember != nil
	}
}
func newMember() th.Predicate {
	return func(update tg.Update) bool {
		return update.Message != nil &&
			len(update.Message.NewChatMembers) > 0
	}
}

func ReplyMessageIsMinus() th.Predicate {
	return func(update tg.Update) bool {
		return update.Message != nil &&
			update.Message.ReplyToMessage != nil &&
			update.Message.Text == "-"
	}
}

func Delete(ChatID tg.ChatID, MessageID int) *tg.DeleteMessageParams {
	return &tg.DeleteMessageParams{
		ChatID:    ChatID,
		MessageID: MessageID,
	}
}

// UpdatesWithSecret set secretToken to FastHTTPWebhookServer and SetWebhookParams
func UpdatesWithSecret(b *tg.Bot, secretToken, publicURL, endPoint string) (<-chan tg.Update, tg.FastHTTPWebhookServer, error) {
	whs := tg.FastHTTPWebhookServer{
		Logger:      b.Logger(),
		Server:      &fasthttp.Server{},
		Router:      router.New(),
		SecretToken: secretToken,
	}
	updates, err := b.UpdatesViaWebhook(endPoint,
		tg.WithWebhookServer(whs),
		tg.WithWebhookSet(&tg.SetWebhookParams{
			URL:         publicURL + endPoint,
			SecretToken: secretToken,
		}))
	return updates, whs, err
}

// UpdatesWithNgrok start ngrok.Tunnel with os.Getenv("NGROK_AUTHTOKEN") and SecretToken
// for close ngrok.Tunnel use ngrok.Tunnel.Session().Close()
func UpdatesWithNgrok(b *tg.Bot, secretToken, forwardsTo, endPoint string) (<-chan tg.Update, *ngrok.Tunnel, error) {
	tun, err := ngrok.Listen(context.Background(), nc.HTTPEndpoint(
		nc.WithForwardsTo(forwardsTo)),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		return nil, nil, err
	}
	publicURL := tun.URL()
	defer func() {
		if err != nil {
			err = tun.Session().Close()
			b.Logger().Errorf("error close session of tunnel %v", err)
		}
	}()
	updates, whs, err := UpdatesWithSecret(b, secretToken, publicURL, endPoint)
	if err != nil {
		return nil, nil, err
	}
	go func() {
		for {
			conn, err := tun.Accept()
			if err != nil {
				b.Logger().Errorf("error accept connection %v", err)
				return
			}
			b.Logger().Debugf("%s => %s", conn.RemoteAddr().String(), conn.LocalAddr().String())
			go func() {
				err := whs.Server.ServeConn(conn)
				if err != nil {
					b.Logger().Errorf("error serving connection %v: %v", conn, err)
				}
			}()
		}
	}()
	return updates, &tun, err
}
