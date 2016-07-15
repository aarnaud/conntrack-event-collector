package conntrack

import (
	"net"
)

type Flow struct {
	Timestamp int
	Type      string
	Id        int
	Original  Meta
	Reply     Meta
	UNREPLIED bool
	ASSURED   bool
}

type Meta struct {
	Layer3  Layer3
	Layer4  Layer4
	Counter Counter
}

type Layer3 struct {
	Protonum  int
	Protoname string
	Src       net.IP
	Dst       net.IP
}

type Layer4 struct {
	Protonum  int
	Protoname string
	Sport     int
	Dport     int
}

type Counter struct {
	Packets int
	Bytes   int
}
