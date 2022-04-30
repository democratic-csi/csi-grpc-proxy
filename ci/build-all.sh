#!/bin/bash

set -e
#set -x

mkdir -p builds
cd src

export CGO_ENABLED=0
export BINARY_NAME="csi-grpc-proxy"

IMAGE_BUILD_LINUX_ARCHES=("amd64" "arm" "arm64" "ppc64le" "s390x")
IMAGE_BUILD_WINDOWS_ARCHES=("amd64")

if [[ -z "${BINARY_VERSION}" ]]; then
  if [[ -z "${GITHUB_REF}" ]]; then
    : "${BINARY_VERSION:=dev}"
  else
    if [[ $GITHUB_REF == refs/tags/* ]]; then
      BINARY_VERSION=${GITHUB_REF#refs/tags/}
    else
      BINARY_VERSION=${GITHUB_REF#refs/heads/}
    fi
  fi
fi

IMAGE_BUILD=0
if [[ ${BINARY_VERSION} == "docker-image-build" ]]; then
  IMAGE_BUILD=1
fi


for target in $(go tool dist list);do
  export GOOS=$(echo $target | cut -d "/" -f1)
  export GOARCH=$(echo $target | cut -d "/" -f2)

  BUILD=0
  suffix=""
  if [ "${GOOS}" == "freebsd" ];then
    if [[ ${IMAGE_BUILD} -eq 1 ]]; then
      continue;
    fi
    BUILD=1
  fi
  if [ "${GOOS}" == "darwin" ];then
    if [[ ${IMAGE_BUILD} -eq 1 ]]; then
      continue;
    fi
    BUILD=1
  fi
  if [ "${GOOS}" == "linux" ];then
    if [[ ${IMAGE_BUILD} -eq 1 ]]; then
      if [[ ! " ${IMAGE_BUILD_LINUX_ARCHES[@]} " =~ " ${GOARCH} " ]]; then
          continue;
      fi
    fi
    BUILD=1
  fi
  if [ "${GOOS}" == "windows" ];then
    suffix=".exe"
    if [[ ${IMAGE_BUILD} -eq 1 ]]; then
      if [[ ! " ${IMAGE_BUILD_WINDOWS_ARCHES[@]} " =~ " ${GOARCH} " ]]; then
          continue;
      fi
    fi
    BUILD=1
  fi

  if [[ "${BUILD}" -ne 1 ]];then
    continue
  fi

  binary="${BINARY_NAME}-${BINARY_VERSION}-${GOOS}-${GOARCH}${suffix}"
  echo "building ${target} as ${binary}"
  go build -o ../builds/${binary}
done
