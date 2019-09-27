// +build nff

package nff

import (
	"reflect"
	"sync"
	"unsafe"

	"crimea/gateway/router/packet"

	libpacket "github.com/intel-go/nff-go/packet"
)

const (
	PacketDrop   = false
	PacketAccept = true
)

type nffPacketLinux struct {
	raw     packet.Packet
	pkt     *libpacket.Packet
	verdict chan bool
	once    sync.Once
}

func newNFFPacket(pkt *libpacket.Packet) *nffPacketLinux {
	var raw []byte
	slice := (*reflect.SliceHeader)(unsafe.Pointer(&raw))
	slice.Data = uintptr(unsafe.Pointer(pkt.Ether))
	slice.Len = int(pkt.GetPacketLen())
	slice.Cap = int(pkt.GetPacketLen())

	res := &nffPacketLinux{
		raw:     raw,
		pkt:     pkt,
		verdict: make(chan bool),
	}

	return res
}

func (n *nffPacketLinux) SetVerdict(value bool) {
	n.once.Do(func() {
		n.verdict <- value
	})
}

func (n *nffPacketLinux) RawBuffer() packet.Packet {
	return n.raw
}
