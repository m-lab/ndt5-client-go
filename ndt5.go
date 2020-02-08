// Package ndt5 contains an ndt5 client
package ndt5

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/m-lab/ndt7-client-go/mlabns"
)

// MockableMlabNSClient is a mockable mlab-ns client
type MockableMlabNSClient interface {
	Query(ctx context.Context) (fqdn string, err error)
}

// MockableDialer is a mockable connection dialer
type MockableDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Client is an ndt5 client
type Client struct {
	// ClientName is the name of the software running ndt5 tests. It's set by
	// NewClient; you may want to change this value.
	ClientName string

	// ClientVersion is the version of the software running ndt5 tests. It's
	// set by NewClient; you may want to change this value.
	ClientVersion string

	// Dialer is the optional network Dialer. It's set to its
	// default value by NewClient; you may override it.
	Dialer MockableDialer

	// FQDN is the optional server FQDN. We will discover the FQDN of
	// a nearby M-Lab server for you if this field is empty.
	FQDN string

	// MLabNSClient is the mlabns client. We'll configure it with
	// defaults in NewClient and you may override it.
	MLabNSClient MockableMlabNSClient
}

// Output is the output emitted by ndt5
type Output struct {
	CurDownloadSpeed *Speed      `json:",omitempty"`
	CurUploadSpeed   *Speed      `json:",omitempty"`
	InfoMessage      *LogMessage `json:",omitempty"`
}

// LogMessage contains a log message
type LogMessage struct {
	Message string
}

// Speed contains a speed measurement
type Speed struct {
	Count   int64
	Elapsed time.Duration
}

// NewClient creates a new ndt5 client instance.
func NewClient() *Client {
	return &Client{
		ClientName:    "bassosimone-ndt5-client-go",
		ClientVersion: "0.0.1",
		Dialer:        new(net.Dialer),
		MLabNSClient: mlabns.NewClient(
			"ndt", "bassosimone-ndt5-client-go/0.0.1",
		),
	}
}

// Start discovers a ndt5 server (if needed) and starts the whole ndt5 test. On
// success it returns a channel where measurements are posted. This channel is
// closed when the download ends. On failure, the error is non nil and you should
// not attempt using the channel. A side effect of starting the download is that, if
// you did not specify a server FQDN, we will discover a server for you and store
// that value into the c.FQDN field.
func (c *Client) Start(ctx context.Context) (<-chan *Output, error) {
	if c.FQDN == "" {
		fqdn, err := c.getClosestServer(ctx)
		if err != nil {
			return nil, err
		}
		c.FQDN = fqdn
	}
	conn, err := c.Dialer.DialContext(
		ctx, "tcp", net.JoinHostPort(c.FQDN, "3001"))
	if err != nil {
		return nil, err
	}
	ch := make(chan *Output)
	go c.run(ctx, conn, ch)
	return ch, nil
}

func (c *Client) getClosestServer(ctx context.Context) (string, error) {
	return c.MLabNSClient.Query(ctx)
}

func (c *Client) run(ctx context.Context, conn net.Conn, ch chan<- *Output) {
	defer close(ch)
	defer conn.Close()
	c.emitProgress("connected to remote server", ch)
	if err := c.sendLogin(conn); err != nil {
		return
	}
	c.emitProgress("sent login message", ch)
	if err := c.recvKickoff(conn); err != nil {
		return
	}
	c.emitProgress("received kickoff message", ch)
	if err := c.waitInQueue(conn); err != nil {
		return
	}
	c.emitProgress("cleared to run the tests", ch)
	if err := c.recvVersion(conn); err != nil {
		return
	}
	c.emitProgress("got remote server version", ch)
	testIDs, err := c.recvTestIDs(conn)
	if err != nil {
		return
	}
	c.emitProgress("got list of test IDs", ch)
	for _, testID := range testIDs {
		switch testID {
		case nettestDownload:
			c.emitProgress("running the download test", ch)
			c.runDownload(ctx, conn, ch)
		case nettestUpload:
			c.emitProgress("running the upload test", ch)
			c.runUpload(ctx, conn, ch)
		}
	}
	c.emitProgress("receiving the results", ch)
	c.recvResultsAndLogout(conn, ch)
	c.emitProgress("finished", ch)
}

const (
	maxResultsLoops = 128

	msgSrvQueue     uint8 = 1
	msgLogin        uint8 = 2
	msgTestPrepare  uint8 = 3
	msgTestStart    uint8 = 4
	msgTestMsg      uint8 = 5
	msgTestFinalize uint8 = 6
	msgLogout       uint8 = 9

	nettestUpload   uint8 = 1 << 1
	nettestDownload uint8 = 1 << 2
	nettestStatus   uint8 = 1 << 4
)

