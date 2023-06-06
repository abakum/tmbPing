package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/jibber_jabber"
	"github.com/fasthttp/router"
	tg "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/valyala/fasthttp"
	"github.com/xlab/closer"
	"golang.ngrok.com/ngrok"
	nc "golang.ngrok.com/ngrok/config"
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
	bh, nt, err := startH(bot)
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
				err = stopH(bot, bh, nt)
				if err != nil {
					return
				}
				bh, nt, err = startH(bot)
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
		err = stopH(bot, bh, nt)
		stdo.Println("stopH", err)
		stdo.Println("closer done <- true")
		done <- true
		stdo.Println("closer ips.close")
		ips.close()
		wg.Wait()
	})
	closer.Hold()
}
func stopH(bot *tg.Bot, bh *th.BotHandler, nt *ngrok.Tunnel) (err error) {
	if bh != nil {
		bh.Stop()
		stdo.Println("bh.Stop")
	}
	if bot != nil {
		if bot.IsRunningWebhook() {
			err = bot.StopWebhook()
			stdo.Println("StopWebhook", err)
		}
		err = bot.DeleteWebhook(&tg.DeleteWebhookParams{
			DropPendingUpdates: false,
		})
		stdo.Println("DeleteWebhook", err)
	}
	if nt != nil {
		err = (*nt).Session().Close()
		stdo.Println("Session().Close", err)
	}
	return
}

// UpdatesWithSecret set SecretToken to FastHTTPWebhookServer and SetWebhookParams
func UpdatesWithSecret(bot *tg.Bot, endPoint, SecretToken, publicURL string) (updates <-chan tg.Update, whs tg.FastHTTPWebhookServer, err error) {
	whs = tg.FastHTTPWebhookServer{
		Server:      &fasthttp.Server{},
		Router:      router.New(),
		SecretToken: SecretToken,
	}
	updates, err = bot.UpdatesViaWebhook(endPoint, tg.WithWebhookServer(whs), tg.WithWebhookSet(tu.Webhook(publicURL+endPoint).WithSecretToken(SecretToken)))
	return
}

// UpdatesWithNgrok start ngrok.Tunnel with os.Getenv("NGROK_AUTHTOKEN") and SecretToken
// for close ngrok.Tunnel use (*nt).Session().Close()
func UpdatesWithNgrok(bot *tg.Bot, endPoint, SecretToken, forwardsTo string) (updates <-chan tg.Update, nt *ngrok.Tunnel, err error) {
	tun, err := ngrok.Listen(context.Background(), nc.HTTPEndpoint(
		nc.WithForwardsTo(forwardsTo)),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		return
	}
	nt = &tun
	publicURL := tun.URL()
	defer func() {
		if err != nil {
			tun.Session().Close()
			nt = nil
		}
	}()
	updates, whs, err := UpdatesWithSecret(bot, endPoint, SecretToken, publicURL)
	if err != nil {
		return
	}
	stdo.Println(publicURL+endPoint, forwardsTo)
	go func() {
		for {
			conn, err := tun.Accept()
			if err != nil {
				stdo.Printf("error accept connection %v", err)
				return
			}
			stdo.Println(conn.RemoteAddr().String(), "=>", conn.LocalAddr().String())
			go func() {
				err := whs.Server.ServeConn(conn)
				if err != nil {
					stdo.Printf("error serving connection %v: %v", conn, err)
				}
			}()
		}
	}()
	return
}

func startH(bot *tg.Bot) (*th.BotHandler, *ngrok.Tunnel, error) {
	var (
		endPoint = "/" + fmt.Sprint(time.Now().Format("2006010215040501"))
		secret   = endPoint[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(endPoint)-3):]
		err      error
		// tun      ngrok.Tunnel
		updates <-chan tg.Update
		nt      *ngrok.Tunnel
	)
	//try use ngrok.exe client for debuging
	publicURL, forwardsTo, err := ngrokUrlAddr()
	if err != nil {
		//use ngrok-go client
		forwardsTo = localhost
		updates, nt, err = UpdatesWithNgrok(bot, endPoint, secret, forwardsTo)
	} else {
		//for ngrok.exe client without web interface
		if NGROK_API_KEY, _ := os.LookupEnv("NGROK_API_KEY"); NGROK_API_KEY != "" {
			publicURL, forwardsTo, _ = ngrokUrlTo(context.Background(), NGROK_API_KEY)
			stdo.Println(publicURL+endPoint, forwardsTo)
		}
		updates, _, err = UpdatesWithSecret(bot, endPoint, secret, publicURL)
	}
	if err != nil {
		return nil, nil, err
	}

	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		return nil, nil, err
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

	go func() {
		err = bot.StartWebhook(webHookAddress(forwardsTo))
		if err != nil {
			stdo.Println("StartWebhook", err)
			closer.Close()
		}
	}()
	go bh.Start()
	return bh, nt, nil
}

func webHookAddress(forwardsTo string) (hp string) {
	if strings.HasPrefix(forwardsTo, "https://") {
		hp = strings.TrimPrefix(forwardsTo, "https://")
		if !strings.Contains(hp, ":") {
			hp += ":443"
		}
		return
	}
	hp = strings.TrimPrefix(forwardsTo, "http://")
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
