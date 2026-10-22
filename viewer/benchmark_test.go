package viewer

import (
	"strings"
	"testing"
	"time"

	"github.com/akrck02/go-renderer/internal/sceneplan"
)

func TestABenchmarkWritesItsReportToItsOutputOnly(test *testing.T) {
	var report strings.Builder
	measurement := newBenchmark(Measure{BenchmarkFrames: 2, LoadingTime: 1500 * time.Millisecond, Output: &report})
	finished := false
	for !finished {
		finished = measurement.record(sceneplan.FrameStatistics{CPU: time.Millisecond})
	}
	measurement.report()
	if !strings.Contains(report.String(), "loading") || !strings.Contains(report.String(), "1500 ms") {
		test.Fatalf("the report lacks the loading time:\n%s", report.String())
	}
	silent := newBenchmark(Measure{BenchmarkFrames: 2})
	silent.report() // with no output it writes nowhere, and does not fail
}
