package nff

type Packet []byte

type nffRunner interface {
	Read() <-chan Packet
	Write(data Packet)
	Drop()

	Close()
}
