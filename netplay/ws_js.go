//go:build js

package netplay

import (
	"sync"
	"syscall/js"
)

// wsLink talks to a WebSocket relay (browser builds, used for testing
// duels without Steam or sockets).
type wsLink struct {
	mu    sync.Mutex
	ws    js.Value
	in    [][]byte
	queue [][]byte
	open  bool
	dead  bool
	funcs []js.Func
}

// Relay connects to a relay URL such as ws://localhost:8768/room/ABC.
func Relay(url string) Link {
	l := &wsLink{}
	ws := js.Global().Get("WebSocket").New(url)
	ws.Set("binaryType", "arraybuffer")
	l.ws = ws
	on := func(name string, f func(args []js.Value)) {
		fn := js.FuncOf(func(this js.Value, args []js.Value) any { f(args); return nil })
		l.funcs = append(l.funcs, fn)
		ws.Set(name, fn)
	}
	on("onopen", func([]js.Value) {
		l.mu.Lock()
		l.open = true
		q := l.queue
		l.queue = nil
		l.mu.Unlock()
		for _, m := range q {
			l.write(m)
		}
	})
	on("onmessage", func(args []js.Value) {
		u8 := js.Global().Get("Uint8Array").New(args[0].Get("data"))
		b := make([]byte, u8.Get("length").Int())
		js.CopyBytesToGo(b, u8)
		l.mu.Lock()
		l.in = append(l.in, b)
		l.mu.Unlock()
	})
	dead := func([]js.Value) {
		l.mu.Lock()
		l.dead = true
		l.mu.Unlock()
	}
	on("onclose", dead)
	on("onerror", dead)
	return l
}

func (l *wsLink) write(m []byte) {
	u8 := js.Global().Get("Uint8Array").New(len(m))
	js.CopyBytesToJS(u8, m)
	l.ws.Call("send", u8)
}

func (l *wsLink) Send(m []byte, reliable bool) {
	l.mu.Lock()
	if l.dead {
		l.mu.Unlock()
		return
	}
	if !l.open {
		l.queue = append(l.queue, append([]byte(nil), m...))
		l.mu.Unlock()
		return
	}
	l.mu.Unlock()
	l.write(m)
}

func (l *wsLink) Recv() [][]byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.in
	l.in = nil
	return out
}

func (l *wsLink) Alive() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return !l.dead
}

func (l *wsLink) Close() {
	l.mu.Lock()
	l.dead = true
	l.mu.Unlock()
	l.ws.Call("close")
}

func (l *wsLink) Kind() string { return "relay" }
