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
	body := make([]byte, 1)
	body[0] = nettestUpload | nettestDownload | nettestStatus
	return p.cc.WriteMessage(msgLogin, body)
}

func (p *protocolNDT5) ReceiveKickoff() error {
	desired := []byte("123456 654321")
	received := make([]byte, len(desired))
	if err := p.cc.Readn(received); err != nil {
		return err
	}
	if !bytes.Equal(desired, received) {
		return errors.New("ReceiveKickoff: got invalid kickoff")
	}
	return nil
}

func (p *protocolNDT5) WaitInQueue() error {
	mtype, mdata, err := p.cc.ReadMessage()
	if err != nil {
		return err
	}
	if mtype != msgSrvQueue {
		return errors.New("WaitInQueue: unexpected message type")
	}
	if !bytes.Equal(mdata, []byte("0")) {
		// Like libndt, we have chosen not to wait in queue here
		return errors.New("WaitInQueue: server is busy")
	}
	return nil
}

func (p *protocolNDT5) ReceiveVersion() (string, error) {
	mtype, mdata, err := p.cc.ReadMessage()
	if err != nil {
		return "", err
	}
	if mtype != msgLogin {
		return "", errors.New("ReceiveVersion: unexpected message type")
	}
	return string(mdata), nil
}

func (p *protocolNDT5) ReceiveTestIDs() ([]uint8, error) {
	mtype, mdata, err := p.cc.ReadMessage()
	if err != nil {
		return nil, err
	}
	if mtype != msgLogin {
		return nil, errors.New("ReceiveTestIDsList: unexpected message type")
	}
	if len(mdata) == 0 {
		return nil, nil // happends when test suite contains nettestStatus only
	}
	elems := bytes.Split(mdata, []byte(" "))
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
	var (
		mtype uint8
		mdata []byte
	)
	mtype, mdata, err = p.cc.ReadMessage()
	if err != nil {
		return
	}
	if mtype != msgTestPrepare {
		err = fmt.Errorf("ExpectTestPrepare: invalid message type: %d", int(mtype))
		return
	}
	port = string(mdata)
	return
}

func (p *protocolNDT5) ExpectTestStart() error {
	mtype, mdata, err := p.cc.ReadMessage()
	if err != nil {
		return err
	}
	if mtype != msgTestStart {
		return fmt.Errorf("ExpectTestStart: invalid message type: %d", int(mtype))
	}
	if len(mdata) != 0 {
		return errors.New("ExpectTestStart: expected empty message")
	}
	return nil
}

func (p *protocolNDT5) ExpectTestMsg() (string, error) {
	mtype, mdata, err := p.cc.ReadMessage()
	if err != nil {
		return "", err
	}
	if mtype != msgTestMsg {
		return "", fmt.Errorf("ExpectTestMsg: invalid message type: %d", int(mtype))
	}
	if len(mdata) == 0 {
		return "", errors.New("ExpectTestMsg: expected nonempty message")
	}
	return string(mdata), nil
}

func (p *protocolNDT5) ExpectTestFinalize() error {
	mtype, mdata, err := p.cc.ReadMessage()
	if err != nil {
		return err
	}
	if mtype != msgTestFinalize {
		return fmt.Errorf("ExpectTestFinalize: invalid message type: %d", int(mtype))
	}
	if len(mdata) != 0 {
		return errors.New("ExpectTestFinalize: expected empty message")
	}
	return nil
}

func (p *protocolNDT5) SendTestMsg(data []byte) error {
	return p.cc.WriteMessage(msgTestMsg, data)
}

func (p *protocolNDT5) ReceiveTestFinalizeOrTestMsg() (mtype uint8, mdata []byte, err error) {
	mtype, mdata, err = p.cc.ReadMessage()
	if err != nil {
		return
	}
	if mtype == msgTestFinalize {
		return
	}
	if mtype != msgTestMsg {
		err = fmt.Errorf("ReceiveLogoutOrTestMsg: invalid message type: %d", int(mtype))
		return
	}
	return
}

func (p *protocolNDT5) ReceiveLogoutOrResults() (mtype uint8, mdata []byte, err error) {
	mtype, mdata, err = p.cc.ReadMessage()
	if err != nil {
		return
	}
	if mtype == msgLogout {
		return
	}
	if mtype != msgResults {
		err = fmt.Errorf("ReceiveLogoutOrTestMsg: invalid message type: %d", int(mtype))
		return
	}
	return
}
