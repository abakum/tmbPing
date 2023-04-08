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
	deadline := time.Now().Add(dd)
	cus := customers{}
	defer ips.del(ip, false)
	for {
		select {
		case <-done:
			done <- true //done other worker
			stdo.Println("save", ip)
			for _, cu := range cus {
				cu.Tm = &telego.Message{MessageID: cu.Tm.MessageID, From: &telego.User{ID: cu.Tm.From.ID}, Chat: telego.Chat{ID: cu.Tm.Chat.ID}}
				cu.Cmd = ip
				//cu.Status = status
				cu.Tm.Text = status
				//cu.Deadline = deadline
				cu.Tm.Date = deadline.Unix()
				stdo.Println("---Save ", status, deadline)
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
					if cust.Cmd == "❌" || strings.TrimSuffix(cust.Cmd, "❌") == strings.TrimSuffix(status, "⏸️") {
						for _, cu := range cus {
							if cu.Reply != nil {
								bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(cu.Reply.Chat.ID), MessageID: cu.Reply.MessageID})
							}
						}
						return
					}
				}
			} else { //load
				if cust.Cmd == ip && status == "" {
					//status = cust.Status
					status = cust.Tm.Text
					//deadline = cust.Deadline
					deadline = time.Unix(cust.Tm.Date, 0)
					stdo.Println("---Load ", status, deadline)
				}
				cus = append(cus, cust)
			}
			oStatus := status
			stdo.Println(ip, cust, len(ch), status, time.Now().Before(deadline))
			ikbse := []telego.InlineKeyboardButton{
				tu.InlineKeyboardButton("❌").WithCallbackData("❌"),
				tu.InlineKeyboardButton("❎").WithCallbackData("❎"),
				tu.InlineKeyboardButton("…").WithCallbackData("…"),
			}
			ikbs := append([]telego.InlineKeyboardButton{
				tu.InlineKeyboardButton("⏸️").WithCallbackData("⏸️"),
			}, ikbse...)
			if time.Now().Before(deadline) {
				status, err = ping(ip)
				if err != nil {
					stdo.Println("ping", ip, err)
					return
				}
			} else {
				if !strings.HasSuffix(status, "⏸️") {
					status += "⏸️"
				}
				ikbs = append([]telego.InlineKeyboardButton{
					tu.InlineKeyboardButton("🔁").WithCallbackData("🔁"),
					tu.InlineKeyboardButton("🔂").WithCallbackData("🔂"),
				}, ikbse...)
			}
			for i, cu := range cus {
				stdo.Println(i, cu, status, oStatus)
				if cu.Reply == nil || status != oStatus {
					if cu.Reply != nil {
						bot.DeleteMessage(&telego.DeleteMessageParams{ChatID: tu.ID(cu.Reply.Chat.ID), MessageID: cu.Reply.MessageID})
					}
					ikbsl := len(ikbs) - 1
					if chats.allowed(cu.Tm.From.ID) {
						ikbsl++
					}
					cus[i].Reply, _ = bot.SendMessage(tu.MessageWithEntities(tu.ID(cu.Tm.Chat.ID),
						tu.Entity(status),
						tu.Entity("/"+ip).Code(),
					).WithReplyToMessageID(cu.Tm.MessageID).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(ikbs[:ikbsl]...))))
				}
			}
		}
	}
}
