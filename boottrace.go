package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Load-time trace (roadmap #103): how long each startup step takes, from the
// process starting to the first main-menu frame on screen. The steps are
// written into the benchmark report, and to boot_trace.txt beside the exe
// when the game is started with -boottrace.

// bootStart is set while the package initialises, before main runs, so the
// trace also counts package init (embedded assets, tables).
var bootStart = time.Now()

type bootStep struct {
	name string
	at   time.Duration // since bootStart, at the end of the step
}

// bootSetUp is the step that ends with the Game built; the next is the
// window opening.
const bootSetUp = "game set up"

var (
	bootSteps []bootStep
	bootDone  bool
	bootFile  bool // -boottrace
)

// bootMark ends a startup step.
func bootMark(name string) {
	if bootDone {
		return
	}
	bootSteps = append(bootSteps, bootStep{name, time.Since(bootStart)})
}

// bootFrame is called by every Draw until the first menu frame has been
// drawn; it then closes the trace.
func bootFrame(g *Game, drawn bool) {
	if bootDone {
		return
	}
	if !drawn {
		if len(bootSteps) == 0 || bootSteps[len(bootSteps)-1].name != "first frame" {
			bootMark("first frame")
		}
		return
	}
	bootMark("first menu frame drawn")
	bootDone = true
	if bootFile {
		exe, _ := os.Executable()
		_ = os.WriteFile(filepath.Join(filepath.Dir(exe), "boot_trace.txt"), []byte(bootReport()), 0o644)
	}
}

// bootReport lists each step with its own duration; the slowest is marked.
func bootReport() string {
	var sb strings.Builder
	sb.WriteString("Startup (ms; from process start to the first main-menu frame):\n")
	slow, slowD := -1, time.Duration(0)
	prev := time.Duration(0)
	for i, s := range bootSteps {
		if d := s.at - prev; d > slowD {
			slow, slowD = i, d
		}
		prev = s.at
	}
	prev = 0
	for i, s := range bootSteps {
		mark := ""
		if i == slow {
			mark = "  <- slowest"
		}
		fmt.Fprintf(&sb, "  %-26s %8.1f  (at %8.1f)%s\n", s.name, ms(s.at-prev), ms(s.at), mark)
		prev = s.at
	}
	return sb.String()
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
