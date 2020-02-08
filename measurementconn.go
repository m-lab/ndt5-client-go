package ndt5

import (
	"context"
	"net"
)

type measurementconnBinaryFactory struct{}

func (bccf *measurementconnBinaryFactory) DialContext(ctx context.Context, address string) (MeasurementConn, error) {
	conn, err := new(net.Dialer).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
