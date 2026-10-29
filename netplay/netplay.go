// Package netplay carries duel messages between two players. A Link is a
// two-way message pipe; it can run over a direct TCP connection (LAN or a
// forwarded port), Steam's relay network (in the game package) or a
// WebSocket relay (browser builds, for testing).
package netplay

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Version is bumped whenever the duel protocol changes; mismatched
// builds refuse to pair.
const Version = 1

// DefaultPort is where a direct-connect host listens.
const DefaultPort = 7777

// BeaconPort carries LAN room announcements.
const BeaconPort = 7778

// Link is a message pipe to the other player.
type Link interface {
	Send(msg []byte, reliable bool)
	// Recv drains every message that has arrived; it never blocks.
	Recv() [][]byte
	Alive() bool
	Close()
	Kind() string
}

// ---------------------------------------------------------------- framed stream

// streamLink frames messages over a stream (4-byte length, then the body).
type streamLink struct {
	conn  net.Conn
	mu    sync.Mutex
	in    [][]byte
	inMu  sync.Mutex
	dead  bool
	kind  string
	wmu   sync.Mutex
	close sync.Once
}

const maxFrame = 1 << 20

func newStreamLink(c net.Conn, kind string) *streamLink {
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
		_ = tc.SetKeepAlive(true)
	}
	l := &streamLink{conn: c, kind: kind}
	go l.readLoop()
	return l
}

func (l *streamLink) readLoop() {
	var hdr [4]byte
	for {
		if _, err := readFull(l.conn, hdr[:]); err != nil {
			break
		}
		n := binary.LittleEndian.Uint32(hdr[:])
		if n > maxFrame {
			break
		}
		buf := make([]byte, n)
		if _, err := readFull(l.conn, buf); err != nil {
			break
		}
		l.inMu.Lock()
		l.in = append(l.in, buf)
		l.inMu.Unlock()
	}
	l.mu.Lock()
	l.dead = true
	l.mu.Unlock()
}

func readFull(c net.Conn, b []byte) (int, error) {
	n := 0
	for n < len(b) {
		k, err := c.Read(b[n:])
		n += k
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (l *streamLink) Send(msg []byte, reliable bool) {
	if !l.Alive() {
		return
	}
	frame := make([]byte, 4+len(msg))
	binary.LittleEndian.PutUint32(frame, uint32(len(msg)))
	copy(frame[4:], msg)
	l.wmu.Lock()
	defer l.wmu.Unlock()
	_ = l.conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if _, err := l.conn.Write(frame); err != nil {
		l.mu.Lock()
		l.dead = true
		l.mu.Unlock()
	}
}

func (l *streamLink) Recv() [][]byte {
	l.inMu.Lock()
	out := l.in
	l.in = nil
	l.inMu.Unlock()
	return out
}

func (l *streamLink) Alive() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return !l.dead
}

func (l *streamLink) Close() {
	l.close.Do(func() {
		l.mu.Lock()
		l.dead = true
		l.mu.Unlock()
		_ = l.conn.Close()
	})
}

func (l *streamLink) Kind() string { return l.kind }

// ---------------------------------------------------------------- host

// Server waits for one guest on a TCP port and announces itself on the LAN.
type Server struct {
	ln     net.Listener
	mu     sync.Mutex
	link   Link
	err    error
	Port   int
	stop   chan struct{}
	closed sync.Once
}

// Host starts listening; name is shown to LAN players browsing rooms.
func Host(port int, name string) (*Server, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, Port: ln.Addr().(*net.TCPAddr).Port, stop: make(chan struct{})}
	go s.acceptLoop()
	go s.beacon(name)
	return s, nil
}

func (s *Server) acceptLoop() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			s.mu.Lock()
			if s.link == nil {
				s.err = err
			}
			s.mu.Unlock()
			return
		}
		s.mu.Lock()
		if s.link != nil && s.link.Alive() {
			s.mu.Unlock()
			_ = c.Close() // the room is full
			continue
		}
		s.link = newStreamLink(c, "direct")
		s.mu.Unlock()
	}
}

// beacon announces the room on the local network once a second.
func (s *Server) beacon(name string) {
	addr := &net.UDPAddr{IP: net.IPv4bcast, Port: BeaconPort}
	c, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return
	}
	defer c.Close()
	msg := []byte(fmt.Sprintf("AORB%d|%s|%d", Version, sanitize(name), s.Port))
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		s.mu.Lock()
		busy := s.link != nil && s.link.Alive()
		s.mu.Unlock()
		if !busy {
			_, _ = c.Write(msg)
		}
		select {
		case <-s.stop:
			return
		case <-t.C:
		}
	}
}

func sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '|' || r < 32 {
			return -1
		}
		return r
	}, s)
	if len(s) > 24 {
		s = s[:24]
	}
	return s
}

// Poll returns the guest's link once someone has connected.
func (s *Server) Poll() (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.link, s.err
}

