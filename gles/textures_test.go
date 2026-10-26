package gles

import (
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
)

// triangle is a small triangle on the ground, with UVs when given.
func triangle(material *scene.Material, uvs []float32) *scene.Node {
	return &scene.Node{Mesh: &scene.Mesh{Positions: []float32{-1, 0, -1, 1, 0, -1, 0, 0, 1},
		Normals: []float32{0, 1, 0, 0, 1, 0, 0, 1, 0}, Indices: []uint32{0, 1, 2}, UVs: uvs, Material: material}}
}

func newTexturedScene(test *testing.T) (*recorder, *SceneRenderer, camera.Lens) {
	test.Helper()
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake, overlay.Fonts{})
	if err != nil {
		test.Fatal(err)
	}
	lens := camera.Lens{Position: graphics.Vector{0, 5, 5, 1}, Up: graphics.Vector{0, 1, 0, 0}, FieldOfViewRadians: 1, Near: 0.1, Far: 100}
	return fake, renderer, lens
}

func TestATextureOnAHundredMeshesIsUploadedOnceAndAgainAfterAChange(test *testing.T) {
	fake, renderer, lens := newTexturedScene(test)
	material := &scene.Material{Texture: &scene.Texture{Picture: draw2d.NewBlankImage(16, 16)}}
	world := scene.New()
	for range 100 {
		world.Nodes = append(world.Nodes, triangle(material, []float32{0, 0, 1, 0, 0, 1}))
	}
	renderer.Draw(world, lens, 640, 480)
	if renderer.textures.uploads != 1 {
		test.Errorf("the first frame uploaded the texture %d times, wanted once", renderer.textures.uploads)
	}
	fake.forget()
	renderer.Draw(world, lens, 640, 480)
	if fake.textureUploads != 0 {
		test.Errorf("a frame with nothing changed uploaded %d textures", fake.textureUploads)
	}
	material.Texture.Picture.Changed()
	renderer.Draw(world, lens, 640, 480)
	if fake.textureUploads != 1 {
		test.Errorf("a changed picture was uploaded %d times, wanted once", fake.textureUploads)
	}
}

func TestOnlyAMeshDrawnWithATextureSendsUVs(test *testing.T) {
	fake, renderer, lens := newTexturedScene(test)
	world := scene.New()
	world.Nodes = []*scene.Node{triangle(&scene.Material{}, nil)}
	renderer.Draw(world, lens, 640, 480)
	if buffers := fake.bufferUploads[ArrayBuffer]; buffers != 1 {
		test.Errorf("a mesh without a texture filled %d vertex buffers, wanted its vertices only", buffers)
	}
	fake.forget()
	world.Nodes = []*scene.Node{triangle(&scene.Material{Texture: &scene.Texture{Picture: draw2d.NewBlankImage(16, 16)}}, nil)}
	renderer.Draw(world, lens, 640, 480)
	if buffers := fake.bufferUploads[ArrayBuffer]; buffers != 2 {
		test.Errorf("a textured mesh filled %d vertex buffers, wanted its vertices and its UVs", buffers)
	}
}

func TestATextureIsSampledAsItAsks(test *testing.T) {
	cases := []struct {
		name            string
		texture         scene.Texture
		wrap, minifying int32
		mipmaps         int
	}{
		{"smooth, repeated", scene.Texture{Picture: draw2d.NewBlankImage(16, 16)}, int32(Repeat), int32(LinearMipmapLinear), 1},
		{"sharp, mirrored", scene.Texture{Picture: draw2d.NewBlankImage(16, 16), Sharp: true, Wrap: scene.Mirror}, int32(MirroredRepeat), int32(Nearest), 0},
		{"clamped", scene.Texture{Picture: draw2d.NewBlankImage(16, 16), Wrap: scene.Clamp}, int32(ClampToEdge), int32(LinearMipmapLinear), 1},
		{"not a power of two", scene.Texture{Picture: draw2d.NewBlankImage(12, 16)}, int32(ClampToEdge), int32(Linear), 0},
	}
	for _, sample := range cases {
		fake, renderer, lens := newTexturedScene(test)
		world := scene.New()
		world.Nodes = []*scene.Node{triangle(&scene.Material{Texture: &sample.texture}, nil)}
		renderer.Draw(world, lens, 640, 480)
		if fake.parameters[TextureWrapS] != sample.wrap || fake.parameters[TextureMinFilter] != sample.minifying || fake.mipmaps != sample.mipmaps {
			test.Errorf("%s: wrap %#x, minifying %#x, %d mipmaps; wanted %#x, %#x, %d", sample.name,
				fake.parameters[TextureWrapS], fake.parameters[TextureMinFilter], fake.mipmaps, sample.wrap, sample.minifying, sample.mipmaps)
		}
	}
}

func TestTheSceneShaderMultipliesItsColourByTheTexture(test *testing.T) {
	fake, _, _ := newTexturedScene(test)
	shaders := fake.sourcesWith("surfaceTexture")
	if len(shaders) != 1 || !strings.Contains(shaders[0], "color *= textureColor(") {
		test.Fatalf("no scene shader multiplies by the surface texture: %d found", len(shaders))
	}
}

func TestAMeshOnceDrawnTexturedKeepsItsUVsInStepWhenItChanges(test *testing.T) {
	fake, renderer, _ := newTexturedScene(test)
	mesh := triangle(nil, nil).Mesh
	renderer.uploadMesh(mesh, &scene.Material{Texture: &scene.Texture{Picture: draw2d.NewBlankImage(8, 8)}})
	fake.forget()
	mesh.Dirty = true
	renderer.uploadMesh(mesh, nil) // a simpler version of it, drawn with its parent's material
	if buffers := fake.bufferUploads[ArrayBuffer]; buffers != 2 {
		test.Errorf("a changed mesh with a UV buffer filled %d vertex buffers, wanted its vertices and its UVs", buffers)
	}
}
