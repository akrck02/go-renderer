package opengl

import (
	"image/color"
	"testing"

	"github.com/go-gl/gl/v4.1-core/gl"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
)

// The frame the texture tests draw, in pixels.
const texturedWidth, texturedHeight = 256, 128

// checkerPicture is a 64-pixel picture of four squares, light and dark: a texture whose repeats
// can be counted on a frame.
func checkerPicture() *draw2d.Image {
	const side, square = 64, 32
	picture := draw2d.NewBlankImage(side, side)
	for y := range side {
		for x := range side {
			grey := uint8(255)
			if (x/square+y/square)%2 == 1 {
				grey = 64
			}
			picture.Pixels.SetNRGBA(x, y, color.NRGBA{grey, grey, grey, 255})
		}
	}
	return picture
}

// texturedQuad is a square two units a side facing the camera, at x, with UVs from 0 to 1.
func texturedQuad(x float32, material *scene.Material) *scene.Node {
	mesh := &scene.Mesh{Positions: []float32{-1, -1, 0, 1, -1, 0, 1, 1, 0, -1, 1, 0},
		Normals: []float32{0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1}, UVs: []float32{0, 0, 1, 0, 1, 1, 0, 1},
		Indices: []uint32{0, 1, 2, 0, 2, 3}, Material: material}
	return &scene.Node{Mesh: mesh, Transform: graphics.Translate(graphics.Vector{x, 0, 0, 0})}
}

// twoTintedQuads shows one checker texture, repeated eight times a side, on two quads: red and
// smooth on the left, blue and sharp on the right.
func twoTintedQuads() (*scene.Scene, camera.Lens) {
	smooth := &scene.Texture{Picture: checkerPicture()}
	sharp := &scene.Texture{Picture: smooth.Picture, Sharp: true}
	repeats := [2]float32{8, 8}
	world := scene.New()
	world.Environment.SunDirection = graphics.Vector{0, 0, 1, 0}
	world.Nodes = []*scene.Node{
		texturedQuad(-1.1, &scene.Material{BaseColor: graphics.Vector{1, 0.2, 0.2, 1}, Texture: smooth, TextureScale: repeats}),
		texturedQuad(1.1, &scene.Material{BaseColor: graphics.Vector{0.2, 0.3, 1, 1}, Texture: sharp, TextureScale: repeats}),
	}
	lens := camera.Lens{Position: graphics.Vector{0, 0, 4.2, 1}, Target: graphics.Vector{0, 0, 0, 1},
		Up: graphics.Vector{0, 1, 0, 0}, FieldOfViewRadians: 0.6, Near: 0.1, Far: 100}
	return world, lens
}

func newTextureTestRenderer(test testing.TB) *SceneRenderer {
	test.Helper()
	openTestContext(test, texturedWidth, texturedHeight)
	renderer, err := NewSceneRenderer(overlay.Fonts{})
	if err != nil {
		test.Fatal(err)
	}
	renderer.Shadows.Enabled = false
	return renderer
}

// TestATintedTextureRepeatsAcrossItsSurface draws one picture eight times across each of two quads,
// tinted red and smooth, blue and sharp, and compares the frame with its reference picture.
func TestATintedTextureRepeatsAcrossItsSurface(test *testing.T) {
	renderer := newTextureTestRenderer(test)
	world, lens := twoTintedQuads()
	renderer.Draw(world, lens, texturedWidth, texturedHeight, 0)
	checkReferencePicture(test, "two-tinted-textures", texturedWidth, texturedHeight)
}

// TestATextureOnAHundredMeshesIsUploadedOnce draws a hundred meshes that share a texture: it is
// uploaded once, and once more after it changes, however many frames are drawn.
func TestATextureOnAHundredMeshesIsUploadedOnce(test *testing.T) {
	renderer := newTextureTestRenderer(test)
	world, lens := twoTintedQuads()
	material := world.Nodes[0].Mesh.Material
	world.Nodes = nil
	for index := range 100 {
		world.Nodes = append(world.Nodes, texturedQuad(float32(index%10)*0.2-1, material))
	}
	renderer.Draw(world, lens, texturedWidth, texturedHeight, 0)
	renderer.Draw(world, lens, texturedWidth, texturedHeight, 0)
	if renderer.textures.uploads != 1 {
		test.Errorf("the texture was uploaded %d times, wanted once", renderer.textures.uploads)
	}
	material.Texture.Picture.Changed()
	renderer.Draw(world, lens, texturedWidth, texturedHeight, 0)
	renderer.Draw(world, lens, texturedWidth, texturedHeight, 0)
	if renderer.textures.uploads != 2 {
		test.Errorf("after a change the texture was uploaded %d times in all, wanted twice", renderer.textures.uploads)
	}
}

// TestAFrameWithTexturesAllocatesNothing draws textured meshes, smooth and sharp, frame after
// frame: once the first frame has uploaded them, no frame makes garbage.
func TestAFrameWithTexturesAllocatesNothing(test *testing.T) {
	renderer := newTextureTestRenderer(test)
	world, lens := twoTintedQuads()
	renderer.DrawIn(world, lens, Area{Width: texturedWidth, Height: texturedHeight}, 0)
	allocations := testing.AllocsPerRun(10, func() {
		renderer.DrawIn(world, lens, Area{Width: texturedWidth, Height: texturedHeight}, 0)
		gl.Finish()
	})
	if allocations != 0 {
		test.Fatalf("a frame allocates %v times", allocations)
	}
}