// Close stops listening (an established link stays open).
func (s *Server) Close() {
	s.closed.Do(func() {
		close(s.stop)
		_ = s.ln.Close()
	})
}

// ---------------------------------------------------------------- join

// Dialer connects to a host in the background.
type Dialer struct {
	mu   sync.Mutex
	link Link
	err  error
	done bool
}

// Join dials addr ("1.2.3.4:7777", a host name, or a room code).
func Join(target string) *Dialer {
	d := &Dialer{}
	go func() {
		addr, err := ResolveTarget(target)
		var c net.Conn
		if err == nil {
			c, err = net.DialTimeout("tcp", addr, 6*time.Second)
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		d.done = true
		if err != nil {
			d.err = err
			return
		}
		d.link = newStreamLink(c, "direct")
	}()
	return d
}

// Poll reports the connection once the attempt has finished.
func (d *Dialer) Poll() (link Link, done bool, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.link, d.done, d.err
}

// ---------------------------------------------------------------- room codes

// A direct room code is the host's IPv4 address and port written in ten
// easy-to-read characters, e.g. "K3M9Q-2X7RD".
const codeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ" // no I, L, O, U

// EncodeAddr turns an IPv4 address and port into a room code.
func EncodeAddr(ip net.IP, port int) string {
	v4 := ip.To4()
	if v4 == nil {
		return ""
	}
	var n uint64
	for _, b := range v4 {
		n = n<<8 | uint64(b)
	}
	n = n<<16 | uint64(uint16(port))
	var out [10]byte
	for i := 9; i >= 0; i-- {
		out[i] = codeAlphabet[n&31]
		n >>= 5
	}
	return string(out[:5]) + "-" + string(out[5:])
}

// DecodeAddr reads a room code back into "ip:port".
func DecodeAddr(code string) (string, error) {
	code = strings.ToUpper(strings.NewReplacer("-", "", " ", "", "O", "0", "I", "1", "L", "1", "U", "V").Replace(code))
	if len(code) != 10 {
		return "", errors.New("a room code has 10 characters")
	}
	var n uint64
	for _, c := range code {
		k := strings.IndexRune(codeAlphabet, c)
		if k < 0 {
			return "", fmt.Errorf("%q isn't used in room codes", c)
		}
		n = n<<5 | uint64(k)
	}
	port := int(n & 0xffff)
	n >>= 16
	ip := net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	return net.JoinHostPort(ip.String(), strconv.Itoa(port)), nil
}

// ResolveTarget accepts a room code, "host", or "host:port".
func ResolveTarget(t string) (string, error) {
	t = strings.TrimSpace(t)
	if t == "" {
		return "", errors.New("enter a room code or an address")
	}
	if a, err := DecodeAddr(t); err == nil {
		return a, nil
	}
	if _, _, err := net.SplitHostPort(t); err == nil {
		return t, nil
	}
	return net.JoinHostPort(t, strconv.Itoa(DefaultPort)), nil
}

// LANAddrs lists this machine's private IPv4 addresses (best first).
func LANAddrs() []net.IP {
	var out []net.IP
	ifs, _ := net.Interfaces()
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if v4 := ipn.IP.To4(); v4 != nil && v4.IsPrivate() {
				out = append(out, v4)
			}
		}
	}
	return out
}

// ---------------------------------------------------------------- LAN browser

// Room is a host found on the local network.
type Room struct {
	Name string
	Addr string
	Seen time.Time
}

// Browser listens for LAN room announcements.
type Browser struct {
	mu    sync.Mutex
	rooms map[string]*Room
	conn  *net.UDPConn
}

// Browse starts listening; it returns nil if the port is taken (another
// copy of the game on this machine is already listening).
func Browse() *Browser {
	c, err := net.ListenUDP("udp4", &net.UDPAddr{Port: BeaconPort})
	if err != nil {
		return nil
	}
	b := &Browser{rooms: map[string]*Room{}, conn: c}
	go func() {
		buf := make([]byte, 256)
		for {
			n, from, err := c.ReadFromUDP(buf)
			if err != nil {
				return
			}
			parts := strings.Split(string(buf[:n]), "|")
			if len(parts) != 3 || parts[0] != fmt.Sprintf("AORB%d", Version) {
				continue
			}
			port, err := strconv.Atoi(parts[2])
			if err != nil {
				continue
			}
			addr := net.JoinHostPort(from.IP.String(), strconv.Itoa(port))
			b.mu.Lock()
			b.rooms[addr] = &Room{Name: parts[1], Addr: addr, Seen: time.Now()}
			b.mu.Unlock()
		}
	}()
	return b
}

// Rooms lists hosts heard from in the last few seconds.
func (b *Browser) Rooms() []Room {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []Room
	for k, r := range b.rooms {
		if time.Since(r.Seen) > 4*time.Second {
			delete(b.rooms, k)
			continue
		}
		out = append(out, *r)
	}
	return out
}

func (b *Browser) Close() {
	if b != nil {
		_ = b.conn.Close()
	}
}
