package main

import (
	"strings"
	"time"

	tg "github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// send ip to ch for add it to ping list
func worker(ip string, ch cCustomer) {
	// var buttons *telego.InlineKeyboardMarkup
	var (
		err error
		status,
		statusOld string
		tl       = start(me, ip)
		deadline = time.Now().Add(dd)
		cus      = customers{}
		ikbs     = []tg.InlineKeyboardButton{
			tu.InlineKeyboardButton("🔁").WithCallbackData("🔁"),
			tu.InlineKeyboardButton("🔂").WithCallbackData("🔂"),
			tu.InlineKeyboardButton("⏸️").WithCallbackData("⏸️"),
			tu.InlineKeyboardButton("❌").WithCallbackData("❌"),
			tu.InlineKeyboardButton("…").WithCallbackData("…"),
			tu.InlineKeyboardButton("❎").WithCallbackData("❎"),
		}
		ikbsf int
	)
	defer ips.del(ip, false)
	for {
		select {
		case <-done:
			done <- true //done other worker
			for i, cu := range cus {
				cu.Tm = &tg.Message{MessageID: cu.Tm.MessageID, From: &tg.User{ID: cu.Tm.From.ID}, Chat: tg.Chat{ID: cu.Tm.Chat.ID}}
				cu.Cmd = ip
				if i == 0 {
					cu.Tm.From.FirstName = status
					cu.Tm.Date = deadline.Unix()
					ltf.Println("saved ", ip, status, deadline)
				}
				save <- cu
			}
			ltf.Println("done", ip)
			return
		case cust, ok := <-ch:
			if !ok {
				ltf.Println("channel closed", ip)
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
								ltf.Println("bot.DeleteMessage", cu)
								re := cu.Reply
								if re != nil {
									// bot.DeleteMessage(&tg.DeleteMessageParams{ChatID: tu.ID(re.Chat.ID), MessageID: re.MessageID})
									bot.DeleteMessage(tu.Delete(tu.ID(re.Chat.ID), re.MessageID))
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
					ltf.Println("loaded ", ip, status, deadline)
				}
				cus = append(cus, cust)
			}
			statusOld = status
			ltf.Println(ip, cust, len(ch), status, time.Now().Before(deadline))
			if time.Now().Before(deadline) {
				status, err = ping(ip)
				if err != nil {
					// status = "⁉️"
					status = "❗"
					ltf.Println("ping", ip, err)
					//return
				}
			} else {
				if !strings.HasSuffix(status, "⏸️") {
					status += "⏸️"
				}
			}
			for i, cu := range cus {
				re := cu.Reply
				ltf.Println(i, fcRfRc(cu.Tm), ip, fcRfRc(re), status, statusOld)
				if re == nil || status != statusOld {
					if re != nil {
						// bot.DeleteMessage(&tg.DeleteMessageParams{ChatID: tu.ID(re.Chat.ID), MessageID: re.MessageID})
						bot.DeleteMessage(tu.Delete(tu.ID(re.Chat.ID), re.MessageID))
					}
					ikbsf = 0
					if !chats.allowed(tf(cu.Tm.Chat.Type == "private", cu.Tm.From.ID, cu.Tm.Chat.ID)) {
						ikbsf = len(ikbs) - 1
					}
					cus[i].Reply, err = bot.SendMessage(tu.MessageWithEntities(tu.ID(cu.Tm.Chat.ID),
						tu.Entity(status),
						tu.Entity(ip).Code(),
						tu.Entity("⚡").TextLink(tl),
					).WithReplyToMessageID(cu.Tm.MessageID).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(ikbs[ikbsf:]...))))
					if err != nil {
						letf.Println("delete", ip)
						ips.del(ip, false)
					}
				}
			}
		}
	}
}
