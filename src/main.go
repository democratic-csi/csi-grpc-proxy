package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Get env var or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	fmt.Printf("request %s %s %s\n", req.Method, req.URL, req.Proto)
	fmt.Printf("request headers %v\n", req.Header)

	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = "localhost"
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Host = "localhost"
	}

	var dialer func() (net.Conn, error)

	if strings.HasPrefix(target, "unix://") {
		addr := strings.TrimPrefix(target, "unix://")
		dialer = func() (net.Conn, error) {
			return net.Dial("unix", addr)
		}
	} else {
		url, _ := url.Parse(target)
		addr := url.Host
		dialer = func() (net.Conn, error) {
			return net.Dial("tcp", addr)
		}
	}

	transport := &http2.Transport{
		// So http2.Transport doesn't complain the URL scheme isn't 'https'
		AllowHTTP: true,
		// Pretend we are dialing a TLS endpoint. (Note, we ignore the passed tls.Config)
		DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
			return dialer()
		},
	}

	proxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: transport,
	}

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

func main() {
	bindTo := getEnv("BIND_TO", "unix:///csi-data/csi.sock")
	proxyTo := getEnv("PROXY_TO", "unix:///tmp/csi.sock")

	h2s := &http2.Server{}

	handler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		serveReverseProxy(proxyTo, res, req)
	})

	server := &http.Server{
		Addr:    bindTo,
		Handler: h2c.NewHandler(handler, h2s),
	}

	fmt.Printf("listening on [%s], proxy to [%s]\n", bindTo, proxyTo)

	if strings.HasPrefix(bindTo, "unix://") {
		addr := strings.TrimPrefix(bindTo, "unix://")

		fi, err := os.Stat(addr)
		if err == nil {
			if fi.Mode()&os.ModeSocket != 0 {
				fmt.Printf("removing existing listen socket %s\n", addr)
				os.Remove(addr)
			}
		}

		unixListener, err := net.Listen("unix", addr)
		if err != nil {
			panic(err)
		}
		server.Serve(unixListener)
	} else {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}
}
