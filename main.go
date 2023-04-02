package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/ngrok/ngrok-api-go/v5"
	"github.com/ngrok/ngrok-api-go/v5/tunnels"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/xlab/closer"
)

func ngrokUrlAddr() (PublicURL string, host string, err error) {
	web_addr := os.Getenv("web_addr")
	if web_addr == "" {
		web_addr = "localhost:4040"
	}
	// https://mholt.github.io/json-to-go/
	var ngrok struct {
		Tunnels []struct {
			Name      string `json:"name"`
			ID        string `json:"ID"`
			URI       string `json:"uri"`
			PublicURL string `json:"public_url"`
			Proto     string `json:"proto"`
			Config    struct {
				Addr    string `json:"addr"`
				Inspect bool   `json:"inspect"`
			} `json:"config"`
		} `json:"tunnels"`
		URI string `json:"uri"`
	}
	resp, err := http.Get("http://" + web_addr + "/api/tunnels")
	if err != nil {
		fmt.Println("ngrokUrlAddr http.Get error:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("ngrokUrlAddr http.Get resp.StatusCode: %v", resp.StatusCode)
		fmt.Println(err)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ngrokUrlAddr io.ReadAll error:", err)
		return
	}
	err = json.Unmarshal(body, &ngrok)
	if err != nil {
		fmt.Println("ngrokUrlAddr json.Unmarshal error:", err)
		return
	}
	for _, tunnel := range ngrok.Tunnels {
		PublicURL = tunnel.PublicURL
		u, err := url.Parse(tunnel.Config.Addr)
		if err != nil {
			fmt.Println("ngrokUrlAddr url.Parse error:", err)
			return PublicURL, tunnel.Config.Addr, err
		}
		host = u.Host
		if PublicURL != "" && host != "" {
			break
		}
	}
	return
}

func ngrokUrlTo(ctx context.Context, NGROK_API_KEY string) (PublicURL string, host string, err error) {
	// construct the api client
	clientConfig := ngrok.NewClientConfig(NGROK_API_KEY)

	// list all online tunnels
	tunnels := tunnels.NewClient(clientConfig)
	iter := tunnels.List(nil)
	err = iter.Err()
	if err != nil {
		fmt.Println("ngrokUrlTo tunnels.NewClient.List error:", err)
		return
	}
	for iter.Next(ctx) {
		err = iter.Err()
		if err != nil {
			fmt.Println("ngrokUrlTo tunnels.NewClient.Next error:", err)
			return
		}
		PublicURL = iter.Item().PublicURL
		u, err := url.Parse(iter.Item().ForwardsTo)
		if err != nil {
			fmt.Println("ngrokUrlTo url.Parse error:", err)
			return PublicURL, iter.Item().ForwardsTo, err
		}
		host = u.Host
		if PublicURL != "" && host != "" {
			break
		}
	}
	return
}

func manInTheMiddle(bot *telego.Bot) bool {
	// Receive information about webhook
	info, err := bot.GetWebhookInfo()
	if err != nil {
		return false
	}
	fmt.Printf("Webhook Info: %+v\n", info)
	if info.IPAddress == "" || info.URL == "" {
		return false
	}

	//test ip of webhook
	u, err := url.Parse(info.URL)
	if err != nil {
		return false
	}
	ips, err := net.LookupIP(strings.Split(u.Host, ":")[0])
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.String() == info.IPAddress {
			return false
		}
	}
	fmt.Printf("manInTheMiddle GetWebhookInfo.IPAddress: %v but GetWebhookInfo.URL ip:%v\n", info.IPAddress, ips)
	return true
}

type customer struct {
	tm    *telego.Message
	del   bool
	reply *telego.Message
}
type mCustomer map[string]customer
type cCustomer chan customer
type mcCustomer map[string]cCustomer
type sCustomer struct {
	sync.RWMutex
	mcCustomer
	mCustomer
}

