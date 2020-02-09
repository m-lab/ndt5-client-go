// Package ndt5 contains an ndt5 client.
package ndt5

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"time"

	"github.com/m-lab/ndt7-client-go/mlabns"
)

// MockableMlabNSClient is a mockable mlab-ns client
type MockableMlabNSClient interface {
	Query(ctx context.Context) (fqdn string, err error)
}

// MeasurementConn is a measurement connection.
type MeasurementConn interface {
	// SetDeadline sets the read and write deadlines.
	SetDeadline(deadline time.Time) error

	// AllocReadBuffer configures the buffer to be used
	// by ReadDiscard. You MUST call this method before you
	// call ReadDiscard, or the code will crash.
	AllocReadBuffer(size int)

	// ReadDiscard reads and discard bytes. Returns the
	// number of discarded bytes or an error.
	ReadDiscard() (int64, error)

	// SetPreparedMessage sets the message that you want to
	// send in WritePreparedMessage. You MUST call this method
	// before you call WritePreparedMessage, or we'll crash.
	SetPreparedMessage(b []byte)

	// WritePreparedMessage writes the previously prepared
	// message. Returns number of bytes written or error.
	WritePreparedMessage() (int, error)

	// Close closes the measurement connection.
	Close() error
}

// Frame is an ndt5 frame
type Frame struct {
	Message []byte // message body
	Raw     []byte // the whole raw message
	Type    uint8  // type of message
}

const (
	maxMessageSize = math.MaxUint16
	maxFrameSize   = 3 + maxMessageSize
)

// NewFrame creates a new frame
func NewFrame(mtype uint8, message []byte) (*Frame, error) {
	// <type: uint8> <length: uint16> <message: [0..65535]byte>
	if len(message) > maxMessageSize {
		return nil, errors.New("message too large for frame")
	}
	b := make([]byte, len(message)+3)
	b[0] = mtype
	binary.BigEndian.PutUint16(b[1:3], uint16(len(message)))
	copy(b[3:], message)
	return &Frame{
		Message: message,
		Raw:     b,
		Type:    mtype,
	}, nil
}

// FrameReadWriteObserver observes when ndt5 frames are
// read or written on the control conn. You MUST NOT change
// the frames that you see, but you can log them.
type FrameReadWriteObserver interface {
	OnRead(frame *Frame)
	OnWrite(frame *Frame)
}

type defaultFrameReadWriteObserver struct{}

func (*defaultFrameReadWriteObserver) OnRead(frame *Frame)  {}
func (*defaultFrameReadWriteObserver) OnWrite(frame *Frame) {}

// FrameReadWriteObserverFactory creates a new
// instance of FrameReadWriteObserver.
type FrameReadWriteObserverFactory interface {
	New(out chan<- *Output) FrameReadWriteObserver
}

type defaultFrameReadWriteObserverFactory struct{}

func (*defaultFrameReadWriteObserverFactory) New(out chan<- *Output) FrameReadWriteObserver {
	return new(defaultFrameReadWriteObserver)
}

// ControlConn is a control connection.
type ControlConn interface {
	// SetFrameReadWriteObserver sets the observer for the
	// events where a ndt5 frame is read or written
	SetFrameReadWriteObserver(observer FrameReadWriteObserver)

	// SetDeadline sets the read and write dealines for the conn.
	SetDeadline(deadline time.Time) error

	// WriteLogin writes the login message using the proper convention
	// required by the current transport.
	WriteLogin(versionCompat string, testSuite byte) error

	// ReadKickoffMessage reads the kickoff message into b. Depending
	// on the transport we may not read an actual message from the network
	// rather we'd just pretend doing so.
	ReadKickoffMessage(b []byte) error

	// ReadFrame reads the next ndt5 frame.
	ReadFrame() (*Frame, error)

	// WriteMessage writes a ndt5 frame containing the specified ndt5
	// message type and message data as body.
	WriteMessage(mtype uint8, data []byte) error

	// WriteFrame writes the specified frame.
	WriteFrame(frame *Frame) error

	// Close closes the connection.
	Close() error
}

// ConnectionsFactory creates connections. There are several ndt5
// transports (e.g. raw TCP, WebSocket) and, for each of them, there
// is a specific ConnectionFactory that you can use.
type ConnectionsFactory interface {
	// DialControlConn dials a control connection. The code shall check
	// whether the address contain a port and use the default port for
	// the specific transport otherwise.
	DialControlConn(ctx context.Context, address string) (ControlConn, error)

	// DialMeasurementConn dials a measurement connection with the
	// specified address. The caller is supposed to compose such address
	// by joining together the FQDN currently being used with the port
	// that has been indicated by the ndt5 server.
	DialMeasurementConn(ctx context.Context, address string) (MeasurementConn, error)
}

