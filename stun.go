package main

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/abakum/go-stun/stun"
)

func GetExternalIP(timeout time.Duration, servers ...string) (ip, message string, err error) {
	type IPfromSince struct {
		IP, From string
		Since    time.Duration
		Err      error
	}

	var done atomic.Bool

	ch := make(chan *IPfromSince)
	defer close(ch)

	t := time.AfterFunc(timeout, func() {
		if !done.Load() {
			ch <- &IPfromSince{"", strings.Join(servers, ","), timeout, fmt.Errorf("timeout")}
		}
	})
	defer t.Stop()
	for _, server := range servers {
		go func(s string) {
			client := stun.NewClient()
			client.SetServerAddr(s)
			t := time.Now()
			ip, err := client.GetExternalIP()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err, "from", s)
				return
			}
			if !done.Load() {
				ch <- &IPfromSince{ip, s, time.Since(t), nil}
			}
		}(server)
	}
	i := <-ch
	done.Store(true)

	message = fmt.Sprint(i.Err, " get external IP")
	if i.Err == nil {
		message = fmt.Sprint("External IP: ", i.IP)
	}
	message += fmt.Sprint(" from ", i.From, " since ", i.Since.Seconds(), "s")

	if i.Err != nil {
		return "127.0.0.1", message, fmt.Errorf("%s", message)
	}

	return i.IP, message, nil
}
