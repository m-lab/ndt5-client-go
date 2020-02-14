package emitter

import (
	"fmt"
	"testing"

	"github.com/m-lab/ndt5-client-go/cmd/ndt5-client/internal/mocks"
)

func TestHumanReadableOnDebug(t *testing.T) {
	sw := &mocks.SavingWriter{}
	hr := HumanReadable{sw}
	err := hr.OnDebug("test")
	if err != nil {
		t.Fatal(err)
	}
	if string(sw.Data[0]) != "\rtest\n" {
		t.Fatal("OnDebug(): unexpected output")
	}

	hr = HumanReadable{&mocks.FailingWriter{}}
	err = hr.OnDebug("test")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestHumanReadableOnError(t *testing.T) {
	sw := &mocks.SavingWriter{}
	hr := HumanReadable{sw}
	err := hr.OnError("test")
	if err != nil {
		t.Fatal(err)
	}
	if string(sw.Data[0]) != "\rtest\n" {
		t.Fatal("OnDebug(): unexpected output")
	}

	hr = HumanReadable{&mocks.FailingWriter{}}
	err = hr.OnError("test")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestHumanReadableOnWarning(t *testing.T) {
	sw := &mocks.SavingWriter{}
	hr := HumanReadable{sw}
	err := hr.OnWarning("test")
	if err != nil {
		t.Fatal(err)
	}
	if string(sw.Data[0]) != "\rtest\n" {
		t.Fatal("OnDebug(): unexpected output")
	}

	hr = HumanReadable{&mocks.FailingWriter{}}
	err = hr.OnWarning("test")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestHumanReadableOnInfo(t *testing.T) {
	sw := &mocks.SavingWriter{}
	hr := HumanReadable{sw}
	err := hr.OnInfo("test")
	if err != nil {
		t.Fatal(err)
	}
	if string(sw.Data[0]) != "\rtest\n" {
		t.Fatal("OnDebug(): unexpected output")
	}

	hr = HumanReadable{&mocks.FailingWriter{}}
	err = hr.OnInfo("test")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestHumanReadableOnSpeed(t *testing.T) {
	sw := &mocks.SavingWriter{}
	hr := HumanReadable{sw}
	err := hr.OnSpeed("test", "speed")
	if err != nil {
		t.Fatal(err)
	}
	if string(sw.Data[0]) != "\rtest: speed\n" {
		t.Fatal("OnDebug(): unexpected output")
	}

	hr = HumanReadable{&mocks.FailingWriter{}}
	err = hr.OnSpeed("test", "speed")
	if err != mocks.ErrMocked {
		t.Fatal("Not the error we expected")
	}
}

func TestHumanReadableOnSummary(t *testing.T) {
	expected := `         Server: test
         Client: test
        Latency:    10.0 ms
       Download:   100.0 Mbit/s
         Upload:   100.0 Mbit/s
 Retransmission:    1.00 %
`
	summary := &Summary{
		Client: "test",
		Server: "test",
		Download: ValueUnitPair{
			Value: 100.0,
			Unit:  "Mbit/s",
		},
		Upload: ValueUnitPair{
			Value: 100.0,
			Unit:  "Mbit/s",
		},
		DownloadRetrans: ValueUnitPair{
			Value: 1.0,
			Unit:  "%",
		},
		RTT: ValueUnitPair{
			Value: 10.0,
			Unit:  "ms",
		},
	}
	sw := &mocks.SavingWriter{}
	j := HumanReadable{sw}
	err := j.OnSummary(summary)
	if err != nil {
		t.Fatal(err)
	}

	if len(sw.Data) != 1 {
		t.Fatal("invalid length")
	}
	if string(sw.Data[0]) != expected {
		fmt.Println(string(sw.Data[0]))
		fmt.Println(expected)
		t.Fatal("OnSummary(): unexpected data")
	}
}

func TestHumanReadableOnSummaryFailure(t *testing.T) {
	sw := &mocks.FailingWriter{}
	j := HumanReadable{sw}
	err := j.OnSummary(&Summary{})
	if err == nil {
		t.Fatal("OnSummary(): expected err, got nil")
	}
}

func TestNewHumanReadableConstructor(t *testing.T) {
	hr := NewHumanReadable()
	if hr == nil {
		t.Fatal("NewHumanReadable() did not return a HumanReadable")
	}
}

func TestNewHumanReadableWithWriter(t *testing.T) {
	hr := NewHumanReadableWithWriter(&mocks.SavingWriter{})
	if hr == nil {
		t.Fatal("NewHumanReadableWithWriter() did not return a HumanReadable")
	}
}
