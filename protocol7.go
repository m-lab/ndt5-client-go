package ndt5

import (
	"context"
	"net"
)

// ProtocolFactory7 is a factory that creates the ndt7 protocol
type ProtocolFactory7 struct {
	ConnectionsFactory *WSConnectionsFactory
}

// NewProtocolFactory7 creates a new ProtocolFactory7 instance
func NewProtocolFactory7() *ProtocolFactory7 {
	return &ProtocolFactory7{
		ConnectionsFactory: NewWSConnectionsFactory(new(net.Dialer)),
	}
}

// NewProtocol implements ProtocolFactory.NewProtocol
func (p *ProtocolFactory7) NewProtocol(
	ctx context.Context, fqdn, userAgent string, ch chan<- *Output,
) (Protocol, error) {
	return &protocol7{factory: p.ConnectionsFactory}, nil
}

type protocol7 struct {
	factory *WSConnectionsFactory
}

func (p *protocol7) SendLogin() error {
	return nil
}

func (p *protocol7) ReceiveKickoff() error {
	return nil
}

func (p *protocol7) WaitInQueue() error {
	return nil
}

func (p *protocol7) ReceiveVersion() (string, error) {
	return "v5.0-NDTinGo", nil
}

func (p *protocol7) ReceiveTestIDs() ([]uint8, error) {
	return []uint8{nettestUpload, nettestDownload}, nil
}

func (p *protocol7) ExpectTestPrepare() (string, error) {
	return "443", nil
}

func (p *protocol7) DialDownloadConn(
	ctx context.Context, address, userAgent string,
) (MeasurementConn, error) {
	conn, err := p.factory.DialEx(
		ctx, address, "/ndt/v7/download", userAgent,
		"net.measurementlab.ndt.v7",
	)
	if err != nil {
		return nil, err
	}
	return &wsMeasurementConn{conn: conn}, nil
}

func (p *protocol7) DialUploadConn(
	ctx context.Context, address, userAgent string,
) (MeasurementConn, error) {
	conn, err := p.factory.DialEx(
		ctx, address, "/ndt/v7/upload", userAgent,
		"net.measurementlab.ndt.v7",
	)
	if err != nil {
		return nil, err
	}
	return &wsMeasurementConn{conn: conn}, nil
}

func (p *protocol7) ExpectTestStart() error {
	return nil
}

func (p *protocol7) ExpectTestMsg() (string, error) {
	return "", nil
}

func (p *protocol7) ExpectTestFinalize() error {
	return nil
}

func (p *protocol7) SendTestMsg(data []byte) error {
	return nil
}

func (p *protocol7) ReceiveTestFinalizeOrTestMsg() (uint8, []byte, error) {
	return msgTestFinalize, nil, nil
}

func (p *protocol7) ReceiveLogoutOrResults() (uint8, []byte, error) {
	return msgLogout, nil, nil
}

func (p *protocol7) Close() error {
	return nil
}
