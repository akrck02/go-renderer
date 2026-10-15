package metal

import (
	"testing"
	"unsafe"

	"github.com/akrck02/go-renderer/graphics"
)

// The shader's static_asserts check its side; these check the Go side against the same numbers.
func TestUniformBlocksHaveTheShadersSizes(t *testing.T) {
	sizes := map[string][2]uintptr{
		"FrameUniforms":         {unsafe.Sizeof(frameUniforms{}), 528},
		"DrawUniforms":          {unsafe.Sizeof(drawUniforms{}), 128},
		"SkyUniforms":           {unsafe.Sizeof(skyUniforms{}), 352},
		"ConstellationUniforms": {unsafe.Sizeof(constellationUniforms{}), 176},
		"OverlaySizes":          {unsafe.Sizeof(overlaySizes{}), 32},
	}
	for name, size := range sizes {
		if size[0] != size[1] {
			t.Errorf("%s is %d bytes in Go, %d in the shader", name, size[0], size[1])
		}
	}
}

// A few offsets after the vectors, where a forgotten field would shift everything that follows.
func TestUniformFieldsSitWhereTheShaderReadsThem(t *testing.T) {
	var frame frameUniforms
	var sky skyUniforms
	offsets := map[string][2]uintptr{
		"frame.windDirection":  {unsafe.Offsetof(frame.windDirection), 4*64 + 11*16},
		"frame.verticalScale":  {unsafe.Offsetof(frame.verticalScale), 4*64 + 11*16 + 4*8},
		"frame.shadowsEnabled": {unsafe.Offsetof(frame.shadowsEnabled), 4*64 + 11*16 + 4*8 + 13*4},
		"sky.cloudOffset":      {unsafe.Offsetof(sky.cloudOffset), 2*64 + 9*16},
		"sky.moonEnabled":      {unsafe.Offsetof(sky.moonEnabled), 2*64 + 9*16 + 2*8 + 13*4},
	}
	for name, offset := range offsets {
		if offset[0] != offset[1] {
			t.Errorf("%s is at byte %d, the shader reads it at %d", name, offset[0], offset[1])
		}
	}
}

func TestMetalClipSpaceKeepsThePictureAndHalvesTheDepthRange(t *testing.T) {
	projection := graphics.PerspectiveOpenGL(0.8, 1.5, 1, 100)
	converted := metalClipSpace(projection)
	for _, depth := range []float32{-1, -10, -100} {
		point := graphics.Vec4{2, 3, depth, 1}
		opengl, metal := projection.Apply(point), converted.Apply(point)
		if opengl[0] != metal[0] || opengl[1] != metal[1] || opengl[3] != metal[3] {
			t.Fatalf("x, y and w must not change: %v and %v", opengl, metal)
		}
		openglDepth, metalDepth := opengl[2]/opengl[3], metal[2]/metal[3]
		if difference := metalDepth - (openglDepth*0.5 + 0.5); difference > 1e-6 || difference < -1e-6 {
			t.Errorf("depth %v in OpenGL should be %v in Metal, got %v", openglDepth, openglDepth*0.5+0.5, metalDepth)
		}
	}
}
