package emitter

import (
	"fmt"
	"io"
	"os"
)

// HumanReadable is a human readable emitter. It emits the events generated
// by running a ndt5 test as pleasant stdout messages.
type HumanReadable struct {
	out io.Writer
}

// NewHumanReadable returns a new human readable emitter.
func NewHumanReadable() Emitter {
	return HumanReadable{os.Stdout}
}

// NewHumanReadableWithWriter returns a new human readable emitter using the
// specified writer.
func NewHumanReadableWithWriter(w io.Writer) Emitter {
	return HumanReadable{w}
}

// OnDebug handles debug messages.
func (h HumanReadable) OnDebug(m string) error {
	_, err := fmt.Fprintf(h.out, "\r%s\n", m)
	return err
}

// OnError handles error messages.
func (h HumanReadable) OnError(m string) error {
	_, failure := fmt.Fprintf(h.out, "\r%s\n", m)
	return failure
}

// OnWarning handles warning messages.
func (h HumanReadable) OnWarning(m string) error {
	_, err := fmt.Fprintf(h.out, "\r%s\n", m)
	return err
}

// OnInfo handles info messages.
func (h HumanReadable) OnInfo(m string) error {
	_, err := fmt.Fprintf(h.out, "\r%s\n", m)
	return err
}

// OnSpeed handles a speed reporting event during a test.
func (h HumanReadable) OnSpeed(test string, speed string) error {
	_, err := fmt.Fprintf(h.out, "\r%s: %s\n", test, speed)
	return err
}

// OnSummary handles the summary event.
func (h HumanReadable) OnSummary(s *Summary) error {
	const summaryFormat = `%15s: %s
%15s: %s
%15s: %7.1f %s
%15s: %7.1f %s
%15s: %7.1f %s
%15s: %7.2f %s
`
	_, err := fmt.Fprintf(h.out, summaryFormat,
		"Server", s.ServerFQDN,
		"Client", s.ClientIP,
		"Latency", s.MinRTT.Value, s.MinRTT.Unit,
		"Download", s.Download.Value, s.Upload.Unit,
		"Upload", s.Upload.Value, s.Upload.Unit,
		"Retransmission", s.DownloadRetrans.Value, s.DownloadRetrans.Unit)
	if err != nil {
		return err
	}

	return nil
}
