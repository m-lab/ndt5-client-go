package ndt5

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type protocolNDT5Factory struct{}

func (p *protocolNDT5Factory) NewProtocol(cc ControlConn) Protocol {
	return &protocolNDT5{cc: cc}
}

type protocolNDT5 struct {
	cc ControlConn
}

func (p *protocolNDT5) SendLogin() error {
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

func (p *protocolNDT5) ReceiveKickoff() error {
	received := make([]byte, len(kickoffMessage))
	if err := p.cc.ReadKickoffMessage(received); err != nil {
		return err
	}
	if !bytes.Equal(kickoffMessage, received) {
		return ErrInvalidKickoff
	}
	return nil
}

func (p *protocolNDT5) WaitInQueue() error {
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

func (p *protocolNDT5) ReceiveVersion() (string, error) {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return "", err
	}
	if frame.Type != msgLogin {
		return "", fmt.Errorf("ReceiveVersion: %w", ErrUnexpectedMessage)
	}
	return string(frame.Message), nil
}

func (p *protocolNDT5) ReceiveTestIDs() ([]uint8, error) {
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

func (p *protocolNDT5) ExpectTestPrepare() (port string, err error) {
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

func (p *protocolNDT5) ExpectTestStart() error {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return err
	}
	if frame.Type != msgTestStart {
		return fmt.Errorf("ExpectTestStart: %w", ErrUnexpectedMessage)
	}
	return nil
}

func (p *protocolNDT5) ExpectTestMsg() (string, error) {
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

func (p *protocolNDT5) ExpectTestFinalize() error {
	frame, err := p.cc.ReadFrame()
	if err != nil {
		return err
	}
	if frame.Type != msgTestFinalize {
		return fmt.Errorf("ExpectTestFinalize: %w", ErrUnexpectedMessage)
	}
	return nil
}

func (p *protocolNDT5) SendTestMsg(data []byte) error {
	return p.cc.WriteMessage(msgTestMsg, data)
}

func (p *protocolNDT5) ReceiveTestFinalizeOrTestMsg() (uint8, []byte, error) {
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

func (p *protocolNDT5) ReceiveLogoutOrResults() (uint8, []byte, error) {
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
