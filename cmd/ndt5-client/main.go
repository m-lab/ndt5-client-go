package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/rtx"
	"github.com/m-lab/ndt5-client-go"
	"github.com/m-lab/ndt5-client-go/cmd/ndt5-client/internal/emitter"
	"github.com/m-lab/ndt5-client-go/internal/trafficshaping"
)

const (
	clientName     = "ndt5-client-go-cmd"
	clientVersion  = "0.1.0"
	defaultTimeout = 55 * time.Second
)

var (
	flagHostname = flag.String("hostname", "", "Measurement server hostname")
	flagProtocol = flagx.Enum{
		Options: []string{"ndt5", "ndt5+wss"},
		Value:   "ndt5",
	}
	flagFormat = flagx.Enum{
		Options: []string{"human", "json"},
		Value:   "human",
	}
	flagNSURL    = flag.String("ns-url", "https://locate.measurementlab.net/", "Base URL to locate service")
	flagThrottle = flag.Int64("throttle", 0, "Throttle connections to given rate for testing (bits/sec)")
	flagTimeout  = flag.Duration(
		"timeout", defaultTimeout, "time after which the test is aborted")
	flagVerbose = flag.Bool("verbose", false, "Log ndt5 messages")
	flagQuiet   = flag.Bool("quiet", false, "emit summary and errors only")
)

func init() {
	flag.Var(
		&flagProtocol,
		"protocol",
		`Protocol to use: "ndt5" or "ndt5+wss"`,
	)
	flag.Var(
		&flagFormat,
		"format",
		`Output format: "human" or "json"`,
	)
}

func main() {
	flag.Parse()
	var dialer ndt5.NetDialer = new(net.Dialer)
	if *flagThrottle > 0 {
		dialer = trafficshaping.NewDialerWithBitrate(*flagThrottle)
	}
	factory5 := ndt5.NewProtocolFactory5()
	switch flagProtocol.Value {
	case "ndt5":
		factory5.ConnectionsFactory = ndt5.NewRawConnectionsFactory(dialer)
	case "ndt5+wss":
		factory5.ConnectionsFactory = ndt5.NewWSConnectionsFactory(dialer)
	}
	if *flagVerbose {
		factory5.ObserverFactory = new(verboseFrameReadWriteObserverFactory)
	}
	client := ndt5.NewClient(clientName, clientVersion, *flagNSURL)
	client.ProtocolFactory = factory5
	client.FQDN = *flagHostname

	var e emitter.Emitter
	if flagFormat.Value == "json" {
		e = emitter.NewJSON(os.Stdout)
	} else {
		e = emitter.NewHumanReadable()
	}

	if *flagQuiet {
		e = emitter.NewQuiet(e)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *flagTimeout)
	defer cancel()
	out, err := client.Start(ctx)
	rtx.Must(err, "client.Start failed")
	for ev := range out {
		if ev.DebugMessage != nil {
			e.OnDebug(strings.Trim(ev.DebugMessage.Message, "\t\n "))
		}
		if ev.InfoMessage != nil {
			e.OnInfo(strings.Trim(ev.InfoMessage.Message, "\t\n "))
		}
		if ev.WarningMessage != nil {
			e.OnWarning(ev.WarningMessage.Error.Error())
		}
		if ev.ErrorMessage != nil {
			e.OnError(ev.ErrorMessage.Error.Error())
		}
		if ev.CurDownloadSpeed != nil {
			e.OnSpeed("download", computeSpeed(ev.CurDownloadSpeed))
		}
		if ev.CurUploadSpeed != nil {
			e.OnSpeed("upload", computeSpeed(ev.CurUploadSpeed))
		}
	}

	summary := makeSummary(client.FQDN, client.Result)
	e.OnSummary(summary)
}

func makeSummary(FQDN string, result ndt5.TestResult) *emitter.Summary {
	s := emitter.NewSummary(FQDN)

	if serverIP, ok := result.Web100["NDTResult.S2C.ServerIP"]; ok {
		s.ServerIP = serverIP
	}

	if clientIP, ok := result.Web100["NDTResult.S2C.ClientIP"]; ok {
		s.ClientIP = clientIP
	}

	if UUID, ok := result.Web100["NDTResult.S2C.UUID"]; ok {
		s.DownloadUUID = UUID
	}

	elapsed := result.ClientMeasuredDownload.Elapsed.Seconds()
	s.Download = emitter.ValueUnitPair{
		Value: (8.0 * float64(result.ClientMeasuredDownload.Count)) /
			float64(elapsed) / 1000.0 / 1000.0,
		Unit: "Mbit/s",
	}

	s.Upload = emitter.ValueUnitPair{
		// Upload coming from the NDT server is in kbit/second.
		Value: result.ServerMeasuredUpload / 1000,
		Unit:  "Mbit/s",
	}

	// Here we use the MinRTT provided by the server, assuming they are
	// symmetrical.
	if rtt, ok := result.Web100["TCPInfo.MinRTT"]; ok {
		rtt, err := strconv.ParseFloat(rtt, 64)
		if err == nil {
			s.MinRTT = emitter.ValueUnitPair{
				// TCPInfo.MinRTT is in microseconds.
				Value: rtt / 1000.0,
				Unit:  "ms",
			}
		}
	}

	if bytesRetrans, ok := result.Web100["TCPInfo.BytesRetrans"]; ok {
		if bytesSent, ok := result.Web100["TCPInfo.BytesSent"]; ok {
			retrans, err1 := strconv.ParseFloat(bytesRetrans, 64)
			sent, err2 := strconv.ParseFloat(bytesSent, 64)

			if err1 == nil && err2 == nil {
				s.DownloadRetrans = emitter.ValueUnitPair{
					Value: retrans / sent * 100,
					Unit:  "%",
				}
			}
		}
	}
	return s
}

func computeSpeed(speed *ndt5.Speed) string {
	elapsed := speed.Elapsed.Seconds() * 1e06
	formatted := float64(8*speed.Count) / elapsed
	return fmt.Sprintf("%11.4f Mbit/s", formatted)
}

type verboseFrameReadWriteObserverFactory struct{}

func (of *verboseFrameReadWriteObserverFactory) New(out chan<- *ndt5.Output) ndt5.FrameReadWriteObserver {
	return &verboseFrameReadWriteObserver{out: out}
}

type verboseFrameReadWriteObserver struct {
	out chan<- *ndt5.Output
}

func (observer *verboseFrameReadWriteObserver) OnRead(frame *ndt5.Frame) {
	observer.log("< ", frame)
}

func (observer *verboseFrameReadWriteObserver) OnWrite(frame *ndt5.Frame) {
	observer.log("> ", frame)
}

func (observer *verboseFrameReadWriteObserver) log(prefix string, frame *ndt5.Frame) {
	observer.out <- &ndt5.Output{
		DebugMessage: &ndt5.LogMessage{
			Message: observer.reformat(prefix, hex.Dump(frame.Raw)),
		},
	}
}

func (observer *verboseFrameReadWriteObserver) reformat(prefix, message string) string {
	builder := new(strings.Builder)
	for _, line := range strings.Split(message, "\n") {
		// We don't bother with checking errors here
		if len(line) > 0 {
			builder.WriteString(prefix)
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