func (c *Client) sendLogin(conn net.Conn) error {
	body := make([]byte, 1)
	body[0] = nettestUpload | nettestDownload | nettestStatus
	return c.msgWriteLegacy(conn, msgLogin, body)
}

func (c *Client) recvKickoff(conn net.Conn) error {
	desired := []byte("123456 654321")
	got := make([]byte, len(desired))
	if err := c.readn(conn, got); err != nil {
		return err
	}
	if !bytes.Equal(desired, got) {
		return errors.New("recvKickoff: got invalid kickoff")
	}
	return nil
}

func (c *Client) waitInQueue(conn net.Conn) error {
	mtype, mdata, err := c.msgReadLegacy(conn)
	if err != nil {
		return err
	}
	if mtype != msgSrvQueue {
		return errors.New("waitInQueue: unexpected message type")
	}
	if !bytes.Equal(mdata, []byte("0")) {
		return errors.New("waitInQueue: server is busy")
	}
	return nil
}

func (c *Client) recvVersion(conn net.Conn) error {
	mtype, _, err := c.msgReadLegacy(conn)
	if err != nil {
		return err
	}
	if mtype != msgLogin {
		return errors.New("recvVersion: unexpected message type")
	}
	return nil
}

func (c *Client) recvTestIDs(conn net.Conn) ([]uint8, error) {
	mtype, mdata, err := c.msgReadLegacy(conn)
	if err != nil {
		return nil, err
	}
	if mtype != msgLogin {
		return nil, errors.New("recvTestIDs: unexpected message type")
	}
	elems := bytes.Split(mdata, []byte(" "))
	if len(elems) < 1 {
		return nil, errors.New("recvTestIDs: zero test IDs")
	}
	var testIDs []uint8
	for _, elem := range elems {
		val, err := strconv.ParseUint(string(elem), 10, 8)
		if err != nil {
			return nil, err
		}
		testIDs = append(testIDs, uint8(val))
	}
	return testIDs, nil
}

func (c *Client) runUpload(ctx context.Context, conn net.Conn, ch chan<- *Output) error {
	testdata := c.makeBuffer()
	portnum, err := c.expectTestPrepare(conn)
	if err != nil {
		return err
	}
	c.emitProgress("got test prepare message", ch)
	testconn, err := c.Dialer.DialContext(
		ctx, "tcp", net.JoinHostPort(c.FQDN, portnum),
	)
	if err != nil {
		return err
	}
	c.emitProgress("created measurement connection", ch)
	if err := testconn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	if err := c.expectTestStart(conn); err != nil {
		return err
	}
	c.emitProgress("got test start message", ch)
	testch := make(chan *Speed)
	go func() {
		defer testconn.Close()
		defer close(testch)
		var (
			begin = time.Now()
			count int64
		)
		for {
			num, err := testconn.Write(testdata)
			if err != nil {
				return
			}
			count += int64(num)
			testch <- &Speed{Count: count, Elapsed: time.Since(begin)}
		}
	}()
	for speed := range testch {
		c.emit(&Output{CurUploadSpeed: speed}, ch)
	}
	c.emitProgress("goroutine terminated", ch)
	speed, err := c.expectTestMsg(conn)
	if err != nil {
		return err
	}
	c.emitProgress(fmt.Sprintf("server-measured speed: %s", speed), ch)
	if err := c.expectTestFinalize(conn); err != nil {
		return err
	}
	c.emitProgress("test terminated", ch)
	return nil
}

