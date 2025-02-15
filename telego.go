package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/router"
	"github.com/loophole/cli/cmd"
	tg "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/valyala/fasthttp"
	"github.com/xlab/closer"
	"golang.ngrok.com/ngrok"
	nc "golang.ngrok.com/ngrok/config"
)

// concat tc from text and caption of message or post
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

// match pattern in text and caption of message or post
func anyWithMatch(pattern *regexp.Regexp) th.Predicate {
	return func(update tg.Update) bool {
		tc, _ := tmtc(update)
		return pattern.MatchString(tc)
	}
}

// is command in text and caption of update
func AnyCommand() th.Predicate {
	return func(update tg.Update) bool {
		_, ctm := tmtc(update)
		if ctm == nil {
			return false
		}
		return strings.HasPrefix(ctm.Text, "/") || strings.HasPrefix(ctm.Caption, "/")
	}
}

// is command in text and caption of update
func leftChat() th.Predicate {
	return func(update tg.Update) bool {
		return update.Message != nil &&
			update.Message.LeftChatMember != nil
	}
}

// is message about new member
func newMember() th.Predicate {
	return func(update tg.Update) bool {
		return update.Message != nil &&
			len(update.Message.NewChatMembers) > 0
	}
}

// is reply to bot message as delete command
func ReplyMessageIsMinus() th.Predicate {
	return func(update tg.Update) bool {
		return update.Message != nil &&
			update.Message.ReplyToMessage != nil &&
			update.Message.Text == "-"
	}
}

// func Delete(ChatID tg.ChatID, MessageID int) *tg.DeleteMessageParams {
// 	return &tg.DeleteMessageParams{
// 		ChatID:    ChatID,
// 		MessageID: MessageID,
// 	}
// }

// set secretToken to FastHTTPWebhookServer and SetWebhookParams
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

// start ngrok.Tunnel with NGROK_AUTHTOKEN in env (optional) and SecretToken
func UpdatesWithNgrok(b *tg.Bot, secretToken, endPoint string) (<-chan tg.Update, error) {
	var (
		err error
		tun ngrok.Tunnel
	)
	// If NGROK_AUTHTOKEN in env and account is free and is already open need return
	// else case ngrok.Listen hang
	ctx, ca := context.WithTimeout(context.Background(), time.Second)
	sess, err := ngrok.Connect(ctx, ngrok.WithAuthtokenFromEnv()) //even without NGROK_AUTHTOKEN in env
	if err != nil {
		return nil, Errorf("tunnel already open %w", err)
	}
	sess.Close()
	ca()

	ctx, ca = context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			ca()
		}
	}()
	tun, err = ngrok.Listen(
		ctx,
		nc.HTTPEndpoint(),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		return nil, srcError(err)
	}
	publicURL := tun.URL()
	if secretToken == "" {
		secretToken = tun.ID()
	}
	if endPoint == "" {
		endPoint = "/" + secretToken
	}

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
				ltf.Println("Serve ok")
				return nil
			}
			letf.Println("Serve", err)
			return srcError(err)
		},
		// Override default stop func to close Ngrok tunnel
		StopFunc: func(_ context.Context) error {
			ltf.Println("StopFunc")
			ca() //need for NGROK_AUTHTOKEN in env
			return nil
		},
	}
	return b.UpdatesViaWebhook(endPoint,
		tg.WithWebhookServer(fws),
		tg.WithWebhookSet(whp))
}

