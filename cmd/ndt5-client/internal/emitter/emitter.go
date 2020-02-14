// Package emitter contains the ndt7-client emitter.
package emitter

// Emitter is a generic emitter. When an event occurs, the
// corresponding method will be called. An error will generally
// mean that it's not possible to write the output. A common
// case where this happen is where the output is redirected to
// a file on a full hard disk.
//
// See the documentation of the main package for more details
// on the sequence in which events may occur.
type Emitter interface {
	// OnDebug is emitted on debug messages.
	OnDebug(string) error

	// OnError is emitted on error mesages.
	OnError(string) error

	// OnWarning is emitted on warning messages.
	OnWarning(string) error

	// OnInfo is emitted on info messages.
	OnInfo(string) error

	// OnUploadEvent is emitted during the upload.
	OnSpeed(string, string) error

	// OnSummary is emitted after the test is over.
	OnSummary(s *Summary) error
}
