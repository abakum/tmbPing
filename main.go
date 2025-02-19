package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/jibber_jabber"
	tg "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/xlab/closer"
)

func main() {
	var (
		err error
	)
	defer closer.Close()
	closer.Bind(func() {
		if err != nil {
			let.Println(err)
			SendError(bot, err)
			defer os.Exit(1)
		}
		PrintOk("stopH", stopH(bot, bh))
		DeleteWebhook(bot)
		ltf.Println("closer done <- true")
		done <- true
		ltf.Println("closer ips.close")
		ips.close()
		wg.Wait()
		// pressEnter()
	})
	ul, err = jibber_jabber.DetectLanguage()
	if err != nil {
		ul = "en"
	}
	if len(chats) == 0 {
		err = Errorf(dic.add(ul,
			"en:Usage: %s AllowedChatID1 AllowedChatID2 AllowedChatIDx\n",
			"ru:Использование: %s РазрешённыйChatID1 РазрешённыйChatID2 РазрешённыйChatIDх\n",
		), os.Args[0])
		return
	} else {
		li.Println(dic.add(ul,
			"en:Allowed ChatID:",
			"ru:Разрешённые ChatID:",
		), chats)
	}
	ex, err := os.Getwd()
	if err == nil {
		tmbPingJson = filepath.Join(ex, tmbPingJson)
	}
	li.Println(filepath.FromSlash(tmbPingJson))

	bot, err = tg.NewBot(os.Getenv("TOKEN"), tg.WithLogger(tg.Logger(Logger{}))) // tg.WithDefaultDebugLogger()
	// bot, err = tg.NewBot(os.Getenv("TOKEN"))

	if err != nil {
		if errors.Is(err, tg.ErrInvalidToken) {
			err = Errorf(dic.add(ul,
				"en:set TOKEN=BOT_TOKEN",
				"ru:Присвойте BOT_TOKEN переменной окружения TOKEN",
			))
		}
		err = srcError(err)
		return
	}

	me, err = bot.GetMe()
	if err != nil {
		err = srcError(err)
		return
	}

	// bot.DeleteMyCommands(nil)
	tacker = time.NewTicker(tt)
	defer tacker.Stop()
	bh, err := startH(bot)
	if err != nil {
		return
	}

	_ = manInTheMiddle(bot)

	wg.Add(1)
	go saver()

	wg.Add(1)
	// main loop
	go func() {
		defer wg.Done()
		ticker = time.NewTicker(dd)
		defer ticker.Stop()
		// tacker = time.NewTicker(tt)
		defer tacker.Stop()
		for {
			select {
			case <-done:
				ltf.Println("Ticker done")
				done <- true
				return
			case t := <-ticker.C:
				ltf.Println("Tick at", t)
				ips.update(customer{})
			case t := <-tacker.C:
				ltf.Println("Tack at", t)
				PrintOk("stopH", stopH(bot, bh))
				bh, err = startH(bot)
				if err != nil {
					letf.Println(err)
					return
				}
			}
		}
	}()

	err = loader()
	if err != nil {
		return
	}
	li.Println(ngrokAPI(os.Getenv("NGROK_API_KEY")))
	closer.Hold()
}

// drain buffered chan c
// func drain(c chan bool) {
// 	for len(c) > 0 {
// 		<-c
// 	}
// 	ltf.Println("drain chan done")
// }

// stop handler, webhook, polling
func stopH(bot *tg.Bot, bh *th.BotHandler) (err error) {
	quit(quitChannel)
	quit(quit2Channel)
	if bot != nil {
		if bot.IsRunningWebhook() {
			ltf.Println("StopWebhook")
			err = srcError(bot.StopWebhook())
		} else if bot.IsRunningLongPolling() {
			ltf.Println("StopLongPolling")
			bot.StopLongPolling()
			// drain(getUpdates)
			// go func() {
			// 	time.Sleep(time.Second * 8)
			// 	getUpdates <- false
			// }()
			// ltf.Println("getUpdates", <-getUpdates)
		}
	}
	if bh != nil {
		ltf.Println("bh.Stop")
		bh.Stop()
	}
	return
}

