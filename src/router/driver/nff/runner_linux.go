package nff

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"unsafe"

	"github.com/intel-go/nff-go/devices"
	"github.com/intel-go/nff-go/flow"
	"github.com/intel-go/nff-go/low"
	libpacket "github.com/intel-go/nff-go/packet"
)

// nff-go changes the port type from uint8 to uint16
type portType = uint16

const (
	defaultPort portType = 0
)

type nffRunnerLinux struct {
	ifName         string
	port           portType
	device         devices.Device
	current        *nffPacketLinux
	incomming      chan Packet
	originalDriver string
	network        string
	hwaddr         net.HardwareAddr
}

func newRunner(nicName string) (*nffRunnerLinux, error) {
	runner := &nffRunnerLinux{
		ifName:    nicName,
		incomming: make(chan Packet),
	}

	if err := runner.register(); err != nil {
		return nil, err
	}

	if err := runner.init(); err != nil {
		return nil, err
	}

	return runner, nil
}

func (n *nffRunnerLinux) Read() <-chan Packet {
	return n.incomming
}

func (n *nffRunnerLinux) Write(data Packet) {
	if n.packetNotChanged(data) {
		n.current.SetVerdict(PacketAccept)
		return
	}

	// Try to overwrite the original packet
	if ok := n.current.pkt.PacketBytesChange(0, []byte(data)); ok {
		n.current.SetVerdict(PacketAccept)
		return
	}

	// Original packet doesn't have sufficient size
	n.dropOldAndSendNewPacket(data)
}

func (n *nffRunnerLinux) Drop() {
	n.current.SetVerdict(PacketDrop)
}

func (n *nffRunnerLinux) Close() {
	log.Println("Stop NFF-Go SystemStop")
	flow.SystemStop()
	log.Println("Setup Systemd ExecStartPost config for DPDK PMD driver unregister")
	if err := n.unregister(); err != nil {
		log.Printf("Couldn't setupUnregister DPDK: %s\n", err.Error())
	}
}

func (n *nffRunnerLinux) register() error {
	var err error
	// 1. Init Binder by nic name
	n.device, err = devices.New(n.ifName)
	if err != nil {
		return err
	}

	// 2. Store originalDriver and network for recovery
	n.originalDriver, err = n.device.CurrentDriver()
	if err != nil {
		return err
	}
	n.hwaddr, n.network, err = getNetworkInfo(n.ifName)
	if err != nil {
		return err
	}

	// 3. check which dpdk driver we want to use
	var dpdkDriver string
	if driver := os.Getenv("DPDK_DRIVER"); driver == "" {
		dpdkDriver = devices.FindDefaultDpdkDriver(n.ifName)
	} else {
		dpdkDriver = driver
	}

	log.Printf("Binding PMD driver %s to NIC %s\n", n.ifName, dpdkDriver)

	return n.device.Bind(dpdkDriver)
}

func (n *nffRunnerLinux) unregister() error {
	// 1. Bind original driver
	// sudo ./devbind -b virtio-pci 0000:00:04.0
	if err := n.device.Bind(n.originalDriver); err != nil {
		return fmt.Errorf("failed to bind device: %s", err.Error())
	}

	// 2. Bring interface up
	cmd := exec.Command("ip", "link", "set", "dev", n.ifName, "up")
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to bring interface %s up: %s", n.ifName, err.Error())
	}

	// 3. Restore IP configuration
	// ip address replace [ip]/[mask-digits] dev [nic]
	log.Println("Try to restore NIC configuration")
	cmd = exec.Command("ip", "address", "replace", n.network, "dev", n.ifName)
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to configure network interface: %s", err.Error())
	}

	return nil
}

func (n *nffRunnerLinux) init() error {
	log.Println("Initiating nff-go flow system")
	config := &flow.Config{
		HWTXChecksum: true,
	}
	flow.SystemInit(config)
	log.Println("Initiated nff-go flow system")

	// NOTE: Must be called after flow.SystemInit
	n.port = getEthPort(n.hwaddr)

	// Main flow from receiver (port 0)
	log.Println("Setting receiver on port", n.port)

	mainFlow, err := flow.SetReceiver(n.port)
	if err != nil {
		return err
	}
	log.Println("Set receiver on port", n.port)

	log.Println("Setting drop handler on port", n.port)
	// Packet handler: rewrite and filter packet
	if err := flow.SetHandlerDrop(mainFlow, n.handle, nil); err != nil {
		return err
	}
	log.Println("Set drop handler on port", n.port)

	// Send packet back to port 0
	log.Println("Setting sender on port", n.port)
	if err := flow.SetSender(mainFlow, n.port); err != nil {
		return err
	}
	log.Println("Set sender on port", n.port)

	go func() {
		log.Println("Starting flow system")
		if err := flow.SystemStart(); err != nil {
			log.Println("Couldn't start NFF system:", err.Error())
		}
	}()

	return nil
}

func (n *nffRunnerLinux) handle(pkt *libpacket.Packet, _ flow.UserContext) bool {
	if pkt == nil {
		log.Println("packet is empty - dropping it.")
		return PacketDrop
	}
	nffpacket := newNFFPacket(pkt)
	n.current = nffpacket
	n.incomming <- nffpacket.RawBuffer()
	return <-nffpacket.verdict
}

func (n *nffRunnerLinux) dropOldAndSendNewPacket(data Packet) {
	// Drop the original one
	n.current.SetVerdict(PacketDrop)

	// Create a new packet
	pkt, err := libpacket.NewPacket()
	if err != nil {
		log.Println("Couldn't create a new packet:", err.Error())
		return
	}

	// Fill the new created packet
	if ok := libpacket.GeneratePacketFromByte(pkt, []byte(data)); !ok {
		log.Println("Couldn't generate a new packet from bytes")
	}

	// Send the new created packet
	if ok := pkt.SendPacket(n.port); !ok {
		log.Printf("Failed to send packet to port(%d)\n", n.port)
		return
	}
}

func (n *nffRunnerLinux) packetNotChanged(data Packet) bool {
	// XXX Dirty check
	var buf1, buf2 []byte = []byte(n.current.raw), []byte(data)
	return len(buf1) > 0 && len(buf2) > 0 && unsafe.Pointer(&buf1[0]) == unsafe.Pointer(&buf2[0])
}

func getNetworkInfo(nicName string) (net.HardwareAddr, string, error) {
	iface, err := net.InterfaceByName(nicName)
	if err != nil {
		return net.HardwareAddr(nil), "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return net.HardwareAddr(nil), "", err
	}

	if len(addrs) == 0 {
		return net.HardwareAddr(nil), "", fmt.Errorf("interface does not have address")
	}

	return iface.HardwareAddr, addrs[0].String(), nil
}

func getEthPort(hwaddr net.HardwareAddr) portType {
	for p := 0; p < low.GetPortsNumber(); p++ {
		portMACAddress := low.GetPortMACAddress(portType(p))
		if hardwareAddressEqual(hwaddr, net.HardwareAddr(portMACAddress[:])) {
			return portType(p)
		}
	}
	return defaultPort
}

func hardwareAddressEqual(addr1, addr2 net.HardwareAddr) bool {
	for i, j := 0, 0; i < len(addr1) && j < len(addr2); i, j = i+1, j+1 {
		if addr1[i] != addr2[j] {
			return false
		}
	}
	return true
}
