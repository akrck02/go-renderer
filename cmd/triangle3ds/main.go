//go:build nintendo3ds

// A first drawing on the Nintendo 3DS through nintendo3ds: a triangle of three colours turning on the
// top screen, and on the bottom one how long frames take and how often the collector runs.
//
//	tinygo build -target=nintendo3ds -o triangle3ds.3dsx ./cmd/triangle3ds
//
// The shader is assembled with devkitPro's picasso:
//
//go:generate picasso -o triangle.shbin triangle.v.pica
package main

import (
	_ "embed"
	"math"
	"runtime"
	"time"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/nintendo3ds"
)

//go:embed triangle.shbin
var shaderBinary []byte

// Position (x, y, z) and colour (r, g, b, a) of each corner, around the origin. The view is
// right-handed, as everywhere in go-renderer: what is visible lies between z = 0 and z = -1.
var corners = []float32{
	0, 80, -0.5, 1, 0.2, 0.2, 1,
	-70, -40, -0.5, 0.2, 1, 0.2, 1,
	70, -40, -0.5, 0.2, 0.4, 1, 1,
}

const skyBlue = 0x68B0D8FF

func main() {
	screen, err := nintendo3ds.OpenScreen()
	if err != nil {
		println(err.Error())
		return
	}
	defer screen.Close()
	nintendo3ds.OpenSecondScreenConsole()
	println("triangle through citro3d")
	checkProjection()

	shader, err := nintendo3ds.LoadShader(shaderBinary)
	if err != nil {
		println(err.Error())
		return
	}
	defer shader.Free()
	triangle, err := nintendo3ds.NewVertexBuffer(corners, []uint8{3, 4})
	if err != nil {
		println(err.Error())
		return
	}
	defer triangle.Free()
	println("press START to exit")
	turnTriangle(screen, shader, triangle)
}

// turnTriangle draws the triangle turning until START is pressed or the system asks to close.
func turnTriangle(screen *nintendo3ds.Screen, shader *nintendo3ds.Shader, triangle *nintendo3ds.VertexBuffer) {
	shader.Use()
	projectionLocation := shader.Uniform("projection")
	modelViewLocation := shader.Uniform("modelView")
	projection := nintendo3ds.OrthographicTilt(0, 400, 0, 240, 0, 1)
	centre := graphics.Translate(graphics.Vector{200, 120, 0, 1})
	started := time.Now()
	report := frameReport{last: started}
	for nintendo3ds.Running() {
		if nintendo3ds.ReadControls().Pressed&nintendo3ds.ButtonStart != 0 {
			return
		}
		turn := time.Since(started).Seconds() * math.Pi / 2 // a quarter turn a second
		modelView := centre.Multiply(graphics.Rotate(graphics.Vector{0, 0, 1, 0}, turn))

		nintendo3ds.BeginFrame()
		screen.Clear(skyBlue)
		nintendo3ds.SetMatrix(projectionLocation, projection)
		nintendo3ds.SetMatrix(modelViewLocation, modelView)
		triangle.DrawTriangles()
		nintendo3ds.EndFrame()
		report.count()
	}
}

// frameReport counts frames and, once a second, writes how many there were and what the collector
// did on the second screen.
type frameReport struct {
	frames     int
	last       time.Time
	statistics runtime.MemStats
}

func (report *frameReport) count() {
	report.frames++
	if time.Since(report.last) < time.Second {
		return
	}
	runtime.ReadMemStats(&report.statistics)
	print("\x1b[6;0H")
	println("frames a second:", report.frames, "   ")
	println("collections:    ", report.statistics.NumGC, "   ")
	println("allocated (KB): ", report.statistics.TotalAlloc/1024, "   ")
	report.frames, report.last = 0, time.Now()
}

// checkProjection compares nintendo3ds.OrthographicTilt with citro3d's own Mtx_OrthoTilt.
func checkProjection() {
	ours := nintendo3ds.GPUMatrix(nintendo3ds.OrthographicTilt(0, 400, 0, 240, 0, 1))
	theirs := nintendo3ds.ReferenceOrthographicTilt(0, 400, 0, 240, 0, 1)
	for index := range ours {
		difference := ours[index] - theirs[index]
		if difference > 1e-6 || difference < -1e-6 {
			println("FAIL projection differs at", index)
			for row := 0; row < 16; row += 4 {
				println("  ours  ", int(ours[row]*1000), int(ours[row+1]*1000), int(ours[row+2]*1000), int(ours[row+3]*1000))
				println("  theirs", int(theirs[row]*1000), int(theirs[row+1]*1000), int(theirs[row+2]*1000), int(theirs[row+3]*1000))
			}
			return
		}
	}
	println("ok   projection matches citro3d")
}
