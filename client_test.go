package ndt5_test

import (
	"context"
	"testing"

	"github.com/bassosimone/ndt5-client-go"
	"github.com/bassosimone/ndt5-client-go/internal/trafficshaping"
)

const (
	clientName    = "ndt5-client-go-testing"
	clientVersion = "0.1.0"
)

func TestIntegrationClientRaw(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	client := ndt5.NewClient(clientName, clientVersion)
	client.ConnectionsFactory = ndt5.NewRawConnectionsFactory(
		trafficshaping.NewDialer(),
	)
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
	client := ndt5.NewClient(clientName, clientVersion)
	client.ConnectionsFactory = ndt5.NewWSConnectionsFactory(
		trafficshaping.NewDialer(),
	)
	out, err := client.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for ev := range out {
		t.Logf("%+v", ev)
	}
}
