package conntrack

import (
	"net"
)

type Flow struct {
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"`
	Id        int    `json:"id"`
	Original  Meta   `json:"original"`
	Reply     Meta   `json:"reply"`
	UNREPLIED bool
	ASSURED   bool
}

type Meta struct {
	Layer3  Layer3  `json:"layer3"`
	Layer4  Layer4  `json:"layer4"`
	Counter Counter `json:"counter"`
}

type Layer3 struct {
	Protonum  int    `json:"protonum"`
	Protoname string `json:"protoname"`
	Src       net.IP `json:"src"`
	Dst       net.IP `json:"dst"`
}

type Layer4 struct {
	Protonum  int    `json:"protonum"`
	Protoname string `json:"protoname"`
	Sport     int    `json:"sport"`
	Dport     int    `json:"dport"`
}

type Counter struct {
	Packets int `json:"packets"`
	Bytes   int `json:"bytes"`
}
