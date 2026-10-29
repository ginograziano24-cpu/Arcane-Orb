// Command relay is a tiny WebSocket relay for testing duels between two
// browser builds: the first two clients on /room/<code> are paired and
// every binary message from one goes to the other.
package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

type client struct {
	c  net.Conn
	rw *bufio.ReadWriter
	mu sync.Mutex
}

var (
	mu    sync.Mutex
	rooms = map[string][]*client{}
)

func main() {
	addr := flag.String("addr", ":8768", "listen address")
	flag.Parse()
	http.HandleFunc("/room/", func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimPrefix(r.URL.Path, "/room/")
		key := r.Header.Get("Sec-WebSocket-Key")
		if key == "" || code == "" {
			http.Error(w, "websocket only", 400)
			return
		}
		h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
		hj, _ := w.(http.Hijacker)
		conn, rw, err := hj.Hijack()
		if err != nil {
			return
		}
		rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + base64.StdEncoding.EncodeToString(h[:]) + "\r\n\r\n")
		rw.Flush()
		me := &client{c: conn, rw: rw}
		mu.Lock()
		if len(rooms[code]) >= 2 {
			mu.Unlock()
			conn.Close()
			return
		}
		rooms[code] = append(rooms[code], me)
		mu.Unlock()
		log.Printf("room %s: %d client(s)", code, len(rooms[code]))
		defer func() {
			mu.Lock()
			cs := rooms[code]
			for i, c := range cs {
				if c == me {
					rooms[code] = append(cs[:i], cs[i+1:]...)
				}
			}
			for _, c := range rooms[code] {
				c.c.Close() // the other side's partner left
			}
			delete(rooms, code)
			mu.Unlock()
			conn.Close()
		}()
		for {
			op, msg, err := readFrame(rw)
			if err != nil || op == 8 {
				return
			}
			if op != 2 && op != 1 {
				continue
			}
			mu.Lock()
			for _, c := range rooms[code] {
				if c != me {
					c.mu.Lock()
					writeFrame(c.rw, msg)
					c.mu.Unlock()
				}
			}
			mu.Unlock()
		}
	})
	log.Printf("relay on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func readFrame(r *bufio.ReadWriter) (byte, []byte, error) {
	var h [2]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return 0, nil, err
	}
	op := h[0] & 15
	n := uint64(h[1] & 127)
	switch n {
	case 126:
		var b [2]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, nil, err
		}
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, nil, err
		}
		n = binary.BigEndian.Uint64(b[:])
	}
	var mask [4]byte
	if h[1]&128 != 0 {
		if _, err := io.ReadFull(r, mask[:]); err != nil {
			return 0, nil, err
		}
	}
	msg := make([]byte, n)
	if _, err := io.ReadFull(r, msg); err != nil {
		return 0, nil, err
	}
	if h[1]&128 != 0 {
		for i := range msg {
			msg[i] ^= mask[i%4]
		}
	}
	return op, msg, nil
}

func writeFrame(w *bufio.ReadWriter, msg []byte) {
	hdr := []byte{0x82}
	switch n := len(msg); {
	case n < 126:
		hdr = append(hdr, byte(n))
	case n < 65536:
		hdr = append(hdr, 126, byte(n>>8), byte(n))
	default:
		hdr = append(hdr, 127, 0, 0, 0, 0, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	w.Write(hdr)
	w.Write(msg)
	w.Flush()
}
