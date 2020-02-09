package ndt5

import (
	"context"
	"encoding/binary"
	"net"
	"time"
)

type rawConnectionsFactory struct {
	dialer *net.Dialer
}

func newRawConnectionsFactory() *rawConnectionsFactory {
	return &rawConnectionsFactory{
		dialer: new(net.Dialer),
	}
}

func (rcf *rawConnectionsFactory) DialControlConn(ctx context.Context, address string) (ControlConn, error) {
	_, _, err := net.SplitHostPort(address)
	if err != nil {
		address = net.JoinHostPort(address, "3001")
	}
	return rcf.dialControlConn(ctx, address)
}

func (rcf *rawConnectionsFactory) dialControlConn(ctx context.Context, address string) (ControlConn, error) {
	conn, err := rcf.dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return &rawControlConn{conn: conn}, nil
}

func (rcf *rawConnectionsFactory) DialMeasurementConn(ctx context.Context, address string) (MeasurementConn, error) {
	conn, err := rcf.dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

type rawControlConn struct {
	conn net.Conn
}

func (cc *rawControlConn) SetDeadline(deadline time.Time) error {
	return cc.conn.SetDeadline(deadline)
}

func (cc *rawControlConn) ReadFrame() (*Frame, error) {
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
	return &Frame{
		Message: b[3:size],
		Raw:     b[:size],
		Type:    b[0],
	}, nil
}

func (cc *rawControlConn) WriteMessage(mtype uint8, data []byte) error {
	frame, err := NewFrame(mtype, data)
	if err != nil {
		return err
	}
	return cc.WriteFrame(frame)
}

func (cc *rawControlConn) WriteFrame(frame *Frame) error {
	_, err := cc.conn.Write(frame.Raw)
	return err
}

func (cc *rawControlConn) Readn(data []byte) error {
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

func (cc *rawControlConn) Close() error {
	return cc.conn.Close()
}