func DeleteWebhook(bot *tg.Bot) {
	if bot != nil {
		PrintOk("DeleteWebhook", bot.DeleteWebhook(&tg.DeleteWebhookParams{
			DropPendingUpdates: false,
		}))
	}
}

// start handler and webhook or polling
func startH(bot *tg.Bot) (*th.BotHandler, error) {
	updates, err := webHook(bot)
	if err != nil {
		PrintOk("webHook", err)
		DeleteWebhook(bot)
		if tt != ttm {
			tt = ttm
			tacker.Reset(ttm) // next try after ttm
		}
		// updates, err = bot.UpdatesViaLongPolling(nil)
		updates, err = bot.UpdatesViaLongPolling(&tg.GetUpdatesParams{Timeout: int(refresh.Seconds())})
		if err != nil {
			return nil, err
		}
		// SendError(bot, fmt.Errorf("updatesViaLongPolling"))
		letf.Println("updatesViaLongPolling")
	} else {
		// SendError(bot, fmt.Errorf("updatesViaWebHook"))
		ltf.Println("updatesViaWebHook")
	}

	bh, err := th.NewBotHandler(bot, updates, th.WithStopTimeout(time.Second*8))
	if err != nil {
		return nil, srcError(err)
	}

	//AnyCallbackQueryWithMessage
	bh.Handle(bhAnyCallbackQueryWithMessage, th.AnyCallbackQueryWithMessage())
	//delete reply message with - or / in text
	bh.Handle(bhReplyMessageIsMinus, ReplyMessageIsMinus())
	//anyWithIP
	bh.Handle(bhAnyWithMatch, anyWithMatch(reIP))
	//AnyCommand
	bh.Handle(bhAnyCommand, AnyCommand())
	//leftChat
	bh.Handle(bhLeftChat, leftChat())
	//newMember
	bh.Handle(bhNewMember, newMember())
	//anyWithYYYYMMDD Easter Egg expected "name YYYY.?MM.?DD"
	bh.Handle(bhEasterEgg, anyWithMatch(reYYYYMMDD))

	go bh.Start()

	return bh, nil
}

// handler IP
func bhAnyWithMatch(bot *tg.Bot, update tg.Update) {
	tc, ctm := tmtc(update)
	if ctm == nil {
		return
	}
	ok, ups := allowed(ul, ctm.From.ID, ctm.Chat.ID)
	keys, _ := set(reIP.FindAllString(tc, -1))
	ltf.Println("bh.Handle anyWithIP", keys, ctm)
	if ok {
		for _, ip := range keys {
			ips.write(ip, customer{Tm: ctm})
		}
	} else {
		ikbsf = len(ikbs) - 1
		news := ""
		for _, ip := range keys {
			if ips.read(ip) {
				ips.write(ip, customer{Tm: ctm})
			} else {
				news += ip + " "
			}
		}
		if len(news) > 1 {
			bot.SendMessage(tu.MessageWithEntities(tu.ID(ctm.Chat.ID),
				tu.Entity("/"+strings.TrimRight(news, " ")).Code(),
				tu.Entity(ups),
			).WithReplyToMessageID(ctm.MessageID).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(ikbs[ikbsf:]...))))
		}
		return
	}
}

