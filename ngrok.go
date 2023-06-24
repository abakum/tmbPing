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
	"time"

	"github.com/mymmrac/telego"
	"github.com/ngrok/ngrok-api-go/v5"
	"github.com/ngrok/ngrok-api-go/v5/tunnels"
)

func ngrokWeb() (publicURL string, forwardsTo string, err error) {
	web_addr := Getenv("web_addr", "localhost:4040")
	var client struct {
		Tunnels []struct {
			// Name      string `json:"name"`
			// ID        string `json:"ID"`
			// URI       string `json:"uri"`
			PublicURL string `json:"public_url"`
			// Proto     string `json:"proto"`
			Config struct {
				Addr string `json:"addr"`
				// Inspect bool   `json:"inspect"`
			} `json:"config"`
		} `json:"tunnels"`
		// URI string `json:"uri"`
	}
	resp, err := http.Get("http://" + web_addr + "/api/tunnels")
	if err != nil {
		return "", "", srcError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("http.Get resp.StatusCode: %v", resp.StatusCode)
		return "", "", Errorf("http.Get resp.StatusCode: %v", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", srcError(err)
	}
	err = json.Unmarshal(body, &client)
	if err != nil {
		return "", "", srcError(err)
	}
	for _, tunnel := range client.Tunnels {
		if true { //free version allow only one tunnel
			return tunnel.PublicURL, tunnel.Config.Addr, nil
		}
	}
	return "", "", Errorf("not found online client")
}
func ngrokAPI(NGROK_API_KEY string) (publicURL string, forwardsTo string, err error) {
	if NGROK_API_KEY == "" {
		return "", "", Errorf("empty NGROK_API_KEY")
	}

	// construct the api client
	clientConfig := ngrok.NewClientConfig(NGROK_API_KEY)

	// list all online client
	client := tunnels.NewClient(clientConfig)
	iter := client.List(nil)
	err = iter.Err()
	if err != nil {
		return "", "", srcError(err)
	}

	ctx, ca := context.WithTimeout(context.Background(), time.Second*3)
	defer ca()
	for iter.Next(ctx) {
		if true { //free version allow only one tunnel
			return iter.Item().PublicURL, iter.Item().ForwardsTo, nil
		}
	}
	err = iter.Err()
	if err != nil {
		return "", "", srcError(err)
	} else {
		return "", "", Errorf("not found online client")
	}
}

func ngrokAPI_() (publicURL string, forwardsTo string, err error) {
	NGROK_API_KEY := os.Getenv("NGROK_API_KEY")
	if NGROK_API_KEY == "" {
		return "", "", Errorf("not NGROK_API_KEY in env")
	}

	// construct the api client
	clientConfig := ngrok.NewClientConfig(NGROK_API_KEY)

	// list all online client
	client := tunnels.NewClient(clientConfig)
	iter := client.List(nil)
	err = iter.Err()
	if err != nil {
		return "", "", srcError(err)
	}

	ctx, ca := context.WithTimeout(context.Background(), time.Second*3)
	defer ca()
	for iter.Next(ctx) {
		err = iter.Err()
		if err != nil {
			return "", "", srcError(err)
		}
		if true { //free version allow only one tunnel
			return iter.Item().PublicURL, iter.Item().ForwardsTo, nil
		}
	}
	return "", "", Errorf("not found online client")
}

func manInTheMiddle(bot *telego.Bot) bool {
	// Receive information about webhook
	info, err := bot.GetWebhookInfo()
	if err != nil {
		return false
	}
	// stdo.Printf("Webhook Info: %+v\n", info)
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
	letf.Printf("manInTheMiddle GetWebhookInfo.IPAddress: %v but GetWebhookInfo.URL ip:%v\n", info.IPAddress, ips)
	return true
}
