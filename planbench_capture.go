//go:build capture && js

package main

import (
	"fmt"
	"syscall/js"
	"time"
)

// __planBench(reps) times the flush planner on the next frame's lists:
// planList for both lists, then the vertex gather, each repeated reps
// times. The means land in __planBenchRes as text.
func init() {
	js.Global().Set("__planBench", js.FuncOf(func(this js.Value, args []js.Value) any {
		reps := args[0].Int()
		flushBenchHook = func(b *Batch, segQ, segG [][]quad) {
			flushBenchHook = nil
			var tq, tf, tg, ts, tr []float64
			nq, ng := 0, 0
			for r := 0; r < reps; r++ {
				planT = [2]time.Duration{}
				t0 := time.Now()
				qs, _ := b.planList(segQ, 0)
				t1 := time.Now()
				ts = append(ts, planT[0].Seconds()*1000)
				tr = append(tr, planT[1].Seconds()*1000)
				b.fillVerts(qs, 2)
				t2 := time.Now()
				gs, _ := b.planList(segG, 1)
				t3 := time.Now()
				tq = append(tq, t1.Sub(t0).Seconds()*1000)
				tf = append(tf, t2.Sub(t1).Seconds()*1000)
				tg = append(tg, t3.Sub(t2).Seconds()*1000)
				nq, ng = len(qs), len(gs)
			}
			med := func(v []float64) float64 {
				t := 0.0
				for _, x := range v {
					t += x
				}
				return t / float64(len(v))
			}
			js.Global().Set("__planBenchRes", fmt.Sprintf("q %d g %d plan %v | sort %.3f runs %.3f | opaque %.3f fill %.3f glow %.3f",
				nq, ng, b.planned[0], med(ts), med(tr), med(tq), med(tf), med(tg)))
		}
		return nil
	}))
}