// handler EasterEgg
func bhEasterEgg(bot *tg.Bot, update tg.Update) {
	tc, ctm := tmtc(update)
	if ctm == nil {
		return
	}
	if ctm.Chat.Type != "private" {
		return
	}
	keys, _ := set(reYYYYMMDD.FindAllString(tc, -1))
	ltf.Println("bh.Handle anyWithYYYYMMDD", keys)
	for _, key := range keys {
		fss := reYYYYMMDD.FindStringSubmatch(key)
		bd, err := time.ParseInLocation("20060102150405", strings.Join(fss[2:], "")+"120000", time.Local)
		if err == nil {
			nbd := fmt.Sprintf("%s %s", fss[1], bd.Format("2006-01-02"))
			tl := start(me, nbd)
			entitys := []tu.MessageEntityCollection{tu.Entity("⚡").TextLink(tl)}
			entitys = append(entitys, tu.Entity(nbd).Code())
			entitys = append(entitys, tu.Entity("🔗"+"\n").TextLink("t.me/share/url?url="+tl))
			le := len(entitys) + 1
			for _, year := range la(bd) {
				// b, a, ok := strings.Cut(year, " ")
				// entitys = append(entitys, tu.Entity(b).Hashtag())
				// if ok {
				// 	entitys = append(entitys, tu.Entityf(" n%s\n", a))
				// }
				entitys = append(entitys, tu.Entity(year+"\n"))
			}
			if len(entitys) > le {
				entitys[len(entitys)-1] = entitys[len(entitys)-1].Spoiler()
			}
			bot.SendMessage(tu.MessageWithEntities(tu.ID(ctm.Chat.ID), entitys...).WithReplyToMessageID(ctm.MessageID))
		}
	}
}

// handler Callback
func bhAnyCallbackQueryWithMessage(bot *tg.Bot, update tg.Update) {
	uc := update.CallbackQuery
	if uc == nil {
		return
	}
	tm := uc.Message
	if tm == nil {
		return
	}
	my := true
	if tm.Chat.Type != "private" && tm.ReplyToMessage != nil {
		my = uc.From.ID == tm.ReplyToMessage.From.ID
	}
	ip := reIP.FindString(tm.Text)
	Data := update.CallbackQuery.Data
	if strings.HasPrefix(Data, "…") {
		ip = ""
	}
	ups := fmt.Sprintf("%s %s @%s #%d%s", uc.From.FirstName, uc.From.LastName, uc.From.Username, uc.From.ID, notAllowed(my, 0, ul)) //tm.From.LanguageCode
	bot.AnswerCallbackQuery(&tg.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID, Text: ups + tf(ips.count() == 0, "∅", ip+Data), ShowAlert: !my})
	if !my {
		return
	}
	if Data == "❎" {
		// bot.DeleteMessage(&tg.DeleteMessageParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID})
		bot.DeleteMessage(tu.Delete(tu.ID(tm.Chat.ID), tm.MessageID))
		return
	}
	if Data == "…" { //chats.allowed(uc.From.ID) &&
		rm := tu.InlineKeyboard(tm.ReplyMarkup.InlineKeyboard[0])
		if len(tm.ReplyMarkup.InlineKeyboard) == 1 {
			if ips.count() == 0 {
				return
			}
			rm = tu.InlineKeyboard(tm.ReplyMarkup.InlineKeyboard[0], tu.InlineKeyboardRow(ikbs[:len(ikbs)-1]...))
		}
		bot.EditMessageReplyMarkup(&tg.EditMessageReplyMarkupParams{ChatID: tu.ID(tm.Chat.ID), MessageID: tm.MessageID, ReplyMarkup: rm})
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
}

// handler DeleteMessage
func bhReplyMessageIsMinus(bot *tg.Bot, update tg.Update) {
	re := update.Message.ReplyToMessage
	err := bot.DeleteMessage(tu.Delete(tu.ID(re.Chat.ID), re.MessageID))
	if err != nil {
		let.Println(err)
		bot.EditMessageText(&tg.EditMessageTextParams{ChatID: tu.ID(re.Chat.ID), MessageID: re.MessageID, Text: "-"})
	}
}

