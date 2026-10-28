//go:build !windows

package main

import "fmt"

func crashNotice(msg string) { fmt.Println(msg) }
