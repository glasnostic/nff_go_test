#!/usr/bin/env bash

BASEDIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
source $BASEDIR/message.sh

function curl_test() {
  local target=$1
  message "curl test to $target"
  curl  --noproxy $target --connect-timeout 10 -m 60 -v $target:8080
}

function iperf_test() {
  local target=$1
  message "iperf test to $target"
  iperf3 -p 3091 -c $target -M 1450 -t 86400 # run for one day
}

function ping_test() {
  local target=$1
  message "ping test to $target"
  ping -c 4 $target
}

ping_test $1
curl_test $1
iperf_test $1
