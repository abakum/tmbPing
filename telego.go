package main

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fasthttp/router"
	tg "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/valyala/fasthttp"
	"github.com/xlab/closer"
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
func UpdatesWithSecret(b *tg.Bot, secretToken, publicURL, endPoint string) (<-chan tg.Update, error) {
	whs := tg.FastHTTPWebhookServer{
		Logger:      b.Logger(),
		Server:      &fasthttp.Server{},
		Router:      router.New(),
		SecretToken: secretToken,
	}
	whp := &tg.SetWebhookParams{
		URL:         publicURL + endPoint,
		SecretToken: secretToken,
	}
	return b.UpdatesViaWebhook(endPoint,
		tg.WithWebhookServer(whs),
		tg.WithWebhookSet(whp))
}

// UpdatesWithNgrok start ngrok.Tunnel with NGROK_AUTHTOKEN in env (optional) and SecretToken
func UpdatesWithNgrok(b *tg.Bot, secretToken, forwardsTo, endPoint string) (<-chan tg.Update, error) {
	var (
		err error
		tun ngrok.Tunnel
	)
	// If NGROK_AUTHTOKEN in env and account is free and is already open need return
	// else case ngrok.Listen hang
	ct, ca := context.WithTimeout(context.Background(), time.Second)
	sess, err := ngrok.Connect(ct, ngrok.WithAuthtokenFromEnv()) //even without NGROK_AUTHTOKEN in env
	if err != nil {
		return nil, err
	}
	sess.Close()
	ca()

	ct, ca = context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			ca()
		}
	}()
	tun, err = ngrok.Listen(
		ct,
		nc.HTTPEndpoint(nc.WithForwardsTo(forwardsTo)),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		return nil, err
	}
	publicURL := tun.URL()
	ltf.Println(publicURL, tun.ForwardsTo(), tun.ID())
	whs := tg.FastHTTPWebhookServer{
		Logger:      b.Logger(),
		Server:      &fasthttp.Server{},
		Router:      router.New(),
		SecretToken: secretToken,
	}
	whp := &tg.SetWebhookParams{
		URL:         publicURL + endPoint,
		SecretToken: secretToken,
	}
	fws := tg.FuncWebhookServer{
		Server: whs,
		// Override default start func to use Ngrok tunnel
		StartFunc: func(address string) error {
			ltf.Println("StartFunc", address)
			err := whs.Server.Serve(tun) //always return error
			if err.Error() == "failed to accept connection: Tunnel closed" {
				return nil
			}
			letf.Println("Serve", err)
			return err
		},
		// Override default stop func to close Ngrok tunnel
		StopFunc: func(_ context.Context) error {
			ltf.Println("StopFunc")
			ca() //need for NGROK_AUTHTOKEN in env
			return nil
			// err := whs.Server.ShutdownWithContext(ctx)
			// if err != nil {
			// 	letf.Println("ShutdownWithContext", err)
			// }
			// return err
		},
	}
	return b.UpdatesViaWebhook(endPoint,
		tg.WithWebhookServer(fws),
		tg.WithWebhookSet(whp))
}

func addressWebHook(forwardsTo string) (hp string) {
	hp = strings.TrimPrefix(forwardsTo, ":")
	_, err := strconv.Atoi(hp)
	if err == nil {
		hp = "127.0.0.1:" + hp
	}
	for k, v := range map[string]string{
		"http://":  ":80",
		"https://": ":443",
	} {
		if strings.HasPrefix(hp, k) {
			hp = strings.TrimPrefix(hp, k)
			if !strings.Contains(hp, ":") {
				hp += v
			}
			break
		}
	}
	return
}

func ngrokWebHook(bot *tg.Bot) (updates <-chan tg.Update, err error) {
	var (
		endPoint = "/" + fmt.Sprint(time.Now().Format("2006010215040501"))
		secret   = endPoint[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(endPoint)-3)+1:]
	)
	//try ngrok.exe client with web interface with web_addr in env for debug
	publicURL, forwardsTo, err := ngrokWeb()
	if err != nil {
		//for ngrok.exe client without web interface but with NGROK_API_KEY in env
		publicURL, forwardsTo, err = ngrokAPI()
		if err != nil {
			//use ngrok-go client
			forwardsTo = Getenv("forwardsTo", "https://localhost")
			updates, err = UpdatesWithNgrok(bot, secret, forwardsTo, endPoint)
			if err != nil {
				ltf.Println(forwardsTo)
				updates, err = UpdatesWithSecret(bot, secret, forwardsTo, endPoint) //no ngrok
			} else {
				ltf.Println("UpdatesWithNgrok", forwardsTo)
			}
		} else {
			ltf.Println("ngrokAPI", publicURL, forwardsTo)
			updates, err = UpdatesWithSecret(bot, secret, publicURL, endPoint) //publicURL from ngrokAPI
		}
	} else {
		ltf.Println("ngrokWeb", publicURL, forwardsTo)
		updates, err = UpdatesWithSecret(bot, secret, publicURL, endPoint)
	}
	if err != nil {
		return nil, err
	}

	go func() {
		err = bot.StartWebhook(addressWebHook(forwardsTo))
		if err != nil {
			letf.Println("StartWebhook", err)
			closer.Close()
		}
	}()

	return updates, err
}