// Protocol manages a ControlConn. We currently only support the
// ndt5 control protocol. You may still want to override the protocol
// instance used by the client for testing purposes.
type Protocol interface {
	SendLogin() error
	ReceiveKickoff() error
	WaitInQueue() error
	ReceiveVersion() (version string, err error)
	ReceiveTestIDs() (ids []uint8, err error)
	ExpectTestPrepare() (portnum string, err error)
	ExpectTestStart() error
	ExpectTestMsg() (info string, err error)
	ExpectTestFinalize() error
	SendTestMsg(data []byte) error
	ReceiveTestFinalizeOrTestMsg() (mtype uint8, mdata []byte, err error)
	ReceiveLogoutOrResults() (mtype uint8, mdata []byte, err error)
}

// ProtocolFactory creates a Protocol.
type ProtocolFactory interface {
	NewProtocol(cc ControlConn) Protocol
}

// Client is an ndt5 client.
type Client struct {
	// ConnectionsFactory creates connections. It's set to its
	// default value by NewClient; you may override it.
	//
	// By changing this field before starting the experiment
	// you can choose the specific transport you want.
	//
	// The default transport is the raw TCP transport that was
	// initially introduced with the NDT codebase.
	ConnectionsFactory ConnectionsFactory

	// ObserverFactory allows you to observe frame events. It's set to its
	// default value by NewClient; you may override it.
	ObserverFactory FrameReadWriteObserverFactory

	// ProtocolFactory creates a ControlManager. It's set to its
	// default value by NewClient; you may override it.
	//
	// This is generally only required for testing.
	ProtocolFactory ProtocolFactory

	// FQDN is the optional server FQDN. We will discover the FQDN of
	// a nearby M-Lab server for you if this field is empty.
	//
	// Setting this field allows you test use a specific server.
	FQDN string

	// MLabNSClient is the mlabns client. We'll configure it with
	// defaults in NewClient and you may override it.
	MLabNSClient MockableMlabNSClient
}

// Output is the output emitted by ndt5
type Output struct {
	CurDownloadSpeed *Speed      `json:",omitempty"`
	CurUploadSpeed   *Speed      `json:",omitempty"`
	DebugMessage     *LogMessage `json:",omitempty"`
	ErrorMessage     *Failure    `json:",omitempty"`
	InfoMessage      *LogMessage `json:",omitempty"`
	WarningMessage   *Failure    `json:",omitempty"`
}

// LogMessage contains a log message
type LogMessage struct {
	Message string
}

// Failure contains an error
type Failure struct {
	Error error
}

// Speed contains a speed measurement
type Speed struct {
	Count   int64         // number of bytes transferred
	Elapsed time.Duration // nanoseconds since beginning
}

// NewClient creates a new ndt5 client instance.
//
// TODO(bassosimone): here we MUST receive the user agent
// like we already do in ndt7.
func NewClient() *Client {
	return &Client{
		ConnectionsFactory: NewRawConnectionsFactory(),
		ObserverFactory:    new(defaultFrameReadWriteObserverFactory),
		ProtocolFactory:    new(protocolNDT5Factory),
		MLabNSClient: mlabns.NewClient(
			"ndt_ssl", "bassosimone-ndt5-client-go/0.0.1",
		),
	}
}

// Start discovers a ndt5 server (if needed) and starts the whole ndt5 test. On
// success it returns a channel where measurements are posted. This channel is
// closed when the test ends. On failure, the error is non nil and you should
// not attempt using the channel. A side effect of starting the test is that, if
// you did not specify a server FQDN, we will discover a server for you and store
// that value into the c.FQDN field. This is done without locking.
func (c *Client) Start(ctx context.Context) (<-chan *Output, error) {
	if c.FQDN == "" {
		fqdn, err := c.MLabNSClient.Query(ctx)
		if err != nil {
			return nil, err
		}
		c.FQDN = fqdn
	}
	cc, err := c.ConnectionsFactory.DialControlConn(ctx, c.FQDN)
	if err != nil {
		return nil, err
	}
	ch := make(chan *Output)
	cc.SetFrameReadWriteObserver(c.ObserverFactory.New(ch))
	go c.run(ctx, cc, ch)
	return ch, nil
}

const (
	maxResultsLoops = 128

	msgSrvQueue      uint8 = 1
	msgLogin         uint8 = 2
	msgTestPrepare   uint8 = 3
	msgTestStart     uint8 = 4
	msgTestMsg       uint8 = 5
	msgTestFinalize  uint8 = 6
	msgResults       uint8 = 8
	msgLogout        uint8 = 9
	msgExtendedLogin uint8 = 11

	nettestUpload   uint8 = 1 << 1
	nettestDownload uint8 = 1 << 2
	nettestStatus   uint8 = 1 << 4
)

