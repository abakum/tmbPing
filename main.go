package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry/jibber_jabber"
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
	if len(s.mcCustomer) == 0 {
		defer ticker.Reset(dd)
		if s.save {
			saveDone <- true
			stdo.Println("del saveDone <- true")
		}
	}
}
func (s *sCustomer) add(ip string) (ch cCustomer) {
	stdo.Println("sCustomer.add ", ip)
	ch = make(cCustomer, 10)
	go worker(ip, ch)
	s.Lock()
	defer s.Unlock()
	if len(s.mcCustomer) == 0 {
		defer ticker.Reset(refresh)
	}
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

func (s *sCustomer) count() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.mcCustomer)
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

type mss map[string]string

func (m mss) add(key string, vals ...string) (val string) {
	var b0, a0, b, a string
	var ok bool
	for k, v := range vals {
		if k == 0 {
			b0, a0, ok = strings.Cut(v, ":")
			if !ok {
				b0 = "en"
				a0 = v
			}
			_, ok = m[b0+":"+a0]
			if !ok {
				m[b0+":"+a0] = a0
			}
		} else {
			b, a, ok = strings.Cut(v, ":")
			if !ok {
				b = "ru"
				a = v
			}
			_, ok = m[b+":"+a0]
			if !ok {
				m[b+":"+a0] = a
			}
		}
	}
	val, ok = m[key+":"+a0]
	if !ok {
		val = a0
	}
	return
}

var (
	chats       AAA
	done        chan bool
	ips         sCustomer
	bot         *telego.Bot
	refresh     time.Duration = time.Second * 60
	dd          time.Duration = time.Hour * 8
	stdo        *log.Logger
	save        cCustomer
	saveDone    chan bool
	tmbPingJson string = "tmbPing.json"
	ticker      *time.Ticker
	dic         mss
	reYYYYMMDD  *regexp.Regexp
	me          *telego.User
)

