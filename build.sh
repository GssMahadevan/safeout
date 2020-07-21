#!/bin/bash

# Esnure go is installed
go version >/dev/null 2>&1
if [ $? -ne 0 ];then
	echo Please install golang and put go-binary  in PATH
	exit
fi
		
d=prebuilt-binary
mkdir -p $d >/dev/null 2>&1
set -x
# Build for  native host processor
go build -ldflags="-s -w"  -o $d/safeout safeout.go 

# Cross Build for ARM (raspberry)
GOARCH=arm GOARM=7 go build -ldflags="-s -w"  -o $d/safeout.a7 safeout.go 



#check for upx
upx --help >/dev/null 2>&1
if [ $? -ne 0 ];then
	echo Please install upx for reducing the size of binary and put upx-binary  in PATH
	exit
fi

upx $d/safeout
upx $d/safeout.a7
	