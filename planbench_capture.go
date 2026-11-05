//go:build capture && js

package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"syscall/js"
	"time"
	"unsafe"
)

// __planBench(reps) times the flush planner on the next frame's lists:
// planList for both lists, then the vertex gather, each repeated reps
// times. The means land in __planBenchRes as text.
func init() {
	forcePieces = true
	js.Global().Set("__jobBench", js.FuncOf(func(this js.Value, args []js.Value) any {
		frames := args[0].Int()
		var sum [32]time.Duration
		n := 0
		sideJobHook = func(job int, d time.Duration) {
			if job == 0 {
				n++
			}
			if job < len(sum) {
				sum[job] += d
			}
			if n > frames {
				sideJobHook = nil
				out := ""
				for i, t := range sum {
					if t > 0 {
						out += fmt.Sprintf(" j%d:%.3f", i, t.Seconds()*1000/float64(frames))
					}
				}
				js.Global().Set("__jobBenchRes", out)
			}
		}
		return nil
	}))
	// __hashFrame() hashes the next frame's drawn quads, in draw order,
	// and lists its runs (__hashRes): two builds that should draw the same
	// thing must match.
	js.Global().Set("__hashFrame", js.FuncOf(func(this js.Value, args []js.Value) any {
		flushBenchHook = func(b *Batch, segQ, segG [][]quad) {
			flushBenchHook = nil
			out := ""
			for l, seg := range [][][]quad{segQ, segG} {
				qs, runs := b.planList(seg, l)
				h := fnv.New64a()
				for _, oi := range b.runIdx {
					q := qs[oi]
					h.Write(unsafe.Slice((*byte)(unsafe.Pointer(&q.v[0])), int(unsafe.Sizeof(q.v))))
					var d [8]byte
					*(*uint64)(unsafe.Pointer(&d[0])) = math.Float64bits(q.depth)
					h.Write(d[:])
				}
				rs := ""
				for _, r := range runs {
					rs += fmt.Sprintf("%d:%d,", r.tex, r.last-r.first)
				}
				for _, q := range seg {
					out += fmt.Sprint(len(q), " ")
				}
				out += fmt.Sprintf("list%d n=%d hash=%x runs=%d [%s]\n", l, len(qs), h.Sum64(), len(runs), rs)
			}
			js.Global().Set("__hashRes", out)
		}
		return nil
	}))
	js.Global().Set("__offBench", js.FuncOf(func(this js.Value, args []js.Value) any {
		flushBenchHook = func(b *Batch, segQ, segG [][]quad) {
			flushBenchHook = nil
			W, H := float32(b.cam.W), float32(b.cam.H)
			out := ""
			for l, seg := range [][][]quad{segQ, segG} {
				for si, qs := range seg {
					off, tiny := 0, 0
					for i := range qs {
						v := &qs[i].v
						x0, x1, y0, y1 := v[0].DstX, v[0].DstX, v[0].DstY, v[0].DstY
						for k := 1; k < 4; k++ {
							x0, x1 = min(x0, v[k].DstX), max(x1, v[k].DstX)
							y0, y1 = min(y0, v[k].DstY), max(y1, v[k].DstY)
						}
						if x1 < 0 || y1 < 0 || x0 > W || y0 > H {
							off++
						} else if (x1-x0)*(y1-y0) < 0.25 {
							tiny++
						}
					}
					if off+tiny > 0 {
						out += fmt.Sprintf(" L%d/s%d:%d/%d off %d tiny %d;", l, si, len(qs), 0, off, tiny)
					}
				}
			}
			js.Global().Set("__offRes", out)
		}
		return nil
	}))
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
