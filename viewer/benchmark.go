package viewer

import (
	"fmt"
	"io"
	"time"

	"github.com/akrck02/go-renderer/opengl"
)

// warmupFrames are drawn before measuring, while meshes are uploaded to the GPU.
const warmupFrames = 10

// benchmark accumulates the statistics of a number of frames, in two rounds: first with GPU timers
// (which wait for the GPU and inflate the processor time), then without them, to measure the
// processor time and the time between frames as they really are.
type benchmark struct {
	frames         int
	drawn          int
	loadingTime    time.Duration
	firstFrame     time.Duration
	total          opengl.FrameStatistics
	last           opengl.FrameStatistics
	processorAlone time.Duration // processor time in the round without timers
	betweenFrames  time.Duration // wall time of the round without timers
	untimedStarted time.Time
}

func newBenchmark(frames int, loadingTime time.Duration) *benchmark {
	return &benchmark{frames: frames, loadingTime: loadingTime}
}

// timing reports whether the next frame should use GPU timers (the first round).
func (measurement *benchmark) timing() bool {
	return measurement.drawn < warmupFrames+measurement.frames
}

// record adds a frame and reports whether the measurement is complete.
func (measurement *benchmark) record(statistics opengl.FrameStatistics) bool {
	measurement.drawn++
	if measurement.drawn == 1 {
		measurement.firstFrame = statistics.CPU
	}
	if measurement.drawn <= warmupFrames {
		return false
	}
	if measurement.drawn > warmupFrames+measurement.frames {
		return measurement.recordUntimed(statistics)
	}
	if measurement.drawn == warmupFrames+measurement.frames {
		measurement.untimedStarted = time.Now()
	}
	measurement.total.CPU += statistics.CPU
	measurement.total.GPU.Seabed += statistics.GPU.Seabed
	measurement.total.GPU.Shadows += statistics.GPU.Shadows
	measurement.total.GPU.Sky += statistics.GPU.Sky
	measurement.total.GPU.Scene += statistics.GPU.Scene
	measurement.last = statistics
	return false
}

// recordUntimed adds a frame of the second round.
func (measurement *benchmark) recordUntimed(statistics opengl.FrameStatistics) bool {
	measurement.processorAlone += statistics.CPU
	if measurement.drawn < warmupFrames+2*measurement.frames {
		return false
	}
	measurement.betweenFrames = time.Since(measurement.untimedStarted)
	return true
}

func (measurement *benchmark) average(total time.Duration) string {
	return fmt.Sprintf("%6.2f ms", float64(total.Microseconds())/1000/float64(measurement.frames))
}

func (measurement *benchmark) report(output io.Writer) {
	gpu := measurement.total.GPU
	fmt.Fprintf(output, "loading      %8.0f ms\n", float64(measurement.loadingTime.Milliseconds()))
	fmt.Fprintf(output, "first frame  %8.0f ms (uploads to the GPU)\n", float64(measurement.firstFrame.Milliseconds()))
	fmt.Fprintf(output, "average of %d frames:\n", measurement.frames)
	fmt.Fprintf(output, "  CPU          %s (with GPU timers)\n", measurement.average(measurement.total.CPU))
	fmt.Fprintf(output, "  CPU alone    %s (without timers)\n", measurement.average(measurement.processorAlone))
	fmt.Fprintf(output, "  frame        %s (between frames without timers or vsync)\n", measurement.average(measurement.betweenFrames))
	fmt.Fprintf(output, "  GPU shadows  %s\n", measurement.average(gpu.Shadows))
	fmt.Fprintf(output, "  GPU sky      %s\n", measurement.average(gpu.Sky))
	fmt.Fprintf(output, "  GPU scene    %s\n", measurement.average(gpu.Scene))
	fmt.Fprintf(output, "  GPU seabed   %s\n", measurement.average(gpu.Seabed))
	fmt.Fprintf(output, "  GPU total    %s\n", measurement.average(gpu.Total()))
	last := measurement.last
	fmt.Fprintf(output, "per frame: %d draw calls, %d triangles, %d instances; shadows: %d draw calls, %d triangles\n",
		last.DrawCalls, last.Triangles, last.Instances, last.ShadowDrawCalls, last.ShadowTriangles)
}
