package emitter

import (
	"encoding/json"
	"fmt"
	"io"
)

// jsonEmitter is a jsonEmitter emitter. It emits messages consistent with
// the cmd/ndt5-client/main.go documentation for `-format=json`.
type jsonEmitter struct {
	io.Writer
}

// NewJSON creates a new JSON emitter
func NewJSON(w io.Writer) Emitter {
	return jsonEmitter{w}
}

func (j jsonEmitter) emitData(data []byte) error {
	_, err := j.Write(append(data, byte('\n')))
	return err
}

func (j jsonEmitter) emitInterface(any interface{}) error {
	data, err := json.Marshal(any)
	if err != nil {
		return err
	}
	return j.emitData(data)
}

type batchEvent struct {
	Key   string
	Value interface{}
}

// OnDebug emits debug events.
func (j jsonEmitter) OnDebug(m string) error {
	return j.emitInterface(batchEvent{
		Key:   "debug",
		Value: m,
	})
}

// OnError emits error events.
func (j jsonEmitter) OnError(m string) error {
	return j.emitInterface(batchEvent{
		Key:   "error",
		Value: m,
	})
}

// OnWarning emits warning events.
func (j jsonEmitter) OnWarning(m string) error {
	return j.emitInterface(batchEvent{
		Key:   "warning",
		Value: m,
	})
}

// OnInfo emits info events.
func (j jsonEmitter) OnInfo(m string) error {
	return j.emitInterface(batchEvent{
		Key:   "info",
		Value: m,
	})
}

// OnSpeed emits speed events.
func (j jsonEmitter) OnSpeed(test string, speed string) error {
	return j.emitInterface(batchEvent{
		Key:   "speed",
		Value: fmt.Sprintf("%s: %s", test, speed),
	})
}

// OnSummary handles the summary event, emitted after the test is over.
func (j jsonEmitter) OnSummary(s *Summary) error {
	return j.emitInterface(s)
}
