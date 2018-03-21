package config

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"gitlab.com/OpenWifiPortal/go-libs/clientAMQP"
	"net"
)

var (
	Config *ServiceConfig
)

//ServerConfig is the server config struct
type ServiceConfig struct {
	ClientAMQPConfig clientAMQP.ClientConfig
	NatOnly          bool
}

func GetMacAddr() (addr string) {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
				// Don't use random as we have a real address
				addr = i.HardwareAddr.String()
				break
			}
		}
	}
	return
}

func GetId() (uuid string) {
	uuid = fmt.Sprintf("%.6x", sha256.Sum256([]byte(GetMacAddr())))
	return
}