/*
	 func (s *sCustomer) status() (bs string){
		s.RLock()
	    defer s.RUnlock()
		sPing :="✅" //:white_check_mark:
		if stopwatch {
			sPing="⏱️" //:stopwatch:
		}
		bs := "Bot ищет в чате IP адреса для 🏓" //:ping:
		If len(s.mCustomer) > 0 {
			bs := "🔁" //:repeat:
			if repeatOne{
				bs="🔂" //:repeat_one:
			}
			bs+="🏓"
			for ip,c := range s.mCustomer{
				//message_id = ips(i)(1): message_idB = ips(i)(2): dt = ips(i)(3)
				sPing += "\n"
				sPing += fmt.Sprintf("[:inbox_tray:](https://t.me/c/%v/%v) ",chat_id, message_id)
				if len(message_idB)>0{
					sPing += fmt.Sprintf("[:outbox_tray:](https://t.me/c/%v/%v) ",chat_id,message_idB)
				}
				sPing += pingS(ip, dPing(ip))
			}
		}
		bs += sPing
	}
*/
func (s *sCustomer) del(ip string, closed bool) {
	fmt.Println("sCustomer.del ", ip)
	s.Lock()
	defer s.Unlock()
	if !closed {
		ch, ok := s.mcCustomer[ip]
		if ok {
			close(ch)
		}
	}
	delete(s.mcCustomer, ip)
	//delete(s.mCustomer, ip)
}
func (s *sCustomer) add(ip string) (ch cCustomer) {
	fmt.Println("sCustomer.add ", ip)
	ch = make(cCustomer, 10)
	go worker(ip, ch)
	s.Lock()
	defer s.Unlock()
	s.mcCustomer[ip] = ch
	//s.mCustomer[ip] = mCustomer{}
	return
}

func (s *sCustomer) write(ip string, c customer) {
	fmt.Println("sCustomer.write ", ip, c)
	defer func() {
		// recover from panic caused by writing to a closed channel
		if err := recover(); err != nil {
			fmt.Println("sCustomer.write error:", err)
			s.del(ip, true)
			return
		}
	}()
	s.RLock()
	ch, ok := s.mcCustomer[ip]
	s.RUnlock()
	if ok {
		ch <- c
	} else {
		s.add(ip) <- c
	}
	/* if c.tm!=nil{
		s.Lock()
		defer s.Unlock()
		s.mCustomer[ip]=c
	} */
}

func m2kv[K comparable, V any](m map[K]V) (keys []K, vals []V) {
	keys = make([]K, 0, len(m))
	vals = make([]V, 0, len(m))
	for key, val := range m {
		keys = append(keys, key)
		vals = append(vals, val)
	}
	return
}

func (s *sCustomer) update() {
	s.RLock()
	k, _ := m2kv(s.mcCustomer)
	s.RUnlock()
	for _, ip := range k {
		s.write(ip, customer{})
	}
}
func ping(ip string) (status string, err error) {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return
	}
	defer pinger.Stop()
	pinger.SetPrivileged(runtime.GOOS == "windows")
	pinger.Count = 3
	pinger.Interval = time.Millisecond * 100
	pinger.Timeout = pinger.Interval*time.Duration(pinger.Count-1) + time.Millisecond*time.Duration(pinger.Count*100)
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return
	}
	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats
	if stats.PacketsRecv == pinger.Count {
		status = "✅"
		fmt.Printf("%v echoReply %d<rtt~%d<%d\n", ip, stats.MinRtt.Milliseconds(), stats.AvgRtt.Milliseconds(), stats.MaxRtt.Milliseconds())
	} else {
		status = "❗"
		fmt.Printf("%v %d/%d packets received\n", ip, stats.PacketsRecv, pinger.Count)
	}
	return
}

type customers []customer

