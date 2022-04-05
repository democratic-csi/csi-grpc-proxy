package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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

// Get proxy instance
func getProxy(target string) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.URL.Scheme = "http"
		req.URL.Host = "localhost"
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

	return proxy
}

func main() {
	bindTo := getEnv("BIND_TO", "unix:///csi-data/csi.sock")
	proxyTo := getEnv("PROXY_TO", "unix:///tmp/csi.sock")
	waitForSocketTimeout, _ := strconv.Atoi(getEnv("PROXY_TO_INITIAL_TIMEOUT", "60"))

	h2s := &http2.Server{}

	proxy := getProxy(proxyTo)
	handler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		fmt.Printf("request %s %s %s\n", req.Method, req.URL, req.Proto)
		fmt.Printf("request headers %v\n", req.Header)

		proxy.ServeHTTP(res, req)
	})

	server := &http.Server{
		Addr:    bindTo,
		Handler: h2c.NewHandler(handler, h2s),
	}

	fmt.Printf("listening on [%s], proxy to [%s]\n", bindTo, proxyTo)

	if waitForSocketTimeout > 0 && strings.HasPrefix(proxyTo, "unix://") {
		proxyToFile := strings.TrimPrefix(proxyTo, "unix://")
		err := WaitForSocket(proxyToFile, waitForSocketTimeout)
		if err != nil {
			panic(err)
		}
	}

	if strings.HasPrefix(bindTo, "unix://") {
		addr := strings.TrimPrefix(bindTo, "unix://")

		if IsSocket(addr) {
			fmt.Printf("removing existing listen socket %s\n", addr)
			os.Remove(addr)
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

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func IsSocket(filename string) bool {
	exists := FileExists(filename)
	if !exists {
		return false
	}

	fi, err := os.Stat(filename)
	if err == nil {
		if fi.Mode()&os.ModeSocket != 0 {
			return true
		}
	}

	return false
}

func WaitForSocket(filename string, timeout int) error {
	for {
		if IsSocket(filename) {
			return nil
		}
		if timeout <= 0 {
			return errors.New("timeout reached waiting for socket")
		} else {
			fmt.Printf("waiting for socket [%s] to appear, %ds remaining\n", filename, timeout)
			time.Sleep(1 * time.Second)
			timeout--
		}
	}
}
