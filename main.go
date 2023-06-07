package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
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
	ul, err = jibber_jabber.DetectLanguage()
	if err != nil {
		ul = "en"
	}
	chats = os.Args[1:]
	if len(chats) == 0 {
		err = fmt.Errorf(dic.add(ul,
			"en:Usage: %s AllowedChatID1 AllowedChatID2 AllowedChatIDx\n",
			"ru:Использование: %s РазрешённыйChatID1 РазрешённыйChatID2 РазрешённыйChatIDх\n",
		), os.Args[0])
		return
	} else {
		stdo.Println(dic.add(ul,
			"en:Allowed ChatID:",
			"ru:Разрешённые ChatID:",
		), chats)
	}
	ex, err := os.Getwd()
	if err == nil {
		tmbPingJson = filepath.Join(ex, tmbPingJson)
	}
	stdo.Println(filepath.FromSlash(tmbPingJson))

	token, ok := os.LookupEnv("TOKEN")
	if !ok {
		err = fmt.Errorf(dic.add(ul,
			"en:set TOKEN=BOT_TOKEN",
			"ru:Присвойте BOT_TOKEN переменной окружения TOKEN",
		))
		return
	}
	bot, err = tg.NewBot(token, tg.WithDefaultDebugLogger())
	if err != nil {
		return
	}
	me, err = bot.GetMe()
	if err != nil {
		return
	}
	// bot.DeleteMyCommands(nil)
	bh, err := startH(bot)
	if err != nil {
		return
	}

	_ = manInTheMiddle(bot)

	wg.Add(1)
	go saver()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker = time.NewTicker(dd)
		defer ticker.Stop()
		tacker = time.NewTicker(dd)
		defer tacker.Stop()
		for {
			select {
			case <-done:
				stdo.Println("Ticker done")
				done <- true
				return
			case t := <-ticker.C:
				stdo.Println("Tick at", t)
				ips.update(customer{})
			case t := <-tacker.C:
				stdo.Println("Tack at", t)
				stopH(bot, bh)
				bh, err = startH(bot)
				if err != nil {
					return
				}
			}
		}
	}()

	loader()
	closer.Bind(func() {
		if err != nil {
			stdo.Println("Error", err)
		}
		stdo.Println("stopH", stopH(bot, bh))
		stdo.Println("closer done <- true")
		done <- true
		stdo.Println("closer ips.close")
		ips.close()
		wg.Wait()
	})
	closer.Hold()
}
func stopH(bot *tg.Bot, bh *th.BotHandler) (err error) {
	if bh != nil {
		bh.Stop()
		stdo.Println("bh.Stop")
	}
	if bot != nil {
		err = bot.DeleteWebhook(&tg.DeleteWebhookParams{
			DropPendingUpdates: false,
		})
		stdo.Println("DeleteWebhook", err)

		if bot.IsRunningWebhook() {
			stdo.Println("IsRunningWebhook")
			err = bot.StopWebhook()
			stdo.Println("StopWebhook", err)
		}
	}
	return
}

func startH(bot *tg.Bot) (*th.BotHandler, error) {
	var (
		endPoint = "/" + fmt.Sprint(time.Now().Format("2006010215040501"))
		secret   = endPoint[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(endPoint)-3)+1:]
		updates  <-chan tg.Update
	)
	//try use ngrok.exe client with web interface for debuging
	publicURL, forwardsTo, err := ngrokWeb()
	if err != nil {
		//for ngrok.exe client without web interface at os.Getenv("web_addr")
		NGROK_API_KEY := os.Getenv("NGROK_API_KEY")
		if NGROK_API_KEY != "" {
			ctx, ca := context.WithTimeout(context.Background(), time.Second*3)
			defer ca()
			publicURL, forwardsTo, err = ngrokAPI(ctx, NGROK_API_KEY)
		}
		if err != nil {
			//use ngrok-go client
			forwardsTo = os.Getenv("forwardsTo")
			if forwardsTo == "" {
				forwardsTo = "https://localhost"
			}
			if os.Getenv("NGROK_AUTHTOKEN") != "" {
				updates, err = UpdatesWithNgrok(bot, secret, forwardsTo, endPoint)
			} else {
				updates, err = UpdatesWithSecret(bot, secret, forwardsTo, endPoint)
			}
		} else {
			updates, err = UpdatesWithSecret(bot, secret, publicURL, endPoint) //publicURL from ngrokAPI
		}
	} else {
		updates, err = UpdatesWithSecret(bot, secret, publicURL, endPoint) //publicURL from ngrokWeb
	}
	if err != nil {
		return nil, err
	}

	go func() {
		err = bot.StartWebhook(webHookAddress(forwardsTo))
		if err != nil {
			stdo.Println("After StartWebhook", err)
			closer.Close()
		}
	}()

	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		return nil, err
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

func webHookAddress(forwardsTo string) (hp string) {
	hp = strings.TrimPrefix(forwardsTo, ":")
	_, err := strconv.Atoi(hp)
	if err == nil {
		hp = "localhost:" + hp
	}
	for k, v := range map[string]string{
		"http://":  ":80",
		"https://": ":443",
		"":         ":80",
	} {
		if strings.HasPrefix(hp, k) {
			hp = strings.TrimPrefix(hp, k)
			if !strings.Contains(hp, ":") {
				hp += v
			}
			break
		}
	}
	stdo.Println(hp)
	return
}

func bhAnyWithMatch(bot *tg.Bot, update tg.Update) {
	tc, ctm := tmtc(update)
	if ctm == nil {
		return
	}
	ok, ups := allowed(ul, ctm.From.ID, ctm.Chat.ID)
	keys, _ := set(reIP.FindAllString(tc, -1))
	stdo.Println("bh.Handle anyWithIP", keys, ctm)
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

func bhEasterEgg(bot *tg.Bot, update tg.Update) {
	tc, ctm := tmtc(update)
	if ctm == nil {
		return
	}
	if ctm.Chat.Type != "private" {
		return
	}
	keys, _ := set(reYYYYMMDD.FindAllString(tc, -1))
	stdo.Println("bh.Handle anyWithYYYYMMDD", keys)
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
		bot.DeleteMessage(Delete(tu.ID(tm.Chat.ID), tm.MessageID))
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
func bhReplyMessageIsMinus(bot *tg.Bot, update tg.Update) {
	re := update.Message.ReplyToMessage
	err := bot.DeleteMessage(Delete(tu.ID(re.Chat.ID), re.MessageID))
	if err != nil {
		bot.EditMessageText(&tg.EditMessageTextParams{ChatID: tu.ID(re.Chat.ID), MessageID: re.MessageID, Text: "-"})
	}
}

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
				stdo.Println(string(ds))
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
func bhNewMember(bot *tg.Bot, update tg.Update) {
	tm := update.Message
	if !chats.allowed(tm.Chat.ID) {
		return
	}
	for _, nu := range tm.NewChatMembers {
		stdo.Println(nu.ID)
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
func start(me *tg.User, s string) string {
	return fmt.Sprintf("t.me/%s?start=%s", me.Username, base64.StdEncoding.EncodeToString([]byte(s)))
}
