package ndt5

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

// ProtocolFactory5 is a factory that creates the ndt5 protocol
type ProtocolFactory5 struct {
	// ConnectionsFactory creates connections. It's set to its
	// default value by NewClient; you may override it.
	//
	// By changing this field before starting the experiment
	// you can choose the specific transport you want.
	//
	// The default transport is the raw TCP transport that was
	// initially introduced with the NDT codebase.
	ConnectionsFactory ConnectionsFactory

	// ObserverFactory allows you to observe frame events. It's set to its
	// default value by NewClient; you may override it.
	ObserverFactory FrameReadWriteObserverFactory
}

// NewProtocolFactory5 creates a new ProtocolFactory5 instance
func NewProtocolFactory5() *ProtocolFactory5 {
	return &ProtocolFactory5{
		ConnectionsFactory: NewRawConnectionsFactory(new(net.Dialer)),
		ObserverFactory:    new(defaultFrameReadWriteObserverFactory),
	}
}

// NewProtocol implements ProtocolFactory.NewProtocol
func (p *ProtocolFactory5) NewProtocol(
	ctx context.Context, fqdn, userAgent string, ch chan<- *Output) (Protocol, error) {
	cc, err := p.ConnectionsFactory.DialControlConn(ctx, fqdn, userAgent)
	if err != nil {
		return nil, err
	}
	cc.SetFrameReadWriteObserver(p.ObserverFactory.New(ch))
	if err := cc.SetDeadline(time.Now().Add(45 * time.Second)); err != nil {
		return nil, fmt.Errorf("cannot set control connection deadline: %w", err)
	}
	return &protocol5{cc: cc, connectionsFactory: p.ConnectionsFactory}, nil
}

type protocol5 struct {
	cc                 ControlConn
	connectionsFactory ConnectionsFactory
}

func (p *protocol5) SendLogin() error {
	const ndt5VersionCompat = "v3.7.0"
	flags := nettestUpload | nettestDownload | nettestStatus
	return p.cc.WriteLogin(ndt5VersionCompat, flags)
}

var (
	// ErrExpectedNonEmptyMessage indicates we expected a message
	// with no body and we received one with body.
	ErrExpectedNonEmptyMessage = errors.New("expected non-empty message")

	// ErrInvalidKickoff indicates that the kickoff message we did
	// receive from the server isn't what we expected
	ErrInvalidKickoff = errors.New("ReceiveKickoff: get invalid kickoff bytes")

	// ErrServerBusy indicates that the server is busy
	ErrServerBusy = errors.New("WaitInQueue: server is busy")

	// ErrUnexpectedMessage indicates we received a message that
	// we were not expecting at this stage.
	ErrUnexpectedMessage = errors.New("unexpected message type")

	kickoffMessage = []byte("123456 654321")
)

func (p *protocol5) ReceiveKickoff() error {
	received := make([]byte, len(kickoffMessage))
	if err := p.cc.ReadKickoffMessage(received); err != nil {
		return err
	}
	if !bytes.Equal(kickoffMessage, received) {
		return ErrInvalidKickoff
	}
	return nil
}

func (p *protocol5) WaitInQueue() error {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return err
	}
	if frame.Type != msgSrvQueue {
		return fmt.Errorf("WaitInQueue: %w", ErrUnexpectedMessage)
	}
	if !bytes.Equal(frame.Message, []byte("0")) {
		// Like libndt, we have chosen not to wait in queue here
		return ErrServerBusy
	}
	return nil
}

func (p *protocol5) ReceiveVersion() (string, error) {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return "", err
	}
	if frame.Type != msgLogin {
		return "", fmt.Errorf("ReceiveVersion: %w", ErrUnexpectedMessage)
	}
	return string(frame.Message), nil
}

func (p *protocol5) ReceiveTestIDs() ([]uint8, error) {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return nil, err
	}
	if frame.Type != msgLogin {
		return nil, fmt.Errorf("ReceiveTestIDsList: %w", ErrUnexpectedMessage)
	}
	if len(frame.Message) == 0 {
		return nil, nil // happends when test suite contains nettestStatus only
	}
	elems := bytes.Split(frame.Message, []byte(" "))
	var testIDs []uint8
	for _, elem := range elems {
		val, err := strconv.ParseUint(string(elem), 10, 8)
		if err != nil {
			return nil, err
		}
		testIDs = append(testIDs, uint8(val))
	}
	return testIDs, nil
}

func (p *protocol5) ExpectTestPrepare() (port string, err error) {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return
	}
	if frame.Type != msgTestPrepare {
		err = fmt.Errorf("ExpectTestPrepare: %w", ErrUnexpectedMessage)
		return
	}
	port = string(frame.Message)
	return
}

func (p *protocol5) DialDownloadConn(
	ctx context.Context, address, userAgent string,
) (MeasurementConn, error) {
	return p.connectionsFactory.DialMeasurementConn(ctx, address, userAgent)
}

func (p *protocol5) DialUploadConn(
	ctx context.Context, address, userAgent string,
) (MeasurementConn, error) {
	return p.connectionsFactory.DialMeasurementConn(ctx, address, userAgent)
}

func (p *protocol5) ExpectTestStart() error {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return err
	}
	if frame.Type != msgTestStart {
		return fmt.Errorf("ExpectTestStart: %w", ErrUnexpectedMessage)
	}
	return nil
}

func (p *protocol5) ExpectTestMsg() (string, error) {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return "", err
	}
	if frame.Type != msgTestMsg {
		return "", fmt.Errorf("ExpectTestMsg: %w", ErrUnexpectedMessage)
	}
	if len(frame.Message) == 0 {
		return "", fmt.Errorf("ExpectTestMsg: %w", ErrExpectedNonEmptyMessage)
	}
	return string(frame.Message), nil
}

func (p *protocol5) ExpectTestFinalize() error {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return err
	}
	if frame.Type != msgTestFinalize {
		return fmt.Errorf("ExpectTestFinalize: %w", ErrUnexpectedMessage)
	}
	return nil
}

func (p *protocol5) SendTestMsg(data []byte) error {
	return p.cc.WriteMessage(msgTestMsg, data)
}

func (p *protocol5) ReceiveTestFinalizeOrTestMsg() (uint8, []byte, error) {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return 0, nil, err
	}
	if frame.Type == msgTestFinalize {
		return msgTestFinalize, nil, nil
	}
	if frame.Type != msgTestMsg {
		err = fmt.Errorf("ReceiveLogoutOrTestMsg: %w", ErrUnexpectedMessage)
		return 0, nil, err
	}
	return msgTestMsg, frame.Message, nil
}

func (p *protocol5) ReceiveLogoutOrResults() (uint8, []byte, error) {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return 0, nil, err
	}
	if frame.Type == msgLogout {
		return msgLogout, nil, nil
	}
	if frame.Type != msgResults {
		err = fmt.Errorf("ReceiveLogoutOrTestMsg: %w", ErrUnexpectedMessage)
		return 0, nil, err
	}
	return msgResults, frame.Message, nil
}

func (p *protocol5) Close() error {
	return p.cc.Close()
}
