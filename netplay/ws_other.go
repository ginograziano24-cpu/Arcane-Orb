//go:build !js

package netplay

// Relay is only available in browser builds.
func Relay(url string) Link { return nil }