func worker(ip string, ch cCustomer) {
	var err error
	status := ""
	// timeout:=8*60
	timeout := 3 //3 min
	cus := customers{}
	defer ips.del(ip, false)
	for {
		select {
		case <-done:
			fmt.Println("worker done", ip)
			done <- true //done other worker
			return
		case cust, ok := <-ch:
			if !ok {
				fmt.Println("worker channel closed", ip)
				return
			}
			if cust.tm == nil { //update
				timeout--
				if timeout < 1 {
					fmt.Println("worker timeout", ip)
					return
				}
			} else {
				tmr := cust.tm.ReplyToMessage
				if tmr != nil && tmr.From.ID == me.ID { //reply to me
					nCus := customers{}
					for _, cu := range cus {
						if cu.tm.Chat.ID == tmr.Chat.ID {
							if cu.reply != nil && tmr.Chat.ID == cu.reply.Chat.ID && tmr.MessageID != cu.reply.MessageID {
								bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(cu.reply.Chat.ID), MessageID: cu.reply.MessageID})
							}
						} else {
							nCus = append(nCus, cu)
						}
					}
					fmt.Println(cus)
					fmt.Println(nCus)
					cus = nCus
					fmt.Println(cus)
					if len(cus) == 0 {
						fmt.Println("worker forget", ip, tmr.Chat.ID, tmr.MessageID)
						return
					}
					continue
				} else {
					cus = append(cus, cust)
				}
			}
			fmt.Println("worker", ip, cust, len(ch))
			oStatus := status
			status, err = ping(ip)
			if err != nil {
				fmt.Println("ping", ip, err)
				return
			}
			for i, cu := range cus {
				fmt.Println(i, cu, status, oStatus)
				if cu.reply == nil || status != oStatus {
					if status != "✅" {
						cus[i].reply, _ = bot.SendMessage(tu.MessageWithEntities(tu.ID(cu.tm.Chat.ID),
							tu.Entity(status),
							tu.Entity("/"+ip).Code(),
						).WithReplyToMessageID(cu.tm.MessageID).WithReplyMarkup(inlineKeyboard))

					} else {
						cus[i].reply, _ = bot.SendMessage(tu.MessageWithEntities(tu.ID(cu.tm.Chat.ID),
							tu.Entity(status),
							tu.Entity("/"+ip).Code(),
						).WithReplyToMessageID(cu.tm.MessageID))
					}
				}
			}
			if status == "✅" {
				fmt.Println("worker stop", ip)
				return
			}
		}
	}
}

type AAA []string

func (a AAA) allowed(ChatID int64) bool {
	s := strconv.FormatInt(ChatID, 10)
	for _, v := range a {
		if v == s {
			return true
		}
	}
	fmt.Println(s, "not in", a)
	return false
}

var chats AAA
var done chan bool
var ips sCustomer
var bot *telego.Bot
var me *telego.User
var inlineKeyboard *telego.InlineKeyboardMarkup

