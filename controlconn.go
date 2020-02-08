package ndt5

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"net"
	"time"
)

type controlconnFactory struct{}

func (*controlconnFactory) DialContext(ctx context.Context, address string) (ControlConn, error) {
	conn, err := new(net.Dialer).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return &controlconn{conn: conn}, nil
}

type controlconn struct {
	conn net.Conn
}

func (cc *controlconn) SetDeadline(deadline time.Time) error {
	return cc.conn.SetDeadline(deadline)
}

const (
	maxBodySize  = math.MaxUint16
	maxFrameSize = 3 + maxBodySize
)

func (cc *controlconn) ReadFrame() (*IncomingFrame, error) {
	// <type: uint8> <length: uint16> <message: [0..65536]byte>
	b := make([]byte, maxFrameSize)
	if err := cc.Readn(b[:1]); err != nil {
		return nil, err
	}
	if err := cc.Readn(b[1:3]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint16(b[1:3]) + 3
	if err := cc.Readn(b[3:size]); err != nil {
		return nil, err
	}
	return &IncomingFrame{
		Message: b[3:size],
		Raw:     b[:size],
		Type:    b[0],
	}, nil
}

func (cc *controlconn) WriteMessage(mtype uint8, data []byte) error {
	// <type: uint8> <length: uint16> <message: [0..65536]byte>
	b := []byte{mtype}
	if _, err := cc.conn.Write(b); err != nil {
		return err
	}
	if len(data) > math.MaxUint16 {
		return errors.New("msgWriteLegacy: message too long")
	}
	b = make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(len(data)))
	if _, err := cc.conn.Write(b); err != nil {
		return err
	}
	_, err := cc.conn.Write(data)
	return err
}

func (cc *controlconn) Readn(data []byte) error {
	// We don't care too much about performance when reading
	// control messages, hence this simple implementation
	for off := 0; off < len(data); {
		curr := make([]byte, 1)
		if _, err := cc.conn.Read(curr); err != nil {
			return err
		}
		data[off] = curr[0]
		off++
	}
	return nil
}

func (cc *controlconn) Close() error {
	return cc.conn.Close()
}
