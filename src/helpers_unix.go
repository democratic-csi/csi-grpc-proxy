//go:build !windows
// +build !windows

package main

import (
	"errors"
	"net"
)

func getWinioDialer(addr string) func() (net.Conn, error) {
	panic(errors.New("winio not available on this platform"))
}

func getWinioListener(addr string) (net.Listener, error) {
	return nil, errors.New("winio not available on this platform")
}
