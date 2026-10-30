//go:build debug && js

package main

import (
	"fmt"
	"syscall/js"
)

// __dbg lets browser test scripts trigger debug actions by name.
func init() {
	js.Global().Set("__crit", js.FuncOf(func(this js.Value, args []js.Value) any {
		if dbgGame == nil || dbgGame.level == nil {
			return "nolevel"
		}
		out := fmt.Sprintf("quads %d; ", lastQuads)
		for _, c := range dbgGame.level.Critters {
			out += fmt.Sprintf("k%d x%.1f z%.1f g%.2f t%.1f; ", c.Kind, c.X, c.Z, c.gone, c.T)
		}
		return out
	}))
	js.Global().Set("__dbg", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			a := args[0].String()
			dbgPending = append(dbgPending, a)
		}
		return nil
	}))
}
