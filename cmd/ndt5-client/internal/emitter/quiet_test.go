package emitter

import (
	"os"
	"testing"

	"github.com/m-lab/ndt5-client-go/cmd/ndt5-client/internal/mocks"
)

func TestNewQuiet(t *testing.T) {
	e := jsonEmitter{os.Stdout}
	if NewQuiet(e) == nil {
		t.Fatal("NewQuiet() did not return an Emitter")
	}
}

func TestQuiet_OnDebug(t *testing.T) {
	sw := &mocks.SavingWriter{}
	e := jsonEmitter{sw}
	quiet := Quiet{e}
	err := quiet.OnDebug("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 0 {
		t.Fatal("OnStarting(): unexpected data")
	}
}

func TestQuiet_OnError(t *testing.T) {
	// The only thing to test here is that errors from the underlying emitter
	// are passed back to the caller.
	sw := &mocks.FailingWriter{}
	e := jsonEmitter{sw}
	quiet := Quiet{e}
	err := quiet.OnError("test")
	if err != mocks.ErrMocked {
		t.Fatal("OnError(): unexpected error type or nil")
	}
}

func TestQuiet_OnWarning(t *testing.T) {
	sw := &mocks.SavingWriter{}
	e := jsonEmitter{sw}
	quiet := Quiet{e}
	err := quiet.OnWarning("download")
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 0 {
		t.Fatal("OnConnected(): unexpected data")
	}
}

func TestQuiet_OnInfo(t *testing.T) {
	sw := &mocks.SavingWriter{}
	e := jsonEmitter{sw}
	quiet := Quiet{e}
	err := quiet.OnInfo("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 0 {
		t.Fatal("OnDownloadEvent(): unexpected data")
	}
}

func TestQuiet_OnSpeed(t *testing.T) {
	sw := &mocks.SavingWriter{}
	e := jsonEmitter{sw}
	quiet := Quiet{e}
	err := quiet.OnSpeed("test", "speed")
	if err != nil {
		t.Fatal(err)
	}
	if len(sw.Data) != 0 {
		t.Fatal("OnUploadEvent(): unexpected data")
	}
}
func TestQuiet_OnSummary(t *testing.T) {
	// The only thing to test here is that errors from the underlying emitter
	// are passed back to the caller.
	sw := &mocks.FailingWriter{}
	e := jsonEmitter{sw}
	quiet := Quiet{e}
	err := quiet.OnSummary(&Summary{})
	if err != mocks.ErrMocked {
		t.Fatal("OnSummary(): unexpected error type or nil")
	}
}
