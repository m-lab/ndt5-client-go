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

func (bcc *controlconn) SetDeadline(deadline time.Time) error {
	return bcc.conn.SetDeadline(deadline)
}

func (bcc *controlconn) ReadMessage() (mtype uint8, data []byte, err error) {
	// <type: uint8> <length: uint16> <message: [0..65536]byte>
	b := make([]byte, 1)
	if err = bcc.Readn(b); err != nil {
		return
	}
	mtype = b[0]
	b = make([]byte, 2)
	if err = bcc.Readn(b); err != nil {
		return
	}
	length := binary.BigEndian.Uint16(b)
	data = make([]byte, length)
	err = bcc.Readn(data)
	return
}

func (bcc *controlconn) WriteMessage(mtype uint8, data []byte) error {
	// <type: uint8> <length: uint16> <message: [0..65536]byte>
	b := []byte{mtype}
	if _, err := bcc.conn.Write(b); err != nil {
		return err
	}
	if len(data) > math.MaxUint16 {
		return errors.New("msgWriteLegacy: message too long")
	}
	b = make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(len(data)))
	if _, err := bcc.conn.Write(b); err != nil {
		return err
	}
	_, err := bcc.conn.Write(data)
	return err
}

func (bcc *controlconn) Readn(data []byte) error {
	// We don't care too much about performance when reading
	// control messages, hence this simple implementation
	for off := 0; off < len(data); {
		curr := make([]byte, 1)
		if _, err := bcc.conn.Read(curr); err != nil {
			return err
		}
		data[off] = curr[0]
		off++
	}
	return nil
}

func (bcc *controlconn) Close() error {
	return bcc.conn.Close()
}
