package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
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

// Get proxy instance
func getProxy(target string) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		// must rewrite scheme regardless
		req.URL.Scheme = "http"

		rewriteHost := getEnv("REWRITE_HOST", "1")
		if rewriteHost == "1" {
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.URL.Host = "localhost"
			req.Host = "localhost"
		}
	}

	var dialer func() (net.Conn, error)

	if strings.HasPrefix(target, "unix://") {
		addr := strings.TrimPrefix(target, "unix://")
		dialer = func() (net.Conn, error) {
			return net.Dial("unix", addr)
		}
	} else if strings.HasPrefix(target, "winio://") {
		addr := strings.TrimPrefix(target, "winio://")
		dialer = getWinioDialer(addr)
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
			log.Printf("Dialing upstream: %s\n", target)
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

func run() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		//log.Println("inline defer executing")
	}()
	defer cancel()

	bindTo := getEnv("BIND_TO", "unix:///csi-data/csi.sock")
	proxyTo := getEnv("PROXY_TO", "unix:///tmp/csi.sock")
	waitForSocketTimeout, _ := strconv.Atoi(getEnv("PROXY_TO_INITIAL_TIMEOUT", "60"))

	signalChan := make(chan os.Signal, 1)
	signal.Notify(
		signalChan,
		syscall.SIGTERM,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
	)

	h2s := &http2.Server{}

	proxy := getProxy(proxyTo)
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

	log.Printf("listening on [%s], proxy to [%s]\n", bindTo, proxyTo)

	if waitForSocketTimeout > 0 && strings.HasPrefix(proxyTo, "unix://") {
		proxyToFile := strings.TrimPrefix(proxyTo, "unix://")
		err := WaitForSocket(proxyToFile, waitForSocketTimeout)
		if err != nil {
			panic(err)
		}
	}

	go func() {
		if strings.HasPrefix(bindTo, "unix://") {
			addr := strings.TrimPrefix(bindTo, "unix://")

			if IsSocket(addr) {
				log.Printf("removing existing listen socket %s\n", addr)
				os.Remove(addr)
			}

			unixListener, err := net.Listen("unix", addr)
			if err != nil {
				panic(err)
			}
			defer unixListener.Close()

			server.Serve(unixListener)
		} else if strings.HasPrefix(bindTo, "winio://") {
			addr := strings.TrimPrefix(bindTo, "winio://")

			winioListener, err := getWinioListener(addr)
			if err != nil {
				panic(err)
			}
			defer winioListener.Close()

			server.Serve(winioListener)
		} else {
			server.Addr = bindTo
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				panic(err)
			} else {
				log.Println("tcp server gracefully stopped listening")
			}
		}
	}()

	<-signalChan
	log.Print("signal caught, shutting down..\n")
	//log.Printf("server conns %v\n", server.)

	// TODO: this does not seem to actually wait for in-flight requests
	// https://github.com/golang/go/issues/17721
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v\n", err)
		return 1
	} else {
		log.Printf("graceful shutdown complete\n")
	}

	return 0
}

func main() {
	code := run()
	log.Printf("exiting with exit code %d\n", code)
	os.Exit(code)
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
