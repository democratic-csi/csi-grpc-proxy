FROM golang:alpine as builder

RUN apk add --no-cache \
    git \
    ca-certificates

COPY ./src/ $GOPATH/src/csi-grpc-proxy/

WORKDIR $GOPATH/src/csi-grpc-proxy

RUN go get \
 && CGO_ENABLED=0 go build -o $GOPATH/bin

FROM alpine:3.15

COPY --from=builder /go/bin/csi-grpc-proxy /usr/bin/csi-grpc-proxy

CMD ["csi-grpc-proxy"]
