package sceneplan

import "time"

// FrameStatistics describes the last frame drawn by a scene renderer. Counts are always filled;
// times only when the renderer is profiling.
type FrameStatistics struct {
	DrawCalls       int // main pass
	ShadowDrawCalls int
	Triangles       int // main pass, instances included
	ShadowTriangles int
	Instances       int // instances drawn in the main pass

	CPU time.Duration // time spent drawing on the processor (without waiting for the GPU)
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

// Statistics returns the counts of the frame planned last, with the processor time it took.
func (planner *Planner) Statistics(processorTime time.Duration) FrameStatistics {
	main, shadow := planner.counters[MainPass], planner.counters[ShadowPass]
	return FrameStatistics{CPU: processorTime,
		DrawCalls: main.drawCalls, Triangles: main.triangles, Instances: main.instances,
		ShadowDrawCalls: shadow.drawCalls, ShadowTriangles: shadow.triangles}
}
