package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"

	"github.com/fasthttp/router"
	"github.com/mymmrac/telego"
	"github.com/valyala/fasthttp"
	"github.com/xlab/closer"
)

func main() {
	var (
		err     error
		cleanup func()
	)
	defer closer.Close()

	// Register the cleanup function
	closer.Bind(func() {
		if err != nil {
			fmt.Println(err)
			defer os.Exit(1)
		}
		if cleanup != nil {
			cleanup()
		}
		fmt.Println("Done")
	})
	botToken := os.Getenv("TOKEN")

	// Note: Please keep in mind that default logger may expose sensitive information, use in development only
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		return
	}

	for {
		restart := false

		// Create a new Ngrok tunnel to connect local network with the Internet & have HTTPS domain for bot
		fmt.Println("Create a new Ngrok tunnel")
		ctx, ca := context.WithCancel(context.Background())

		tun, err := ngrok.Listen(ctx,
			// Forward connections to localhost:8080 (optional)
			config.HTTPEndpoint(), //config.WithForwardsTo(":8080")
			// Authenticate into Ngrok using NGROK_AUTHTOKEN env (optional)
			ngrok.WithAuthtokenFromEnv(),
		)
		if err != nil {
			return
		}

		// Handle stop signal (Ctrl+C)
		cleanup = func() {
			fmt.Println("Stopping...")

			// Close ngrok tunnel
			fmt.Println("Cancel ngrok.Listen")
			ca()

			// Stop reviving updates from update channel and shutdown webhook server
			bot.StopWebhook()
			fmt.Println("StopWebhook done")

			// Unset webhook on telegram server but keep updates for next start
			bot.DeleteWebhook(&telego.DeleteWebhookParams{DropPendingUpdates: false})
			fmt.Println("DeleteWebhook done")
		}

		// Set SecretToken - let there be a little more security
		secret := tun.ID()

		// Prepare fast HTTP server
		srv := &fasthttp.Server{}
		fwhs := telego.FuncWebhookServer{
			Server: telego.FastHTTPWebhookServer{
				Logger:      bot.Logger(),
				Server:      srv,
				Router:      router.New(),
				SecretToken: secret,
			},
			// Override default start func to use Ngrok tunnel
			StartFunc: func(_ string) error {
				bot.Logger().Debugf("Serve")
				err := srv.Serve(tun)
				if err != nil {
					if err.Error() == "failed to accept connection: Tunnel closed" {
						bot.Logger().Debugf("serverClosed")
						return nil
					}
					bot.Logger().Errorf("Serve %s", err)
				}
				return err
			},
		}

		// Get an update channel from webhook using Ngrok
		updates, err := bot.UpdatesViaWebhook("/"+secret,
			// Set func server with fast http server inside that will be used to handle webhooks
			telego.WithWebhookServer(fwhs),
			// Calls SetWebhook before starting webhook and provide dynamic Ngrok tunnel URL
			telego.WithWebhookSet(&telego.SetWebhookParams{
				URL:         tun.URL() + "/" + secret,
				SecretToken: secret,
			}),
		)
		if err != nil {
			return
		}
		bot.GetWebhookInfo()

		// Start server for receiving requests from the Telegram
		go func() {
			_ = bot.StartWebhook("")
		}()

		// Loop through all updates when they came
		for update := range updates {
			fmt.Printf("Update: %+v\n", update)

			if update.Message != nil {
				// Restart ngrok tunnel on command /restart
				if strings.HasPrefix(update.Message.Text, "/restart") {
					restart = true
					cleanup()
					// To prevent "Too Many Requests"
					time.Sleep(time.Second) //to prevent Too Many Requests
				}

				// Stop bot on command /stop
				if strings.HasPrefix(update.Message.Text, "/stop") {
					return
				}
			}
		}
		if !restart {
			break
		}
	}
}
