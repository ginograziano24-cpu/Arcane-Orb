package main

import (
	"math"
	"strconv"
	"testing"
)

var integerTextSink string

func BenchmarkSmallIntegerText(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		integerTextSink = itoa(i % 100)
	}
}
func BenchmarkLargeIntegerText(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		integerTextSink = itoa(1000000 + i%1000000)
	}
}

func TestIntegerTextAllSignsAndBounds(t *testing.T) {
	for _, n := range []int{0, 1, -1, 9, 10, 99, 100, -100, 123456789, math.MaxInt, math.MinInt} {
		s := itoa(n)
		v, e := strconv.Atoi(s)
		if e != nil || v != n {
			t.Fatalf("%d -> %q -> %d (%v)", n, s, v, e)
		}
	}
}
