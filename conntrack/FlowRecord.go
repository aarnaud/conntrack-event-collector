package conntrack

import (
	"fmt"
	"net"
	"time"
)

type FlowRecord struct {
	SPort       int
	DPort       int
	Bytes       uint64
	Packets     uint64
	TS          time.Time
	Protocol    string
	Source      net.IP
	Destination net.IP
	State       string
	Unreplied   bool
	Assured     bool
	TTL         uint64
}

func (flow FlowRecord) String() string {
	return fmt.Sprintf("%s %s:%d -> %s:%d", flow.TS.Format("2006-01-02 15:04:05"), flow.Source, flow.SPort, flow.Destination, flow.DPort)
}
