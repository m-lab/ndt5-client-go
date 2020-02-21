package emitter

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/m-lab/ndt5-client-go/cmd/ndt5-client/internal/mocks"
)

func TestJSONOnDebug(t *testing.T) {
	sw := &mocks.SavingWriter{}
	j := NewJSON(sw)
	err := j.OnDebug("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 1 {
		t.Fatal("invalid length")
	}
	var event struct {
		Key   string
		Value string
	}
	err = json.Unmarshal(sw.Data[0], &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Key != "debug" {
		t.Fatal("Unexpected event key")
	}
	if event.Value != "test" {
		t.Fatal("Unexpected event value")
	}

	j = NewJSON(&mocks.FailingWriter{})
	err = j.OnDebug("test")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestJSONOnError(t *testing.T) {
	sw := &mocks.SavingWriter{}
	j := NewJSON(sw)
	err := j.OnError("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 1 {
		t.Fatal("invalid length")
	}
	var event struct {
		Key   string
		Value string
	}
	err = json.Unmarshal(sw.Data[0], &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Key != "error" {
		t.Fatal("Unexpected event key")
	}
	if event.Value != "test" {
		t.Fatal("Unexpected event value")
	}

	j = NewJSON(&mocks.FailingWriter{})
	err = j.OnError("test")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestJSONOnWarning(t *testing.T) {
	sw := &mocks.SavingWriter{}
	j := NewJSON(sw)
	err := j.OnWarning("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 1 {
		t.Fatal("invalid length")
	}
	var event struct {
		Key   string
		Value string
	}
	err = json.Unmarshal(sw.Data[0], &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Key != "warning" {
		t.Fatal("Unexpected event key")
	}
	if event.Value != "test" {
		t.Fatal("Unexpected event value")
	}

	j = NewJSON(&mocks.FailingWriter{})
	err = j.OnWarning("test")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestJSONOnInfo(t *testing.T) {
	sw := &mocks.SavingWriter{}
	j := NewJSON(sw)
	err := j.OnInfo("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 1 {
		t.Fatal("invalid length")
	}
	var event struct {
		Key   string
		Value string
	}
	err = json.Unmarshal(sw.Data[0], &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Key != "info" {
		t.Fatal("Unexpected event key")
	}
	if event.Value != "test" {
		t.Fatal("Unexpected event value")
	}

	j = NewJSON(&mocks.FailingWriter{})
	err = j.OnInfo("test")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestJSONOnSpeed(t *testing.T) {
	sw := &mocks.SavingWriter{}
	j := NewJSON(sw)
	err := j.OnSpeed("test", "speed")
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 1 {
		t.Fatal("invalid length")
	}
	var event struct {
		Key   string
		Value string
	}
	err = json.Unmarshal(sw.Data[0], &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Key != "speed" {
		t.Fatal("Unexpected event key")
	}
	if event.Value != "test: speed" {
		t.Fatal("Unexpected event value")
	}

	j = NewJSON(&mocks.FailingWriter{})
	err = j.OnSpeed("test", "speed")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestNewJSONConstructor(t *testing.T) {
	if NewJSON(&mocks.SavingWriter{}) == nil {
		t.Fatal("NewJSONWithWriter did not return an Emitter")
	}
}

func TestEmitInterfaceFailure(t *testing.T) {
	j := jsonEmitter{Writer: os.Stdout}
	// See https://stackoverflow.com/a/48901259
	x := map[string]interface{}{
		"foo": make(chan int),
	}
	err := j.emitInterface(x)
	switch err.(type) {
	case *json.UnsupportedTypeError:
		// nothing
	default:
		t.Fatal("Expected a json.UnsupportedTypeError here")
	}
}

func TestJSONOnSummary(t *testing.T) {
	summary := &Summary{}
	sw := &mocks.SavingWriter{}
	j := NewJSON(sw)
	err := j.OnSummary(summary)
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 1 {
		t.Fatal("invalid length")
	}

	var output Summary
	err = json.Unmarshal(sw.Data[0], &output)
	if err != nil {
		t.Fatal(err)
	}
	if output.Client != summary.Client ||
		output.Server != summary.Server ||
		output.Download != summary.Download ||
		output.Upload != summary.Upload ||
		output.DownloadRetrans != summary.DownloadRetrans ||
		output.RTT != summary.RTT {
		t.Fatal("OnSummary(): unexpected output")
	}

}
