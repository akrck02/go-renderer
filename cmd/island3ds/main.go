//go:build nintendo3ds

// The sample island on the Nintendo 3DS, drawn by ctr's scene renderer, with the camera going
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

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
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
	screen, err := ctr.Open()
	if err != nil {
		println(err.Error())
		return
	}
	defer screen.Close()
	ctr.OpenBottomConsole()
	println("sample island through ctr")

	world := sample.Island()
	renderer, err := ctr.NewSceneRenderer(screen)
	if err != nil {
		println(err.Error())
		return
	}
	println("press START to exit")

	camera := models.Camera{Target: graphics.Vector{0, 0, 0, 1}, Up: graphics.Vector{0, 1, 0, 0}, Far: 20000}
	started := time.Now()
	lastReport := started
	frames := 0
	var processing, drawing float32
	var statistics runtime.MemStats
	for ctr.Running() {
		if ctr.ButtonsPressed()&ctr.ButtonStart != 0 {
			break
		}
		turn := 2 * math.Pi * time.Since(started).Seconds() / orbitPeriod.Seconds()
		camera.Position = graphics.Vector{float32(orbitRadius * math.Sin(turn)), orbitHeight, float32(orbitRadius * math.Cos(turn)), 1}

		renderer.Draw(world, camera)
		processing += ctr.ProcessingTime()
		drawing += ctr.DrawingTime()
		frames++

		if time.Since(lastReport) < time.Second {
			continue
		}
		drawn := renderer.Statistics(0)
		runtime.ReadMemStats(&statistics)
		print("\x1b[4;0H")
		println("frames a second:  ", frames, "   ")
		println("processor (ms):   ", int(processing/float32(frames)*10)/10, ".", int(processing/float32(frames)*10)%10, "  ")
		println("GPU (ms):         ", int(drawing/float32(frames)*10)/10, ".", int(drawing/float32(frames)*10)%10, "  ")
		println("draw calls:       ", drawn.DrawCalls, "   ")
		println("triangles:        ", drawn.Triangles, "   ")
		println("collections:      ", statistics.NumGC, "   ")
		println("allocated (KB):   ", statistics.TotalAlloc/1024, "   ")
		frames, processing, drawing, lastReport = 0, 0, 0, time.Now()
	}
}
