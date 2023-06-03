package main

import (
	"regexp"
	"strings"

	tg "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func tmtc(update tg.Update) (tc string, m *tg.Message) {
	if update.Message == nil {
		return "", nil
	}
	for _, tm := range []*tg.Message{update.EditedMessage,
		update.EditedChannelPost,
		update.Message,
		update.ChannelPost} {
		//edit = i < 2
		if tm != nil {
			m = tm
			tc += tm.Text + " "
			tc += tm.Caption + " "
			if tm.ReplyToMessage != nil {
				tc += tm.ReplyToMessage.Text + " "
				tc += tm.ReplyToMessage.Caption + " "
			}
			break
		}
	}
	return
}
func anyWithMatch(pattern *regexp.Regexp) th.Predicate {
	return func(update tg.Update) bool {
		tc, _ := tmtc(update)
		return pattern.MatchString(tc)
	}
}
func AnyCommand() th.Predicate {
	return func(update tg.Update) bool {
		_, ctm := tmtc(update)
		if ctm == nil {
			return false
		}
		return strings.HasPrefix(ctm.Text, "/") || strings.HasPrefix(ctm.Caption, "/")
	}
}
func leftChat() th.Predicate {
	return func(update tg.Update) bool {
		return update.Message != nil && update.Message.LeftChatMember != nil
	}
}
func newMember() th.Predicate {
	return func(update tg.Update) bool {
		return update.Message != nil && len(update.Message.NewChatMembers) > 0
	}
}
func Delete(ChatID tg.ChatID, MessageID int) *tg.DeleteMessageParams {
	return &tg.DeleteMessageParams{
		ChatID:    ChatID,
		MessageID: MessageID,
	}
}
