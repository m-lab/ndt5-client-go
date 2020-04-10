package ndt5_test

import (
	"context"
	"testing"

	"github.com/m-lab/ndt5-client-go"
	"github.com/m-lab/ndt5-client-go/internal/trafficshaping"
)

const (
	clientName    = "ndt5-client-go-testing"
	clientVersion = "0.1.0"
)

func TestIntegrationClientRaw(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	protocolFactory := ndt5.NewProtocolFactory5()
	protocolFactory.ConnectionsFactory = ndt5.NewRawConnectionsFactory(
		trafficshaping.NewDialer(),
	)
	client := ndt5.NewClient(clientName, clientVersion, "https://mlab-sandbox.appspot.com")
	client.ProtocolFactory = protocolFactory
	out, err := client.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for ev := range out {
		t.Logf("%+v", ev)
	}
}

func TestIntegrationClientWSS(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	protocolFactory := ndt5.NewProtocolFactory5()
	protocolFactory.ConnectionsFactory = ndt5.NewWSConnectionsFactory(
		trafficshaping.NewDialer(),
		nil,
	)
	client := ndt5.NewClient(clientName, clientVersion, "https://mlab-sandbox.appspot.com")
	client.ProtocolFactory = protocolFactory
	out, err := client.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for ev := range out {
		t.Logf("%+v", ev)
	}
}