// run performs the ndt5 experiment. This function takes ownership of
// the conn argument and will close the ch argument when done.
func (c *Client) run(ctx context.Context, cc ControlConn, ch chan<- *Output) {
	defer close(ch)
	defer cc.Close()
	if err := cc.SetDeadline(time.Now().Add(45 * time.Second)); err != nil {
		c.emitError(fmt.Errorf("cannot set control connection deadline: %w", err), ch)
		return
	}
	proto := c.ProtocolFactory.NewProtocol(cc)
	c.emitProgress(fmt.Sprintf("connected to remote server: %s", c.FQDN), ch)
	if err := proto.SendLogin(); err != nil {
		c.emitError(fmt.Errorf("cannot send login message: %w", err), ch)
		return
	}
	c.emitProgress("sent login message", ch)
	if err := proto.ReceiveKickoff(); err != nil {
		c.emitError(fmt.Errorf("cannot receive kickoff message: %w", err), ch)
		return
	}
	c.emitProgress("received the kickoff message", ch)
	if err := proto.WaitInQueue(); err != nil {
		c.emitError(fmt.Errorf("cannot wait in queue: %w", err), ch)
		return
	}
	c.emitProgress("cleared to run the tests", ch)
	version, err := proto.ReceiveVersion()
	if err != nil {
		c.emitError(fmt.Errorf("cannot receive server's version: %w", err), ch)
		return
	}
	c.emitProgress(fmt.Sprintf("got remote server version: %s", version), ch)
	testIDs, err := proto.ReceiveTestIDs()
	if err != nil {
		c.emitError(fmt.Errorf("cannot receive test IDs: %w", err), ch)
		return
	}
	c.emitProgress(fmt.Sprintf("got list of test IDs: %+v", testIDs), ch)
	for _, testID := range testIDs {
		switch testID {
		case nettestDownload:
			c.emitProgress("running the download test", ch)
			if err := c.runDownload(ctx, proto, ch); err != nil {
				c.emitWarning(fmt.Errorf("download failed: %w", err), ch)
				// don't stop testing
			}
		case nettestUpload:
			c.emitProgress("running the upload test", ch)
			if err := c.runUpload(ctx, proto, ch); err != nil {
				c.emitWarning(fmt.Errorf("upload failed: %w", err), ch)
				// don't stop testing
			}
		}
	}
	c.emitProgress("receiving the results", ch)
	if err := c.recvResultsAndLogout(proto, ch); err != nil {
		c.emitError(fmt.Errorf("recvResultsAndLogout failed: %w", err), ch)
		return
	}
	c.emitProgress("finished successfully", ch)
}

func (c *Client) runUpload(ctx context.Context, proto Protocol, ch chan<- *Output) error {
	testdata := c.makeBuffer()
	portnum, err := proto.ExpectTestPrepare()
	if err != nil {
		err = fmt.Errorf("cannot get TestPrepare message: %w", err)
		return err
	}
	c.emitProgress("got TestPrepare message", ch)
	testconn, err := c.ConnectionsFactory.DialMeasurementConn(
		ctx, net.JoinHostPort(c.FQDN, portnum),
	)
	if err != nil {
		err = fmt.Errorf("cannot create measurement connection: %w", err)
		return err
	}
	c.emitProgress("created measurement connection", ch)
	if err := testconn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		err = fmt.Errorf("cannot set measurement connection deadline: %w", err)
		return err
	}
	if err := proto.ExpectTestStart(); err != nil {
		err = fmt.Errorf("cannot get TestStart message: %w", err)
		return err
	}
	c.emitProgress("got TestStart message", ch)
	testconn.SetPreparedMessage(testdata)
	testch := make(chan *Speed)
	go c.uploader(testconn, testch)
	c.emitProgress("uploader goroutine forked off", ch)
	for speed := range testch {
		c.emit(&Output{CurUploadSpeed: speed}, ch)
	}
	c.emitProgress("uploader goroutine terminated", ch)
	speed, err := proto.ExpectTestMsg()
	if err != nil {
		err = fmt.Errorf("cannot get TestMsg message: %w", err)
		return err
	}
	// TODO(bassosimone): this information should probably be
	// parsed and emitted in a much more actionable way
	c.emitProgress(fmt.Sprintf("server-measured speed: %s", speed), ch)
	if err := proto.ExpectTestFinalize(); err != nil {
		err = fmt.Errorf("cannot get TestFinalize message: %w", err)
		return err
	}
	c.emitProgress("test terminated", ch)
	return nil
}

// uploader runs the async uploader. It takes ownership of the testconn
// and closes the testch when it is done.
func (c *Client) uploader(testconn MeasurementConn, testch chan<- *Speed) {
	defer testconn.Close()
	defer close(testch)
	var (
		begin = time.Now()
		count int64
	)
	for {
		num, err := testconn.WritePreparedMessage()
		if err != nil {
			return
		}
		count += int64(num)
		// TODO(bassosimone): this should be nonblocking
		testch <- &Speed{Count: count, Elapsed: time.Since(begin)}
	}
}

