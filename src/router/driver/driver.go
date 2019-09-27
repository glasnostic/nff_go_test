package driver

import (
	"github.com/glasnostic/example/router/driver/nff"
	"github.com/glasnostic/example/router/packet"
)

const (
	nffName = "dpdk"
)

type Runner interface {
	Run(packetHandler packet.Handler)
}

func New(driverName, nicName string) (Runner, error) {
	switch driverName {
	default:
		return nff.New(nicName)
	}
}
