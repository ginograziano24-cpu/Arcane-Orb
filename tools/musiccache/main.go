// Command musiccache generates the immutable menu music bank before packaging.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"orb/mus"
	"os"
	"path/filepath"
)

func main() {
	out := flag.String("o", "assets/music/groove.aopcm", "output bank")
	preview := flag.String("review-join", "", "optional four-second WAV around the loop join")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *preview != "" {
		if err := writeJoinPreview(*out, *preview); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func writeJoinPreview(bankPath, out string) error {
	b, err := os.ReadFile(bankPath)
	if err != nil {
		return err
	}
	c, err := mus.ParsePCMCache(b)
	if err != nil {
		return err
	}
	end := int64(binary.LittleEndian.Uint32(b[16:]))
	stream := c.NewStream()
	if _, err = io.CopyN(io.Discard, stream, (end-2*mus.SampleRate)*4); err != nil {
		return err
	}
	pcm := make([]byte, 4*mus.SampleRate*4)
	if _, err = io.ReadFull(stream, pcm); err != nil {
		return err
	}
	wav := make([]byte, 44+len(pcm))
	copy(wav, "RIFF")
	binary.LittleEndian.PutUint32(wav[4:], uint32(len(wav)-8))
	copy(wav[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:], 16)
	binary.LittleEndian.PutUint16(wav[20:], 1)
	binary.LittleEndian.PutUint16(wav[22:], 2)
	binary.LittleEndian.PutUint32(wav[24:], mus.SampleRate)
	binary.LittleEndian.PutUint32(wav[28:], mus.SampleRate*4)
	binary.LittleEndian.PutUint16(wav[32:], 4)
	binary.LittleEndian.PutUint16(wav[34:], 16)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:], uint32(len(pcm)))
	copy(wav[44:], pcm)
	return os.WriteFile(out, wav, 0644)
}
func run(out string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return err
	}
	tmp := out + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	err = mus.WriteMenuCache(f)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(tmp, out); err != nil {
		return err
	}
	b, err := os.ReadFile(out)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(b)
	source := sha256.New()
	files, err := filepath.Glob("mus/*.go")
	if err != nil {
		return err
	}
	for _, p := range files {
		if filepath.Ext(p) == ".go" && len(p) > 8 && p[len(p)-8:] != "_test.go" {
			data, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			source.Write([]byte(filepath.Base(p)))
			source.Write([]byte{0})
			source.Write(data)
		}
	}
	info := map[string]any{"file": out, "bytes": len(b), "sha256": hex.EncodeToString(hash[:]), "synthSourceSHA256": hex.EncodeToString(source.Sum(nil)), "sampleRate": mus.SampleRate, "format": "stereo signed16 little-endian, custom loop header", "regenerate": "go run ./tools/musiccache", "loopJoin": "100ms continuation-to-head crossfade; listening review pending"}
	meta, _ := json.MarshalIndent(info, "", "  ")
	if err = os.WriteFile(out+".json", meta, 0644); err != nil {
		return err
	}
	fmt.Println(string(meta))
	return nil
}
