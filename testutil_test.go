package ndt5_test

import (
	"context"
	"errors"
	"net"
)

const UserAgent = "ndt5-client-go-testing/0.1.0"

var ErrMocked = errors.New("mocked error")

type RecordParametersDialer struct {
	Address string
	Network string
}

func (d *RecordParametersDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *RecordParametersDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	d.Network = network
	d.Address = address
	conn, _ := net.Pipe()
	return conn, nil
}

type AlwaysFailingDialer struct{}

func (d *AlwaysFailingDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *AlwaysFailingDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	return nil, ErrMocked
}

type PipeDialer struct {
	ServerConn net.Conn
	ClientConn net.Conn
}

func NewPipeDialer() *PipeDialer {
	d := new(PipeDialer)
	d.ClientConn, d.ServerConn = net.Pipe()
	return d
}

func (d *PipeDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *PipeDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	return d.ClientConn, nil
}
