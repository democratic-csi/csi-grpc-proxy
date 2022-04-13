#!/bin/bash

set -e
#set -x

mkdir -p builds
cd src

export CGO_ENABLED=0
export BINARY_NAME="csi-grpc-proxy"

if [[ -z "${GITHUB_REF}" ]]; then
  : "${BINARY_VERSION:=dev}"
else
  if [[ $GITHUB_REF == refs/tags/* ]]; then
    BINARY_VERSION=${GITHUB_REF#refs/tags/}
  else
    BINARY_VERSION=${GITHUB_REF#refs/heads/}
  fi
fi

for target in $(go tool dist list);do
  export GOOS=$(echo $target | cut -d "/" -f1)
  export GOARCH=$(echo $target | cut -d "/" -f2)

  BUILD=0
  suffix=""
  if [ "${GOOS}" == "freebsd" ];then
    BUILD=1
  fi
  if [ "${GOOS}" == "darwin" ];then
    BUILD=1
  fi
  if [ "${GOOS}" == "linux" ];then
    BUILD=1
  fi
  if [ "${GOOS}" == "windows" ];then
    suffix=".exe"
    BUILD=1
  fi

  if [[ "${BUILD}" -ne 1 ]];then
    continue
  fi

  binary="${BINARY_NAME}-${BINARY_VERSION}-${GOOS}-${GOARCH}${suffix}"
  echo "building ${target} as ${binary}"
  go build -o ../builds/${binary}
done
