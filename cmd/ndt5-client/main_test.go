package main

import (
	"os"
	"testing"
)

func TestIntegrationMainRaw(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	origValue := flagProtocol.Value
	flagProtocol.Value = "ndt5"
	defer func() {
		flagProtocol.Value = origValue
	}()
	main()
}

func TestIntegrationMainWSS(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	origValue := flagProtocol.Value
	flagProtocol.Value = "ndt5+wss"
	defer func() {
		flagProtocol.Value = origValue
	}()
	main()
}

func TestMain(m *testing.M) {
	// Do not use production servers for CI.
	*flagHostname = "ndt-mlab4-lga0t.mlab-sandbox.measurement-lab.org"
	*flagThrottle = 1 << 18 // be gentle on CI servers
	code := m.Run()
	os.Exit(code)
}
