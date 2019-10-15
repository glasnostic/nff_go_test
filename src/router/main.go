package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"

	"github.com/glasnostic/example/router/driver"
	"github.com/glasnostic/example/router/packet/handler"
)

var (
	nicName    string
	driverName string
	local      net.IP
	localMac   net.HardwareAddr
	client     net.IP
	clientMac  net.HardwareAddr
	server     net.IP
	serverMac  net.HardwareAddr
)

const (
	defaultDriver = "dpdk"
	defaultNIC    = "eth0"
)

func main() {
	log.Println("===== Example setup =====")
	setup()
	log.Println("===== Example start =====")

	drv, err := driver.New(driverName, nicName)
	mustSuccess(err, "Failed to create driver with error")
	hdl := handler.NewRewriter(localMac, clientMac, serverMac, local, client, server)

	log.Println("======= Start running driver =======")
	go drv.Run(hdl)

	// Terminate _Example_ when receiving SIGINT | SIGTERM
	sig := make(chan os.Signal)
	signal.Notify(sig, unix.SIGINT, unix.SIGTERM)
	<-sig
	log.Println("====== Example end ======")
}

func setup() {
	clientIPString := os.Getenv("CLIENT")
	client = net.ParseIP(clientIPString)
	mustHaveIP(client, "client ip")

	serverIPString := os.Getenv("SERVER")
	server = net.ParseIP(serverIPString)
	mustHaveIP(server, "server ip")

	driverName = os.Getenv("DRIVER")
	if driverName == "" {
		driverName = defaultDriver
	}

	nicName = os.Getenv("NIC")
	if nicName == "" {
		nicName = defaultNIC
	}

	mustSuccess(loadMAC(), "Failed to load MAC")
	mustSuccess(setRlimit(), "Failed to setrlimit")
}

func loadMAC() error {
	// get localIP and mac
	nic, err := net.InterfaceByName(nicName)
	if err != nil {
		return fmt.Errorf("given NIC %s must exist and be accessible", nicName)
	}
	localMac = nic.HardwareAddr
	local, err = getFirstIP(nic)
	if err != nil {
		return err
	}
	log.Printf("Using IP %v bound to nic %s", local, nic.Name)
	clientMac, err = parseMACAllowEmpty(os.Getenv("CLIENT_MAC"))
	if err != nil {
		return err
	}
	serverMac, err = parseMACAllowEmpty(os.Getenv("SERVER_MAC"))
	if err != nil {
		return err
	}
	return nil
}

func parseMACAllowEmpty(mac string) (net.HardwareAddr, error) {
	if mac == "" {
		return nil, nil
	} else {
		parsedMac, err := net.ParseMAC(mac)
		if err != nil {
			return nil, err
		}
		return parsedMac, nil
	}
}

func getFirstIP(nic *net.Interface) (net.IP, error) {
	addrs, err := nic.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			return v.IP, nil
		case *net.IPAddr:
			return v.IP, nil
		}
	}
	return nil, fmt.Errorf("no IP bound to nic %s", nic.Name)
}

func setRlimit() error {
	rLimit := &unix.Rlimit{
		Max: unix.RLIM_INFINITY,
		Cur: unix.RLIM_INFINITY,
	}
	return unix.Setrlimit(unix.RLIMIT_MEMLOCK, rLimit)
}

func mustSuccess(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func mustHaveIP(ip net.IP, msg string) {
	if ip == nil {
		log.Fatalf("%s must be given but missing!\n", msg)
	}
}
