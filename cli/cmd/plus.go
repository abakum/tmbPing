//go:build !desktop
// +build !desktop

package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/beevik/guid"
	"github.com/loophole/cli/config"
	"github.com/loophole/cli/internal/app/loophole"
	lm "github.com/loophole/cli/internal/app/loophole/models"
	"github.com/loophole/cli/internal/pkg/communication"
	"github.com/loophole/cli/internal/pkg/token"
	"github.com/loophole/cli/internal/pkg/urlmaker"
)

// GoExecute runs command parsing chain from go.
// Look example ../example/main.go.
func GoExecute(ctx context.Context, version, commit, mode string, args ...string) error {
	config.Config.Version = version
	config.Config.CommitHash = commit
	config.Config.ClientMode = mode

	rootCmd.Version = fmt.Sprintf("%s (%s)", config.Config.Version, config.Config.CommitHash)

	rootCmd.SetArgs(args)
	return rootCmd.ExecuteContext(ctx)
}

// Like GoExecute(ctx, version, commit, mode, "--hostname", hostname, "http", p, h)
// but without communication.Fatal in case of an unsuccessful attempt loophole.RegisterTunnel
func ForwarDPort(version, commit, mode, hostname, p, h string, quitChannel chan bool) (siteAddr string, err error) {
	// rootInit
	// cobra.OnInitialize(initLogger)

	// Execute
	config.Config.Version = version
	config.Config.CommitHash = commit
	config.Config.ClientMode = mode
	rootCmd.Version = fmt.Sprintf("%s (%s)", config.Config.Version, config.Config.CommitHash)

	// initServeCommand
	remoteEndpointSpecs.SiteID = hostname
	remoteEndpointSpecs.TunnelID = guid.NewString()

	// httpCmdRun
	loggedIn := token.IsTokenSaved()
	idToken := token.GetIdToken()
	communication.ApplicationStart(loggedIn, idToken)

	checkVersion()

	localEndpointSpecs.Host = h
	port, _ := strconv.ParseInt(p, 10, 32)
	localEndpointSpecs.Port = int32(port)

	exposeConfig := lm.ExposeHTTPConfig{
		Local:  localEndpointSpecs,
		Remote: remoteEndpointSpecs,
	}
	authMethod, err := loophole.RegisterTunnel(&exposeConfig.Remote)
	if err != nil {
		// communication.Fatal(err.Error())
		return "", err
	}
	return urlmaker.GetSiteURL("https", exposeConfig.Remote.SiteID, exposeConfig.Remote.Domain), loophole.ForwarDPort(exposeConfig, authMethod, quitChannel)
}
