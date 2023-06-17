package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime/debug"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoutil"
	"github.com/xlab/closer"
)

const (
	ansiReset = "\u001B[0m"
	ansiRedBG = "\u001B[41m"
	BUG       = ansiRedBG + "Ж" + ansiReset
	// BUG = "💥"
)

var (
	// Log error with time and source of error
	letf = log.New(os.Stdout, BUG, log.Ltime|log.Lshortfile)
	// Log debug with time and source of debug
	ltf = log.New(os.Stdout, " ", log.Ltime|log.Lshortfile)

	// Log error with time for custom loger
	let = log.New(os.Stdout, BUG, log.Ltime)
	// Log debug with time for custom loger
	lt = log.New(os.Stdout, " ", log.Ltime)
)

// Custom loger type
type customLogger struct{}

// Custom logger method for debug
func (customLogger) Debugf(format string, args ...any) {
	lt.Print(woToken(format, args...))
}

// Custom logger method for error
func (customLogger) Errorf(format string, args ...any) {
	let.Print(woToken(format, args...))
}

func main() {
	var (
		err error
	)
	defer closer.Close()

	// Register the cleanup function
	closer.Bind(func() {
		if err != nil {
			let.Println(err)
			defer os.Exit(1)
		}
	})

	// Try create bot with empty botToken
	botToken := ""

	// Print debug
	ltf.Println("Try create bot with empty botToken")
	// 15:37:21 main.go:66: Try create bot with empty botToken
	_, err = telego.NewBot(botToken)
	// Print error and ignore it
	letf.Println(err)
	//💥15:37:21 main.go:70: telego: invalid token

	botToken = os.Getenv("TOKEN")

	// Create bot with fake APIServer and health check
	ltf.Println("Create bot with fake APIServer and health check")
	// 15:37:21 main.go:76: Create bot with fake APIServer and health check
	_, err = telego.NewBot(botToken,
		// Change bot API server URL (default: https://api.telegram.org)
		telego.WithAPIServer("https://api.telegram.com"),

		// Enables basic health check that will call getMe method before returning bot instance (default: false)
		telego.WithHealthCheck(),

		// Create you custom logger that implements telego.Logger (default: telego has build in default logger)
		// Note: Please keep in mind that logger may expose sensitive information, use in development only or configure
		// it not to leak unwanted content
		telego.WithLogger(telego.Logger(customLogger{})),
	)
	// 15:37:21 bot.go:166: API call to: "https://api.telegram.com/bot/getMe", with data: null
	//💥15:37:22 bot.go:104: Execution error getMe: request call: fasthttp do request: tls: failed to verify certificate: x509: ...
	// Print error
	letf.Println(err)
	//💥15:37:22 main.go:93: telego: health check: telego: getMe(): internal execution: request call: fasthttp do request: tls: failed to verify certificate: x509: ...

	ltf.Println("Create bot with custom logger")
	// 15:37:22 main.go:96: Create bot with custom logger
	bot, err := telego.NewBot(botToken,
		// Create you custom logger that implements telego.Logger (default: telego has build in default logger)
		// Note: Please keep in mind that logger may expose sensitive information, use in development only or configure
		// it not to leak unwanted content
		telego.WithLogger(telego.Logger(customLogger{})),
	)
	if err != nil {
		// Wrap error and print it on exit
		err = srcError(err)
		return
	}

	// Debug print over customLogger
	bot.Logger().Debugf("Call method DeleteMessage")
	// 15:37:22 main.go:111: Call method DeleteMessage

	// Call method DeleteMessage
	err = bot.DeleteMessage(telegoutil.Delete(telegoutil.ID(1), 1))
	// 15:37:22 bot.go:166: API call to: "https://api.telegram.org/bot/deleteMessage", with data: {"chat_id":1,"message_id":1}
	// 15:37:22 bot.go:107: API response deleteMessage: Ok: false, Err: [400 "Bad Request: chat not found"]
	if err != nil {
		// Print error over customLogger
		bot.Logger().Errorf("%+v\n", err)
		//💥15:37:22 main.go:120: telego: deleteMessage(): api: 400 "Bad Request: chat not found"
		// Wrap error and print it on exit
		err = srcError(err)
		//💥15:37:22 main.go:123: telego: deleteMessage(): api: 400 "Bad Request: chat not found"
	}
}

// Get source of code
func src(deep int) (s string) {
	s = string(debug.Stack())
	str := strings.Split(s, "\n")
	if l := len(str); l <= deep {
		deep = l - 1
		for k, v := range str {
			lt.Println(k, v)
		}
	}
	s = str[deep]
	s = strings.Split(s, " +0x")[0]
	_, s = path.Split(s)
	s += ":"
	return
}

// Hide bot token
func woToken(format string, args ...any) (s string) {
	// Add source of code to error
	s = src(10) + " " + fmt.Sprintf(format, args...)
	btStart := strings.Index(s, "/bot") + 4
	if btStart > 4-1 {
		btLen := strings.Index(s[btStart:], "/")
		if btLen > 0 {
			s = s[:btStart] + s[btStart+btLen:]
		}
	}
	return
}

// Wrap source of code and error to error
func srcError(err error) error {
	return fmt.Errorf(src(8)+" %w", err)
}
