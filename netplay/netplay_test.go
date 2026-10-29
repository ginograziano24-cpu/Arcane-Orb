package netplay

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestCodeRoundTrip(t *testing.T) {
	for _, c := range []struct {
		ip   string
		port int
	}{{"192.168.1.42", 7777}, {"10.0.0.1", 65535}, {"255.255.255.255", 1}, {"0.0.0.0", 0}} {
		code := EncodeAddr(net.ParseIP(c.ip), c.port)
		got, err := DecodeAddr(code)
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		want := net.JoinHostPort(c.ip, fmtInt(c.port))
		if got != want {
			t.Fatalf("code %s decoded to %s, want %s", code, got, want)
		}
		// lower case and look-alike letters are forgiven
		if got2, _ := DecodeAddr(lowerish(code)); got2 != want {
			t.Fatalf("lenient decode of %s failed: %s", code, got2)
		}
	}
	if _, err := DecodeAddr("SHORT"); err == nil {
		t.Fatal("short code accepted")
	}
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func lowerish(s string) string { return string(bytes.ToLower([]byte(s))) }

func TestResolveTarget(t *testing.T) {
	cases := map[string]string{"1.2.3.4": "1.2.3.4:7777", "1.2.3.4:9000": "1.2.3.4:9000", "example.com": "example.com:7777"}
	for in, want := range cases {
		if got, err := ResolveTarget(in); err != nil || got != want {
			t.Fatalf("%s -> %s %v, want %s", in, got, err, want)
		}
	}
}

func TestHostJoinExchange(t *testing.T) {
	s, err := Host(0, "tester")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d := Join(net.JoinHostPort("127.0.0.1", fmtInt(s.Port)))
	var guest, host Link
	deadline := time.Now().Add(5 * time.Second)
	for (guest == nil || host == nil) && time.Now().Before(deadline) {
		if l, done, err := d.Poll(); done {
			if err != nil {
				t.Fatal(err)
			}
			guest = l
		}
		host, _ = s.Poll()
		time.Sleep(10 * time.Millisecond)
	}
	if guest == nil || host == nil {
		t.Fatal("no connection")
	}
	big := bytes.Repeat([]byte("x"), 50000)
	guest.Send([]byte("hello"), true)
	guest.Send(big, false)
	host.Send([]byte("welcome"), true)
	var gotHost, gotGuest [][]byte
	for time.Now().Before(deadline) && (len(gotHost) < 2 || len(gotGuest) < 1) {
		gotHost = append(gotHost, host.Recv()...)
		gotGuest = append(gotGuest, guest.Recv()...)
		time.Sleep(5 * time.Millisecond)
	}
	if len(gotHost) != 2 || string(gotHost[0]) != "hello" || !bytes.Equal(gotHost[1], big) {
		t.Fatalf("host got %d messages", len(gotHost))
	}
	if len(gotGuest) != 1 || string(gotGuest[0]) != "welcome" {
		t.Fatalf("guest got %q", gotGuest)
	}
	// a second guest is turned away while the room is full
	d2 := Join(net.JoinHostPort("127.0.0.1", fmtInt(s.Port)))
	for time.Now().Before(deadline) {
		if l, done, _ := d2.Poll(); done {
			if l != nil {
				time.Sleep(200 * time.Millisecond)
				if l.Alive() {
					t.Fatal("second guest kept a connection")
				}
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	guest.Close()
	for i := 0; i < 100 && host.Alive(); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if host.Alive() {
		t.Fatal("host didn't notice the guest leaving")
	}
}
