#!/bin/bash 
set -e
set -x       
clear   

go test -v -cover -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.