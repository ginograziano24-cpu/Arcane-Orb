//go:build !js

package main

import (
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"orb/netplay"
)

const relayBuild = false

// publicCode asks a public echo service for this machine's internet
// address and turns it into a room code (port forwarding still needed).
func publicCode(port int) string {
	c := &http.Client{Timeout: 4 * time.Second}
	resp, err := c.Get("https://api.ipify.org")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	ip := net.ParseIP(strings.TrimSpace(string(b)))
	if ip == nil {
		return ""
	}
	return netplay.EncodeAddr(ip, port)
}
