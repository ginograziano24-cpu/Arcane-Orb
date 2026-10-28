package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"
)

// crashPath is the crash log, next to the save so players can find and
// send it.
func crashPath() string { return filepath.Join(filepath.Dir(savePath()), "crash.log") }

// guard catches a panic in the game loop, writes what happened to the
// crash log, tells the player where it is, and exits. The run in progress
// was already saved at its last safe point, so Continue picks it back up.
func (g *Game) guard(where string) {
	v := recover()
	if v == nil {
		return
	}
	stack := debug.Stack()
	fmt.Fprintf(os.Stderr, "panic: %v\n%s\n", v, stack)
	p := crashPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	// keep the log small: start over once it grows past 256 KB
	flag := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if fi, err := os.Stat(p); err == nil && fi.Size() > 256<<10 {
		flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	if f, err := os.OpenFile(p, flag, 0o644); err == nil {
		fmt.Fprintf(f, "==== Arcane Orb %s crashed %s ====\n", gameVersion, time.Now().Format(time.RFC3339))
		fmt.Fprintf(f, "os %s/%s  go %s  in %s  state %d\n", runtime.GOOS, runtime.GOARCH, runtime.Version(), where, g.state)
		if r := g.run; r != nil {
			fmt.Fprintf(f, "run: %s seed %s act %d depth %d challenge %q asc %d endless %v\n", r.Arch.ID, r.SeedCode, r.Act, r.Depth, r.Challenge, r.Asc, r.Endless)
			fmt.Fprintf(f, "build: %v\n", r.Order)
		}
		if st := g.settings; st != nil {
			fmt.Fprintf(f, "settings: quality %d res %.2f dyn %v classic %v cb %d speed %.2f\n", st.Quality, st.RenderRes, st.DynRes, st.Classic, st.ColorBlind, st.GameSpeed)
		}
		fmt.Fprintf(f, "panic: %v\n%s\n", v, stack)
		f.Close()
	}
	writeSave(&g.save)
	steamShutdown()
	crashNotice("Arcane Orb ran into a problem and has to close.\n\nYour progress is saved, and Continue on the main menu resumes your run.\n\nA crash report was written to:\n" + p + "\n\nSending it to the developer helps fix this.")
	os.Exit(1)
}
