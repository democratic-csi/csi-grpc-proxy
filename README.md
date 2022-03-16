# csi-grpc-proxy

`csi-grpc-proxy` is a Go reverse proxy over UDS and/or H2C.

The express purpose is to alleviate the issues associated with the strict
handling of the `:authority` header by the various `grpc` server
implementations. Several `grpc` clients send/set the `uds` file path as the
`host` / `:authority` header when connecting via `uds`, which is (seemingly)
non-conformat to the spec, therefore server implementations reject the request
outright (ie: `nginx`, `envoy`, `nodejs`, and anything `nghttp2`-based).

This proxy always overrides the `host` / `:authority` header as `localhost`
before sending the request upstream. Additionally the `x-forwarded-host` header
is set to the original value, otherwise the request is unaltered.

- https://github.com/grpc/grpc-go/pull/3730/files
- https://github.com/dotnet/aspnetcore/issues/18522
- https://github.com/nodejs/help/issues/3422

## usage

required environment vars:

- `BIND_TO`: sets the listening url as http or UDS address
- `PROXY_TO`: sets the upstream proxy as http or UDS address

## docker

```
docker pull democraticcsi/csi-grpc-proxy
docker run --rm -d \
    -e BIND_TO=unix:///csi-data/csi.sock \
    -e PROXY_TO=unix:///tmp/csi.sock \
    democraticcsi/csi-grpc-proxy
```

# development

```
cd src/
go mod init csi-grpc-proxy
go get
BIND_TO="unix:///tmp/csi.sock" PROXY_TO="unix:///tmp/csi.sock.internal" go run ./main.go

go fmt ./

CGO_ENABLED=0 go build

go tool dist list
GOOS=linux GOARCH=arm64 go build -o csi-grpc-proxy

# upgrade go version
go mod edit -go 1.18
# edit Dockerfile as appropriate
# edit github-release.yaml as appropriate
```

# links

- https://github.com/Zetanova/grpc-proxy
- https://pkg.go.dev/net/http#Request
- https://opensource.com/article/21/1/go-cross-compiling
