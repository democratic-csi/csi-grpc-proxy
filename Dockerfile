FROM golang:1.18-alpine3.15 as builder

RUN apk add --no-cache \
    build-base \
    git \
    ca-certificates

COPY ./src/ $GOPATH/src/csi-grpc-proxy/

WORKDIR $GOPATH/src/csi-grpc-proxy

RUN go get \
 && CGO_ENABLED=0 go build -o $GOPATH/bin

FROM alpine:3.15

LABEL org.opencontainers.image.source https://github.com/democratic-csi/csi-grcp-proxy

COPY --from=builder /go/bin/csi-grpc-proxy /usr/bin/csi-grpc-proxy

CMD ["csi-grpc-proxy"]