func (c *Client) runDownload(ctx context.Context, proto Protocol, ch chan<- *Output) error {
	const readBufferSize = 1 << 20
	portnum, err := proto.ExpectTestPrepare()
	if err != nil {
		err = fmt.Errorf("cannot get TestPrepare message: %w", err)
		return err
	}
	c.emitProgress("got test prepare message", ch)
	testconn, err := c.ConnectionsFactory.DialMeasurementConn(
		ctx, net.JoinHostPort(c.FQDN, portnum),
	)
	if err != nil {
		err = fmt.Errorf("cannot create measurement connection: %w", err)
		return err
	}
	c.emitProgress("created measurement connection", ch)
	if err := testconn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		err = fmt.Errorf("cannot set measurement connection deadline: %w", err)
		return err
	}
	if err := proto.ExpectTestStart(); err != nil {
		err = fmt.Errorf("cannot get TestStart message: %w", err)
		return err
	}
	c.emitProgress("got test start message", ch)
	testconn.AllocReadBuffer(readBufferSize)
	testch := make(chan *Speed)
	go c.downloader(testconn, testch)
	c.emitProgress("downloader goroutine forked off", ch)
	var lastSample *Speed
	for speed := range testch {
		c.emit(&Output{CurDownloadSpeed: speed}, ch)
		lastSample = speed
	}
	c.emitProgress("downloader goroutine terminated", ch)
	speed, err := proto.ExpectTestMsg()
	if err != nil {
		return err
	}
	// TODO(bassosimone): this information should probably be
	// parsed and emitted in a much more actionable way
	c.emitProgress(fmt.Sprintf("server-measured speed: %s kbit/s", speed), ch)
	var clientSpeed float64
	if lastSample != nil {
		elapsed := float64(lastSample.Elapsed / time.Millisecond)
		clientSpeed = 8 * float64(lastSample.Count) / elapsed
	}
	clientSpeedStr := fmt.Sprintf("%f", clientSpeed)
	c.emitProgress(fmt.Sprintf("client-measured speed: %s kbit/s", clientSpeedStr), ch)
	if err := proto.SendTestMsg([]byte(clientSpeedStr)); err != nil {
		err = fmt.Errorf("cannot seend TestMsg message: %w", err)
		return err
	}
	for i := 0; i < maxResultsLoops; i++ {
		mtype, mdata, err := proto.ReceiveTestFinalizeOrTestMsg()
		if err != nil {
			err = fmt.Errorf("cannot get message: %w", err)
			return err
		}
		if mtype == msgTestFinalize {
			c.emitProgress("test terminated", ch)
			return nil
		}
		// TODO(bassosimone): save these messages
		c.emitProgress(fmt.Sprintf("web100: %s", string(mdata)), ch)
	}
	return errors.New("download: too many results")
}

// downloader is like uploader but for the download.
func (c *Client) downloader(testconn MeasurementConn, testch chan<- *Speed) {
	defer testconn.Close()
	defer close(testch)
	var (
		begin = time.Now()
		count int64
	)
	for {
		num, err := testconn.ReadDiscard()
		if err != nil {
			return
		}
		count += num
		// TODO(bassosimone): this should be nonblocking
		testch <- &Speed{Count: count, Elapsed: time.Since(begin)}
	}
}

func (c *Client) recvResultsAndLogout(proto Protocol, ch chan<- *Output) error {
	for i := 0; i < maxResultsLoops; i++ {
		mtype, mdata, err := proto.ReceiveLogoutOrResults()
		if err != nil {
			err = fmt.Errorf("cannot get message: %w", err)
			return err
		}
		if mtype == msgLogout {
			return nil
		}
		// TODO(bassosimone): save these messages?
		c.emitProgress(fmt.Sprintf("server: %s", string(mdata)), ch)
	}
	return errors.New("recvResultsAndLogout: too many results")
}

func (c *Client) makeBuffer() []byte {
	// See https://stackoverflow.com/a/31832326
	b := make([]byte, 1<<17)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	var letterRunes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	for i := range b {
		b[i] = letterRunes[rnd.Intn(len(letterRunes))]
	}
	return b
}

func (c *Client) emitError(err error, ch chan<- *Output) {
	c.emit(&Output{ErrorMessage: &Failure{Error: err}}, ch)
}

func (c *Client) emitWarning(err error, ch chan<- *Output) {
	c.emit(&Output{ErrorMessage: &Failure{Error: err}}, ch)
}

func (c *Client) emitProgress(msg string, ch chan<- *Output) {
	c.emit(&Output{InfoMessage: &LogMessage{Message: msg}}, ch)
}

func (c *Client) emit(msg *Output, ch chan<- *Output) {
	ch <- msg
}
