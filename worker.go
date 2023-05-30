package main

import (
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func worker(ip string, ch cCustomer) {
	// var buttons *telego.InlineKeyboardMarkup
	var err error
	status := ""
	tl := start(me, ip)
	deadline := time.Now().Add(dd)
	cus := customers{}
	defer ips.del(ip, false)
	for {
		select {
		case <-done:
			done <- true //done other worker
			for i, cu := range cus {
				cu.Tm = &telego.Message{MessageID: cu.Tm.MessageID, From: &telego.User{ID: cu.Tm.From.ID}, Chat: telego.Chat{ID: cu.Tm.Chat.ID}}
				cu.Cmd = ip
				if i == 0 {
					cu.Tm.From.FirstName = status
					cu.Tm.Date = deadline.Unix()
					stdo.Println("saved ", ip, status, deadline)
				}
				save <- cu
			}
			stdo.Println("done", ip)
			return
		case cust, ok := <-ch:
			if !ok {
				stdo.Println("channel closed", ip)
				return
			}
			if cust.Tm == nil { //update
				switch cust.Cmd {
				case "⏸️":
					deadline = time.Now().Add(-refresh)
				case "🔁":
					deadline = time.Now().Add(dd)
				case "🔂":
					deadline = time.Now().Add(refresh)
				default:
					if strings.HasSuffix(cust.Cmd, "❌") {
						tsX := strings.TrimSuffix(cust.Cmd, "❌") // empty|pause|connect|disconnect
						// if tsX == "" || strings.HasSuffix(status, tsX) || strings.HasPrefix(status, tsX) || (strings.HasPrefix(status, "❗") && tsX == "⁉️") {
						if tsX == "" || strings.HasSuffix(status, tsX) || strings.HasPrefix(status, tsX) || (strings.HasPrefix(status, "❗") && tsX == "❗") {
							for _, cu := range cus {
								stdo.Println("bot.DeleteMessage", cu)
								if cu.Reply != nil {
									bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(cu.Reply.Chat.ID), MessageID: cu.Reply.MessageID})
								}
							}
							return
						}
					}
				}
			} else { //load
				if cust.Cmd == ip && cust.Tm.Date > 0 {
					status = cust.Tm.From.FirstName
					deadline = time.Unix(cust.Tm.Date, 0)
					stdo.Println("loaded ", ip, status, deadline)
				}
				cus = append(cus, cust)
			}
			oStatus := status
			stdo.Println(ip, cust, len(ch), status, time.Now().Before(deadline))
			ikbs := []telego.InlineKeyboardButton{
				tu.InlineKeyboardButton("🔁").WithCallbackData("🔁"),
				tu.InlineKeyboardButton("🔂").WithCallbackData("🔂"),
				tu.InlineKeyboardButton("⏸️").WithCallbackData("⏸️"),
				tu.InlineKeyboardButton("❌").WithCallbackData("❌"),
				tu.InlineKeyboardButton("…").WithCallbackData("…"),
				tu.InlineKeyboardButton("❎").WithCallbackData("❎"),
			}
			if time.Now().Before(deadline) {
				status, err = ping(ip)
				if err != nil {
					// status = "⁉️"
					status = "❗"
					stdo.Println("ping", ip, err)
					//return
				}
			} else {
				if !strings.HasSuffix(status, "⏸️") {
					status += "⏸️"
				}
			}
			for i, cu := range cus {
				stdo.Println(i, fcRfRc(cu.Tm), ip, fcRfRc(cu.Reply), status, oStatus)
				if cu.Reply == nil || status != oStatus {
					if cu.Reply != nil {
						bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(cu.Reply.Chat.ID), MessageID: cu.Reply.MessageID})
					}
					ikbsf := 0
					if !chats.allowed(tf(cu.Tm.Chat.Type == "private", cu.Tm.From.ID, cu.Tm.Chat.ID)) {
						ikbsf = len(ikbs) - 1
					}
					//ikbsl := len(ikbs) - 1
					//if chats.allowed(cu.Tm.From.ID) {
					//	ikbsl++
					//}
					cus[i].Reply, _ = bot.SendMessage(tu.MessageWithEntities(tu.ID(cu.Tm.Chat.ID),
						tu.Entity(status),
						tu.Entity(ip).Code(),
						tu.Entity("⚡").TextLink(tl),
					).WithReplyToMessageID(cu.Tm.MessageID).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(ikbs[ikbsf:]...))))
				}
			}
		}
	}
}