func main() {
	chats = AAA{"-1001788229970", "-1001970078580", "1208474684", "-1001784778261"}
	inlineKeyboard = tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			// tu.InlineKeyboardButton("❗").WithCallbackData("❗"),
			// tu.InlineKeyboardButton("✅").WithCallbackData("✅"),
			// tu.InlineKeyboardButton("🔁").WithCallbackData("repeat"),
			// tu.InlineKeyboardButton("🔂").WithCallbackData("repeat_one"),
			// tu.InlineKeyboardButton("⏸️").WithCallbackData("pause"),
			tu.InlineKeyboardButton("❎").WithCallbackData("close"),
			tu.InlineKeyboardButton("❌").WithCallbackData("close all"),
		),
	)
	done = make(chan bool)
	ips = sCustomer{mcCustomer: mcCustomer{}, mCustomer: mCustomer{}}
	defer closer.Close()
	numFL := "(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])"
	reIP := regexp.MustCompile(numFL + "(\\.(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){2}\\." + numFL)
	deb := false
	publicURL := "https://localhost"
	addr := "localhost:443"
	publicURL, addr, err := ngrokUrlAddr()
	if err != nil {
		if NGROK_API_KEY := os.Getenv("NGROK_API_KEY"); NGROK_API_KEY != "" {
			publicURL, addr, _ = ngrokUrlTo(context.Background(), NGROK_API_KEY)
		}
	}
	// Note: Please keep in mind that default logger may expose sensitive information,
	// use in development only
	bot, err = telego.NewBot(os.Getenv("TOKEN"), telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Println(err)
		closer.Close()
	}
	me, _ = bot.GetMe()
	// bot.DeleteMyCommands(nil)
	/* 	closer.Bind(func() {
	   		fmt.Println("Press Enter")
	   		os.Stdin.Read([]byte{0})
	   	})
	*/
	endPoint := "/" + fmt.Sprint(time.Now().Format("2006010215040501"))
	// Set up a webhook on Telegram side
	_ = bot.SetWebhook(tu.Webhook(publicURL + endPoint)) //.WithSecretToken(endPoint)

	closer.Bind(func() {
		bot.DeleteWebhook(&telego.DeleteWebhookParams{
			DropPendingUpdates: true,
		})
		fmt.Println("closer bot.DeleteWebhook")
	})

	_ = manInTheMiddle(bot)

	// Get an update channel from webhook.
	// (more on configuration in examples/updates_webhook/main.go)
	updates, _ := bot.UpdatesViaWebhook(endPoint)

	// Start server for receiving requests from the Telegram
	go func() {
		_ = bot.StartWebhook(addr)
	}()

	// Stop server receiving requests from the Telegram
	closer.Bind(func() {
		_ = bot.StopWebhook()
		fmt.Println("closer bot.StopWebhook")
	})

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				fmt.Println("Ticker done")
				done <- true
				return
			case t := <-ticker.C:
				fmt.Println("Tick at", t)
				ips.update()
			}
		}
	}()

	closer.Bind(func() {
		done <- true
		fmt.Println("closer done <- true")
	})

	if deb {
		// Loop through all updates when they came
		for update := range updates {
			fmt.Printf("Update: %+v\n", update)
		}
	} else {
		// Create bot handler and specify from where to get updates
		bh, _ := th.NewBotHandler(bot, updates)

		// Stop handling updates
		closer.Bind(func() {
			bh.Stop()
			fmt.Println("closer bh.Stop")
		})
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tm := update.CallbackQuery.Message
			switch update.CallbackQuery.Data {
			case "close":
				bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID})
			case "close_all":
			}
		}, th.AnyCallbackQueryWithMessage())

		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tc, ctm := tmtc(update)
			if !chats.allowed(ctm.Chat.ID) {
				return
			}
			keys := reIP.FindAllString(tc, -1)
			fmt.Println("bh.Handle anyWithIP", keys, ctm)
			for _, ip := range keys {
				ips.write(ip, customer{tm: ctm})
			}
		}, anyWithIP(reIP))

		// Register new handler with match on any command
		// Handlers will match only once and in order of registration,
		// so this handler will be called on any command except `/start` command
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tm := update.Message
			bot.SendMessage(tu.MessageWithEntities(tu.ID(tm.Chat.ID),
				tu.Entity("Ожидался список IP адресов\n"),
				tu.Entity("/127.0.0.1 127.0.0.2 127.0.0.254").Code(),
				tu.Entity("🏓"), //:ping:
			).WithReplyToMessageID(tm.MessageID).WithReplyMarkup(inlineKeyboard))
		}, th.AnyCommand())

		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tm := update.Message
			bot.SendMessage(tu.MessageWithEntities(tu.ID(tm.Chat.ID),
				tu.Entity("Он улетел, но обещал вернуться❗\n    "),
				tu.Entity("Милый...").Bold(), tu.Entity("😍\n        "),
				tu.Entity("Милый...").Italic(), tu.Entity("😢"),
			).WithReplyToMessageID(tm.MessageID))
		}, leftChat())

		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tm := update.Message
			if !chats.allowed(tm.Chat.ID) {
				return
			}
			for _, nu := range tm.NewChatMembers {
				fmt.Println(nu.ID)
				bot.SendMessage(tu.MessageWithEntities(tu.ID(tm.Chat.ID),
					tu.Entity("Здорово, селяне!\n"),
					tu.Entity("Карета готова?\n").Strikethrough(),
					tu.Entity("Телега готова!🏓"),
					// tu.Entity("Начните личный чат\n").TextLink("https://t.me/rtk85bot"),
				).WithReplyToMessageID(tm.MessageID))
				return
			}

		}, newMember())

		// Start handling updates
		bh.Start()
	}
	fmt.Println("os.Exit(0)")
}

func tmtc(update telego.Update) (tc string, m *telego.Message) {
	for _, tm := range []*telego.Message{update.EditedMessage,
		update.EditedChannelPost,
		update.Message,
		update.ChannelPost} {
		//edit = i < 2
		if tm != nil {
			m = tm
			tc += tm.Text + " "
			tc += tm.Caption + " "
			tm = tm.ReplyToMessage
			if tm != nil {
				//m = tm
				tc += tm.Text + " "
				tc += tm.Caption + " "
			}
			break
		}
	}
	return
}
func anyWithIP(pattern *regexp.Regexp) th.Predicate {
	return func(update telego.Update) bool {
		tc, _ := tmtc(update)
		return pattern.MatchString(tc)
	}
}
func leftChat() th.Predicate {
	return func(update telego.Update) bool {
		return update.Message != nil && update.Message.LeftChatMember != nil
	}
}
func newMember() th.Predicate {
	return func(update telego.Update) bool {
		return update.Message != nil && update.Message.NewChatMembers != nil
	}
}
