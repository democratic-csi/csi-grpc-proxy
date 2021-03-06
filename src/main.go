package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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

var VALID_NETWORKS = [...]string{"unix", "tcp", "tcp4", "tcp6", "npipe"}

// Get proxy instance
func getProxy(network, addr string) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		// must rewrite scheme regardless
		req.URL.Scheme = "http"

		rewriteHost := getEnv("REWRITE_HOST", "1")
		rewriteHostHostname := getEnv("REWRITE_HOST_HOSTNAME", "localhost")

		// rewrite host regardless of value
		if rewriteHost == "1" {
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.URL.Host = rewriteHostHostname
			req.Host = rewriteHostHostname
		}

		// rewrite host only if value is invalid/non-compliant
		if rewriteHost == "2" {
			// rfc is defined as: host [ ":" port ]
			// crude checking, could definitely be made more robust
			if strings.Contains(req.Host, "/") {
				req.Header.Set("X-Forwarded-Host", req.Host)
				req.URL.Host = rewriteHostHostname
				req.Host = rewriteHostHostname
			}
		}
	}

	var dialer func() (net.Conn, error)

	switch network {
	case "unix", "tcp", "tcp4", "tcp6":
		dialer = func() (net.Conn, error) {
			return net.Dial(network, addr)
		}
	case "npipe":
		dialer = getWinioDialer(addr)
	default:
		panic(fmt.Errorf("invalid PROXY_TO nextwork: %s", network))
	}

	transport := &http2.Transport{
		// So http2.Transport doesn't complain the URL scheme isn't 'https'
		AllowHTTP: true,
		// Pretend we are dialing a TLS endpoint. (Note, we ignore the passed tls.Config)
		DialTLS: func(_, _ string, _ *tls.Config) (net.Conn, error) {
			log.Printf("Dialing upstream: %s://%s\n", network, addr)
			return dialer()
		},
	}

	proxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: transport,
		ModifyResponse: func(r *http.Response) error {
			// intercept response here and modify as desired
			//log.Printf("%v", r.Body)
			return nil
		},
	}

	return proxy
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		//log.Println("inline defer executing")
	}()
	defer cancel()

	bindTo := getEnv("BIND_TO", "unix:///csi-data/csi.sock")
	proxyTo := getEnv("PROXY_TO", "unix:///tmp/csi.sock")
	waitForSocketTimeout, _ := strconv.Atoi(getEnv("PROXY_TO_INITIAL_TIMEOUT", "60"))

	bindToNetwork, bindToAddr, found := strings.Cut(bindTo, "://")
	if !found {
		panic(fmt.Errorf("invalid BIND_TO: %s", bindTo))
	}

	if !StringInSlice(bindToNetwork, VALID_NETWORKS[:]) {
		panic(fmt.Errorf("invalid BIND_TO network: %s", bindToNetwork))
	}

	proxyToNetwork, proxyToAddr, found := strings.Cut(proxyTo, "://")
	if !found {
		panic(fmt.Errorf("invalid PROXY_TO: %s", proxyTo))
	}

	if !StringInSlice(proxyToNetwork, VALID_NETWORKS[:]) {
		panic(fmt.Errorf("invalid PROXY_TO network: %s", proxyToNetwork))
	}

	log.Printf("BIND_TO [%s], PROXY_TO [%s]\n", bindTo, proxyTo)

	h2s := &http2.Server{}
	proxy := getProxy(proxyToNetwork, proxyToAddr)
	handler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		log.Printf("request (%s://%s) %s %s %s\n", req.URL.Scheme, req.Host, req.Method, req.URL, req.Proto)
		log.Printf("request headers %v\n", req.Header)

		proxy.ServeHTTP(res, req)
	})

	server := &http.Server{
		Handler: h2c.NewHandler(handler, h2s),
		ConnContext: func(ctx context.Context, conn net.Conn) context.Context {
			// intercept connections as they happen

			//log.Printf("conn: from %s to %s\n", conn.RemoteAddr(), conn.LocalAddr())
			return ctx

			//if c2 := ctx.Value("conn"); c2 != nil {
			//	log.Printf("existing: %s\n", c2.(net.Conn).RemoteAddr())
			//}
			//
			//return context.WithValue(ctx, "conn", conn)
		},
		ConnState: func(conn net.Conn, newState http.ConnState) {
			// intercept connection state changes

			// in our case the series is
			// new -> active -> hijacked
			//log.Printf("conn: %s state change %v\n", conn.RemoteAddr(), newState)
		},
	}

	//server.RegisterOnShutdown(func () {
	//	log.Println("registered shutdown function 1 invoked")
	//})
	//server.RegisterOnShutdown(func () {
	//	log.Println("registered shutdown function 2 invoked")
	//})
	//defer server.Shutdown(ctx)
	//defer server.Close()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(
		signalChan,
		syscall.SIGTERM,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
	)

	go func() {
		var returncode int = 0
		<-signalChan
		log.Print("signal caught, shutting down..\n")
		//log.Printf("server conns %v\n", server.)

		// TODO: this does not seem to actually wait for in-flight requests
		// https://github.com/golang/go/issues/17721
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("server shutdown error: %v\n", err)
			returncode = 1
		} else {
			log.Printf("graceful shutdown complete\n")
			returncode = 0
		}

		log.Printf("exiting with exit code %d\n", returncode)
		os.Exit(returncode)
	}()

	if waitForSocketTimeout > 0 {
		func(network, addr string) {
			var err error
			switch network {
			case "unix":
				if runtime.GOOS == "windows" {
					err = WaitForDial(network, addr, waitForSocketTimeout)
				} else {
					err = WaitForSocket(addr, waitForSocketTimeout)
				}
			case "tcp", "tcp4", "tcp6":
				err = WaitForDial(network, addr, waitForSocketTimeout)
			case "npipe":
				err = WaitForFile(addr, waitForSocketTimeout)
			default:
				panic(fmt.Errorf("invalid PROXY_TO nextwork: %s", network))
			}

			if err != nil {
				panic(err)
			}

			log.Printf("PROXY_TO [%s] is ready!", proxyTo)
		}(proxyToNetwork, proxyToAddr)
	}

	go func(network, addr string) {
		var err error
		var listener net.Listener
		switch network {
		case "unix":
			if runtime.GOOS == "windows" {
				// uds stat on windows just flat out fails, indescriminately delete here and ignore errors
				// https://github.com/golang/go/issues/33357
				log.Printf("removing existing listen socket %s\n", addr)
				os.Remove(addr)
			} else {
				if IsSocket(addr) {
					log.Printf("removing existing listen socket %s\n", addr)
					os.Remove(addr)
				}
			}

			listener, err = net.Listen(network, addr)
		case "tcp", "tcp4", "tcp6":
			server.Addr = addr
			listener, err = net.Listen(network, addr)
		case "npipe":
			listener, err = getWinioListener(addr)
		default:
			panic(fmt.Errorf("invalid BIND_TO nextwork: %s", network))
		}

		if err != nil {
			panic(err)
		}
		defer listener.Close()

		log.Printf("BIND_TO [%s] is ready!", bindTo)
		err = server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		} else {
			log.Println("server gracefully stopped listening")
		}
	}(bindToNetwork, bindToAddr)

	// block forever
	select {}
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}

	return !info.IsDir()
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
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
			log.Printf("waiting for socket [%s] to appear, %ds remaining\n", filename, timeout)
			time.Sleep(1 * time.Second)
			timeout--
		}
	}
}

func WaitForFile(filename string, timeout int) error {
	for {
		if FileExists(filename) {
			return nil
		}
		if timeout <= 0 {
			return errors.New("timeout reached waiting for file")
		} else {
			log.Printf("waiting for file [%s] to appear, %ds remaining\n", filename, timeout)
			time.Sleep(1 * time.Second)
			timeout--
		}
	}
}

func WaitForDial(network, addr string, timeout int) error {
	for {
		conn, err := net.DialTimeout(network, addr, 1*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}

		if timeout <= 0 {
			return errors.New("timeout reached waiting for dial")
		} else {
			log.Printf("waiting for successful dial [%s://%s], %ds remaining\n", network, addr, timeout)
			time.Sleep(1 * time.Second)
			timeout--
		}
	}
}
