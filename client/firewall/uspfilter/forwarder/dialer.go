package forwarder

import (
	"context"
	"net"
	"net/netip"
	"strconv"

	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// RoutedDialFunc opens the outbound connection of a routed flow instead of a
// host socket to the original destination. src is the peer that opened the
// flow and dst is the address and port it dialled.
type RoutedDialFunc func(ctx context.Context, network string, src, dst netip.AddrPort) (net.Conn, error)

// SetDialer sets the dialer for routed flows. A nil dialer restores host
// sockets to the original destination.
func (f *Forwarder) SetDialer(dial RoutedDialFunc) {
	if dial == nil {
		f.dialer.Store(nil)
		return
	}
	f.dialer.Store(&dial)
}

func (f *Forwarder) dialRouted(network string, id stack.TransportEndpointID) (net.Conn, error) {
	if dial := f.dialer.Load(); dial != nil {
		src := netip.AddrPortFrom(addrToNetipAddr(id.RemoteAddress), id.RemotePort)
		dst := netip.AddrPortFrom(addrToNetipAddr(id.LocalAddress), id.LocalPort)
		return (*dial)(f.ctx, network, src, dst)
	}

	addr := net.JoinHostPort(f.determineDialAddr(id.LocalAddress).String(), strconv.Itoa(int(id.LocalPort)))
	return (&net.Dialer{}).DialContext(f.ctx, network, addr)
}
