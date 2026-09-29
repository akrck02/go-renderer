package viewer

import (
	"fmt"
	"io"
	"time"

	"github.com/akrck02/go-renderer/opengl"
)

// warmupFrames are drawn before measuring, while meshes are uploaded to the GPU.
const warmupFrames = 10

// benchmark accumulates the statistics of a number of frames.
type benchmark struct {
	frames      int
	drawn       int
	loadingTime time.Duration
	firstFrame  time.Duration
	total       opengl.FrameStatistics
	last        opengl.FrameStatistics
}

func newBenchmark(frames int, loadingTime time.Duration) *benchmark {
	return &benchmark{frames: frames, loadingTime: loadingTime}
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
	measurement.total.CPU += statistics.CPU
	measurement.total.GPU.Seabed += statistics.GPU.Seabed
	measurement.total.GPU.Shadows += statistics.GPU.Shadows
	measurement.total.GPU.Sky += statistics.GPU.Sky
	measurement.total.GPU.Scene += statistics.GPU.Scene
	measurement.last = statistics
	return measurement.drawn-warmupFrames >= measurement.frames
}

func (measurement *benchmark) average(total time.Duration) string {
	return fmt.Sprintf("%6.2f ms", float64(total.Microseconds())/1000/float64(measurement.frames))
}

func (measurement *benchmark) report(output io.Writer) {
	gpu := measurement.total.GPU
	fmt.Fprintf(output, "loading      %8.0f ms\n", float64(measurement.loadingTime.Milliseconds()))
	fmt.Fprintf(output, "first frame  %8.0f ms (uploads to the GPU)\n", float64(measurement.firstFrame.Milliseconds()))
	fmt.Fprintf(output, "average of %d frames:\n", measurement.frames)
	fmt.Fprintf(output, "  CPU          %s\n", measurement.average(measurement.total.CPU))
	fmt.Fprintf(output, "  GPU shadows  %s\n", measurement.average(gpu.Shadows))
	fmt.Fprintf(output, "  GPU sky      %s\n", measurement.average(gpu.Sky))
	fmt.Fprintf(output, "  GPU scene    %s\n", measurement.average(gpu.Scene))
	fmt.Fprintf(output, "  GPU seabed   %s\n", measurement.average(gpu.Seabed))
	fmt.Fprintf(output, "  GPU total    %s\n", measurement.average(gpu.Total()))
	last := measurement.last
	fmt.Fprintf(output, "per frame: %d draw calls, %d triangles, %d instances; shadows: %d draw calls, %d triangles\n",
		last.DrawCalls, last.Triangles, last.Instances, last.ShadowDrawCalls, last.ShadowTriangles)
}
