//go:build windows
// +build windows

package main

import (
	"net"

	"github.com/Microsoft/go-winio"
)

func getWinioDialer(addr string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		return winio.DialPipe(addr, nil)
	}
}

func getWinioListener(addr string) (net.Listener, error) {
	return winio.ListenPipe(addr, nil)
}
