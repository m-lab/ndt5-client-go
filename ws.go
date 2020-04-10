package ndt5

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/m-lab/go/rtx"
)

// WSConnectionsFactory creates ndt5+wss connections
type WSConnectionsFactory struct {
	Dialer *websocket.Dialer
	URL    *url.URL
}

// defaultURL creates the default url for connecting to the NDT wss server.
func defaultURL() *url.URL {
	return &url.URL{
		Scheme: "wss",
		Path:   "/ndt_protocol",
	}
}

// NewWSConnectionsFactory returns a factory for ndt5+wss connections
func NewWSConnectionsFactory(dialer NetDialer, u *url.URL) *WSConnectionsFactory {
	if u == nil {
		u = defaultURL()
	}
	const bufferSize = 1 << 20
	return &WSConnectionsFactory{
		Dialer: &websocket.Dialer{
			NetDial:          dialer.Dial,
			NetDialContext:   dialer.DialContext,
			HandshakeTimeout: 10 * time.Second,
			ReadBufferSize:   bufferSize,
			WriteBufferSize:  bufferSize,
		},
		URL: u,
	}
}

// DialControlConn implements ConnectionsFactory.DialControlConn
func (cf *WSConnectionsFactory) DialControlConn(
	ctx context.Context, address, userAgent string) (ControlConn, error) {
	u := *cf.URL
	u.Host = net.JoinHostPort(address, "3010")
	conn, err := cf.DialEx(ctx, u, "ndt", userAgent)
	if err != nil {
		return nil, err
	}
	return &wsControlConn{
		conn:     conn,
		observer: new(defaultFrameReadWriteObserver),
	}, nil
}

// DialMeasurementConn implements ConnectionsFactory.DialMeasurementConn.
func (cf *WSConnectionsFactory) DialMeasurementConn(
	ctx context.Context, address, userAgent string) (MeasurementConn, error) {
	u := *cf.URL
	u.Host = address
	conn, err := cf.DialEx(ctx, u, "ndt", userAgent)
	if err != nil {
		return nil, err
	}
	return &wsMeasurementConn{conn: conn}, nil
}

// DialEx is the extended WebSocket dial function
func (cf *WSConnectionsFactory) DialEx(
	ctx context.Context, u url.URL, wsProtocol, userAgent string,
) (*websocket.Conn, error) {
	headers := http.Header{}
	headers.Add("Sec-WebSocket-Protocol", wsProtocol)
	headers.Add("User-Agent", userAgent)
	conn, _, err := cf.Dialer.DialContext(ctx, u.String(), headers)
	return conn, err
}

type wsControlConn struct {
	conn     *websocket.Conn
	observer FrameReadWriteObserver
}

func (cc *wsControlConn) SetFrameReadWriteObserver(observer FrameReadWriteObserver) {
	cc.observer = observer
}

func (cc *wsControlConn) SetDeadline(deadline time.Time) (err error) {
	if err = cc.conn.SetReadDeadline(deadline); err == nil {
		err = cc.conn.SetWriteDeadline(deadline)
	}
	return
}

type wsLoginMessage struct {
	Msg   string `json:"msg"`
	Tests string `json:"tests"`
}

func (cc *wsControlConn) WriteLogin(versionCompat string, testSuite byte) error {
	return cc.writeJSON(msgExtendedLogin, wsLoginMessage{
		Msg:   versionCompat,
		Tests: strconv.Itoa(int(testSuite)),
	})
}

func (cc *wsControlConn) ReadKickoffMessage(b []byte) error {
	// Here we pretend that we're reading the kickoff message but there is
	// no such kickoff message on the wire with WebSocket
	copy(b, kickoffMessage)
	return nil
}

type wsMessage struct {
	Msg              string `json:"msg"`
	ThroughputValue  string
	TotalSentByte    string
	UnsentDataAmount string
}

func (cc *wsControlConn) ReadFrame() (*Frame, error) {
	// <type: uint8> <length: uint16> <message: [0..65536]byte>
	mtype, mdata, err := cc.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if mtype != websocket.BinaryMessage {
		return nil, errors.New("ws: expected BinaryMessage")
	}
	if len(mdata) > maxFrameSize {
		return nil, errors.New("ws: WebSocket frame too large")
	}
	if len(mdata) < 3 {
		return nil, errors.New("ws: WebSocket frame too small")
	}
	size := binary.BigEndian.Uint16(mdata[1:3]) + 3
	if int(size) != len(mdata) {
		return nil, errors.New("ws: did not receive a complete ndt5 frame")
	}
	// Here the value is a JSON message
	var msg wsMessage
	if err := json.Unmarshal(mdata[3:size], &msg); err != nil {
		return nil, err
	}
	// There is a bunch of JSON message possibilities. The approach here
	// is to reconstruct what a raw client would have sent us.
	var messagevalue string
	if msg.ThroughputValue != "" && msg.UnsentDataAmount != "" && msg.TotalSentByte != "" {
		messagevalue = fmt.Sprintf(
			"%s %s %s", msg.ThroughputValue, msg.UnsentDataAmount, msg.TotalSentByte,
		)
	} else {
		messagevalue = msg.Msg
	}
	// We don't bother with fixing up the raw message; indeed we want
	// such message to contain JSON for debugging. Upstream users will
	// be using just the Message and Type fields anyway.
	frame := &Frame{
		Message: []byte(messagevalue),
		Raw:     mdata,
		Type:    mdata[0],
	}
	cc.observer.OnRead(frame)
	return frame, nil
}

func (cc *wsControlConn) WriteMessage(mtype uint8, data []byte) error {
	return cc.writeJSON(mtype, wsMessage{Msg: string(data)})
}

func (cc *wsControlConn) writeJSON(mtype uint8, record interface{}) error {
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}
	frame, err := NewFrame(mtype, body)
	if err != nil {
		return err
	}
	return cc.WriteFrame(frame)
}

func (cc *wsControlConn) WriteFrame(frame *Frame) error {
	cc.observer.OnWrite(frame)
	return cc.conn.WriteMessage(websocket.BinaryMessage, frame.Raw)
}

func (cc *wsControlConn) Close() error {
	return cc.conn.Close()
}

type wsMeasurementConn struct {
	conn     *websocket.Conn
	prepared *websocket.PreparedMessage
	prepsiz  int
}

func (mc *wsMeasurementConn) SetDeadline(deadline time.Time) (err error) {
	if err = mc.conn.SetReadDeadline(deadline); err == nil {
		err = mc.conn.SetWriteDeadline(deadline)
	}
	return
}

func (mc *wsMeasurementConn) AllocReadBuffer(bufsiz int) {
	// Nothing we can actually do here
}

func (mc *wsMeasurementConn) ReadDiscard() (int64, error) {
	_, reader, err := mc.conn.NextReader()
	if err != nil {
		return 0, err
	}
	return io.Copy(ioutil.Discard, reader)
}

func (mc *wsMeasurementConn) SetPreparedMessage(b []byte) {
	pm, err := websocket.NewPreparedMessage(
		websocket.BinaryMessage, b,
	)
	rtx.PanicOnError(err, "websocket.NewPreparedMessage failed unexpectedly")
	mc.prepared = pm
	mc.prepsiz = len(b)
}

func (mc *wsMeasurementConn) WritePreparedMessage() (int, error) {
	// We assume the prepared message has been initialized
	err := mc.conn.WritePreparedMessage(mc.prepared)
	return mc.prepsiz, err
}

func (mc *wsMeasurementConn) Close() error {
	return mc.conn.Close()
}
