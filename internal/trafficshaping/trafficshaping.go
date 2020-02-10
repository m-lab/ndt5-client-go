// Package trafficshaping contains code to perform traffic shaping.
package trafficshaping

import (
	"context"
	"net"

	"github.com/google/martian/v3/trafficshape"
)

// Dialer is a dialer performing shaping.
type Dialer struct {
	bitrate int64
	dialer  *net.Dialer
}

// NewDialerWithBitrate returns a new dialer with the specified throttled bitrate.
func NewDialerWithBitrate(bitrate int64) *Dialer {
	return &Dialer{bitrate: bitrate, dialer: new(net.Dialer)}
}

// NewDialer returns a new dialer with the default throttled bitrate.
func NewDialer() *Dialer {
	return NewDialerWithBitrate(1 << 20)
}

// Dial dials a shaped network connection.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext is like Dial but with a context.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	listener := trafficshape.NewListener(new(net.TCPListener))
	listener.SetReadBitrate(d.bitrate)
	listener.SetWriteBitrate(d.bitrate)
	return listener.GetTrafficShapedConn(conn), nil
}