func (c *Client) runDownload(ctx context.Context, conn net.Conn, ch chan<- *Output) error {
	testdata := make([]byte, 1<<20)
	portnum, err := c.expectTestPrepare(conn)
	if err != nil {
		return err
	}
	c.emitProgress("got test prepare message", ch)
	testconn, err := c.Dialer.DialContext(
		ctx, "tcp", net.JoinHostPort(c.FQDN, portnum),
	)
	if err != nil {
		return err
	}
	c.emitProgress("created measurement connection", ch)
	if err := testconn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	if err := c.expectTestStart(conn); err != nil {
		return err
	}
	c.emitProgress("got test start message", ch)
	testch := make(chan *Speed)
	go func() {
		defer testconn.Close()
		defer close(testch)
		var (
			begin = time.Now()
			count int64
		)
		for {
			num, err := testconn.Read(testdata)
			if err != nil {
				return
			}
			count += int64(num)
			testch <- &Speed{Count: count, Elapsed: time.Since(begin)}
		}
	}()
	for speed := range testch {
		c.emit(&Output{CurDownloadSpeed: speed}, ch)
	}
	c.emitProgress("goroutine terminated", ch)
	speed, err := c.expectTestMsg(conn)
	if err != nil {
		return err
	}
	c.emitProgress(fmt.Sprintf("server-measured speed: %s", speed), ch)
	// TODO(bassosimone): send real download speed
	if err := c.msgWriteLegacy(conn, msgTestMsg, []byte("0")); err != nil {
		return err
	}
	for i := 0; i < maxResultsLoops; i++ {
		mtype, mdata, err := c.msgReadLegacy(conn)
		if err != nil {
			return err
		}
		if mtype == msgLogout {
			c.emitProgress("test terminated", ch)
			return nil
		}
		// TODO(bassosimone): save these messages
		c.emitProgress(fmt.Sprintf("web100: %s", string(mdata)), ch)
	}
	return errors.New("download: too many results")
}

func (c *Client) recvResultsAndLogout(conn net.Conn, ch chan<- *Output) error {
	for i := 0; i < maxResultsLoops; i++ {
		mtype, mdata, err := c.msgReadLegacy(conn)
		if err != nil {
			return err
		}
		if mtype == msgLogout {
			return nil
		}
		// TODO(bassosimone): save these messages
		c.emitProgress(fmt.Sprintf("server: %s", string(mdata)), ch)
	}
	return errors.New("recvResultsAndLogout: too many results")
}

func (c *Client) expectTestPrepare(conn net.Conn) (port string, err error) {
	var (
		mtype uint8
		mdata []byte
	)
	mtype, mdata, err = c.msgReadLegacy(conn)
	if err != nil {
		return
	}
	if mtype != msgTestPrepare {
		err = errors.New("expectTestPrepare: invalid message type")
		return
	}
	port = string(mdata)
	return
}

func (c *Client) expectTestStart(conn net.Conn) error {
	mtype, mdata, err := c.msgReadLegacy(conn)
	if err != nil {
		return err
	}
	if mtype != msgTestStart {
		return errors.New("expectTestStart: invalid message type")
	}
	if len(mdata) != 0 {
		return errors.New("expectTestStart: expected empty message")
	}
	return nil
}

func (c *Client) expectTestMsg(conn net.Conn) (string, error) {
	mtype, mdata, err := c.msgReadLegacy(conn)
	if err != nil {
		return "", err
	}
	if mtype != msgTestMsg {
		return "", errors.New("expectTestMsg: invalid message type")
	}
	if len(mdata) == 0 {
		return "", errors.New("expectTestMsg: expected nonempty message")
	}
	return string(mdata), nil
}

func (c *Client) expectTestFinalize(conn net.Conn) error {
	mtype, mdata, err := c.msgReadLegacy(conn)
	if err != nil {
		return err
	}
	if mtype != msgTestFinalize {
		return errors.New("expectTestFinalize: invalid message type")
	}
	if len(mdata) != 0 {
		return errors.New("expectTestFinalize: expected empty message")
	}
	return nil
}

func (c *Client) msgWriteLegacy(conn net.Conn, mtype uint8, data []byte) error {
	b := []byte{mtype}
	if _, err := conn.Write(b); err != nil {
		return err
	}
	if len(data) > math.MaxUint16 {
		return errors.New("msgWriteLegacy: message too long")
	}
	b = make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(len(data)))
	if _, err := conn.Write(b); err != nil {
		return err
	}
	_, err := conn.Write(data)
	return err
}

func (c *Client) msgReadLegacy(conn net.Conn) (mtype uint8, data []byte, err error) {
	b := make([]byte, 1)
	if err = c.readn(conn, b); err != nil {
		return
	}
	mtype = b[0]
	b = make([]byte, 2)
	if err = c.readn(conn, b); err != nil {
		return
	}
	length := binary.BigEndian.Uint16(b)
	data = make([]byte, length)
	err = c.readn(conn, data)
	return
}

func (c *Client) readn(conn net.Conn, data []byte) error {
	for off := 0; off < len(data); {
		curr := make([]byte, 1)
		if _, err := conn.Read(curr); err != nil {
			return err
		}
		data[off] = curr[0]
		off++
	}
	return nil
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

func (c *Client) emitProgress(msg string, ch chan<- *Output) {
	c.emit(&Output{InfoMessage: &LogMessage{Message: msg}}, ch)
}

func (c *Client) emit(msg *Output, ch chan<- *Output) {
	ch <- msg
}
