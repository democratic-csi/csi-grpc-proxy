# local testing
# docker build --pull -t foobar --build-arg TARGETPLATFORM="linux/amd64" .
# docker run --rm -ti foobar

FROM alpine:3.15 as builder

# https://github.com/BretFisher/multi-platform-docker-build
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETVARIANT
RUN printf "I'm building for TARGETPLATFORM=${TARGETPLATFORM}" \
    && printf ", TARGETARCH=${TARGETARCH}" \
    && printf ", TARGETVARIANT=${TARGETVARIANT} \n" \
    && printf "With uname -s : " && uname -s \
    && printf "and  uname -m : " && uname -mm

RUN mkdir -p /binary; mkdir -p /builds;

COPY builds /builds

RUN ls -l /builds

# linux/amd64,linux/arm64,linux/arm/v7,linux/s390x,linux/ppc64le
RUN case ${TARGETPLATFORM} in \
        "linux/amd64")   BINARY=csi-grpc-proxy-docker-image-build-linux-amd64   ;; \
        "linux/arm64")   BINARY=csi-grpc-proxy-docker-image-build-linux-arm64   ;; \
        "linux/arm/v7")  BINARY=csi-grpc-proxy-docker-image-build-linux-arm     ;; \
        "linux/ppc64le") BINARY=csi-grpc-proxy-docker-image-build-linux-ppc64le ;; \
        "linux/s390x")   BINARY=csi-grpc-proxy-docker-image-build-linux-s390x   ;; \
    esac \
  && cp /builds/${BINARY} /binary/csi-grpc-proxy \
  && chmod +x /binary/csi-grpc-proxy

RUN ls -l /binary

FROM alpine:3.15

LABEL org.opencontainers.image.source https://github.com/democratic-csi/csi-grpc-proxy

COPY --from=builder /binary/csi-grpc-proxy /usr/bin/csi-grpc-proxy

CMD ["csi-grpc-proxy"]
