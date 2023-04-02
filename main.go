package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/xlab/closer"
)

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

func (s *sCustomer) update() {
	s.RLock()
	k, _ := m2kv(s.mcCustomer)
	s.RUnlock()
	for _, ip := range k {
		s.write(ip, customer{})
	}
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
				if cust.del {
					for _, cu := range cus {
						if cu.reply != nil {
							bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(cu.reply.Chat.ID), MessageID: cu.reply.MessageID})
						}
					}
					return
				}
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
	chats = os.Args[1:]
	fmt.Println(chats)
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
			case "close all":
				ips.write(reIP.FindString(tm.Text), customer{del: true})
				bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID})
			case "close":
				bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID})
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
