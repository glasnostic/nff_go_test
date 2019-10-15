package nff

import (
	"log"
	"sync"

	"github.com/glasnostic/example/router/packet"
)

type nff struct {
	runner nffRunner

	stop chan struct{}

	*sync.WaitGroup
}

func New(nicName string) (*nff, error) {
	runner, err := newRunner(nicName)
	if err != nil {
		return nil, err
	}
	res := &nff{
		runner:    runner,
		stop:      make(chan struct{}),
		WaitGroup: &sync.WaitGroup{},
	}

	return res, nil
}

func (n *nff) Run(packetHandler packet.Handler) {
	log.Println("NFF Run called")
	n.Add(1)
	defer n.Done()

	var pktMeta packet.Metadata

	for {
		select {
		case <-n.stop:
			return
		case pkt := <-n.runner.Read():
			pktMeta.Reset()
			pktMeta.Packet = pkt
			action, err := packetHandler.Handle(&pktMeta)
			if err != nil {
				log.Println("NFF drops packet due to error:", err)
				n.runner.Drop()
				continue
			}
			n.handle(action, pktMeta.Packet)
		}
	}
}

func (n *nff) Suspend() {
	n.stop <- struct{}{} // send close signal
	n.Wait()             // wait current Run finished
}

func (n *nff) Close() {
	log.Println("Stopping NFF NetworkDriver")
	defer log.Println("Stopped NFF NetworkDriver")

	close(n.stop)    // close stop channel
	n.runner.Close() // close runner
	n.Wait()         // wait current Run finished
}

func (n *nff) handle(action packet.Action, pkt Packet) {
	switch action {
	case packet.Drop:
		// drop this packet
		n.runner.Drop()
	case packet.Pass, packet.Rewrite, packet.New:
		// pass this packet to TX
		n.runner.Write(pkt)
	default:
		// drop this packet
		log.Println("NFF drops packet due to un-expected action returned!")
		n.runner.Drop()
	}
}
