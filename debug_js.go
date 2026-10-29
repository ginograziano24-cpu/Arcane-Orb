//go:build debug && js

package main

import "syscall/js"

// __dbg lets browser test scripts trigger debug actions by name.
func init() {
	js.Global().Set("__dbg", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			a := args[0].String()
			dbgPending = append(dbgPending, a)
		}
		return nil
	}))
}
