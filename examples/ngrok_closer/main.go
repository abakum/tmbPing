package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"runtime/debug"
	"strings"

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
		bot     *telego.Bot
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
		if bot != nil {
			// Unset webhook on telegram server but keep updates for next start
			bot.DeleteWebhook(&telego.DeleteWebhookParams{DropPendingUpdates: false})
			fmt.Println("DeleteWebhook done")
		}
		fmt.Println("Done")
	})
	botToken := os.Getenv("TOKEN")

	// Note: Please keep in mind that default logger may expose sensitive information, use in development only
	bot, err = telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		err = srcError(err)
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
			err = srcError(err)
			return
		}

		// Handle stop signal
		cleanup = func() {
			fmt.Println("Stopping...")

			// Close ngrok tunnel
			fmt.Println("Cancel ngrok.Listen")
			ca()

			// Stop reviving updates from update channel and shutdown webhook server
			bot.StopWebhook()
			fmt.Println("StopWebhook done")
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
				// Restart ngrok tunnel
				return srcError(err)
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

// Get source of code
func src(deep int) (s string) {
	s = string(debug.Stack())
	str := strings.Split(s, "\n")
	if l := len(str); l <= deep {
		deep = l - 1
		for k, v := range str {
			fmt.Println(k, v)
		}
	}
	s = str[deep]
	s = strings.Split(s, " +0x")[0]
	_, s = path.Split(s)
	s += ":"
	return
}

// Wrap source of code and message to error
func Errorf(format string, args ...any) error {
	return fmt.Errorf(src(8)+" %w", fmt.Errorf(format, args...))
}

// Wrap source of code and error to error
func srcError(err error) error {
	return fmt.Errorf(src(8)+" %w", err)
}
