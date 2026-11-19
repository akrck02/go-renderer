package metal

import (
	"testing"
	"unsafe"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/scene"
)

// The shader's static_asserts check its side; these check the Go side against the same numbers.
func TestUniformBlocksHaveTheShadersSizes(test *testing.T) {
	sizes := map[string][2]uintptr{
		"FrameUniforms":         {unsafe.Sizeof(frameUniforms{}), 528},
		"DrawUniforms":          {unsafe.Sizeof(drawUniforms{}), 272},
		"SkyUniforms":           {unsafe.Sizeof(skyUniforms{}), 352},
		"ConstellationUniforms": {unsafe.Sizeof(constellationUniforms{}), 176},
		"OverlaySizes":          {unsafe.Sizeof(overlaySizes{}), 32},
	}
	for name, size := range sizes {
		if size[0] != size[1] {
			test.Errorf("%s is %d bytes in Go, %d in the shader", name, size[0], size[1])
		}
	}
}

// A few offsets after the vectors, where a forgotten field would shift everything that follows.
func TestUniformFieldsSitWhereTheShaderReadsThem(test *testing.T) {
	var frame frameUniforms
	var sky skyUniforms
	var draw drawUniforms
	offsets := map[string][2]uintptr{
		"draw.lightCount":       {unsafe.Offsetof(draw.lightCount), 64 + 16 + 9*4},
		"draw.lightPositions":   {unsafe.Offsetof(draw.lightPositions), 128},
		"draw.lightColors":      {unsafe.Offsetof(draw.lightColors), 128 + 4*16},
		"draw.texturePlacement": {unsafe.Offsetof(draw.texturePlacement), 128 + 8*16},
		"frame.windDirection":   {unsafe.Offsetof(frame.windDirection), 4*64 + 11*16},
		"frame.verticalScale":   {unsafe.Offsetof(frame.verticalScale), 4*64 + 11*16 + 4*8},
		"frame.shadowsEnabled":  {unsafe.Offsetof(frame.shadowsEnabled), 4*64 + 11*16 + 4*8 + 13*4},
		"sky.cloudOffset":       {unsafe.Offsetof(sky.cloudOffset), 2*64 + 9*16},
		"sky.moonEnabled":       {unsafe.Offsetof(sky.moonEnabled), 2*64 + 9*16 + 2*8 + 13*4},
	}
	for name, offset := range offsets {
		if offset[0] != offset[1] {
			test.Errorf("%s is at byte %d, the shader reads it at %d", name, offset[0], offset[1])
		}
	}
}

func TestMetalClipSpaceKeepsThePictureAndHalvesTheDepthRange(test *testing.T) {
	projection := graphics.PerspectiveOpenGL(0.8, 1.5, 1, 100)
	converted := metalClipSpace(projection)
	for _, depth := range []float32{-1, -10, -100} {
		point := graphics.Vector{2, 3, depth, 1}
		opengl, metal := projection.Apply(point), converted.Apply(point)
		if opengl[0] != metal[0] || opengl[1] != metal[1] || opengl[3] != metal[3] {
			test.Fatalf("x, y and w must not change: %v and %v", opengl, metal)
		}
		openglDepth, metalDepth := opengl[2]/opengl[3], metal[2]/metal[3]
		if difference := metalDepth - (openglDepth*0.5 + 0.5); difference > 1e-6 || difference < -1e-6 {
			test.Errorf("depth %v in OpenGL should be %v in Metal, got %v", openglDepth, openglDepth*0.5+0.5, metalDepth)
		}
	}
}

func TestATexturedDrawTellsTheShaderHowToReadItsTexture(test *testing.T) {
	texture := &scene.Texture{Picture: draw2d.NewBlankImage(8, 8), Wrap: scene.Mirror, Sharp: true}
	material := &scene.Material{Texture: texture, TextureScale: [2]float32{4, 2}, TextureOffset: [2]float32{0.5, 0}}
	uniforms := drawUniformsFor(sceneplan.Command{Material: material}, false, nil)
	if uniforms.textured != 1 || uniforms.textureSampling != 2*2+1 || uniforms.texturePlacement != [4]float32{4, 2, 0.5, 0} {
		test.Errorf("textured %d, sampler %d, placement %v", uniforms.textured, uniforms.textureSampling, uniforms.texturePlacement)
	}
	if plain := drawUniformsFor(sceneplan.Command{Material: &scene.Material{}}, false, nil); plain.textured != 0 {
		test.Error("a plain material is drawn textured")
	}
}
