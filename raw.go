package ndt5

import (
	"context"
	"encoding/binary"
	"net"
	"time"
)

// RawConnectionsFactory creates ndt5 connections
type RawConnectionsFactory struct {
	dialer *net.Dialer
}

// NewRawConnectionsFactory creates a factory for ndt5 connections
func NewRawConnectionsFactory() *RawConnectionsFactory {
	return &RawConnectionsFactory{
		dialer: new(net.Dialer),
	}
}

// DialControlConn implements ConnectionsFactory.DialControlConn
func (cf *RawConnectionsFactory) DialControlConn(
	ctx context.Context, address, userAgent string) (ControlConn, error) {
	_, _, err := net.SplitHostPort(address)
	if err != nil {
		address = net.JoinHostPort(address, "3001")
	}
	return cf.dialControlConn(ctx, address)
}

func (cf *RawConnectionsFactory) dialControlConn(
	ctx context.Context, address string) (ControlConn, error) {
	conn, err := cf.dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return &rawControlConn{
		conn:     conn,
		observer: new(defaultFrameReadWriteObserver),
	}, nil
}

// DialMeasurementConn implements ConnectionsFactory.DialMeasurementConn.
func (cf *RawConnectionsFactory) DialMeasurementConn(
	ctx context.Context, address, userAgent string) (MeasurementConn, error) {
	conn, err := cf.dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return &rawMeasurementConn{conn: conn}, nil
}

type rawControlConn struct {
	conn     net.Conn
	observer FrameReadWriteObserver
}

func (cc *rawControlConn) SetFrameReadWriteObserver(observer FrameReadWriteObserver) {
	cc.observer = observer
}

func (cc *rawControlConn) SetDeadline(deadline time.Time) error {
	return cc.conn.SetDeadline(deadline)
}

func (cc *rawControlConn) WriteLogin(versionCompat string, testSuite byte) error {
	// Note that versionCompat is ignored with the legacy login message
	return cc.WriteMessage(msgLogin, []byte{testSuite})
}

func (cc *rawControlConn) ReadKickoffMessage(b []byte) error {
	return cc.readn(b)
}

func (cc *rawControlConn) ReadFrame() (*Frame, error) {
	// <type: uint8> <length: uint16> <message: [0..65536]byte>
	b := make([]byte, maxFrameSize)
	if err := cc.readn(b[:1]); err != nil {
		return nil, err
	}
	if err := cc.readn(b[1:3]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint16(b[1:3]) + 3
	if err := cc.readn(b[3:size]); err != nil {
		return nil, err
	}
	frame := &Frame{
		Message: b[3:size],
		Raw:     b[:size],
		Type:    b[0],
	}
	cc.observer.OnRead(frame)
	return frame, nil
}

func (cc *rawControlConn) WriteMessage(mtype uint8, data []byte) error {
	frame, err := NewFrame(mtype, data)
	if err != nil {
		return err
	}
	return cc.WriteFrame(frame)
}

func (cc *rawControlConn) WriteFrame(frame *Frame) error {
	cc.observer.OnWrite(frame)
	_, err := cc.conn.Write(frame.Raw)
	return err
}

func (cc *rawControlConn) readn(data []byte) error {
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

type rawMeasurementConn struct {
	conn     net.Conn
	prepared []byte
	rbuf     []byte
}

func (mc *rawMeasurementConn) SetDeadline(deadline time.Time) error {
	return mc.conn.SetDeadline(deadline)
}

func (mc *rawMeasurementConn) AllocReadBuffer(bufsiz int) {
	mc.rbuf = make([]byte, bufsiz)
}

func (mc *rawMeasurementConn) ReadDiscard() (int64, error) {
	// We assume the read buffer has been initialized
	count, err := mc.conn.Read(mc.rbuf)
	return int64(count), err
}

func (mc *rawMeasurementConn) SetPreparedMessage(b []byte) {
	mc.prepared = b
}

func (mc *rawMeasurementConn) WritePreparedMessage() (int, error) {
	// We assume the prepared message has been initialized
	return mc.conn.Write(mc.prepared)
}

func (mc *rawMeasurementConn) Close() error {
	return mc.conn.Close()
}
