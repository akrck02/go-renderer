package opengl

import (
	"time"

	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// FrameStatistics describes the last frame drawn by a SceneRenderer. Counts are always filled;
// times only when SceneRenderer.Profiling is on.
type FrameStatistics struct {
	DrawCalls       int // main pass
	ShadowDrawCalls int
	Triangles       int // main pass, instances included
	ShadowTriangles int
	Instances       int // instances drawn in the main pass

	CPU time.Duration // time spent by Draw on the processor (without waiting for the GPU)
	GPU PassTimes     // time spent by the GPU on each pass
}

// PassTimes are GPU durations of the passes of one frame.
type PassTimes struct {
	Seabed, Shadows, Sky, Scene time.Duration
}

// Total returns the sum of the pass durations.
func (times PassTimes) Total() time.Duration {
	return times.Seabed + times.Shadows + times.Sky + times.Scene
}

// passProfiler measures GPU passes with timer queries. Results are read at the end of the frame,
// which waits for the GPU: profiling slows the frame down and is meant for measurements only.
type passProfiler struct {
	queries map[string]uint32
	started []string
}

func newPassProfiler() *passProfiler {
	return &passProfiler{queries: map[string]uint32{}}
}

func (profiler *passProfiler) begin(pass string) {
	query, created := profiler.queries[pass]
	if !created {
		gl.GenQueries(1, &query)
		profiler.queries[pass] = query
	}
	gl.BeginQuery(gl.TIME_ELAPSED, query)
	profiler.started = append(profiler.started, pass)
}

func (profiler *passProfiler) end() {
	gl.EndQuery(gl.TIME_ELAPSED)
}

// collect waits for the passes of this frame and returns their durations.
func (profiler *passProfiler) collect() PassTimes {
	var times PassTimes
	for _, pass := range profiler.started {
		var nanoseconds uint64
		gl.GetQueryObjectui64v(profiler.queries[pass], gl.QUERY_RESULT, &nanoseconds)
		*passField(&times, pass) += time.Duration(nanoseconds)
	}
	profiler.started = profiler.started[:0]
	return times
}

func passField(times *PassTimes, pass string) *time.Duration {
	switch pass {
	case "seabed":
		return &times.Seabed
	case "shadows":
		return &times.Shadows
	case "sky":
		return &times.Sky
	default:
		return &times.Scene
	}
}

func castsShadow(command drawCommand) bool {
	return !command.transparent && command.material.Kind != scene.KindUnlit
}
