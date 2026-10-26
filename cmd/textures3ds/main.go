//go:build nintendo3ds

// The sample island's village on the Nintendo 3DS, close enough to see its textures: cobbles in
// RGBA8, planks and shingles in RGB565, grass, masonry and plaster in ETC1. Every ten seconds the
// textures go off or come back, and the bottom screen shows what frames cost either way.
//
//	tinygo build -target=nintendo3ds -o textures3ds.3dsx ./cmd/textures3ds
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
	"time"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/nintendo3ds"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

// The camera goes round the village's house, close.
var village = graphics.Vector{20, 24, 14, 1}

const (
	orbitRadius = 24
	orbitHeight = 12
	orbitPeriod = 30 * time.Second
	switchEvery = 10 * time.Second // textures on, then off, then on…
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
	world := sample.Island()
	textured := texturedMaterials(world)
	renderer, err := nintendo3ds.NewSceneRenderer(screen)
	if err != nil {
		println(err.Error())
		return
	}
	println("textures on the 3DS; START to exit")
	flyAround(world, textured, renderer)
}

// texturedMaterials keeps each textured material's texture, choosing a format for each so all
// three are drawn.
func texturedMaterials(world *scene.Scene) map[*scene.Material]*scene.Texture {
	formats := map[string]scene.TextureFormat{"cobbles": scene.RGBA8, "planks": scene.RGB565, "slate": scene.RGB565}
	textured := map[*scene.Material]*scene.Texture{}
	world.Walk(func(node *scene.Node, _ graphics.Matrix) {
		if node.Mesh == nil || node.Mesh.Material == nil || node.Mesh.Material.Texture == nil {
			return
		}
		material := node.Mesh.Material
		material.Texture.Format = scene.ETC1
		if format, named := formats[material.Name]; named {
			material.Texture.Format = format
		}
		textured[material] = material.Texture
	})
	return textured
}

// flyAround draws the village from a camera going round it, switching the textures off and on,
// until START is pressed or the system asks to close.
func flyAround(world *scene.Scene, textured map[*scene.Material]*scene.Texture, renderer *nintendo3ds.SceneRenderer) {
	view := camera.Lens{Target: village, Up: graphics.Vector{0, 1, 0, 0}, Far: 2000}
	started := time.Now()
	report := frameReport{last: started, on: true}
	for nintendo3ds.Running() {
		if nintendo3ds.ReadControls().Pressed&nintendo3ds.ButtonStart != 0 {
			return
		}
		on := int(time.Since(started)/switchEvery)%2 == 0
		if on != report.on {
			switchTextures(textured, on)
			report.restart(on)
		}
		turn := 2 * math.Pi * time.Since(started).Seconds() / orbitPeriod.Seconds()
		view.Position = graphics.Vector{village[0] + float32(orbitRadius*math.Sin(turn)), village[1] + orbitHeight, village[2] + float32(orbitRadius*math.Cos(turn)), 1}
		renderer.Draw(world, view)
		report.count()
	}
}

func switchTextures(textured map[*scene.Material]*scene.Texture, on bool) {
	for material, texture := range textured {
		material.Texture = nil
		if on {
			material.Texture = texture
		}
	}
}

// frameReport adds up the frames' times and, once a second, writes their averages on the second
// screen, with whether the textures are on.
type frameReport struct {
	frames              int
	processing, drawing float32 // milliseconds, added up
	last                time.Time
	on                  bool
}

func (report *frameReport) restart(on bool) {
	report.frames, report.processing, report.drawing, report.last, report.on = 0, 0, 0, time.Now(), on
}

func (report *frameReport) count() {
	report.processing += nintendo3ds.ProcessingTime()
	report.drawing += nintendo3ds.DrawingTime()
	report.frames++
	if time.Since(report.last) < time.Second {
		return
	}
	processing, drawing := report.processing/float32(report.frames), report.drawing/float32(report.frames)
	print("\x1b[4;0H")
	if report.on {
		println("textures:          on ")
	} else {
		println("textures:          off")
	}
	println("frames a second:  ", report.frames, "   ")
	println("processor (ms):   ", int(processing*10)/10, ".", int(processing*10)%10, "  ")
	println("GPU (ms):         ", int(drawing*10)/10, ".", int(drawing*10)%10, "  ")
	report.restart(report.on)
}
