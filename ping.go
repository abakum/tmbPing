package main

import (
	"fmt"
	"runtime"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func ping(ip string) (status string, err error) {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return
	}
	defer pinger.Stop()
	pinger.SetPrivileged(runtime.GOOS == "windows")
	pinger.Count = 3
	pinger.Interval = time.Millisecond * 100
	pinger.Timeout = pinger.Interval*time.Duration(pinger.Count-1) + time.Millisecond*time.Duration(pinger.Count*100)
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return
	}
	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats
	if stats.PacketsRecv == pinger.Count {
		status = "✅"
		fmt.Printf("%v echoReply %d<rtt~%d<%d\n", ip, stats.MinRtt.Milliseconds(), stats.AvgRtt.Milliseconds(), stats.MaxRtt.Milliseconds())
	} else {
		status = "❗"
		fmt.Printf("%v %d/%d packets received\n", ip, stats.PacketsRecv, pinger.Count)
	}
	return
}
