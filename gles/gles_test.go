package gles

import (
	"errors"
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/internal/shading"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

func TestShadersAreOpenGLESTwo(test *testing.T) {
	for name, source := range map[string]string{
		"quad vertex": quadVertexShader, "quad fragment": quadFragmentShader,
		"scene vertex": sceneVertexShader, "scene fragment": sceneFragmentShader,
		"backdrop vertex": backdropVertexShader, "backdrop fragment": backdropBlurShader,
		"light vertex": lightVertexShader, "light fragment": lightFragmentShader, "light multiply fragment": lightMultiplyShader,
		"light area vertex": lightAreaVertexShader,
	} {
		if !strings.HasPrefix(source, "#version 100\n") {
			test.Errorf("%s shader does not start with the GLSL ES 1.00 version line", name)
		}
		for _, newer := range []string{"layout(", " in ", " out ", "texture(", "inverse("} {
			if strings.Contains(source, newer) {
				test.Errorf("%s shader uses %q, which OpenGL ES 2 does not have", name, strings.TrimSpace(newer))
			}
		}
		if strings.Contains(name, "fragment") && !strings.Contains(source, "precision ") {
			test.Errorf("%s shader has no default precision, which OpenGL ES requires", name)
		}
		if strings.Contains(source, "\x00") {
			test.Errorf("%s shader holds a zero byte", name)
		}
	}
}

func TestACompileErrorIsReported(test *testing.T) {
	fake := newRecorder()
	fake.failCompile = true
	_, err := NewFlatRenderer(fake)
	if !errors.Is(err, ShaderNotCompiled) || !strings.Contains(err.Error(), "broken on purpose") {
		test.Fatalf("error = %v, want %s with the compiler's log", err, ShaderNotCompiled.Name())
	}
}

func TestALinkErrorIsReported(test *testing.T) {
	fake := newRecorder()
	fake.failLink = true
	_, err := NewFlatRenderer(fake)
	if !errors.Is(err, ProgramNotLinked) || !strings.Contains(err.Error(), "broken on purpose") {
		test.Fatalf("error = %v, want %s with the linker's log", err, ProgramNotLinked.Name())
	}
}

func card() *draw2d.List {
	var list draw2d.List
	area := draw2d.Rect{X: 20, Y: 20, Width: 200, Height: 60}
	list.Rectangle(area, 8, draw2d.Hex("#ffffff"), draw2d.Hex("#dde3ea"), 1)
	list.Text("Lighthouse", 36, 56, 14, draw2d.Regular, draw2d.Hex("#18212e"))
	return &list
}

func TestTheFlatRendererDrawsAFrameInOneCall(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	renderer.Draw(card(), draw2d.Hex("#eef1f5"), 640, 480, 1)
	if len(fake.draws) != 1 {
		test.Fatalf("draw calls = %d, want 1", len(fake.draws))
	}
	if count := fake.draws[0].count; count == 0 || count%6 != 0 {
		test.Fatalf("vertices = %d, want six per rectangle", count)
	}
	if fake.textureUploads != 1 {
		test.Fatalf("atlas uploads = %d on the first frame, want 1", fake.textureUploads)
	}
}

func TestTheAtlasIsUploadedOnlyWhenItChanges(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	renderer.Draw(card(), draw2d.Hex("#eef1f5"), 640, 480, 1)
	fake.forget()
	renderer.Draw(card(), draw2d.Hex("#eef1f5"), 640, 480, 1)
	if fake.textureUploads != 0 {
		test.Fatalf("atlas uploads = %d on a frame with the same glyphs, want 0", fake.textureUploads)
	}
	if fake.bufferUploads[ArrayBuffer] != 1 {
		test.Fatalf("vertex uploads = %d, want the frame's rectangles once", fake.bufferUploads[ArrayBuffer])
	}
}

func TestAnEmptyListDrawsNothing(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	renderer.Draw(&draw2d.List{}, draw2d.Hex("#eef1f5"), 640, 480, 1)
	if len(fake.draws) != 0 || fake.textureUploads != 0 {
		test.Fatalf("draws %d, atlas uploads %d: an empty list should only clear", len(fake.draws), fake.textureUploads)
	}
}

func TestQuadVerticesKeepTheirCornersAndColour(test *testing.T) {
	quad := overlay.Quad{Screen: [4]float32{10, 20, 30, 40}, Atlas: [4]float32{1, 2, 3, 4}, Color: [4]float32{0.5, 0.25, 1, 1}}
	vertices := appendQuadVertices(nil, []overlay.Quad{quad})
	if len(vertices) != 6*floatsPerQuadVertex {
		test.Fatalf("floats = %d, want %d", len(vertices), 6*floatsPerQuadVertex)
	}
	topLeft := vertices[0:8]
	want := []float32{10, 20, 1, 2, 0.5, 0.25, 1, 1}
	for index := range want {
		if topLeft[index] != want[index] {
			test.Fatalf("first corner = %v, want %v", topLeft, want)
		}
	}
	lastCorner := vertices[5*floatsPerQuadVertex : 5*floatsPerQuadVertex+4]
	if lastCorner[0] != 30 || lastCorner[1] != 20 || lastCorner[2] != 3 || lastCorner[3] != 2 {
		test.Fatalf("last corner = %v, want the top-right one", lastCorner)
	}
}

func islandCamera() models.Camera {
	return models.Camera{Position: graphics.Vector{0, 600, 900, 1}, Target: graphics.Vector{0, 0, 0, 1}, Up: graphics.Vector{0, 1, 0, 0}, Far: 20000}
}

func TestTheSceneRendererDrawsTheSampleIsland(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	world := sample.Island()
	renderer.Draw(world, islandCamera(), 1280, 800)
	// the ground, the water and the other meshes, and the forest in a few batches
	if len(fake.draws) < 5 || len(fake.draws) > 60 {
		test.Fatalf("draw calls = %d, want the meshes and a call per batch of trees", len(fake.draws))
	}
	if fake.bufferUploads[ElementArrayBuffer] == 0 {
		test.Fatal("no index buffer was uploaded")
	}
}

func TestMeshesAreUploadedOnce(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	world := sample.Island()
	renderer.Draw(world, islandCamera(), 1280, 800)
	fake.forget()
	renderer.Draw(world, islandCamera(), 1280, 800)
	if uploads := fake.bufferUploads[ElementArrayBuffer]; uploads != 0 {
		test.Fatalf("index uploads on the second frame = %d, want 0", uploads)
	}
	if len(fake.draws) == 0 {
		test.Fatal("the second frame drew nothing")
	}
}

func TestADirtyMeshIsUploadedAgain(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	world := sample.Island()
	renderer.Draw(world, islandCamera(), 1280, 800)
	world.Nodes[0].Mesh.Dirty = true
	fake.forget()
	renderer.Draw(world, islandCamera(), 1280, 800)
	if fake.bufferUploads[ElementArrayBuffer] != 1 {
		test.Fatalf("index uploads = %d, want the changed mesh once", fake.bufferUploads[ElementArrayBuffer])
	}
}

func TestAttributesAreBoundBeforeLinking(test *testing.T) {
	fake := newRecorder()
	if _, err := NewSceneRenderer(fake); err != nil {
		test.Fatal(err)
	}
	want := append(append([]string{}, sceneAttributes...), quadAttributes...)
	if strings.Join(fake.attributes, ",") != strings.Join(want, ",") {
		test.Fatalf("attributes bound = %v, want %v", fake.attributes, want)
	}
}

func TestANewRendererUploadsTheMeshesAgain(test *testing.T) {
	world := sample.Island()
	first, err := NewSceneRenderer(newRecorder())
	if err != nil {
		test.Fatal(err)
	}
	first.Draw(world, islandCamera(), 1280, 800)
	// a phone came back from the background: a new graphics context, a new renderer
	fake := newRecorder()
	second, err := NewSceneRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	second.Draw(world, islandCamera(), 1280, 800)
	if fake.bufferUploads[ElementArrayBuffer] == 0 {
		test.Fatal("the new renderer drew the ground with the old context's buffers")
	}
}

func TestASecondFrameSendsOnlyWhatChanged(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	world := sample.Island()
	renderer.Draw(world, islandCamera(), 1280, 800)
	firstValues := fake.uniformValues
	fake.forget()
	renderer.Draw(world, islandCamera(), 1280, 800)
	// every call is a trip through the bindings: x/mobile's crosses threads every three calls. What
	// is left changes from one draw to the next (a material's colour and kind)
	if fake.uniformValues >= firstValues {
		test.Fatalf("uniform values sent: %d on the first frame, %d on the same frame again; want fewer", firstValues, fake.uniformValues)
	}
	// the three both programs read stay on; the quads' own two (pictures and backdrop) go on for
	// the overlay and off after
	if fake.enables > 2 {
		test.Fatalf("vertex attributes switched on %d times again; want only the quads' own two", fake.enables)
	}
}

func TestAnImageIsDrawnInTheSameCallAndOnlyItsChangesAreSent(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	photo := draw2d.NewBlankImage(64, 48)
	list := card()
	list.Image(photo, draw2d.Rect{X: 20, Y: 100, Width: 120, Height: 90}, 12, draw2d.Hex("#ffffff"))
	renderer.Draw(list, draw2d.Hex("#eef1f5"), 640, 480, 1)
	if len(fake.draws) != 1 {
		test.Fatalf("draw calls = %d with an image, want still 1", len(fake.draws))
	}
	if fake.textureUploads != 2 {
		test.Fatalf("whole uploads = %d on the first frame, want the atlas and the pictures", fake.textureUploads)
	}
	fake.forget()
	photo.Changed()
	renderer.Draw(list, draw2d.Hex("#eef1f5"), 640, 480, 1)
	if fake.textureUploads != 0 || fake.textureAreaUploads != 1 {
		test.Fatalf("whole uploads %d, area uploads %d after the picture changed; want only its area",
			fake.textureUploads, fake.textureAreaUploads)
	}
}

func TestAFlatListWithoutBackdropsIsOneDrawAndNoSnapshot(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	var list draw2d.List
	list.Rectangle(draw2d.Rect{X: 10, Y: 10, Width: 200, Height: 100}, 12, draw2d.Hex("#ffffff20"), draw2d.Transparent, 0)
	renderer.Draw(&list, draw2d.Hex("#202020"), 800, 600, 1)
	if len(fake.draws) != 1 || fake.snapshots != 0 {
		test.Fatalf("no backdrop: one draw and no copy of the frame, got %d draws and %d copies", len(fake.draws), fake.snapshots)
	}
}

func TestPanelsOfOneLayerShareOneSnapshot(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	var list draw2d.List
	list.Rectangle(draw2d.Rect{X: 0, Y: 0, Width: 800, Height: 600}, 0, draw2d.Hex("#808080"), draw2d.Transparent, 0)
	for row := float32(0); row < 4; row++ {
		panel := draw2d.Rect{X: 20, Y: 20 + row*140, Width: 300, Height: 120}
		list.Backdrop(panel, 12, 24)
		list.Rectangle(panel, 12, draw2d.Hex("#ffffff20"), draw2d.Transparent, 0)
	}
	renderer.Draw(&list, draw2d.Hex("#202020"), 800, 600, 1)
	if fake.snapshots != 1 {
		test.Fatalf("four panels of one layer copy the frame once, got %d", fake.snapshots)
	}
	// the ground; then the halvings and two blur passes; then the rest of the quads
	halvings := shading.BackdropHalvings(24)
	if len(fake.draws) != 1+halvings+2+1 {
		test.Fatalf("got %d draws, wanted %d", len(fake.draws), 1+halvings+2+1)
	}
}

func TestALightingAddsItsLightsAndMultipliesTheScreenInTwoDraws(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	var list draw2d.List
	list.Rectangle(draw2d.Rect{Width: 800, Height: 600}, 0, draw2d.Hex("#808080"), draw2d.Transparent, 0)
	list.PointLight(200, 200, 20, 80, draw2d.Hex("#ffeeaa"))
	list.PointLight(600, 200, 20, 80, draw2d.Hex("#aaeeff"))
	list.Lighting(draw2d.Color{0.6, 0.6, 0.8, 0.7})
	list.Text("Score", 10, 30, 16, draw2d.Regular, draw2d.Hex("#ffffff"))
	renderer.Draw(&list, draw2d.Hex("#202020"), 800, 600, 1)
	// the ground; the two lights in one draw; the multiply over the screen; the text after
	if len(fake.draws) != 4 {
		test.Fatalf("got %d draws, wanted 4", len(fake.draws))
	}
	if lights := fake.draws[1].count; lights != 2*cornersPerQuad {
		test.Errorf("the lights draw %d corners, wanted %d", lights, 2*cornersPerQuad)
	}
	if fake.snapshots != 0 {
		test.Errorf("lights copy no frame, got %d copies", fake.snapshots)
	}
}

func TestLightsAndBackdropsDrawBackIntoTheRenderersTarget(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	renderer.Target = fake.CreateFramebuffer() // a picture the caller draws into, not the screen
	var list draw2d.List
	list.Rectangle(draw2d.Rect{Width: 800, Height: 600}, 0, draw2d.Hex("#808080"), draw2d.Transparent, 0)
	list.PointLight(200, 200, 20, 80, draw2d.Hex("#ffeeaa"))
	list.Lighting(draw2d.Color{0.6, 0.6, 0.8, 0.7})
	panel := draw2d.Rect{X: 20, Y: 20, Width: 300, Height: 120}
	list.Backdrop(panel, 12, 24)
	list.Rectangle(panel, 12, draw2d.Hex("#ffffff20"), draw2d.Transparent, 0)
	renderer.Draw(&list, draw2d.Hex("#202020"), 800, 600, 1)
	ground, multiply, panelDraw := fake.draws[0], fake.draws[2], fake.draws[len(fake.draws)-1]
	for name, draw := range map[string]recordedDraw{"the ground": ground, "the multiply": multiply, "the panel": panelDraw} {
		if draw.framebuffer != renderer.Target {
			test.Errorf("%s is drawn into framebuffer %d, not the target %d", name, draw.framebuffer, renderer.Target)
		}
	}
}

func TestALightingWithNoLightsOnlyMultipliesTheAmbient(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	var list draw2d.List
	list.Lighting(draw2d.Color{1, 1, 1, 0.5})
	renderer.Draw(&list, draw2d.Hex("#202020"), 800, 600, 1)
	if len(fake.draws) != 1 || fake.draws[0].count != len(cornerFloats)/2 {
		test.Fatalf("got draws %+v, wanted only the multiply over the screen", fake.draws)
	}
}

func TestADrawUnderALampGetsTheLampsUniforms(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake)
	if err != nil {
		test.Fatal(err)
	}
	world := scene.New()
	world.Nodes = []*scene.Node{{Name: "crate", Mesh: &scene.Mesh{Positions: []float32{-1, 0, -1, 1, 0, -1, 0, 0, 1},
		Normals: []float32{0, 1, 0, 0, 1, 0, 0, 1, 0}, Indices: []uint32{0, 1, 2}}}}
	world.Lights = []scene.PointLight{{Position: graphics.Vector{0, 1, 0, 1}, Color: graphics.Vector{1, 1, 1, 1}, Outer: 3}}
	camera := models.Camera{Position: graphics.Vector{0, 5, 5, 1}, Up: graphics.Vector{0, 1, 0, 0}, FieldOfView: 1, Near: 0.1, Far: 100}
	renderer.Draw(world, camera, 640, 480)
	for _, name := range []string{"lightCount", "lightPositions[0]", "lightColors[0]"} {
		if _, asked := fake.uniforms[name]; !asked {
			test.Errorf("the draw under the lamp never set %s", name)
		}
	}
}
