package ndt5_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/m-lab/ndt5-client-go"
)

func TestUnitRawDialControlConnDefaultPort(t *testing.T) {
	dialer := new(RecordParametersDialer)
	f := ndt5.NewRawConnectionsFactory(dialer)
	f.DialControlConn(context.Background(), "127.0.0.1", UserAgent)
	if dialer.Address != "127.0.0.1:3001" {
		t.Fatal("unexpected address was dialed")
	}
	f.DialControlConn(context.Background(), "localhost", UserAgent)
	if dialer.Address != "localhost:3001" {
		t.Fatal("unexpected address was dialed")
	}
	f.DialControlConn(context.Background(), "::1", UserAgent)
	if dialer.Address != "[::1]:3001" {
		t.Fatal("unexpected address was dialed")
	}
	f.DialControlConn(context.Background(), "127.0.0.1:54321", UserAgent)
	if dialer.Address != "127.0.0.1:54321" {
		t.Fatal("unexpected address was dialed")
	}
	f.DialControlConn(context.Background(), "localhost:54321", UserAgent)
	if dialer.Address != "localhost:54321" {
		t.Fatal("unexpected address was dialed")
	}
	f.DialControlConn(context.Background(), "[::1]:54321", UserAgent)
	if dialer.Address != "[::1]:54321" {
		t.Fatal("unexpected address was dialed")
	}
}

func TestUnitRawDialControlConnSuccess(t *testing.T) {
	f := ndt5.NewRawConnectionsFactory(NewPipeDialer())
	cc, err := f.DialControlConn(context.Background(), "127.0.0.1:3001", UserAgent)
	if err != nil {
		t.Fatal(err)
	}
	if cc == nil {
		t.Fatal("expected non-nil cc here")
	}
	cc.Close()
	mc, err := f.DialMeasurementConn(context.Background(), "127.0.0.1:9001", UserAgent)
	if err != nil {
		t.Fatal("expected ErrMocked here")
	}
	if mc == nil {
		t.Fatal("expected non-nil mc here")
	}
	mc.Close()
}

func TestUnitRawDialControlConnFailure(t *testing.T) {
	f := ndt5.NewRawConnectionsFactory(new(AlwaysFailingDialer))
	cc, err := f.DialControlConn(context.Background(), "127.0.0.1:3001", UserAgent)
	if !errors.Is(err, ErrMocked) {
		t.Fatal("expected ErrMocked here")
	}
	if cc != nil {
		t.Fatal("expected nil cc here")
	}
	mc, err := f.DialMeasurementConn(context.Background(), "127.0.0.1:9001", UserAgent)
	if !errors.Is(err, ErrMocked) {
		t.Fatal("expected ErrMocked here")
	}
	if mc != nil {
		t.Fatal("expected nil mc here")
	}
}

func TestUnitRawControlConnWriteMessageFailure(t *testing.T) {
	f := ndt5.NewRawConnectionsFactory(new(PipeDialer))
	cc, err := f.DialControlConn(context.Background(), "127.0.0.1:3001", UserAgent)
	if err != nil {
		t.Fatal(err)
	}
	err = cc.WriteMessage(1, make([]byte, 1<<20))
	if !errors.Is(err, ndt5.ErrMessageSize) {
		t.Fatal("expected ndt5.ErrMessageSize here")
	}
}

func TestUnitRawControlConnReadFrameFirstReadnFailure(t *testing.T) {
	dialer := NewPipeDialer()
	f := ndt5.NewRawConnectionsFactory(dialer)
	cc, err := f.DialControlConn(context.Background(), "127.0.0.1:3001", UserAgent)
	if err != nil {
		t.Fatal(err)
	}
	dialer.ServerConn.Close() // should cause first readn to fail
	frame, err := cc.ReadFrame()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
	if frame != nil {
		t.Fatal("expected nil frame here")
	}
}

func TestUnitRawControlConnReadFrameSecondReadnFailure(t *testing.T) {
	dialer := NewPipeDialer()
	f := ndt5.NewRawConnectionsFactory(dialer)
	cc, err := f.DialControlConn(context.Background(), "127.0.0.1:3001", UserAgent)
	if err != nil {
		t.Fatal(err)
	}
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		dialer.ServerConn.Write([]byte{1})
		dialer.ServerConn.Close()
		wg.Done()
	}()
	frame, err := cc.ReadFrame()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
	if frame != nil {
		t.Fatal("expected nil frame here")
	}
	wg.Wait()
}

func TestUnitRawControlConnReadFrameThirdReadnFailure(t *testing.T) {
	dialer := NewPipeDialer()
	f := ndt5.NewRawConnectionsFactory(dialer)
	cc, err := f.DialControlConn(context.Background(), "127.0.0.1:3001", UserAgent)
	if err != nil {
		t.Fatal(err)
	}
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		dialer.ServerConn.Write([]byte{1})
		dialer.ServerConn.Write([]byte{0, 4})
		dialer.ServerConn.Close()
		wg.Done()
	}()
	frame, err := cc.ReadFrame()
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected io.EOF here")
	}
	if frame != nil {
		t.Fatal("expected nil frame here")
	}
	wg.Wait()
}