func main() {
	dic = mss{}
	stdo = log.New(os.Stdout, "", log.Lshortfile|log.Ltime)
	ul, err := jibber_jabber.DetectLanguage()
	if err != nil {
		ul = "ru"
	}
	chats = os.Args[1:]
	if len(chats) == 0 {
		stdo.Printf(dic.add(ul,
			"en:Usage: %s AllowedChatID1 AllowedChatID2 AllowedChatIDx\n",
			"ru:Использование: %s РазрешённыйChatID1 РазрешённыйChatID2 РазрешённыйChatIDх\n",
		), os.Args[0])
		os.Exit(1)
	} else {
		stdo.Println(dic.add(ul,
			"en:Allowed ChatID:",
			"ru:Разрешённые ChatID:",
		), chats)
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
	numFL := `(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])`
	reIP := regexp.MustCompile(numFL + `(\.(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){2}\.` + numFL)
	reYYYYMMDD = regexp.MustCompile(`(\p{L}*)\s([12][0-9][0-9][0-9]).?(0[1-9]|1[0-2]).?(0[1-9]|[12][0-9]|30|31)`)
	deb := false
	publicURL := "https://localhost"
	addr := "localhost:443"
	publicURL, addr, err = ngrokUrlAddr()
	if err != nil {
		if NGROK_API_KEY, ok := os.LookupEnv("NGROK_API_KEY"); !ok {
			publicURL, addr, _ = ngrokUrlTo(context.Background(), NGROK_API_KEY)
		}
	}
	token, ok := os.LookupEnv("TOKEN")
	if !ok {
		stdo.Println(dic.add(ul,
			"en:set TOKEN=BOT_TOKEN",
			"ru:Присвойте BOT_TOKEN переменной окружения TOKEN",
		))
		closer.Close()
	}
	// Note: Please keep in mind that default logger may expose sensitive information,
	// use in development only
	bot, err = telego.NewBot(token, telego.WithDefaultDebugLogger())
	if err != nil {
		stdo.Println(err)
		closer.Close()
	}
	me, err = bot.GetMe()
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

	go func() {
		ticker = time.NewTicker(dd)
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

	loader()

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
			tu.InlineKeyboardButton("❌").WithCallbackData("…❌"),
			tu.InlineKeyboardButton("⏸️").WithCallbackData("…⏸️❌"),
			tu.InlineKeyboardButton("✅").WithCallbackData("…✅❌"),
			tu.InlineKeyboardButton("⁉️").WithCallbackData("…⁉️❌"),
			tu.InlineKeyboardButton("❎").WithCallbackData("❎"),
		}
		var ikbsf int
		//AnyCallbackQueryWithMessage
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			uc := update.CallbackQuery
			tm := uc.Message
			my := uc.From.ID == tm.Chat.ID
			if tm.ReplyToMessage != nil {
				my = uc.From.ID == tm.ReplyToMessage.From.ID
			}
			ip := reIP.FindString(tm.Text)
			Data := update.CallbackQuery.Data
			if strings.HasPrefix(Data, "…") {
				ip = ""
			}
			ups := fmt.Sprintf("%s %s @%s #%d%s", uc.From.FirstName, uc.From.LastName, uc.From.Username, uc.From.ID, notAllowed(my, 0, tm.From.LanguageCode))
			bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID, Text: ups + tf(ips.count() == 0, "∅", ip+Data), ShowAlert: !my})
			if !my {
				return
			}
			if Data == "❎" {
				bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID})
				return
			}
			// ok := chats.allowed(uc.From.ID)
			// ikbsf = tf(ok, 0, len(ikbs)-1)
			if chats.allowed(uc.From.ID) && Data == "…" {
				rm := tu.InlineKeyboard(tm.ReplyMarkup.InlineKeyboard[0])
				if len(tm.ReplyMarkup.InlineKeyboard) == 1 {
					if ips.count() == 0 {
						return
					}
					rm = tu.InlineKeyboard(tm.ReplyMarkup.InlineKeyboard[0], tu.InlineKeyboardRow(ikbs[:len(ikbs)-1]...))
				}
				bot.EditMessageReplyMarkup(&telego.EditMessageReplyMarkupParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID, ReplyMarkup: rm})
				return
			}
			if ips.count() == 0 {
				return
			}
			if strings.HasPrefix(Data, "…") {
				ips.update(customer{Cmd: strings.TrimPrefix(Data, "…")})
			} else {
				ips.write(ip, customer{Cmd: Data})
			}
		}, th.AnyCallbackQueryWithMessage())
		//anyWithIP
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tc, ctm := tmtc(update)
			ok, ups := allowed(ctm.From.LanguageCode, ctm.From.ID, ctm.Chat.ID)
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
		}, anyWithMatch(reIP))
		//AnyCommand
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tm := update.Message
			if tm.Chat.Type == "private" {
				p := "/start "
				if strings.HasPrefix(tm.Text, p) {
					ds, err := base64.StdEncoding.DecodeString(strings.Trim(strings.TrimPrefix(tm.Text, p), " "))
					if err == nil {
						stdo.Println(string(ds))
						tm.Text = p + string(ds)
						easterEgg(bot, update)
						return
					}
				}
			}
			ok, ups := allowed(tm.From.LanguageCode, tm.From.ID, tm.Chat.ID)
			mecs := []tu.MessageEntityCollection{
				tu.Entity(dic.add(tm.From.LanguageCode,
					"en:List of IP addresses expected\n",
					"ru:Ожидался список IP адресов\n",
				)),
				tu.Entity("/127.0.0.1 127.0.0.2 127.0.0.254").Code(),
				tu.Entity(ups),
			}
			mecsf := len(mecs) - 1
			if ok {
				mecsf = 0
			}
			ikbsf = len(ikbs) - 1
			if chats.allowed(tm.From.ID) && ips.count() > 0 {
				ikbsf = 0
			}
			bot.SendMessage(tu.MessageWithEntities(tu.ID(tm.Chat.ID),
				mecs[mecsf:]...,
			).WithReplyToMessageID(tm.MessageID).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(ikbs[ikbsf:]...))))
		}, AnyCommand())
		//leftChat
		bh.Handle(func(bot *telego.Bot, update telego.Update) {
			tm := update.Message
			bot.SendMessage(tu.MessageWithEntities(tu.ID(tm.Chat.ID),
				tu.Entity(dic.add(tm.From.LanguageCode,
					"en:He flew away, but promised to return❗\n    ",
					"ru:Он улетел, но обещал вернуться❗\n    ",
				)),
				tu.Entity(dic.add(tm.From.LanguageCode,
					"en:Cute...",
					"ru:Милый...",
				)).Bold(), tu.Entity("😍\n        "),
				tu.Entity(dic.add(tm.From.LanguageCode,
					"en:Cute...",
				)).Italic(), tu.Entity("😢"),
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
					tu.Entity(dic.add(tm.From.LanguageCode,
						"en:Hello villagers!",
						"ru:Здорово, селяне!\n",
					)),
					tu.Entity(dic.add(tm.From.LanguageCode,
						"en:Is the carriage ready?\n",
						"ru:Карета готова?\n",
					)).Strikethrough(),
					tu.Entity(dic.add(tm.From.LanguageCode,
						"en:The cart is ready!🏓",
						"ru:Телега готова!🏓",
					)),
				).WithReplyToMessageID(tm.MessageID))
				return
			}

		}, newMember())
		//anyWithYYYYMMDD Easter Egg expected "name YYYY.?MM.?DD"
		bh.Handle(easterEgg, anyWithMatch(reYYYYMMDD))
		// Start handling updates
		bh.Start()
	}
	stdo.Println("os.Exit(0)")

}

