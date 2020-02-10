package ndt5_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/m-lab/ndt5-client-go"
)

func TestUnitProtocolReceiveKickoffReadKickoffError(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	err := proto.ReceiveKickoff()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
}

func TestUnitProtocolReceiveKickoffReadKickoffWrongBytes(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		dialer.ServerConn.Write([]byte("654321 123456"))
		wg.Done()
	}()
	err := proto.ReceiveKickoff()
	if !errors.Is(err, ndt5.ErrInvalidKickoff) {
		t.Fatal("expected ndt5.ErrInvalidKickoff here")
	}
	wg.Wait()
}

func TestUnitProtocolWaitInQueueReadFrameFailure(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	err := proto.WaitInQueue()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
}

func TestUnitProtocolWaitInQueueWrongMessageType(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(2, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	err := proto.WaitInQueue()
	if !errors.Is(err, ndt5.ErrUnexpectedMessage) {
		t.Fatal("expected ndt5.ErrUnexpectedMessage here")
	}
	wg.Wait()
}

func TestUnitProtocolWaitInQueueServerBusy(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(1, []byte("9999"))
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	err := proto.WaitInQueue()
	if !errors.Is(err, ndt5.ErrServerBusy) {
		t.Fatal("expected ndt5.ErrServerBusy here")
	}
	wg.Wait()
}

func TestUnitProtocolReceiveVersionReadFrameFailure(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	version, err := proto.ReceiveVersion()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
	if version != "" {
		t.Fatal("expected empty version here")
	}
}

func TestUnitProtocolReceiveVersionWrongMessageType(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(1, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	version, err := proto.ReceiveVersion()
	if !errors.Is(err, ndt5.ErrUnexpectedMessage) {
		t.Fatal("expected ndt5.ErrUnexpectedMessage here")
	}
	if version != "" {
		t.Fatal("expected empty version here")
	}
	wg.Wait()
}

func TestUnitProtocolReceiveTestsIDsReadFrameFailure(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	testIDs, err := proto.ReceiveTestIDs()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
	if testIDs != nil {
		t.Fatal("expected nil testIDs here")
	}
}

func TestUnitProtocolReceiveTestIDsWrongMessageType(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(1, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	testIDs, err := proto.ReceiveTestIDs()
	if !errors.Is(err, ndt5.ErrUnexpectedMessage) {
		t.Fatal("expected ndt5.ErrUnexpectedMessage here")
	}
	if testIDs != nil {
		t.Fatal("expected nil testIDs here")
	}
	wg.Wait()
}

func TestUnitProtocolReceiveTestIDsWithNoIDs(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(2, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	testIDs, err := proto.ReceiveTestIDs()
	if err != nil {
		t.Fatal(err)
	}
	if testIDs != nil {
		t.Fatal("expected nil testIDs here")
	}
	wg.Wait()
}

func TestUnitProtocolReceiveTestIDsParseUintError(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(2, []byte("xx"))
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	testIDs, err := proto.ReceiveTestIDs()
	// Apparently the error returned here is not wrapped in a way
	// in which we can extract it using errors.Is.
	if !strings.HasSuffix(err.Error(), "invalid syntax") {
		t.Fatal("expected strconv.ErrSyntax here")
	}
	if testIDs != nil {
		t.Fatal("expected nil testIDs here")
	}
	wg.Wait()
}

func TestUnitProtocolExpectTestPrepareReadFrameFailure(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	portnum, err := proto.ExpectTestPrepare()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
	if portnum != "" {
		t.Fatal("expected empty portnum here")
	}
}

func TestUnitProtocolExpectTestPrepareWrongMessageType(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(1, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	portnum, err := proto.ExpectTestPrepare()
	if !errors.Is(err, ndt5.ErrUnexpectedMessage) {
		t.Fatal("expected ndt5.ErrUnexpectedMessage here")
	}
	if portnum != "" {
		t.Fatal("expected empty portnum here")
	}
	wg.Wait()
}

func TestUnitProtocolExpectTestStartReadFrameFailure(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	err := proto.ExpectTestStart()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
}

func TestUnitProtocolExpectTestStartWrongMessageType(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(1, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	err := proto.ExpectTestStart()
	if !errors.Is(err, ndt5.ErrUnexpectedMessage) {
		t.Fatal("expected ndt5.ErrUnexpectedMessage here")
	}
	wg.Wait()
}

func TestUnitProtocolExpectTestMsgReadFrameFailure(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	msg, err := proto.ExpectTestMsg()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
	if msg != "" {
		t.Fatal("expected empty msg here")
	}
}

func TestUnitProtocolExpectTestMsgWrongMessageType(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(1, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	msg, err := proto.ExpectTestMsg()
	if !errors.Is(err, ndt5.ErrUnexpectedMessage) {
		t.Fatal("expected ndt5.ErrUnexpectedMessage here")
	}
	if msg != "" {
		t.Fatal("expected empty msg here")
	}
	wg.Wait()
}

func TestUnitProtocolExpectTestMsgEmptyMsg(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(5, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	msg, err := proto.ExpectTestMsg()
	if !errors.Is(err, ndt5.ErrExpectedNonEmptyMessage) {
		t.Fatal("expected ndt5.ErrExpectedNonEmptyMessage here")
	}
	if msg != "" {
		t.Fatal("expected empty msg here")
	}
	wg.Wait()
}

func TestUnitProtocolExpectTestFinalizeReadFrameFailure(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	err := proto.ExpectTestFinalize()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
}

func TestUnitProtocolExpectTestFinalizeWrongMessageType(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(1, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	err := proto.ExpectTestFinalize()
	if !errors.Is(err, ndt5.ErrUnexpectedMessage) {
		t.Fatal("expected ndt5.ErrUnexpectedMessage here")
	}
	wg.Wait()
}

func TestUnitProtocolReceiveTestFinalizeOrTestMsgReadFrameFailure(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	_, _, err := proto.ReceiveTestFinalizeOrTestMsg()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
}

func TestUnitProtocolReceiveTestFinalizeOrTestMsgWrongMessageType(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(1, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	_, _, err := proto.ReceiveTestFinalizeOrTestMsg()
	if !errors.Is(err, ndt5.ErrUnexpectedMessage) {
		t.Fatal("expected ndt5.ErrUnexpectedMessage here")
	}
	wg.Wait()
}

func TestUnitProtocolReceiveLogoutOrResultsReadFrameFailure(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	dialer.ServerConn.Close()
	_, _, err := proto.ReceiveLogoutOrResults()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
}

func TestUnitProtocolReceiveLogoutOrResultsWrongMessageType(t *testing.T) {
	dialer, proto := NewMockableProtocol(t)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		frame, _ := ndt5.NewFrame(1, nil)
		dialer.ServerConn.Write(frame.Raw)
		wg.Done()
	}()
	_, _, err := proto.ReceiveLogoutOrResults()
	if !errors.Is(err, ndt5.ErrUnexpectedMessage) {
		t.Fatal("expected ndt5.ErrUnexpectedMessage here")
	}
	wg.Wait()
}

func NewMockableProtocol(t *testing.T) (*PipeDialer, ndt5.Protocol) {
	dialer := NewPipeDialer()
	connfactory := ndt5.NewRawConnectionsFactory(dialer)
	protofactory := ndt5.NewProtocolFactory5()
	protofactory.ConnectionsFactory = connfactory
	ch := make(chan *ndt5.Output, 1) // buffer for connected message
	proto, err := protofactory.NewProtocol(
		context.Background(), "127.0.0.1", UserAgent, ch)
	if err != nil {
		t.Fatal(err)
	}
	return dialer, proto
}
