package tailscale

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	F "github.com/sagernet/sing/common/format"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"tailscale.com/tsnet"
)

type TailscaleConfig struct {
	options     *option.TailscaleOptions
	logFactory  log.Factory
	logger      log.ContextLogger
	nodeServers *sync.Map
}

func NewTailscaleConfig(options *option.TailscaleOptions, logFactory log.Factory) *TailscaleConfig {
	logger := logFactory.NewLogger("experimental/tailscale")
	nodeServers := &sync.Map{}
	return &TailscaleConfig{options: options, logFactory: logFactory, logger: logger, nodeServers: nodeServers}
}

func (t *TailscaleConfig) authKey(name string) string {
	if node, ok := t.options.Nodes[name]; ok {
		if node.AuthKey != "" {
			return node.AuthKey
		}
	}
	return t.options.AuthKey
}

func (t *TailscaleConfig) controlURL(name string) string {
	if node, ok := t.options.Nodes[name]; ok {
		if node.ControlURL != "" {
			return node.ControlURL
		}
	}
	return t.options.ControlURL
}

func (t *TailscaleConfig) ephemeral(name string) bool {
	if node, ok := t.options.Nodes[name]; ok {
		if ephemeral, ok := node.Ephemeral.Get(); ok {
			return ephemeral
		}
	}
	return t.options.Ephemeral
}

func (t *TailscaleConfig) webUI(name string) bool {
	if node, ok := t.options.Nodes[name]; ok {
		if webUI, ok := node.WebUI.Get(); ok {
			return webUI
		}
	}
	return t.options.WebUI
}

func (t *TailscaleConfig) hostname(name string) string {
	if node, ok := t.options.Nodes[name]; ok {
		if node.Hostname != "" {
			return node.Hostname
		}
	}
	return name
}

func (t *TailscaleConfig) LoadOrStoreNode(name string) *TsServer {
	logger := t.logFactory.NewLogger(F.ToString("experimental/tailscale", "[", name, "]"))
	s := &TsServer{
		Server: tsnet.Server{
			Hostname: t.hostname(name),
			Logf: func(format string, args ...any) {
				logger.Debug(fmt.Sprintf(format, args...))
			},
			UserLogf: func(format string, args ...any) {
				logger.Info(fmt.Sprintf(format, args...))
			},
			Ephemeral:    t.ephemeral(name),
			AuthKey:      t.authKey(name),
			ControlURL:   t.controlURL(name),
			RunWebClient: t.webUI(name),
		},
		logger: logger,
	}
	node, _ := t.nodeServers.LoadOrStore(name, s)
	return node.(*TsServer)
}

var (
	_ N.Dialer = (*TsServer)(nil)
)

type TsServer struct {
	tsnet.Server
	logger log.ContextLogger
}

func (s *TsServer) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return s.Server.Dial(ctx, network, destination.String())
}

func (s *TsServer) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return s.Server.ListenPacket("udp", destination.String())
}
