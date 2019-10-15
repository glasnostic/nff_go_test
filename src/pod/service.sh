#!/usr/bin/env bash

# start iperf3 server as daemon
iperf3 -p 3091 -s -D

# start simple http server
python -m SimpleHTTPServer 8080
