#!/bin/bash 
set -e
set -x
clear 
./build.sh
 #./portscanner -start=0.0.0.0 -end=255.255.255.255 -ports=25
 ./portscanner -start=192.168.1.1 -end=192.168.1.255 -ports=22,80,443,8080 -timeout=1s -concurrent=100