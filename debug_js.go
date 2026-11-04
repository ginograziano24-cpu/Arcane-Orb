//go:build debug && js

package main

import (
	"fmt"
	"strings"
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
	trace := map[[2]uint8]int{}
	depths := map[int]int{}
	js.Global().Set("__flushTrace", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 && args[0].Bool() {
			trace, depths = map[[2]uint8]int{}, map[int]int{}
			flushTrace = func(a, b uint8, d float64) { trace[[2]uint8{a, b}]++; depths[int(d)]++ }
			return nil
		}
		flushTrace = nil
		return fmt.Sprintf("%v depths %v texImages %d", trace, depths, len(texImages))
	}))
	js.Global().Set("__perfReset", js.FuncOf(func(this js.Value, args []js.Value) any {
		perf.hist = make([]perfFrame, 240)
		perf.hi = 0
		return nil
	}))
	js.Global().Set("__perf", js.FuncOf(func(this js.Value, args []js.Value) any {
		s := summarize(perf.hist)
		return fmt.Sprintf("calls %.0f fps %.1f world %.2f flush %.2f post %.2f hud %.2f draw %.2f upd %.2f quads %.0f sdf %.0f text %.0f allocs %.0f",
			s.world, s.fps, s.sec[secWorld], s.sec[secFlush], s.sec[secPost], s.sec[secHUD], s.sec[secDraw], s.sec[secUpdate], s.quads, s.sdf, s.text, s.allocs) +
			fmt.Sprintf("  marks scenery %d arena %d rest %d", perfMarks[0], perfMarks[1]-perfMarks[0], perfMarks[2])
	}))
	misses := map[string]int{}
	js.Global().Set("__trMiss", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 && args[0].Bool() {
			misses = map[string]int{}
			trMiss = func(s string) { misses[s]++ }
			return nil
		}
		var b strings.Builder
		for k := range misses {
			b.WriteString(k)
			b.WriteString("\n")
		}
		return b.String()
	}))
	js.Global().Set("__dbg", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			a := args[0].String()
			dbgPending = append(dbgPending, a)
		}
		return nil
	}))
}
