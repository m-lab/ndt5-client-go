package emitter

// Quiet acts as a filter allowing summary and error messages only, and
// doesn't perform any formatting.
// The message is actually emitted by the embedded Emitter.
type Quiet struct {
	emitter Emitter
}

// NewQuiet returns a Summary emitter which emits messages
// via the passed Emitter.
func NewQuiet(e Emitter) Emitter {
	return &Quiet{
		emitter: e,
	}
}

// OnDebug does not emit anything.
func (q Quiet) OnDebug(string) error {
	return nil
}

// OnError emits the error event.
func (q Quiet) OnError(m string) error {
	return q.emitter.OnError(m)
}

// OnWarning does not emit anything.
func (q Quiet) OnWarning(string) error {
	return nil
}

// OnInfo does not emit anything.
func (q Quiet) OnInfo(string) error {
	return nil
}

// OnSpeed does not emit anything.
func (q Quiet) OnSpeed(string, string) error {
	return nil
}

// OnSummary handles the summary event, emitted after the test is over.
func (q Quiet) OnSummary(s *Summary) error {
	return q.emitter.OnSummary(s)
}
