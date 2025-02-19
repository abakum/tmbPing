package main

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

const (
	numFL   = `(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[1-9])`
	tth     = time.Hour * 2
	refresh = time.Second * 60
	dd      = time.Hour * 8
	ttm     = time.Minute * 10
)

var (
	chats       = NewAAA()
	done        = make(chan bool, 10)
	ips         = sCustomer{mcCustomer: mcCustomer{}}
	bot         *tg.Bot
	tt          = tth
	save        = make(cCustomer, 1)
	saveDone    = make(chan bool, 1)
	tmbPingJson = "tmbPing.json"
	ticker,
	tacker *time.Ticker
	dic        = mss{}
	reIP       = regexp.MustCompile(numFL + `(\.(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){2}\.` + numFL)
	reYYYYMMDD = regexp.MustCompile(`(\p{L}*)\s([12][0-9][0-9][0-9]).?(0[1-9]|1[0-2]).?(0[1-9]|[12][0-9]|30|31)`)
	me         *tg.User
	ul         string
	ikbs       = []tg.InlineKeyboardButton{
		tu.InlineKeyboardButton("🔁").WithCallbackData("…🔁"),
		tu.InlineKeyboardButton("🔂").WithCallbackData("…🔂"),
		tu.InlineKeyboardButton("⏸️").WithCallbackData("…⏸️"),
		tu.InlineKeyboardButton("❌").WithCallbackData("…❌"),
		tu.InlineKeyboardButton("✅").WithCallbackData("…✅❌"),
		// tu.InlineKeyboardButton("⁉️").WithCallbackData("…⁉️❌"),
		tu.InlineKeyboardButton("❗").WithCallbackData("…❗❌"),
		tu.InlineKeyboardButton("⏸️").WithCallbackData("…⏸️❌"),
		tu.InlineKeyboardButton("❎").WithCallbackData("❎"),
	}
	ikbsf int
	wg    sync.WaitGroup
	bh    *th.BotHandler
	// getUpdates   = make(chan bool, 2)
	quitChannel  = make(chan bool, 2)
	quit2Channel = make(chan bool, 2)
)

// ping customer
type customer struct {
	Tm    *tg.Message `json:"tm,omitempty"`    //task
	Cmd   string      `json:"cmd,omitempty"`   //command
	Reply *tg.Message `json:"reply,omitempty"` //task reports
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
		ltf.Println("sCustomer.close saveDone <- true")
	}
}

// remove ip from ping list
func (s *sCustomer) del(ip string, closed bool) {
	ltf.Println("sCustomer.del ", ip)
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
		if ticker != nil {
			defer ticker.Reset(dd)
		}
		if s.save {
			saveDone <- true
			ltf.Println("del saveDone <- true")
		}
	}
}

// add ip to ping list
func (s *sCustomer) add(ip string) (ch cCustomer) {
	ltf.Println("sCustomer.add ", ip)
	ch = make(cCustomer, 10)
	go worker(ip, ch)
	s.Lock()
	defer s.Unlock()
	if len(s.mcCustomer) == 0 {
		if ticker != nil {
			defer ticker.Reset(refresh)
		}
	}
	s.mcCustomer[ip] = ch
	return
}

// add ip to ping list
func (s *sCustomer) write(ip string, c customer) {
	ltf.Println("sCustomer.write ", ip, c)
	defer func() {
		// recover from panic caused by writing to a closed channel
		if err := recover(); err != nil {
			letf.Println("sCustomer.write error:", err)
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

func (s *sCustomer) read(ip string) (ok bool) {
	ltf.Println("sCustomer.read ", ip)
	s.RLock()
	defer s.RUnlock()
	_, ok = s.mcCustomer[ip]
	return
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

type AAA []int64

func (a AAA) allowed(ChatID int64) bool {
	for _, v := range a {
		if v == ChatID {
			return true
		}
	}
	ltf.Println(ChatID, "not in", a)
	return false
}

func NewAAA() AAA {
	chats := AAA{}
	for _, s := range os.Args[1:] {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			continue
		}
		chats = append(chats, i)
	}
	return chats

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
