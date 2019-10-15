# NFF-Go Test Environment Using Two VMs

This example is meant as a test to check the stability and 
performance of [nff-go](https://github.com/intel-go/nff-go) running in different 
VM environments.

## Test Setup

We have one VM (called _Pod_) 
running two server processes: a simple HTTP server and [iperf3](https://iperf.fr/). 
From the same VM we also start client processes that connect to the server processes. 

Usually this traffic would go via the loopback device, but instead we want the traffic to go via
an interface owned by nff-go.

For this we start another VM (called _Router_). It's purpose is to swap source 
and destination IP of each incoming packet.

If the _Pod_ is then connecting to the _Router_, the traffic goes back to itself. 
The forward flow (`FF`) and backward flow (`BF`) looks like this:
 
```
+-------------------+          +-------------------+
|                   |    FF    |                   |
|                   +--------->+                   |
|       Pod         |          |       Router      |
|                   |    BF    |                   |
|                   |<---------|                   |
+-------------------+          +-------------------+
```

## Run the example

### Preparations 

First you'll have to checkout this code in both test VMs which must both have Docker installed.

### Router VM 

On the _Router_ VM you have to first build the `router` docker container and initialize hugepages:

```bash
$ ./scripts/huge  
$ docker build -t router -f ./src/router/Dockerfile ./src/router
```

Then you can start the router process via:

```bash
$ docker run -d --privileged --network=host -e "CLIENT=$POD_IP" -e "SERVER=$POD_IP" -e "DPDK_DRIVER=igb_uio" -e "NIC=eth1" router
```

`POD_IP` is the IP of the `Pod VM` and the env variable `DPDK_DRIVER` is configuring the kernel module
used by nff-go (in the example it's `igb_uio`). The used network interface is configured by changing the env variable `NIC`, here it's `eth1`. 

> **Hint**: The `docker run` call returns the container id. You can check the logs with `docker logs <container_id>`. 

> **Note**: The router also works with two _Pod_ VMs. One is then the _Client_ and the other the _Server_ VM.
> To simplify the setup we only use one _Pod_ VM. This means the router parameters `CLIENT` and `SERVER` are all
> set to the same IP, the one of the _Pod_ VM. 

### Pod VM

On the _Pod_ VM, you have to build the `pod` docker container first:

```bash
$ docker build -t pod -f ./src/pod/Dockerfile ./src/pod
```

After that, you can first run the server processes on the _Pod_ VM:

```bash
$ docker run -d -P --privileged --network=host pod service.sh
```

Then you can run the tests via:
```bash
$ docker run -it --privileged --network=host pod client.sh $ROUTER_IP
```

Where `ROUTER_IP` is the IP of the _Router_ VM.  

