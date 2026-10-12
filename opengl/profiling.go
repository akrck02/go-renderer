package opengl

import (
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
)

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
