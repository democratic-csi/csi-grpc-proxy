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
  - `unix:///path/to/socket`
  - `npipe:////./pipe/foo`
  - `tcp://localhost:5216`
  - `tcp://0.0.0.0:5216`
  - `tcp://:5216`
- `PROXY_TO`: sets the upstream proxy as http or UDS address
  - `unix:///path/to/socket`
  - `npipe:////./pipe/csi.sock`
  - `tcp://localhost:5216`
- `REWRITE_HOST`: enables host header rewriting (primary purpose of the proxy)
  - `0`: disabled
  - `1`: enabled, default, unconditionally rewrites host value to
    `REWRITE_HOST_HOSTNAME`
  - `2`: enabled, conditionally rewrites host value to `REWRITE_HOST_HOSTNAME`
    only if the original value is non-compliant
- `REWRITE_HOST_HOSTNAME`: what hostname to use for the rewrite
  - `localhost`: default, compliant values are `host[:port]`
- `PROXY_TO_INITIAL_TIMEOUT`: how long (seconds) to wait for the upstream proxy
  to be available (binding happens _after_ this and only if the upstream
  becomes available before the timeout is reached)

  - `60`: default
  - `0`: disable

You can review notes here for appropriate `BIND_TO` and `PROXY_TO` syntax
https://pkg.go.dev/net#Dial

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
BIND_TO="unix:///tmp/csi.sock" PROXY_TO="unix:///tmp/csi.sock.internal" go run .

# add dep
go get github.com/Microsoft/go-winio

# format
go fmt ./

# build
CGO_ENABLED=0 go build

go tool dist list
GOOS=linux GOARCH=arm64 go build -o csi-grpc-proxy

# upgrade go version
go mod edit -go 1.18
# edit Dockerfile as appropriate
# edit github-release.yaml as appropriate
```

# TODO

- graceful shutdown (handle signals and terminate properly in-flight requests)
  - https://rafallorenz.com/go/handle-signals-to-graceful-shutdown-http-server/
  - https://medium.com/honestbee-tw-engineer/gracefully-shutdown-in-go-http-server-5f5e6b83da5a
  - https://gist.github.com/embano1/e0bf49d24f1cdd07cffad93097c04f0a
- https://github.com/golang/go/issues/33357

# links

- https://github.com/Zetanova/grpc-proxy
- https://pkg.go.dev/net/http#Request
- https://opensource.com/article/21/1/go-cross-compiling
- https://www.digitalocean.com/community/tutorials/building-go-applications-for-different-operating-systems-and-architectures
- https://github.com/gesellix/go-npipe
- https://github.com/gesellix/go-npipe/blob/master/windows/Dockerfile
- https://github.com/microsoft/go-winio