// start ngrok.Tunnel with NGROK_AUTHTOKEN in env (optional) and SecretToken not used serve but loop of accept
func UpdatesWithNgrokAccept(b *tg.Bot, secretToken, endPoint string) (<-chan tg.Update, error) {
	var (
		err error
		tun ngrok.Tunnel
	)
	// If NGROK_AUTHTOKEN in env and account is free and is already open need return
	// else case ngrok.Listen hang
	ctx, ca := context.WithTimeout(context.Background(), time.Second)
	sess, err := ngrok.Connect(ctx, ngrok.WithAuthtokenFromEnv()) //even without NGROK_AUTHTOKEN in env
	if err != nil {
		return nil, Errorf("tunnel already open %w", err)
	}
	sess.Close()
	ca()

	ctx, ca = context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			ca()
		}
	}()
	tun, err = ngrok.Listen(
		ctx,
		nc.HTTPEndpoint(),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		return nil, srcError(err)
	}
	publicURL := tun.URL()
	if secretToken == "" {
		secretToken = tun.ID()
	}
	if endPoint == "" {
		endPoint = "/" + secretToken
	}
	b.Logger().Debugf("%s %s %s %s", publicURL, tun.ForwardsTo(), secretToken, endPoint)

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
		// Override default stop func to close Ngrok tunnel
		StopFunc: func(_ context.Context) error {
			b.Logger().Debugf("StopFunc")
			ca() //need for NGROK_AUTHTOKEN in env
			return nil
		},
	}

	go func() {
		for {
			conn, err := tun.Accept()
			if err != nil {
				b.Logger().Errorf("tun.Accept %v", err)
				return
			}
			b.Logger().Debugf("%s => %s", conn.RemoteAddr().String(), conn.LocalAddr().String())
			go func() {
				err := whs.Server.ServeConn(conn)
				if err != nil {
					b.Logger().Errorf("Server.ServeConn(%v): %v", conn, err)
				}
				b.Logger().Debugf("Server.ServeConn ok")
			}()
		}
	}()

	return b.UpdatesViaWebhook(endPoint,
		tg.WithWebhookServer(fws),
		tg.WithWebhookSet(whp))
}

// convert forwardsTo to host:port
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

