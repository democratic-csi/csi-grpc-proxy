#!/bin/bash

echo "$DOCKER_PASSWORD" | docker login         -u "$DOCKER_USERNAME" --password-stdin
echo "$GHCR_PASSWORD"   | docker login ghcr.io -u "$GHCR_USERNAME"   --password-stdin

export DOCKER_ORG="democraticcsi"
export DOCKER_PROJECT="csi-grpc-proxy"
export DOCKER_REPO="${DOCKER_ORG}/${DOCKER_PROJECT}"

export GHCR_REPO="ghcr.io/${GITHUB_REPOSITORY}"

export MANIFEST_NAME="csi-grpc-proxy-combined:${IMAGE_TAG}"

if [[ -n "${IMAGE_TAG}" ]]; then
  # create local manifest to work with
  buildah manifest rm "${MANIFEST_NAME}" || true
  buildah manifest create "${MANIFEST_NAME}"
  
  # all all the existing linux data to the manifest
  buildah manifest add "${MANIFEST_NAME}" --all "docker.io/${DOCKER_REPO}:${IMAGE_TAG}"
  buildah manifest inspect "${MANIFEST_NAME}"
  
  # import pre-built images
  buildah pull docker-archive:csi-grpc-proxy-windows-1809.tar
  buildah pull docker-archive:csi-grpc-proxy-windows-ltsc2022.tar

  # add pre-built images to manifest
  buildah manifest add "${MANIFEST_NAME}" csi-grpc-proxy-windows:${GITHUB_RUN_ID}-1809
  buildah manifest add "${MANIFEST_NAME}" csi-grpc-proxy-windows:${GITHUB_RUN_ID}-ltsc2022
  buildah manifest inspect "${MANIFEST_NAME}"

  # push manifest
  buildah manifest push --all "${MANIFEST_NAME}" docker://docker.io/${DOCKER_REPO}:${IMAGE_TAG}
  buildah manifest push --all "${MANIFEST_NAME}" docker://${GHCR_REPO}:${IMAGE_TAG}

  # cleanup
  buildah manifest rm "${MANIFEST_NAME}" || true
else
  :
fi
