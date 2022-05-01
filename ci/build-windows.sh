#!/bin/bash

set -e
#set -x

mkdir -p builds
cd src

export CGO_ENABLED=0
export GOOS="windows"
export GOARCH="amd64"
export BINARY_NAME="csi-grpc-proxy-docker-image-build-windows-amd64.exe"

go build -o ../builds/${BINARY_NAME}
