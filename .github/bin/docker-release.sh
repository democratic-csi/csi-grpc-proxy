#!/bin/bash

echo "$DOCKER_PASSWORD" | docker login         -u "$DOCKER_USERNAME" --password-stdin
echo "$GHCR_PASSWORD"   | docker login ghcr.io -u "$GHCR_USERNAME"   --password-stdin

export DOCKER_ORG="democraticcsi"
export DOCKER_PROJECT="csi-grpc-proxy"
export DOCKER_REPO="${DOCKER_ORG}/${DOCKER_PROJECT}"

export GHCR_REPO="ghcr.io/${GITHUB_REPOSITORY}"

if [[ -n "${IMAGE_TAG}" ]]; then
  docker buildx build --progress plain --pull --push --platform "${DOCKER_BUILD_PLATFORM}" -t ${DOCKER_REPO}:${IMAGE_TAG} -t ${GHCR_REPO}:${IMAGE_TAG} .
else
  :
fi
