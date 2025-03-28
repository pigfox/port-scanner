#!/bin/bash 
set -e
set -x          
clear

go build -o portscanner email.go portscanner.go structs.go helpers.go