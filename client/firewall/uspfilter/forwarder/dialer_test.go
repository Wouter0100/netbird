package forwarder

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

func routedID(src, dst netip.AddrPort) stack.TransportEndpointID {
	return stack.TransportEndpointID{
		RemoteAddress: tcpip.AddrFrom4(src.Addr().As4()),
		RemotePort:    src.Port(),
		LocalAddress:  tcpip.AddrFrom4(dst.Addr().As4()),
		LocalPort:     dst.Port(),
	}
}

func TestDialRoutedUsesDialer(t *testing.T) {
	f := &Forwarder{ctx: context.Background()}
	src := netip.MustParseAddrPort("100.64.0.7:41000")
	dst := netip.MustParseAddrPort("10.250.0.10:5432")

	var gotNetwork string
	var gotSrc, gotDst netip.AddrPort
	refused := errors.New("refused")
	f.SetDialer(func(_ context.Context, network string, s, d netip.AddrPort) (net.Conn, error) {
		gotNetwork, gotSrc, gotDst = network, s, d
		return nil, refused
	})

	_, err := f.dialRouted("tcp", routedID(src, dst))
	require.ErrorIs(t, err, refused)
	assert.Equal(t, "tcp", gotNetwork)
	assert.Equal(t, src, gotSrc)
	assert.Equal(t, dst, gotDst)
}

func TestDialRoutedWithoutDialerUsesHostSocket(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		if c, err := l.Accept(); err == nil {
			_ = c.Close()
		}
	}()

	f := &Forwarder{ctx: context.Background()}
	f.SetDialer(func(context.Context, string, netip.AddrPort, netip.AddrPort) (net.Conn, error) {
		return nil, errors.New("must not be called")
	})
	f.SetDialer(nil)

	dst := netip.MustParseAddrPort(l.Addr().String())
	conn, err := f.dialRouted("tcp", routedID(netip.MustParseAddrPort("100.64.0.7:41000"), dst))
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}
