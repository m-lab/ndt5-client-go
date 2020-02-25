package emitter

// ValueUnitPair represents a {"Value": ..., "Unit": ...} pair.
type ValueUnitPair struct {
	Value float64
	Unit  string
}

// Summary is a struct containing the values displayed to the user at
// the end of an ndt5 test.
type Summary struct {
	// ServerFQDN is the FQDN of the server used for this test.
	ServerFQDN string

	// ServerIP is the IP address of the server.
	ServerIP string

	// ClientIP is the IP address of the client.
	ClientIP string

	// DownloadUUID is the UUID of the download test.
	DownloadUUID string

	// Download is the download speed, in Mbit/s. This is measured at the
	// receiver.
	Download ValueUnitPair

	// Upload is the upload speed, in Mbit/s. This is measured at the sender.
	Upload ValueUnitPair

	// DownloadRetrans is the retransmission rate. This is based on the TCPInfo
	// values provided by the server during a download test.
	DownloadRetrans ValueUnitPair

	// MinRTT is the minimum round-trip time reported by the server in the
	// last Measurement of a download test, in milliseconds.
	MinRTT ValueUnitPair
}

// NewSummary returns a new Summary struct for a given FQDN.
func NewSummary(FQDN string) *Summary {
	return &Summary{
		ServerFQDN: FQDN,
	}
}
