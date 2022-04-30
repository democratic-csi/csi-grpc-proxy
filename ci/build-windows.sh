#!/bin/bash

set -e
#set -x

mkdir -p bin
cd src

export CGO_ENABLED=0
export GOOS="windows"
export GOARCH="amd64"
export BINARY_NAME="csi-grpc-proxy.exe"

go build -o ../bin/${BINARY_NAME}
