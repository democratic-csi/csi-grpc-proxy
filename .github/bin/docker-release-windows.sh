#!/bin/bash

echo "$DOCKER_PASSWORD" | docker login         -u "$DOCKER_USERNAME" --password-stdin
echo "$GHCR_PASSWORD"   | docker login ghcr.io -u "$GHCR_USERNAME"   --password-stdin

export DOCKER_ORG="democraticcsi"
export DOCKER_PROJECT="csi-grpc-proxy"
export DOCKER_REPO="${DOCKER_ORG}/${DOCKER_PROJECT}"

export GHCR_REPO="ghcr.io/${GITHUB_REPOSITORY}"

export MANIFEST_NAME="windows"

if [[ -n "${IMAGE_TAG}" ]]; then
  # create local manifest to work with
  buildah manifest create "${MANIFEST_NAME}"
  
  # all all the existing linux data to the manifest
  buildah manifest add "${MANIFEST_NAME}" --all "docker.io/${DOCKER_REPO}:${IMAGE_TAG}"
  
  # import pre-built images
  buildah pull docker-archive:window-1809.tar
  buildah pull docker-archive:windows-ltsc2022.tar

  # add pre-built images to manifest
  buildah manifest add "${MANIFEST_NAME}" windows:1809
  buildah manifest add "${MANIFEST_NAME}" windows:ltsc2022

  # push manifest
  buildah manifest push --all "${MANIFEST_NAME}" docker://docker.io/${DOCKER_REPO}:${IMAGE_TAG}
  buildah manifest push --all "${MANIFEST_NAME}" docker://${GHCR_REPO}:${IMAGE_TAG}
else
  :
fi
