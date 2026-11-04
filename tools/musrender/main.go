// Command musrender renders the game's songs to WAV for listening and
// analysis: musrender -o dir [-secs 120] [-song name] [-curve ramp|low|high|menu]
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	"orb/mus"
)

func main() {
	out := flag.String("o", "musout", "output directory")
	secs := flag.Float64("secs", 120, "seconds per song")
	only := flag.String("song", "", "render only this song")
	curve := flag.String("curve", "ramp", "intensity curve: ramp, low, high, menu, wave")
	solo := flag.String("solo", "", "comma list of parts to measure alone")
	vstats := flag.Bool("voices", false, "report average sounding voices per part")
	prof := flag.String("cpuprofile", "", "write a CPU profile")
	flag.Parse()
	if *prof != "" {
		pf, _ := os.Create(*prof)
		_ = pprof.StartCPUProfile(pf)
		defer pprof.StopCPUProfile()
	}
	_ = os.MkdirAll(*out, 0o755)
	names := mus.Names()
	sort.Strings(names)
	for _, n := range names {
		if *only != "" && n != *only {
			continue
		}
		f := curveFn(*curve, *secs)
		if *solo != "" {
			for _, p := range strings.Split(*solo, ",") {
				mus.Solo = p
				l, r := mus.Render(n, *secs, f)
				fmt.Printf("  %-8s %s\n", p, bandsOf(l, r))
				writeWav(filepath.Join(*out, n+"_"+p+".wav"), l, r)
			}
			mus.Solo = ""
			continue
		}
		if *vstats {
			mus.VoiceStats = map[string]int{}
		}
		t0 := time.Now()
		l, r := mus.Render(n, *secs, f)
		if *vstats {
			nb := mus.VoiceStats["#blocks"] / max(1, len(mus.VoiceStats)-1)
			for k, v := range mus.VoiceStats {
				if k != "#blocks" {
					fmt.Printf("  %-8s %.1f voices\n", k, float64(v)/float64(max(1, nb)))
				}
			}
			mus.VoiceStats = nil
		}
		el := time.Since(t0)
		path := filepath.Join(*out, n+".wav")
		writeWav(path, l, r)
		peak, rms := 0.0, 0.0
		for i := range l {
			peak = math.Max(peak, math.Max(math.Abs(l[i]), math.Abs(r[i])))
			rms += l[i]*l[i] + r[i]*r[i]
		}
		rms = math.Sqrt(rms / float64(2*len(l)))
		fmt.Printf("%-8s %5.1fs render %.2fs (%.0fx) peak %.2f dBFS rms %.1f dBFS\n", n, *secs, el.Seconds(),
			*secs/el.Seconds(), 20*math.Log10(peak), 20*math.Log10(rms))
	}
}

func curveFn(c string, secs float64) func(float64) float64 {
	switch c {
	case "low":
		return func(float64) float64 { return 0.1 }
	case "high":
		return func(float64) float64 { return 0.9 }
	case "menu":
		return func(float64) float64 { return 0.25 }
	case "wave": // a level: builds, a miss drops it, builds again
		return func(t float64) float64 {
			u := math.Mod(t, 70) / 70
			if u < 0.6 {
				return u / 0.6
			}
			return 0.15
		}
	}
	return func(t float64) float64 { return math.Min(1, t/(secs*0.45)) }
}

func writeWav(path string, l, r []float64) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	n := len(l)
	hdr := make([]byte, 44)
	copy(hdr[0:], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:], uint32(36+n*4))
	copy(hdr[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(hdr[16:], 16)
	binary.LittleEndian.PutUint16(hdr[20:], 1)
	binary.LittleEndian.PutUint16(hdr[22:], 2)
	binary.LittleEndian.PutUint32(hdr[24:], mus.SampleRate)
	binary.LittleEndian.PutUint32(hdr[28:], mus.SampleRate*4)
	binary.LittleEndian.PutUint16(hdr[32:], 4)
	binary.LittleEndian.PutUint16(hdr[34:], 16)
	copy(hdr[36:], "data")
	binary.LittleEndian.PutUint32(hdr[40:], uint32(n*4))
	buf := make([]byte, n*4)
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint16(buf[i*4:], uint16(int16(math.Max(-32767, math.Min(32767, l[i]*32767)))))
		binary.LittleEndian.PutUint16(buf[i*4+2:], uint16(int16(math.Max(-32767, math.Min(32767, r[i]*32767)))))
	}
	f.Write(hdr)
	f.Write(buf)
}

// bandsOf reports a signal's level and how its energy spreads by band.
func bandsOf(l, r []float64) string {
	rms := 0.0
	for i := range l {
		rms += l[i]*l[i] + r[i]*r[i]
	}
	rms = math.Sqrt(rms / float64(2*len(l)))
	return fmt.Sprintf("rms %6.1f dBFS", 20*math.Log10(rms+1e-9))
}
