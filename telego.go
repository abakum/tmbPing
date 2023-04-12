package main

import (
	"regexp"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func tmtc(update telego.Update) (tc string, m *telego.Message) {
	for _, tm := range []*telego.Message{update.EditedMessage,
		update.EditedChannelPost,
		update.Message,
		update.ChannelPost} {
		//edit = i < 2
		if tm != nil {
			m = tm
			tc += tm.Text + " "
			tc += tm.Caption + " "
			tm = tm.ReplyToMessage
			if tm != nil {
				//m = tm
				tc += tm.Text + " "
				tc += tm.Caption + " "
			}
			break
		}
	}
	return
}
func anyWithMatch(pattern *regexp.Regexp) th.Predicate {
	return func(update telego.Update) bool {
		tc, _ := tmtc(update)
		return pattern.MatchString(tc)
	}
}
func AnyCommand() th.Predicate {
	return func(update telego.Update) bool {
		_, ctm := tmtc(update)
		return strings.HasPrefix(ctm.Text, "/") || strings.HasPrefix(ctm.Caption, "/")
	}
}
func leftChat() th.Predicate {
	return func(update telego.Update) bool {
		return update.Message != nil && update.Message.LeftChatMember != nil
	}
}
func newMember() th.Predicate {
	return func(update telego.Update) bool {
		return update.Message != nil && update.Message.NewChatMembers != nil
	}
}
