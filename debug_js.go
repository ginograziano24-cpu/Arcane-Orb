//go:build debug && js

package main

import "syscall/js"

// __dbg lets browser test scripts trigger debug actions by name.
func init() {
	js.Global().Set("__dbg", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			dbgPending = append(dbgPending, args[0].String())
		}
		return nil
	}))
}
