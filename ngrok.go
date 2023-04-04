package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/ngrok/ngrok-api-go/v5"
	"github.com/ngrok/ngrok-api-go/v5/tunnels"
)

func ngrokUrlAddr() (PublicURL string, host string, err error) {
	defer stdo.SetPrefix(stdo.Prefix())
	stdo.SetPrefix("ngrokUrlAddr ")
	web_addr := os.Getenv("web_addr")
	if web_addr == "" {
		web_addr = "localhost:4040"
	}
	// https://mholt.github.io/json-to-go/
	var ngrok struct {
		Tunnels []struct {
			Name      string `json:"name"`
			ID        string `json:"ID"`
			URI       string `json:"uri"`
			PublicURL string `json:"public_url"`
			Proto     string `json:"proto"`
			Config    struct {
				Addr    string `json:"addr"`
				Inspect bool   `json:"inspect"`
			} `json:"config"`
		} `json:"tunnels"`
		URI string `json:"uri"`
	}
	resp, err := http.Get("http://" + web_addr + "/api/tunnels")
	if err != nil {
		stdo.Println("ngrokUrlAddr http.Get error:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("ngrokUrlAddr http.Get resp.StatusCode: %v", resp.StatusCode)
		stdo.Println(err)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		stdo.Println("ngrokUrlAddr io.ReadAll error:", err)
		return
	}
	err = json.Unmarshal(body, &ngrok)
	if err != nil {
		stdo.Println("ngrokUrlAddr json.Unmarshal error:", err)
		return
	}
	for _, tunnel := range ngrok.Tunnels {
		PublicURL = tunnel.PublicURL
		u, err := url.Parse(tunnel.Config.Addr)
		if err != nil {
			stdo.Println("ngrokUrlAddr url.Parse error:", err)
			return PublicURL, tunnel.Config.Addr, err
		}
		host = u.Host
		if PublicURL != "" && host != "" {
			break
		}
	}
	return
}

func ngrokUrlTo(ctx context.Context, NGROK_API_KEY string) (PublicURL string, host string, err error) {
	defer stdo.SetPrefix(stdo.Prefix())
	stdo.SetPrefix("ngrokUrlTo ")
	// construct the api client
	clientConfig := ngrok.NewClientConfig(NGROK_API_KEY)

	// list all online tunnels
	tunnels := tunnels.NewClient(clientConfig)
	iter := tunnels.List(nil)
	err = iter.Err()
	if err != nil {
		stdo.Println("ngrokUrlTo tunnels.NewClient.List error:", err)
		return
	}
	for iter.Next(ctx) {
		err = iter.Err()
		if err != nil {
			stdo.Println("ngrokUrlTo tunnels.NewClient.Next error:", err)
			return
		}
		PublicURL = iter.Item().PublicURL
		u, err := url.Parse(iter.Item().ForwardsTo)
		if err != nil {
			stdo.Println("ngrokUrlTo url.Parse error:", err)
			return PublicURL, iter.Item().ForwardsTo, err
		}
		host = u.Host
		if PublicURL != "" && host != "" {
			break
		}
	}
	return
}

func manInTheMiddle(bot *telego.Bot) bool {
	defer stdo.SetPrefix(stdo.Prefix())
	stdo.SetPrefix("manInTheMiddle ")
	// Receive information about webhook
	info, err := bot.GetWebhookInfo()
	if err != nil {
		return false
	}
	stdo.Printf("Webhook Info: %+v\n", info)
	if info.IPAddress == "" || info.URL == "" {
		return false
	}

	//test ip of webhook
	u, err := url.Parse(info.URL)
	if err != nil {
		return false
	}
	ips, err := net.LookupIP(strings.Split(u.Host, ":")[0])
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.String() == info.IPAddress {
			return false
		}
	}
	stdo.Printf("manInTheMiddle GetWebhookInfo.IPAddress: %v but GetWebhookInfo.URL ip:%v\n", info.IPAddress, ips)
	return true
}
