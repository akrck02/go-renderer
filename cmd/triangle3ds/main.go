//go:build nintendo3ds

// A first drawing on the Nintendo 3DS through ctr: a triangle of three colours turning on the
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

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/graphics"
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
	screen, err := ctr.Open()
	if err != nil {
		println(err.Error())
		return
	}
	defer screen.Close()
	ctr.OpenBottomConsole()
	println("triangle through citro3d")
	checkProjection()

	shader, err := ctr.LoadShader(shaderBinary)
	if err != nil {
		println(err.Error())
		return
	}
	defer shader.Free()
	triangle, err := ctr.NewVertexBuffer(corners, []uint8{3, 4})
	if err != nil {
		println(err.Error())
		return
	}
	defer triangle.Free()

	shader.Use()
	projectionLocation := shader.Uniform("projection")
	modelViewLocation := shader.Uniform("modelView")
	projection := ctr.OrthographicTilt(0, 400, 0, 240, 0, 1)
	centre := graphics.Translate(graphics.Vec4{200, 120, 0, 1})

	println("press START to exit")
	started := time.Now()
	lastReport := started
	frames := 0
	var statistics runtime.MemStats
	for ctr.Running() {
		if ctr.ButtonsPressed()&ctr.ButtonStart != 0 {
			break
		}
		turn := time.Since(started).Seconds() * math.Pi / 2 // a quarter turn a second
		modelView := centre.Multiply(graphics.Rotate(graphics.Vec4{0, 0, 1, 0}, turn))

		screen.BeginFrame(skyBlue)
		ctr.SetMatrix(projectionLocation, projection)
		ctr.SetMatrix(modelViewLocation, modelView)
		triangle.DrawTriangles()
		screen.EndFrame()

		frames++
		if time.Since(lastReport) >= time.Second {
			runtime.ReadMemStats(&statistics)
			print("\x1b[6;0H")
			println("frames a second:", frames, "   ")
			println("collections:    ", statistics.NumGC, "   ")
			println("allocated (KB): ", statistics.TotalAlloc/1024, "   ")
			frames = 0
			lastReport = time.Now()
		}
	}
}

// checkProjection compares ctr.OrthographicTilt with citro3d's own Mtx_OrthoTilt.
func checkProjection() {
	ours := ctr.PICAMatrix(ctr.OrthographicTilt(0, 400, 0, 240, 0, 1))
	theirs := ctr.OrthographicTiltFromCitro3D(0, 400, 0, 240, 0, 1)
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
