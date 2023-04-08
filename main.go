package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	Tm    *telego.Message `json:"tm,omitempty"`    //task
	Cmd   string          `json:"cmd,omitempty"`   //command
	Reply *telego.Message `json:"reply,omitempty"` //task reports
}
type cCustomer chan customer
type mcCustomer map[string]cCustomer
type sCustomer struct {
	sync.RWMutex
	mcCustomer
	save bool
}

func (s *sCustomer) close() {
	s.Lock()
	s.save = true
	s.Unlock()
	if len(s.mcCustomer) == 0 {
		saveDone <- true
		stdo.Println("sCustomer.close saveDone <- true")
	}
}

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
	if s.save && len(s.mcCustomer) == 0 {
		saveDone <- true
		stdo.Println("del saveDone <- true")
	}
}
func (s *sCustomer) add(ip string) (ch cCustomer) {
	stdo.Println("sCustomer.add ", ip)
	ch = make(cCustomer, 10)
	go worker(ip, ch)
	s.Lock()
	defer s.Unlock()
	s.mcCustomer[ip] = ch
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

var (
	chats       AAA
	done        chan bool
	ips         sCustomer
	bot         *telego.Bot
	refresh     time.Duration = time.Second * 60
	dd          time.Duration = time.Minute * 2
	stdo        *log.Logger
	save        cCustomer
	saveDone    chan bool
	tmbPingJson string = "tmbPing.json"
)

func main() {
	stdo = log.New(os.Stdout, "", log.Lshortfile|log.Ltime) //log.Ldate
	chats = os.Args[1:]
	if len(chats) == 0 {
		stdo.Printf("Usage: %s AllowedChatID1 AllowedChatID2 AllowedChatIDx\n", os.Args[0])
		os.Exit(1)
	} else {
		stdo.Println("Разрешённые ChatID:", chats)
	}
	// ex, err := os.Executable()
	ex, err := os.Getwd()
	if err == nil {
		// tmbPingJson = filepath.Join(filepath.Dir(ex), tmbPingJson)
		tmbPingJson = filepath.Join(ex, tmbPingJson)
	}
	stdo.Println(filepath.FromSlash(tmbPingJson))
	done = make(chan bool, 10)
	ips = sCustomer{mcCustomer: mcCustomer{}}
	defer closer.Close()
	numFL := "(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])"
	reIP := regexp.MustCompile(numFL + "(\\.(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){2}\\." + numFL)
	deb := false
	publicURL := "https://localhost"
	addr := "localhost:443"
	publicURL, addr, err = ngrokUrlAddr()
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

	go saver()
	closer.Bind(func() {
		ips.close()
		stdo.Println("closer ips.close")
	})
	loader()

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
		ikbs := []telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("🔁").WithCallbackData("…🔁"),
			tu.InlineKeyboardButton("🔂").WithCallbackData("…🔂"),
			tu.InlineKeyboardButton("⏸️").WithCallbackData("…⏸️"),
			tu.InlineKeyboardButton("✅❌").WithCallbackData("…✅❌"),
			tu.InlineKeyboardButton("⁉️❌").WithCallbackData("…⁉️❌"),
			tu.InlineKeyboardButton("❌").WithCallbackData("…❌"),
			tu.InlineKeyboardButton("❎").WithCallbackData("❎"),
		}
		var ikbsf int
		//AnyCallbackQueryWithMessage
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			uc := update.CallbackQuery
			tm := uc.Message
			ip := reIP.FindString(tm.Text)
			my := uc.From.ID == tm.Chat.ID
			if tm.ReplyToMessage != nil {
				my = uc.From.ID == tm.ReplyToMessage.From.ID
			}
			ups := fmt.Sprintf("%s %s @%s #%d%s", uc.From.FirstName, uc.From.LastName, uc.From.Username, uc.From.ID, notAllowed(my))
			ok := chats.allowed(uc.From.ID)
			ikbsf = tf(ok, 0, len(ikbs)-1)
			Data := update.CallbackQuery.Data
			if strings.HasPrefix(Data, "…") {
				bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID, Text: ups + Data, ShowAlert: !my})
				if !my {
					return
				}
				if ok && Data == "…" {
					rm := tu.InlineKeyboard(tm.ReplyMarkup.InlineKeyboard[0])
					if len(tm.ReplyMarkup.InlineKeyboard) == 1 {
						rm = tu.InlineKeyboard(tm.ReplyMarkup.InlineKeyboard[0], tu.InlineKeyboardRow(ikbs[ikbsf:len(ikbs)-1]...))
					}
					bot.EditMessageReplyMarkup(&telego.EditMessageReplyMarkupParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID, ReplyMarkup: rm})
					return
				}
				ips.update(customer{Cmd: strings.TrimPrefix(Data, "…")})
			} else {
				bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID, Text: ups + ip + Data, ShowAlert: !my})
				if !my {
					return
				}
				if Data != "❎" {
					ips.write(ip, customer{Cmd: Data})
				}
			}
			if !my {
				return
			}
			if Data == "❎" || strings.HasSuffix(Data, "❌") {
				bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID})
			}
		}, th.AnyCallbackQueryWithMessage())
		//anyWithIP
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tc, ctm := tmtc(update)
			ok, ups := allowed(ctm.From.ID, ctm.Chat.ID)
			keys, _ := set(reIP.FindAllString(tc, -1))
			stdo.Println("bh.Handle anyWithIP", keys, ctm)
			if ok {
				for _, ip := range keys {
					ips.write(ip, customer{Tm: ctm})
				}
			} else {
				ikbsf = len(ikbs) - 1
				bot.SendMessage(tu.MessageWithEntities(tu.ID(ctm.Chat.ID),
					tu.Entity("/"+strings.Join(keys, " ")).Code(),
					tu.Entity(ups),
				).WithReplyToMessageID(ctm.MessageID).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(ikbs[ikbsf:]...))))
				return
			}
		}, anyWithIP(reIP))

		//AnyCommand
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tm := update.Message
			ok, ups := allowed(tm.From.ID, tm.Chat.ID)
			mecs := []tu.MessageEntityCollection{
				tu.Entity("Ожидался список IP адресов\n"),
				tu.Entity("/127.0.0.1 127.0.0.2 127.0.0.254").Code(),
				tu.Entity(ups),
			}
			mecsf := len(mecs) - 1
			if ok {
				mecsf = 0
			}
			ikbsf = len(ikbs) - 1
			if chats.allowed(tm.From.ID) {
				ikbsf = 0
			}
			bot.SendMessage(tu.MessageWithEntities(tu.ID(tm.Chat.ID),
				mecs[mecsf:]...,
			).WithReplyToMessageID(tm.MessageID).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(ikbs[ikbsf:]...))))
		}, th.AnyCommand())
		//leftChat
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tm := update.Message
			bot.SendMessage(tu.MessageWithEntities(tu.ID(tm.Chat.ID),
				tu.Entity("Он улетел, но обещал вернуться❗\n    "),
				tu.Entity("Милый...").Bold(), tu.Entity("😍\n        "),
				tu.Entity("Милый...").Italic(), tu.Entity("😢"),
			).WithReplyToMessageID(tm.MessageID))
		}, leftChat())
		//newMember
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
func allowed(ChatIDs ...int64) (ok bool, s string) {
	s = notAllowed(false)
	if len(ChatIDs) == 0 {
		return
	}
	s = "\n🏓"
	for _, v := range ChatIDs {
		ok = chats.allowed(v)
		if ok {
			return
		}
	}
	s = fmt.Sprintf("\nБатюшка не благословляет Вас:%d\n🏓", ChatIDs[0])
	return
}

func notAllowed(ok bool) (s string) {
	s = "\n🏓"
	if ok {
		return
	}
	s = "\nБатюшка не благословляет Вас\n🏓"
	return
}
