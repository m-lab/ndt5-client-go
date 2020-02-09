package ndt5

import (
	"context"
	"net"
)

type measurementconnFactory struct{}

func (bccf *measurementconnFactory) DialContext(ctx context.Context, address string) (MeasurementConn, error) {
	conn, err := new(net.Dialer).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
