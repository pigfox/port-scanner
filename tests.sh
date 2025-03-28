#!/bin/bash 
set -e
set -x       
clear   

go test -v -cover -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. -benchmem
ls -lh mem.prof
ls -lh cpu.prof
go tool pprof -alloc_space -top mem.prof