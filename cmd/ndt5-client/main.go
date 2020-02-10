package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/rtx"
	"github.com/m-lab/ndt5-client-go"
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
	flagThrottle = flag.Bool("throttle", false, "Throttle connections for testing")
	flagTimeout  = flag.Duration(
		"timeout", defaultTimeout, "time after which the test is aborted")
	flagVerbose = flag.Bool("verbose", false, "Log ndt5 messages")
)

func init() {
	flag.Var(
		&flagProtocol,
		"protocol",
		`Protocol to use: "ndt5" or "ndt5+wss"`,
	)
}

func main() {
	flag.Parse()
	var dialer ndt5.NetDialer = new(net.Dialer)
	if *flagThrottle {
		dialer = trafficshaping.NewDialer()
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
	client := ndt5.NewClient(clientName, clientVersion)
	client.ProtocolFactory = factory5
	client.FQDN = *flagHostname
	ctx, cancel := context.WithTimeout(context.Background(), *flagTimeout)
	defer cancel()
	out, err := client.Start(ctx)
	rtx.Must(err, "client.Start failed")
	var extra string
	for ev := range out {
		if ev.DebugMessage != nil {
			fmt.Printf("%s%s\n", extra, strings.Trim(ev.DebugMessage.Message, "\t\n "))
			extra = ""
		}
		if ev.InfoMessage != nil {
			fmt.Printf("%s%s\n", extra, strings.Trim(ev.InfoMessage.Message, "\t\n "))
			extra = ""
		}
		if ev.WarningMessage != nil {
			fmt.Printf("%swarning: %s\n", extra, ev.WarningMessage.Error.Error())
			extra = ""
		}
		if ev.ErrorMessage != nil {
			fmt.Printf("%serror: %s\n", extra, ev.ErrorMessage.Error.Error())
			extra = ""
		}
		if ev.CurDownloadSpeed != nil {
			fmt.Printf("\rdownload: %s", computeSpeed(ev.CurDownloadSpeed))
			extra = "\n"
		}
		if ev.CurUploadSpeed != nil {
			fmt.Printf("\rupload:   %s", computeSpeed(ev.CurUploadSpeed))
			extra = "\n"
		}
	}
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
