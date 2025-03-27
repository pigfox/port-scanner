#!/bin/bash 
set -e
set -x
clear 

 ./portscanner -start=0.0.0.0 -end=255.255.255.255 -ports=25