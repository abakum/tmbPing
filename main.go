package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/xlab/closer"
)

type customer struct {
	tm    *telego.Message //task
	cmd   string          //command
	reply *telego.Message //task reports
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
	stdo.Println("sCustomer.del ", ip)
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
	stdo.Println("sCustomer.add ", ip)
	ch = make(cCustomer, 10)
	go worker(ip, ch)
	s.Lock()
	defer s.Unlock()
	s.mcCustomer[ip] = ch
	//s.mCustomer[ip] = mCustomer{}
	return
}

func (s *sCustomer) write(ip string, c customer) {
	stdo.Println("sCustomer.write ", ip, c)
	defer func() {
		// recover from panic caused by writing to a closed channel
		if err := recover(); err != nil {
			stdo.Println("sCustomer.write error:", err)
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

func (s *sCustomer) update(c customer) {
	s.RLock()
	k, _ := m2kv(s.mcCustomer)
	s.RUnlock()
	for _, ip := range k {
		s.write(ip, c)
	}
}

type customers []customer

func worker(ip string, ch cCustomer) {
	log.SetPrefix("worker ")
	var buttons *telego.InlineKeyboardMarkup
	var err error
	status := "?"
	dd := time.Duration(time.Minute * 2)
	deadline := time.Now().Add(dd)
	cus := customers{}
	defer ips.del(ip, false)
	for {
		select {
		case <-done:
			stdo.Println("worker done", ip)
			done <- true //done other worker
			return
		case cust, ok := <-ch:
			if !ok {
				stdo.Println("worker channel closed", ip)
				return
			}
			oStatus := status
			if cust.tm == nil { //update
				switch cust.cmd { //
				case "⏸️":
					deadline = time.Now().Add(-refresh)
					oStatus = cust.cmd
				case "🔁":
					deadline = time.Now().Add(dd)
					oStatus = cust.cmd
				case "🔂":
					deadline = time.Now().Add(refresh)
					oStatus = cust.cmd
				default: // status+"❌" status+"⏸️❌"
					stdo.Println("----------", cust.cmd, status, strings.TrimSuffix(cust.cmd, "❌") == strings.TrimSuffix(status, "⏸️"))
					if cust.cmd == "❌" || strings.TrimSuffix(cust.cmd, "❌") == strings.TrimSuffix(status, "⏸️") {
						for _, cu := range cus {
							if cu.reply != nil {
								bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(cu.reply.Chat.ID), MessageID: cu.reply.MessageID})
							}
						}
						return
					}
				}
			} else {
				cus = append(cus, cust)
			}
			stdo.Println("worker", ip, cust, len(ch), status, time.Now().Before(deadline))
			if time.Now().Before(deadline) {
				status, err = ping(ip)
				if err != nil {
					stdo.Println("ping", ip, err)
					return
				}
				buttons = tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("⏸️").WithCallbackData("⏸️"),
						tu.InlineKeyboardButton("❌").WithCallbackData("❌"),
						tu.InlineKeyboardButton("❎").WithCallbackData("❎"),
					),
				)
			} else {
				buttons = tu.InlineKeyboard(
					tu.InlineKeyboardRow(
						tu.InlineKeyboardButton("🔁").WithCallbackData("🔁"),
						tu.InlineKeyboardButton("🔂").WithCallbackData("🔂"),
						tu.InlineKeyboardButton("❌").WithCallbackData("❌"),
						tu.InlineKeyboardButton("❎").WithCallbackData("❎"),
					),
				)
				if !strings.HasSuffix(status, "⏸️") {
					status += "⏸️"
				}
			}
			for i, cu := range cus {
				stdo.Println(i, cu, status, oStatus)
				if cu.reply == nil || status != oStatus {
					if cu.reply != nil {
						bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(cu.reply.Chat.ID), MessageID: cu.reply.MessageID})
					}
					cus[i].reply, _ = bot.SendMessage(tu.MessageWithEntities(tu.ID(cu.tm.Chat.ID),
						tu.Entity(status),
						tu.Entity("/"+ip).Code(),
					).WithReplyToMessageID(cu.tm.MessageID).WithReplyMarkup(buttons))

				}
			}
			/* if status == "✅" {
				stdo.Println("worker stop", ip)
				deadline = time.Now()
			} */
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
	stdo.Println(s, "not in", a)
	return false
}

var chats AAA
var done chan bool
var ips sCustomer
var bot *telego.Bot
var refresh time.Duration = time.Second * 60
var stdo *log.Logger

func main() {
	stdo = log.New(os.Stdout, "main ", log.Lshortfile|log.Ltime) //log.Ldate
	chats = os.Args[1:]
	if len(chats) == 0 {
		stdo.Printf("Usage: %s AllowedChatID1 AllowedChatID2 AllowedChatIDx\n", os.Args[0])
		os.Exit(1)
	} else {
		stdo.Println("Разрешённые ChatID:", chats)
	}
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
		stdo.Println(err)
		closer.Close()
	}
	// bot.DeleteMyCommands(nil)

	/* 	closer.Bind(func() {
	   		stdo.Println("Press Enter")
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
		stdo.Println("closer bot.DeleteWebhook")
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
		stdo.Println("closer bot.StopWebhook")
	})

	go func() {
		ticker := time.NewTicker(refresh)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				stdo.Println("Ticker done")
				done <- true
				return
			case t := <-ticker.C:
				stdo.Println("Tick at", t)
				ips.update(customer{})
			}
		}
	}()

	closer.Bind(func() {
		done <- true
		stdo.Println("closer done <- true")
	})

	if deb {
		// Loop through all updates when they came
		for update := range updates {
			stdo.Printf("Update: %+v\n", update)
		}
	} else {
		// Create bot handler and specify from where to get updates
		bh, _ := th.NewBotHandler(bot, updates)

		// Stop handling updates
		closer.Bind(func() {
			bh.Stop()
			stdo.Println("closer bh.Stop")
		})
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tm := update.CallbackQuery.Message
			Data := update.CallbackQuery.Data
			if strings.HasPrefix(Data, "…") {
				bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID, Text: "🏓" + Data})
				ips.update(customer{cmd: strings.TrimPrefix(Data, "…")})
			} else {
				if Data != "❎" {
					ip := reIP.FindString(tm.Text)
					bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID, Text: "🏓" + ip + Data})
					ips.write(ip, customer{cmd: Data})
				}
			}
			if Data == "❎" || strings.HasSuffix(Data, "❌") {
				bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID})
			}
		}, th.AnyCallbackQueryWithMessage())

		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tc, ctm := tmtc(update)
			if !chats.allowed(ctm.Chat.ID) {
				return
			}
			keys := reIP.FindAllString(tc, -1)
			stdo.Println("bh.Handle anyWithIP", keys, ctm)
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
				tu.Entity("🏓"),
			).WithReplyToMessageID(tm.MessageID).WithReplyMarkup(tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("🔁").WithCallbackData("…🔁"),
					tu.InlineKeyboardButton("🔂").WithCallbackData("…🔂"),
					tu.InlineKeyboardButton("⏸️").WithCallbackData("…⏸️"),
					tu.InlineKeyboardButton("✅❌").WithCallbackData("…✅❌"),
					tu.InlineKeyboardButton("⁉️❌").WithCallbackData("…⁉️❌"),
					tu.InlineKeyboardButton("❌").WithCallbackData("…❌"),
					tu.InlineKeyboardButton("❎").WithCallbackData("❎"),
				),
			)))
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
				stdo.Println(nu.ID)
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
	stdo.Println("os.Exit(0)")
}
