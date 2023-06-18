package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"

	"github.com/fasthttp/router"
	"github.com/mymmrac/telego"
	"github.com/valyala/fasthttp"
)

func main() {
	botToken := os.Getenv("TOKEN")

	// Note: Please keep in mind that default logger may expose sensitive information, use in development only
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Initialize signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Initialize done chan
	done := make(chan struct{}, 1)

	for {
		restart := false

		// Create a new Ngrok tunnel to connect local network with the Internet & have HTTPS domain for bot
		fmt.Println("Create a new Ngrok tunnel")
		ctx, ca := context.WithCancel(context.Background())

		tun, err := ngrok.Listen(ctx,
			// Forward connections to localhost:8080
			config.HTTPEndpoint(config.WithForwardsTo(":8080")),
			// Authenticate into Ngrok using NGROK_AUTHTOKEN env (optional)
			ngrok.WithAuthtokenFromEnv(),
		)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Handle stop signal (Ctrl+C) or restart
		go func() {
			// Wait for stop signal or restart
			<-sigs

			fmt.Println("Stopping...")

			// Close ngrok tunnel
			fmt.Println("Cancel ngrok.Listen")
			ca()

			// Stop reviving updates from update channel and shutdown webhook server
			bot.StopWebhook()
			fmt.Println("StopWebhook done")

			// Notify that stop is done
			done <- struct{}{}
		}()

		// Set SecretToken - let there be a little more security
		secret := tun.ID()

		// Prepare fast HTTP server
		srv := &fasthttp.Server{}
		whs := telego.FastHTTPWebhookServer{
			Logger:      bot.Logger(),
			Server:      srv,
			Router:      router.New(),
			SecretToken: secret,
		}

		// Get an update channel from webhook using Ngrok
		updates, err := bot.UpdatesViaWebhook("/"+secret,
			// Set fast http server that will be used to handle webhooks
			telego.WithWebhookServer(whs),
			// Calls SetWebhook before starting webhook and provide dynamic Ngrok tunnel URL
			telego.WithWebhookSet(&telego.SetWebhookParams{
				URL:         tun.URL() + "/" + secret,
				SecretToken: secret,
			}),
		)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		bot.GetWebhookInfo()

		// Accept connection from ngrok tunnel
		go func() {
			for {
				conn, err := tun.Accept()
				if err != nil {
					bot.Logger().Errorf("tun.Accept %v", err)
					return
				}
				bot.Logger().Debugf("%s => %s", conn.RemoteAddr().String(), conn.LocalAddr().String())
				go func() {
					err := srv.ServeConn(conn)
					if err != nil {
						bot.Logger().Errorf("Server.ServeConn %v: %v", conn, err)
					}
					bot.Logger().Debugf("Server.ServeConn done")
				}()
			}
		}()

		// Start server for receiving requests from the Telegram
		go func() {
			_ = bot.StartWebhook(":8080")
		}()

		// Loop through all updates when they came
		for update := range updates {
			fmt.Printf("Update: %+v\n", update)

			if update.Message != nil {
				// Restart ngrok tunnel on command /restart
				if strings.HasPrefix(update.Message.Text, "/restart") {
					restart = true
					sigs <- syscall.Signal(0xa)
					<-done
				}

				// Stop bot on command /stop
				if strings.HasPrefix(update.Message.Text, "/stop") {
					sigs <- syscall.SIGTERM
				}
			}
		}
		if !restart {
			break
		}
	}

	// Wait for the stop process to be completed
	<-done

	// Unset webhook on telegram server but keep updates for next start
	bot.DeleteWebhook(&telego.DeleteWebhookParams{DropPendingUpdates: false})
	fmt.Println("DeleteWebhook done")

	fmt.Println("Done")
}
