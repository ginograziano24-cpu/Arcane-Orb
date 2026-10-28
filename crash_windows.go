package main

import (
	"syscall"
	"unsafe"
)

// crashNotice shows a plain Windows message box.
func crashNotice(msg string) {
	defer func() { _ = recover() }()
	user32 := syscall.NewLazyDLL("user32.dll")
	box := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(msg)
	c, _ := syscall.UTF16PtrFromString("Arcane Orb")
	box.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10) // MB_ICONERROR
}
