//go:build with_tailscale

package outbound

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/experimental/tailscale"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-dns"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

var (
	_ adapter.Outbound                = (*Tailscale)(nil)
	_ adapter.InterfaceUpdateListener = (*Tailscale)(nil)
)

type Tailscale struct {
	myOutboundAdapter
	ctx      context.Context
	tsServer *tailscale.TsServer
}

func NewTailscale(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.TailscaleOutboundOptions) (*Tailscale, error) {
	outbound := &Tailscale{
		myOutboundAdapter: myOutboundAdapter{
			protocol:     C.TypeTailscale,
			network:      options.Network.Build(),
			router:       router,
			logger:       logger,
			tag:          tag,
			dependencies: withDialerDependency(options.DialerOptions),
		},
		ctx: ctx,
	}
	if c, ok := ctx.Value("tailscale").(*tailscale.TailscaleConfig); !ok {
		return nil, E.New("missing tailscale config")
	} else {
		outbound.tsServer = c.LoadOrStoreNode(options.Node)
	}
	return outbound, nil
}

func (t *Tailscale) Start() error {
	return t.start()
}

func (t *Tailscale) PostStart() error {
	return t.start()
}

func (t *Tailscale) start() error {
	return t.tsServer.Start()
}

func (t *Tailscale) Close() error {
	return t.tsServer.Close()
}

func (t *Tailscale) InterfaceUpdated() {
	return
}

func (t *Tailscale) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	switch network {
	case N.NetworkTCP:
		t.logger.InfoContext(ctx, "outbound connection to ", destination)
	case N.NetworkUDP:
		t.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	}
	if destination.IsFqdn() {
		destinationAddresses, err := t.router.LookupDefault(ctx, destination.Fqdn)
		if err != nil {
			return nil, err
		}
		return N.DialSerial(ctx, t.tsServer, network, destination, destinationAddresses)
	}
	return t.tsServer.DialContext(ctx, network, destination)
}

func (t *Tailscale) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	t.logger.InfoContext(ctx, "outbound packet connection to ", destination)
	if destination.IsFqdn() {
		destinationAddresses, err := t.router.LookupDefault(ctx, destination.Fqdn)
		if err != nil {
			return nil, err
		}
		packetConn, _, err := N.ListenSerial(ctx, t.tsServer, destination, destinationAddresses)
		if err != nil {
			return nil, err
		}
		return packetConn, err
	}
	return t.tsServer.ListenPacket(ctx, destination)
}

func (t *Tailscale) NewConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext) error {
	return NewDirectConnection(ctx, t.router, t, conn, metadata, dns.DomainStrategyAsIS)
}

func (t *Tailscale) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata adapter.InboundContext) error {
	return NewDirectPacketConnection(ctx, t.router, t, conn, metadata, dns.DomainStrategyAsIS)
}