// send t.C then reset t
func restart(t *time.Ticker, d time.Duration) {
	if t != nil {
		t.Reset(time.Millisecond * 100)
		time.Sleep(time.Millisecond * 150)
		t.Reset(d)
	}
}

// handler Command
func bhAnyCommand(bot *tg.Bot, update tg.Update) {
	tm := update.Message
	if tm == nil {
		return
	}
	if tm.Chat.Type == "private" {
		p := "/start "
		if strings.HasPrefix(tm.Text, p) {
			ds, err := base64.StdEncoding.DecodeString(strings.Trim(strings.TrimPrefix(tm.Text, p), " "))
			if err == nil {
				ltf.Println(string(ds))
				tm.Text = p + string(ds)
				switch {
				case reYYYYMMDD.MatchString(tm.Text):
					bhEasterEgg(bot, update)
				case reIP.MatchString(tm.Text):
					bhAnyWithMatch(bot, update)
				}
				return
			}
		}
		// For owner as first chatID in args
		if tm.From != nil && chats[:1].allowed(tm.From.ID) {
			p = "/restart"
			if strings.HasPrefix(tm.Text, p) {
				restart(tacker, tt)
				return
			}
			p = "/stop"
			if strings.HasPrefix(tm.Text, p) {
				closer.Close()
				return
			}
		}
	}
	ok, ups := allowed(ul, tm.From.ID, tm.Chat.ID)
	mecs := []tu.MessageEntityCollection{
		tu.Entity(dic.add(ul,
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
}

// handler LeftChat
func bhLeftChat(bot *tg.Bot, update tg.Update) {
	tm := update.Message
	bot.SendMessage(tu.MessageWithEntities(tu.ID(tm.Chat.ID),
		tu.Entity(dic.add(ul,
			"en:He flew away, but promised to return❗\n    ",
			"ru:Он улетел, но обещал вернуться❗\n    ",
		)),
		tu.Entity(dic.add(ul,
			"en:Cute...",
			"ru:Милый...",
		)).Bold(), tu.Entity("😍\n        "),
		tu.Entity(dic.add(ul,
			"en:Cute...",
		)).Italic(), tu.Entity("😢"),
	).WithReplyToMessageID(tm.MessageID))

}

// handler NewMember
func bhNewMember(bot *tg.Bot, update tg.Update) {
	tm := update.Message
	if !chats.allowed(tm.Chat.ID) {
		return
	}
	for _, nu := range tm.NewChatMembers {
		ltf.Println(nu.ID)
		bot.SendMessage(tu.MessageWithEntities(tu.ID(tm.Chat.ID),
			tu.Entity(dic.add(ul,
				"en:Hello villagers!",
				"ru:Здорово, селяне!\n",
			)),
			tu.Entity(dic.add(ul,
				"en:Is the carriage ready?\n",
				"ru:Карета готова?\n",
			)).Strikethrough(),
			tu.Entity(dic.add(ul,
				"en:The cart is ready!🏓",
				"ru:Телега готова!🏓",
			))).WithReplyToMessageID(tm.MessageID))
		break
	}
}

// is key in args
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

// message for ChatID
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

// tm info
func fcRfRc(tm *tg.Message) (s string) {
	s = ""
	if tm == nil {
		return
	}
	s = fmt.Sprintf("From:@%s #%d Chat:@%s #%d", tm.From.Username, tm.From.ID, tm.Chat.Title, tm.Chat.ID)
	if tm.ReplyToMessage == nil {
		return
	}
	s = fmt.Sprintf(" Reply From:@%s #%d Reply Chat:@%s #%d", tm.ReplyToMessage.From.Username, tm.ReplyToMessage.From.ID, tm.ReplyToMessage.Chat.Title, tm.ReplyToMessage.Chat.ID)
	return
}

// encode for /start
func start(me *tg.User, s string) string {
	return fmt.Sprintf("t.me/%s?start=%s", me.Username, base64.StdEncoding.EncodeToString([]byte(s)))
}