// start webhook over ngrok channel
func ngrokWebHook(bot *tg.Bot) (updates <-chan tg.Update, err error) {
	var (
		endPoint = "/" + fmt.Sprint(time.Now().Format("2006010215040501"))
		secret   = endPoint[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(endPoint)-3)+1:]
	)
	//try ngrok.exe client with web interface with web_addr in env for debug
	publicURL, forwardsTo, err := ngrokWeb()
	if err != nil {
		lt.Println(err)
		//for ngrok.exe client without web interface but with NGROK_API_KEY in env
		publicURL, forwardsTo, err = ngrokAPI(os.Getenv("NGROK_API_KEY"))
		if err != nil {
			lt.Println(err)
			//use ngrok-go client
			forwardsTo = Getenv("forwardsTo", "https://localhost")
			updates, err = UpdatesWithNgrok(bot, "", endPoint)
			if err == nil {
				ltf.Println("UpdatesWithNgrok")
				if tt != tth {
					tt = tth
					tacker.Reset(tth) // next reconnect after tth
					SendError(bot, Errorf("UpdatesWithNgrok"))
				}
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
		return nil, srcError(err)
	}

	go func() {
		err = bot.StartWebhook(addressWebHook(forwardsTo))
		if err != nil {
			PrintOk("StartWebhook", err)
			closer.Close()
		}
	}()

	return updates, nil
}

// is webhook ip same as ngrok endpoint ip
func manInTheMiddle(bot *tg.Bot) bool {
	// Receive information about webhook
	info, err := bot.GetWebhookInfo()
	if err != nil {
		return false
	}
	// stdo.Printf("Webhook Info: %+v\n", info)
	if info.IPAddress == "" || info.URL == "" {
		return false
	}

	//test ip of webhook
	u, err := url.Parse(info.URL)
	if err != nil {
		return false
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.String() == info.IPAddress {
			return false
		}
	}
	letf.Printf("manInTheMiddle GetWebhookInfo.IPAddress: %v but GetWebhookInfo.URL ip:%v\n", info.IPAddress, ips)
	return true
}

// telego logger interface
type Logger struct{}

// hide bot token
func woToken(format string, args ...any) (s string) {
	s = src(10) + " " + fmt.Sprintf(format, args...)
	btStart := strings.Index(s, "/bot") + 4
	if btStart > 4-1 {
		btLen := strings.Index(s[btStart:], "/")
		if btLen > 0 {
			s = s[:btStart] + s[btStart+btLen:]
		}
	}
	return
}

// bot debug message
func (Logger) Debugf(format string, args ...any) {
	if format == "API response %s: %s" && args[0] == "getUpdates" && len(getUpdates) == 0 {
		getUpdates <- true
	}
	lt.Print(woToken(format, args...))
}

// bot error message
func (Logger) Errorf(format string, args ...any) {
	let.Print(woToken(format, args...))
}

// send error message to first chatID in args
func SendError(bot *tg.Bot, err error) {
	if bot != nil && len(chats) > 0 && err != nil {
		bot.SendMessage(tu.MessageWithEntities(tu.ID(chats[0]),
			tu.Entity("💥"),
			tu.Entity(err.Error()).Code(),
		))
	}
}

// // UpdatesWithSecret set secretToken to FastHTTPWebhookServer and SetWebhookParams
// func (b *Bot) UpdatesWithSecret(secretToken, publicURL, endPoint string) (<-chan Update, FastHTTPWebhookServer, error) {
// 	whs := FastHTTPWebhookServer{
// 		Logger:      b.Logger(),
// 		Server:      &fasthttp.Server{},
// 		Router:      router.New(),
// 		SecretToken: secretToken,
// 	}
// 	updates, err := b.UpdatesViaWebhook(endPoint,
// 		WithWebhookServer(whs),
// 		WithWebhookSet(&SetWebhookParams{
// 			URL:         publicURL + endPoint,
// 			SecretToken: secretToken,
// 		}))
// 	return updates, whs, err
// }

// // UpdatesWithNgrok start ngrok.Tunnel with os.Getenv("NGROK_AUTHTOKEN") and SecretToken
// // for close ngrok.Tunnel use ngrok.Tunnel.Session().Close()
// func (b *Bot) UpdatesWithNgrok(secretToken, forwardsTo, endPoint string) (<-chan Update, *ngrok.Tunnel, error) {
// 	tun, err := ngrok.Listen(context.Background(), config.HTTPEndpoint(
// 		config.WithForwardsTo(forwardsTo)),
// 		ngrok.WithAuthtokenFromEnv(),
// 	)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	publicURL := tun.URL()
// 	defer func() {
// 		if err != nil {
// 			err = tun.Session().Close()
// 			b.log.Errorf("error close session of tunnel %v", err)
// 		}
// 	}()
// 	updates, whs, err := b.UpdatesWithSecret(secretToken, publicURL, endPoint)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	go func() {
// 		for {
// 			conn, err := tun.Accept()
// 			if err != nil {
// 				b.log.Errorf("error accept connection %v", err)
// 				return
// 			}
// 			b.log.Debugf("%s => %s", conn.RemoteAddr().String(), conn.LocalAddr().String())
// 			go func() {
// 				err := whs.Server.ServeConn(conn)
// 				if err != nil {
// 					b.log.Errorf("error serving connection %v: %v", conn, err)
// 				}
// 			}()
// 		}
// 	}()
// 	return updates, &tun, err
// }

// start webhook
func webHook(bot *tg.Bot) (updates <-chan tg.Update, err error) {
	var (
		endPoint = "/" + fmt.Sprint(time.Now().Format("2006010215040501"))
		secret   = endPoint[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(endPoint)-3)+1:]
	)
	forwardsTo := Getenv("forwardsTo", "127.0.0.1:80")
	publicURL := os.Getenv("publicURL")
	k := ""
	funcs := []func() (<-chan tg.Update, error){}
	funcs = append(funcs, func() (<-chan tg.Update, error) {
		k = "loophole.site"
		if strings.HasSuffix(publicURL, "."+k) {
			hostname := strings.TrimSuffix(publicURL, "."+k)
			hostname = strings.TrimPrefix(hostname, "http://")
			hostname = strings.TrimPrefix(hostname, "https://")
			h, p, err := net.SplitHostPort(forwardsTo)
			if err != nil {
				h = "127.0.0.1"
				p = "80"
			}
			ltf.Printf("cli --hostname %s http %s %s", hostname, p, h)
			ctx, cancel = context.WithCancel(context.Background())
			go cmd.GoExecute(ctx, "1.0.0-beta.15", "5cecf33", "cli", "--hostname", hostname, "http", p, h)
			return UpdatesWithSecret(bot, secret, publicURL, endPoint)
		}
		return nil, fmt.Errorf("not used %s", k)
	})
	funcs = append(funcs, func() (<-chan tg.Update, error) {
		k = "devhook.ru"
		if strings.HasSuffix(publicURL, "."+k) {
			devhook := strings.TrimSuffix(publicURL, "."+k)
			// ssh -p 2222 -R %devhook%:80:%forwardsTo% devhook.ru
			ltf.Printf("ssh -p 2222 -R %s:80:%s %s", addressWebHook(devhook), forwardsTo, k)
			return UpdatesWithSecret(bot, secret, publicURL, endPoint)
		}
		return nil, fmt.Errorf("not used %s", k)
	})
	funcs = append(funcs, func() (<-chan tg.Update, error) {
		k = "cloudpub.ru"
		if strings.HasSuffix(publicURL, "."+k) {
			ltf.Println("clo run")
			return UpdatesWithSecret(bot, secret, publicURL, endPoint)
		}
		return nil, fmt.Errorf("not used %s", k)
	})
	funcs = append(funcs, func() (<-chan tg.Update, error) {
		k = "DDNS"
		if publicURL != "" {
			return UpdatesWithSecret(bot, secret, publicURL, endPoint)
		}
		return nil, fmt.Errorf("not used %s", k)
	})
	funcs = append(funcs, func() (<-chan tg.Update, error) {
		k = "getExternalIP"
		if publicURL == "" {
			p := ""
			p, k, err = GetExternalIP(time.Second, "stun.sipnet.ru:3478", "stun.l.google.com:19302", "stun.fitauto.ru:3478")
			if err == nil {
				publicURL = "http://" + p
				return UpdatesWithSecret(bot, secret, publicURL, endPoint)
			}
		}
		return nil, fmt.Errorf("not used %s", k)
	})
	funcs = append(funcs, func() (<-chan tg.Update, error) {
		k = "web_addr"
		if v := os.Getenv(k); v != "" {
			if p, f, err := ngrokWeb(); err == nil {
				publicURL, forwardsTo = p, f
				return UpdatesWithSecret(bot, secret, publicURL, endPoint)
			}
		}
		return nil, fmt.Errorf("not used %s", k)
	})
	funcs = append(funcs, func() (<-chan tg.Update, error) {
		k = "NGROK_API_KEY"
		if v := os.Getenv(k); v != "" {
			if p, f, err := ngrokAPI(v); err == nil {
				publicURL, forwardsTo = p, f
				return UpdatesWithSecret(bot, secret, publicURL, endPoint)
			}
		}
		return nil, fmt.Errorf("not used %s", k)
	})
	funcs = append(funcs, func() (<-chan tg.Update, error) {
		k = "NGROK_AUTHTOKEN"
		if v := os.Getenv(k); v != "" {
			if p, f, err := ngrokAPI(v); err == nil {
				publicURL, forwardsTo = p, f
				updates, err = UpdatesWithNgrok(bot, "", endPoint)
				if err == nil && tt != tth {
					tt = tth
					tacker.Reset(tt)
				}
				return updates, err
			}
		}
		return nil, fmt.Errorf("not used %s", k)
	})
	for _, f := range funcs {
		updates, err = f()
		if err == nil {
			ltf.Printf("%s", k)
			SendError(bot, Errorf(k))
			break
		}
		ltf.Println(k, publicURL, forwardsTo, err)
	}
	if err != nil {
		return nil, srcError(err)
	}
	chErr := make(chan error)
	var once sync.Once
	time.AfterFunc(time.Second, func() { once.Do(func() { chErr <- nil }) })
	go func() {
		err := bot.StartWebhook(addressWebHook(forwardsTo))
		once.Do(func() { chErr <- err })
		tt = ttm
		tacker.Reset(tt)
	}()
	err = <-chErr
	return updates, srcError(err)
}
