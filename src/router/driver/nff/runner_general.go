// +build !linux

package nff

import "errors"

var (
	ErrNetworkDriverNotAvailable = errors.New("Not available")
)

func newRunner(ifName string) (nffRunner, error) {
	return nil, ErrNetworkDriverNotAvailable
}
