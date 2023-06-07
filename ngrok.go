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

	"github.com/mymmrac/telego"
	"github.com/ngrok/ngrok-api-go/v5"
	"github.com/ngrok/ngrok-api-go/v5/tunnels"
)

func ngrokWeb() (string, string, error) {
	web_addr := os.Getenv("web_addr")
	if web_addr == "" {
		web_addr = "localhost:4040"
	}
	var client struct {
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
		stdo.Println("ngrokWeb http.Get error:", err)
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("ngrokWeb http.Get resp.StatusCode: %v", resp.StatusCode)
		stdo.Println(err)
		return "", "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		stdo.Println("ngrokWeb io.ReadAll error:", err)
		return "", "", err
	}
	err = json.Unmarshal(body, &client)
	if err != nil {
		stdo.Println("ngrokWeb json.Unmarshal error:", err)
		return "", "", err
	}
	for _, tunnel := range client.Tunnels {
		if true { //free version allow only one tunnel
			return tunnel.PublicURL, tunnel.Config.Addr, nil
		}
	}
	return "", "", fmt.Errorf("ngrokWeb not found online client")
}

func ngrokAPI(ctx context.Context, NGROK_API_KEY string) (string, string, error) {
	// construct the api client
	clientConfig := ngrok.NewClientConfig(NGROK_API_KEY)

	// list all online client
	client := tunnels.NewClient(clientConfig)
	iter := client.List(nil)
	err := iter.Err()
	if err != nil {
		stdo.Println("ngrokAPI tunnels.NewClient.List error:", err)
		return "", "", err
	}
	for iter.Next(ctx) {
		err = iter.Err()
		if err != nil {
			stdo.Println("ngrokAPI tunnels.NewClient.Next error:", err)
			return "", "", err
		}
		if true { //free version allow only one tunnel
			return iter.Item().PublicURL, iter.Item().ForwardsTo, nil
		}
	}
	return "", "", fmt.Errorf("ngrokAPI not found online client")
}

func manInTheMiddle(bot *telego.Bot) bool {
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
	ips, err := net.LookupIP(u.Hostname())
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