func easterEgg(bot *telego.Bot, update telego.Update) {
	tc, ctm := tmtc(update)
	if ctm.Chat.Type != "private" {
		return
	}
	keys, _ := set(reYYYYMMDD.FindAllString(tc, -1))
	stdo.Println("bh.Handle anyWithYYYYMMDD", keys)
	for _, key := range keys {
		fss := reYYYYMMDD.FindStringSubmatch(key)
		bd, err := time.Parse("20060102", strings.Join(fss[2:], ""))
		if err == nil {
			nbd := fmt.Sprintf("%s %s", fss[1], bd.Format("2006-01-02"))
			tl := fmt.Sprintf("t.me/%s?start=%s", me.Username, base64.StdEncoding.EncodeToString([]byte(nbd)))
			entitys := []tu.MessageEntityCollection{tu.Entity("⚡").TextLink(tl)}
			entitys = append(entitys, tu.Entity(nbd).Code())
			entitys = append(entitys, tu.Entity("🔗"+"\n").TextLink("t.me/share/url?url="+tl))
			le := len(entitys) + 1
			for _, year := range la(bd) {
				entitys = append(entitys, tu.Entity(year+"\n"))
			}
			if len(entitys) > le {
				entitys[len(entitys)-1] = entitys[len(entitys)-1].Spoiler()
			}
			bot.SendMessage(tu.MessageWithEntities(tu.ID(ctm.Chat.ID), entitys...).WithReplyToMessageID(ctm.MessageID))
		}
	}
}

func allowed(key string, ChatIDs ...int64) (ok bool, s string) {
	s = "\n🏓"
	for _, v := range ChatIDs {
		ok = chats.allowed(v)
		if ok {
			return
		}
	}
	s = notAllowed(false, ChatIDs[0], key)
	return
}

func notAllowed(ok bool, ChatID int64, key string) (s string) {
	s = "\n🏓"
	if ok {
		return
	}
	s = dic.add(key,
		"en:\nNot allowed for you",
		"ru:\nБатюшка не благословляет Вас",
	)
	if ChatID != 0 {
		s += fmt.Sprintf(":%d", ChatID)
	}
	s += "\n🏓"
	return
}
