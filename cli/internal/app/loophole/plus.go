//go:build !desktop
// +build !desktop

package loophole

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	lm "github.com/loophole/cli/internal/app/loophole/models"
	"github.com/loophole/cli/internal/pkg/communication"
	"github.com/loophole/cli/internal/pkg/urlmaker"
	"golang.org/x/crypto/ssh"
)

// forwardD like forward, but the client processing loop is in go func.
func forwarD(remoteEndpointSpecs lm.RemoteEndpointSpecs,
	authMethod ssh.AuthMethod, server *http.Server, localEndpoint string,
	protocols []string, quitChannel <-chan bool) error {

	localListenerEndpoint, err := startLocalHTTPServer(remoteEndpointSpecs.TunnelID, server)
	if err != nil {
		communication.TunnelStartFailure(remoteEndpointSpecs.TunnelID, err)
		return err
	}
	serverSSHConnHTTPS, err := connectViaSSH(remoteEndpointSpecs.SiteID, remoteEndpointSpecs.TunnelID, authMethod)
	if err != nil {
		communication.TunnelStartFailure(remoteEndpointSpecs.TunnelID, err)
		return err
	}
	listenerHTTPSOverSSH, err := listenOnRemoteEndpoint(remoteEndpointSpecs.TunnelID, serverSSHConnHTTPS)
	if err != nil {
		communication.TunnelStartFailure(remoteEndpointSpecs.TunnelID, err)
		return err
	}

	go func() {
		communication.TunnelDebug(remoteEndpointSpecs.TunnelID, "Issuing request to provision certificate")
		var netTransport = &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 30 * time.Second,
		}
		var netClient = &http.Client{
			Timeout:   time.Second * 30,
			Transport: netTransport,
		}
		_, err := netClient.Get(urlmaker.GetSiteURL("https", remoteEndpointSpecs.SiteID, remoteEndpointSpecs.Domain))

		if err != nil {
			communication.TunnelError(remoteEndpointSpecs.TunnelID, "TLS Certificate failed to provision. Will be obtained with first request made by any client, therefore first execution may be slower")
		} else {
			communication.TunnelInfo(remoteEndpointSpecs.TunnelID, "TLS Certificate successfully provisioned")
		}
	}()

	communication.TunnelStartSuccess(remoteEndpointSpecs, localEndpoint)

	acceptedClients := make(chan net.Conn)
	tunnelTerminatedOnPurpose := false

	go func(l *net.Listener, tunnelTerminatedOnPurpose *bool) {
		for {
			communication.TunnelDebug(remoteEndpointSpecs.TunnelID, "Waiting to accept")
			client, err := (*l).Accept()
			communication.TunnelDebug(remoteEndpointSpecs.TunnelID, "Accepted")
			if err == io.EOF {
				if !(*tunnelTerminatedOnPurpose) {
					communication.TunnelInfo(remoteEndpointSpecs.TunnelID, err.Error()+" Connection dropped, reconnecting...")
					(*l).Close()
					serverSSHConnHTTPS, err = connectViaSSH(remoteEndpointSpecs.SiteID, remoteEndpointSpecs.TunnelID, authMethod)
					if err != nil {
						communication.TunnelStartFailure(remoteEndpointSpecs.TunnelID, err)
						return
					} else {
						defer serverSSHConnHTTPS.Close()
					}
					defer serverSSHConnHTTPS.Close()
					l, err = listenOnRemoteEndpoint(remoteEndpointSpecs.TunnelID, serverSSHConnHTTPS)
					if err != nil {
						communication.TunnelStartFailure(remoteEndpointSpecs.TunnelID, err)
						return
					} else {
						defer (*l).Close()
					}
					continue
				}
			} else if err != nil {
				communication.TunnelWarn(remoteEndpointSpecs.TunnelID, "Failed to accept connection over HTTPS")
				communication.TunnelWarn(remoteEndpointSpecs.TunnelID, err.Error())
				continue
			}
			communication.TunnelDebug(remoteEndpointSpecs.TunnelID, "Sending client trough channel")
			acceptedClients <- client
		}
	}(listenerHTTPSOverSSH, &tunnelTerminatedOnPurpose)

	go func() {
		for {
			communication.TunnelDebug(remoteEndpointSpecs.TunnelID, "For loop cycle")
			select {
			case <-quitChannel:
				tunnelTerminatedOnPurpose = true
				communication.TunnelStopSuccess(remoteEndpointSpecs.TunnelID)
				defer server.Close()
				defer serverSSHConnHTTPS.Close()
				defer (*listenerHTTPSOverSSH).Close()
				return
			case client := <-acceptedClients:
				communication.TunnelDebug(remoteEndpointSpecs.TunnelID, "Handling client")
				go func() {
					communication.TunnelInfo(remoteEndpointSpecs.TunnelID, "Succeeded to accept connection over HTTPS")
					communication.TunnelDebug(remoteEndpointSpecs.TunnelID, fmt.Sprintf("Dialing into local proxy for HTTPS: %s", localListenerEndpoint.URI()))
					local, err := net.Dial("tcp", localListenerEndpoint.URI())
					if err != nil {
						communication.TunnelError(remoteEndpointSpecs.TunnelID, "Dialing into local proxy for HTTPS failed")
					}
					communication.TunnelDebug(remoteEndpointSpecs.TunnelID, "Dialing into local proxy for HTTPS succeeded")
					handleClient(remoteEndpointSpecs.TunnelID, client, local)
				}()
			}
		}
	}()
	return nil
}

// ForwarDPort is used to forward external URL to locally available port.
// Like ForwardPort but with forwarD instead of forward.
func ForwarDPort(exposeHTTPConfig lm.ExposeHTTPConfig, publicKeyAuthMethod ssh.AuthMethod, quitChannel <-chan bool) error {
	protocol := "http"
	if exposeHTTPConfig.Local.HTTPS {
		protocol = "https"
	}
	localEndpoint := lm.Endpoint{
		Protocol: protocol,
		Host:     exposeHTTPConfig.Local.Host,
		Port:     exposeHTTPConfig.Local.Port,
		Path:     exposeHTTPConfig.Local.Path,
	}

	server, err := createTLSReverseProxy(localEndpoint, exposeHTTPConfig.Remote)
	if err != nil {
		return err
	}
	return forwarD(exposeHTTPConfig.Remote, publicKeyAuthMethod, server, localEndpoint.URI(), []string{"https"}, quitChannel)
}
