package main

import (
	"os"
	"testing"
)

func TestIntegrationMainRaw(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	origValue := flagTransport.Value
	flagTransport.Value = "raw"
	defer func() {
		flagTransport.Value = origValue
	}()
	main()
}

func TestIntegrationMainWSS(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	origValue := flagTransport.Value
	flagTransport.Value = "wss"
	defer func() {
		flagTransport.Value = origValue
	}()
	main()
}

func TestMain(m *testing.M) {
	*flagThrottle = true // be gentle on CI servers
	code := m.Run()
	os.Exit(code)
}
