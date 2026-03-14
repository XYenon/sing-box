package group

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/interrupt"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/stretchr/testify/require"
)

type testOutbound struct {
	outbound.Adapter
	dialErr        error
	listenErr      error
	dialConn       net.Conn
	listenPacket   net.PacketConn
}

func (t *testOutbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	if t.dialErr != nil {
		return nil, t.dialErr
	}
	return t.dialConn, nil
}

func (t *testOutbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	if t.listenErr != nil {
		return nil, t.listenErr
	}
	return t.listenPacket, nil
}

func newTestOutbound(tag string) *testOutbound {
	return &testOutbound{
		Adapter: outbound.NewAdapter("test", tag, []string{N.NetworkTCP, N.NetworkUDP}, nil),
	}
}

func newTestGroup(outbounds []adapter.Outbound, limit uint32) *URLTestGroup {
	history := urltest.NewHistoryStorage()
	return &URLTestGroup{
		ctx:                     context.Background(),
		logger:                  logger.NOP(),
		outbounds:               outbounds,
		tolerance:               50,
		history:                 history,
		interruptGroup:          interrupt.NewGroup(),
		consecutiveFailureLimit: limit,
		close:                   make(chan struct{}),
	}
}

func TestTrackFailure_Disabled(t *testing.T) {
	ob := newTestOutbound("test-a")
	g := newTestGroup([]adapter.Outbound{ob}, 0)

	g.trackFailure(ob)
	g.trackFailure(ob)
	g.trackFailure(ob)

	require.Equal(t, int32(0), g.consecutiveFailures.Load())
}

func TestTrackFailure_BelowLimit(t *testing.T) {
	ob := newTestOutbound("test-a")
	g := newTestGroup([]adapter.Outbound{ob}, 3)

	g.trackFailure(ob)
	require.Equal(t, int32(1), g.consecutiveFailures.Load())

	g.trackFailure(ob)
	require.Equal(t, int32(2), g.consecutiveFailures.Load())
}

func TestTrackFailure_ReachesLimit(t *testing.T) {
	obA := newTestOutbound("test-a")
	obB := newTestOutbound("test-b")
	g := newTestGroup([]adapter.Outbound{obA, obB}, 3)

	// Set obA as selected and give obB a better history so performUpdateCheck switches
	g.selectedOutboundTCP = obA
	g.selectedOutboundUDP = obA
	g.history.StoreURLTestHistory("test-b", &adapter.URLTestHistory{
		Time:  time.Now(),
		Delay: 100,
	})

	g.trackFailure(obA)
	g.trackFailure(obA)
	// Third failure reaches limit=3, triggers performUpdateCheck and resets counter
	g.trackFailure(obA)

	require.Equal(t, int32(0), g.consecutiveFailures.Load())
	// performUpdateCheck should have switched to obB
	require.Equal(t, obB, g.selectedOutboundTCP)
	require.Equal(t, obB, g.selectedOutboundUDP)
}

func TestTrackFailure_ReachesLimit_NoSwitch(t *testing.T) {
	obA := newTestOutbound("test-a")
	g := newTestGroup([]adapter.Outbound{obA}, 2)

	// obA is selected and is the only outbound; performUpdateCheck won't change selection
	g.selectedOutboundTCP = obA
	g.selectedOutboundUDP = obA
	g.history.StoreURLTestHistory("test-a", &adapter.URLTestHistory{
		Time:  time.Now(),
		Delay: 100,
	})

	g.trackFailure(obA)
	g.trackFailure(obA)

	// Counter still resets to 0 after reaching limit
	require.Equal(t, int32(0), g.consecutiveFailures.Load())
	// Selection unchanged
	require.Equal(t, obA, g.selectedOutboundTCP)
}

func TestPerformUpdateCheck_ResetsCounterOnSwitch(t *testing.T) {
	obA := newTestOutbound("test-a")
	obB := newTestOutbound("test-b")
	g := newTestGroup([]adapter.Outbound{obA, obB}, 5)

	g.selectedOutboundTCP = obA
	g.selectedOutboundUDP = obA
	g.consecutiveFailures.Store(4)

	// Give obB a better delay so it gets selected
	g.history.StoreURLTestHistory("test-b", &adapter.URLTestHistory{
		Time:  time.Now(),
		Delay: 50,
	})
	// Remove obA's history so it won't be selected
	g.history.DeleteURLTestHistory("test-a")

	g.performUpdateCheck()

	require.Equal(t, int32(0), g.consecutiveFailures.Load())
	require.Equal(t, obB, g.selectedOutboundTCP)
	require.Equal(t, obB, g.selectedOutboundUDP)
}

func TestPerformUpdateCheck_NoResetWhenNoSwitch(t *testing.T) {
	obA := newTestOutbound("test-a")
	g := newTestGroup([]adapter.Outbound{obA}, 5)

	g.selectedOutboundTCP = obA
	g.selectedOutboundUDP = obA
	g.consecutiveFailures.Store(3)

	g.history.StoreURLTestHistory("test-a", &adapter.URLTestHistory{
		Time:  time.Now(),
		Delay: 100,
	})

	g.performUpdateCheck()

	// Counter should NOT be reset since no switch occurred
	require.Equal(t, int32(3), g.consecutiveFailures.Load())
	require.Equal(t, obA, g.selectedOutboundTCP)
}

func TestConsecutiveFailures_ResetOnSuccess(t *testing.T) {
	obA := newTestOutbound("test-a")
	g := newTestGroup([]adapter.Outbound{obA}, 5)

	g.consecutiveFailures.Store(3)

	// Simulate what DialContext/ListenPacket do on success
	g.consecutiveFailures.Store(0)

	require.Equal(t, int32(0), g.consecutiveFailures.Load())
}
