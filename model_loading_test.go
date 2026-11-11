package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
)

var updateModelReference = flag.Bool("update-model-reference", false, "write model topology reference before optimizing")
var modelBenchmarkSink []*Mesh

func modelTestBank(tb testing.TB) (names []string, data [][]byte) {
	tb.Helper()
	entries, e := modelFS.ReadDir("assets/models")
	if e != nil {
		tb.Fatal(e)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".orbm") {
			b, e := modelFS.ReadFile("assets/models/" + entry.Name())
			if e != nil {
				tb.Fatal(e)
			}
			names = append(names, entry.Name())
			data = append(data, b)
		}
	}
	return
}

func BenchmarkParseModelBank(b *testing.B) {
	_, bank := modelTestBank(b)
	modelBenchmarkSink = make([]*Mesh, len(bank))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, data := range bank {
			m, e := parseModel(data)
			if e != nil {
				b.Fatal(e)
			}
			modelBenchmarkSink[j] = m
		}
	}
}

func BenchmarkBuildModelTopologyBank(b *testing.B) {
	_, bank := modelTestBank(b)
	meshes := make([]*Mesh, len(bank))
	for i, data := range bank {
		m, e := parseModel(data)
		if e != nil {
			b.Fatal(e)
		}
		meshes[i] = m
	}
	modelBenchmarkSink = make([]*Mesh, len(bank))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, source := range meshes {
			m := &Mesh{Tris: source.Tris}
			m.topology()
			modelBenchmarkSink[j] = m
		}
	}
}

func TestAllModelGeometryAndTopologyRemainIdentical(t *testing.T) {
	names, bank := modelTestBank(t)
	got := map[string]string{}
	for i, data := range bank {
		m, e := parseModel(data)
		if e != nil {
			t.Fatal(names[i], e)
		}
		topo := m.topology()
		h := sha256.New()
		// Include authored geometry/material fields and exact shared-corner order.
		for _, tri := range m.Tris {
			binary.Write(h, binary.LittleEndian, tri.P)
			binary.Write(h, binary.LittleEndian, tri.Col)
			binary.Write(h, binary.LittleEndian, tri.Emi)
			binary.Write(h, binary.LittleEndian, tri.Fixed)
		}
		binary.Write(h, binary.LittleEndian, topo.pos)
		binary.Write(h, binary.LittleEndian, topo.tri)
		got[names[i]] = hex.EncodeToString(h.Sum(nil))
	}
	const path = "testdata/model-topology-reference.json"
	if *updateModelReference {
		if e := os.MkdirAll("testdata", 0755); e != nil {
			t.Fatal(e)
		}
		b, _ := json.MarshalIndent(got, "", "  ")
		if e := os.WriteFile(path, b, 0644); e != nil {
			t.Fatal(e)
		}
		return
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var want map[string]string
	if e = json.Unmarshal(b, &want); e != nil {
		t.Fatal(e)
	}
	if len(want) != len(got) {
		t.Fatalf("asset count changed: %d vs %d", len(got), len(want))
	}
	for name, hash := range got {
		if hash != want[name] {
			t.Errorf("geometry/topology changed: %s", name)
		}
	}
}

func TestModelDecoderRecoversAfterCorruptInput(t *testing.T) {
	_, bank := modelTestBank(t)
	for _, bad := range [][]byte{nil, []byte("ORBMbadstream"), bank[0][:len(bank[0])/2]} {
		if _, e := parseModel(bad); e == nil {
			t.Fatal("corrupt model accepted")
		}
		if _, e := parseModel(bank[0]); e != nil {
			t.Fatalf("valid model after error: %v", e)
		}
	}
}
