package main

import (
	"math"
	"testing"
	"time"
)

func TestFramePacingTailAndBudgetStatistics(t *testing.T) {
	frames := make([]perfFrame, 1002)
	for i := 0; i < 1000; i++ {
		frames[i].frame = 10 * time.Millisecond
	}
	frames[998].frame = 30 * time.Millisecond
	frames[999].frame = 100 * time.Millisecond
	frames[1001].frame = -time.Millisecond // invalid/initial frames must not skew tails
	s := summarize(frames)
	if s.n != 1000 || s.over60 != 2 || s.over120 != 1000 || s.over144 != 1000 || s.over33 != 1 {
		t.Fatalf("wrong sample/budget counts: %+v", s)
	}
	for name, pair := range map[string][2]float64{
		"mean": {s.frameMS, 10.11}, "1%": {s.low1, 1000.0 / 21}, "0.1%": {s.low01, 10},
		"p95": {s.p95MS, 10}, "p99": {s.p99MS, 10}, "p999": {s.p999MS, 30.07}, "max": {s.maxMS, 100},
		"stddev": {s.stddevMS, math.Sqrt((998*100+900+10000)/1000.0 - 10.11*10.11)},
	} {
		if math.Abs(pair[0]-pair[1]) > 1e-7 {
			t.Errorf("%s got %.12f want %.12f", name, pair[0], pair[1])
		}
	}
}

func TestFramePacingEmptyAndSingleSample(t *testing.T) {
	if s := summarize(nil); s.n != 0 || s.p99MS != 0 || s.low01 != 0 {
		t.Fatal(s)
	}
	s := summarize([]perfFrame{{frame: 20 * time.Millisecond}})
	if s.low01 != 50 || s.p999MS != 20 || s.stddevMS != 0 {
		t.Fatal(s)
	}
	if framePercentile([]float64{2, 8}, -.1) != 2 || framePercentile([]float64{2, 8}, 1.1) != 8 {
		t.Fatal("percentile bounds")
	}
}

func TestBenchmarkDurationValidation(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{{"30s", true}, {"1s", true}, {"10m", true}, {"0s", false}, {"-1s", false}, {"NaN", false}, {"11m", false}, {"100", false}} {
		if _, ok := benchDurationArg(tc.value, time.Second, 10*time.Minute); ok != tc.valid {
			t.Errorf("%q valid=%v", tc.value, ok)
		}
	}
	if d, ok := benchDurationArg("0s", 0, time.Minute); !ok || d != 0 {
		t.Fatal("zero warmup must be supported")
	}
}

func TestBenchmarkRecorderExcludesWarmupAndItsOwnAllocation(t *testing.T) {
	oldWarm, oldMeasure := benchWarm, benchMeasure
	defer func() { benchWarm, benchMeasure = oldWarm, oldMeasure }()
	benchWarm, benchMeasure = time.Second, 10*time.Second
	start := time.Unix(1000, 0)
	b := &benchRun{start: start}
	f := perfFrame{frame: 10 * time.Millisecond}
	for i := 0; i < 100; i++ {
		benchCollectFrame(b, &f, start.Add(time.Duration(i)*10*time.Millisecond))
	}
	if len(b.frames) != 0 || b.reserved {
		t.Fatal("warmup included in measurement")
	}
	spike := perfFrame{frame: 200 * time.Millisecond}
	benchCollectFrame(b, &spike, start.Add(time.Second))
	if !b.reserved || len(b.frames) != 0 || !b.measureStart.IsZero() {
		t.Fatal("allocation frame included")
	}
	benchCollectFrame(b, &spike, start.Add(time.Second+16*time.Millisecond))
	if len(b.frames) != 0 || b.measureStart.IsZero() {
		t.Fatal("post-allocation interval included")
	}
	benchCollectFrame(b, &f, start.Add(time.Second+32*time.Millisecond))
	if len(b.frames) != 1 || b.frames[0].frame != f.frame || b.warmMax != spike.frame {
		t.Fatal("timing phases mixed")
	}
	if got := testing.AllocsPerRun(100, func() { benchCollectFrame(b, &f, start.Add(2*time.Second)) }); got != 0 {
		t.Fatalf("recorder allocates after reserve: %f", got)
	}
}

func TestBenchmarkSceneSelectionIsExplicitAndValidated(t *testing.T) {
	scenes := benchScenes()
	selected, e := selectBenchScenes(scenes, "boss, menu, multiball")
	if e != nil || len(selected) != 3 || selected[0].name != "boss" || selected[1].name != "menu" || selected[2].name != "multiball" {
		t.Fatal(selected, e)
	}
	for _, bad := range []string{"", "menu,", "menu,menu", "world9000"} {
		if _, e := selectBenchScenes(scenes, bad); e == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	if len(scenes) != len(benchScenes()) {
		t.Fatal("selection changed shared capture scene list")
	}
}

func TestBenchmarkCountsUpdatesWhenDrawTimestampRepeats(t *testing.T) {
	s := summarize([]perfFrame{
		{frame: time.Millisecond, updateCount: 1},
		{updateSeconds: 0.001, updateCount: 1},
	})
	if s.n != 1 || s.updateCount != 2 || s.updateSeconds != 0.001 || s.zeroDrawIntervals != 1 {
		t.Fatalf("lost update in a repeated draw interval: %+v", s)
	}
}
