package ndt5

func NewProtocolNDT5Factory() ProtocolFactory {
	return new(protocolNDT5Factory)
}
