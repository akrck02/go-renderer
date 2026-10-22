//go:build nintendo3ds

// The sample island on the Nintendo 3DS, drawn by nintendo3ds's scene renderer, with the camera going
// round it. The bottom screen shows how fast frames go and what they cost.
//
//	tinygo build -target=nintendo3ds -o island3ds.3dsx ./cmd/island3ds
package main

/*
#include <3ds.h>

// fastCPU asks for the New 3DS's 804 MHz clock and its L2 cache; the original 3DS ignores it.
static void fastCPU(void) {
	osSetSpeedupEnable(true);
}
*/
import "C"

import (
	"math"
	"runtime"
	"time"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/nintendo3ds"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

// The orbit stays well inside the island's fog, which is total at 900 units.
const (
	orbitRadius = 420
	orbitHeight = 220
	orbitPeriod = 40 * time.Second
)

func main() {
	C.fastCPU()
	screen, err := nintendo3ds.OpenScreen()
	if err != nil {
		println(err.Error())
		return
	}
	defer screen.Close()
	nintendo3ds.OpenSecondScreenConsole()
	println("sample island through nintendo3ds")

	world := sample.Island()
	renderer, err := nintendo3ds.NewSceneRenderer(screen)
	if err != nil {
		println(err.Error())
		return
	}
	println("press START to exit")
	flyAround(world, renderer)
}

// flyAround draws the island from a camera going round it, until START is pressed or the system
// asks to close.
func flyAround(world *scene.Scene, renderer *nintendo3ds.SceneRenderer) {
	view := camera.Camera{Target: graphics.Vector{0, 0, 0, 1}, Up: graphics.Vector{0, 1, 0, 0}, Far: 20000}
	started := time.Now()
	report := frameReport{last: started}
	for nintendo3ds.Running() {
		if nintendo3ds.ReadControls().Pressed&nintendo3ds.ButtonStart != 0 {
			return
		}
		turn := 2 * math.Pi * time.Since(started).Seconds() / orbitPeriod.Seconds()
		view.Position = graphics.Vector{float32(orbitRadius * math.Sin(turn)), orbitHeight, float32(orbitRadius * math.Cos(turn)), 1}
		renderer.Draw(world, view)
		report.count(renderer)
	}
}

// frameReport adds up the frames' times and, once a second, writes their averages on the second
// screen.
type frameReport struct {
	frames              int
	processing, drawing float32 // milliseconds, added up
	last                time.Time
	statistics          runtime.MemStats
}

func (report *frameReport) count(renderer *nintendo3ds.SceneRenderer) {
	report.processing += nintendo3ds.ProcessingTime()
	report.drawing += nintendo3ds.DrawingTime()
	report.frames++
	if time.Since(report.last) < time.Second {
		return
	}
	drawn := renderer.Statistics(0)
	runtime.ReadMemStats(&report.statistics)
	processing, drawing := report.processing/float32(report.frames), report.drawing/float32(report.frames)
	print("\x1b[4;0H")
	println("frames a second:  ", report.frames, "   ")
	println("processor (ms):   ", int(processing*10)/10, ".", int(processing*10)%10, "  ")
	println("GPU (ms):         ", int(drawing*10)/10, ".", int(drawing*10)%10, "  ")
	println("draw calls:       ", drawn.DrawCalls, "   ")
	println("triangles:        ", drawn.Triangles, "   ")
	println("collections:      ", report.statistics.NumGC, "   ")
	println("allocated (KB):   ", report.statistics.TotalAlloc/1024, "   ")
	report.frames, report.processing, report.drawing, report.last = 0, 0, 0, time.Now()
}
