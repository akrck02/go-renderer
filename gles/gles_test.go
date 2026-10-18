package gles

import (
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
	"github.com/akrck02/go-renderer/sceneplan"
)

func TestShadersAreOpenGLESTwo(t *testing.T) {
	for name, source := range map[string]string{
		"quad vertex": quadVertexShader, "quad fragment": quadFragmentShader,
		"scene vertex": sceneVertexShader, "scene fragment": sceneFragmentShader,
		"backdrop vertex": backdropVertexShader, "backdrop fragment": backdropBlurShader,
	} {
		if !strings.HasPrefix(source, "#version 100\n") {
			t.Errorf("%s shader does not start with the GLSL ES 1.00 version line", name)
		}
		for _, newer := range []string{"layout(", " in ", " out ", "texture(", "inverse("} {
			if strings.Contains(source, newer) {
				t.Errorf("%s shader uses %q, which OpenGL ES 2 does not have", name, strings.TrimSpace(newer))
			}
		}
		if strings.Contains(name, "fragment") && !strings.Contains(source, "precision ") {
			t.Errorf("%s shader has no default precision, which OpenGL ES requires", name)
		}
		if strings.Contains(source, "\x00") {
			t.Errorf("%s shader holds a zero byte", name)
		}
	}
}

func TestACompileErrorIsReported(t *testing.T) {
	fake := newRecorder()
	fake.failCompile = true
	if _, err := NewFlatRenderer(fake); err == nil || !strings.Contains(err.Error(), "broken on purpose") {
		t.Fatalf("error = %v, want the compiler's log", err)
	}
}

func card() *draw2d.List {
	var list draw2d.List
	area := draw2d.Rect{X: 20, Y: 20, Width: 200, Height: 60}
	list.Rectangle(area, 8, draw2d.Hex("#ffffff"), draw2d.Hex("#dde3ea"), 1)
	list.Text("Lighthouse", 36, 56, 14, draw2d.Regular, draw2d.Hex("#18212e"))
	return &list
}

func TestTheFlatRendererDrawsAFrameInOneCall(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	renderer.Draw(card(), draw2d.Hex("#eef1f5"), 640, 480, 1)
	if len(fake.draws) != 1 {
		t.Fatalf("draw calls = %d, want 1", len(fake.draws))
	}
	if count := fake.draws[0].count; count == 0 || count%6 != 0 {
		t.Fatalf("vertices = %d, want six per rectangle", count)
	}
	if fake.textureUploads != 1 {
		t.Fatalf("atlas uploads = %d on the first frame, want 1", fake.textureUploads)
	}
}

func TestTheAtlasIsUploadedOnlyWhenItChanges(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	renderer.Draw(card(), draw2d.Hex("#eef1f5"), 640, 480, 1)
	fake.forget()
	renderer.Draw(card(), draw2d.Hex("#eef1f5"), 640, 480, 1)
	if fake.textureUploads != 0 {
		t.Fatalf("atlas uploads = %d on a frame with the same glyphs, want 0", fake.textureUploads)
	}
	if fake.bufferUploads[ArrayBuffer] != 1 {
		t.Fatalf("vertex uploads = %d, want the frame's rectangles once", fake.bufferUploads[ArrayBuffer])
	}
}

func TestAnEmptyListDrawsNothing(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	renderer.Draw(&draw2d.List{}, draw2d.Hex("#eef1f5"), 640, 480, 1)
	if len(fake.draws) != 0 || fake.textureUploads != 0 {
		t.Fatalf("draws %d, atlas uploads %d: an empty list should only clear", len(fake.draws), fake.textureUploads)
	}
}

func TestQuadVerticesKeepTheirCornersAndColour(t *testing.T) {
	quad := overlay.Quad{Screen: [4]float32{10, 20, 30, 40}, Atlas: [4]float32{1, 2, 3, 4}, Color: [4]float32{0.5, 0.25, 1, 1}}
	vertices := appendQuadVertices(nil, []overlay.Quad{quad})
	if len(vertices) != 6*floatsPerQuadVertex {
		t.Fatalf("floats = %d, want %d", len(vertices), 6*floatsPerQuadVertex)
	}
	topLeft := vertices[0:8]
	want := []float32{10, 20, 1, 2, 0.5, 0.25, 1, 1}
	for index := range want {
		if topLeft[index] != want[index] {
			t.Fatalf("first corner = %v, want %v", topLeft, want)
		}
	}
	lastCorner := vertices[5*floatsPerQuadVertex : 5*floatsPerQuadVertex+4]
	if lastCorner[0] != 30 || lastCorner[1] != 20 || lastCorner[2] != 3 || lastCorner[3] != 2 {
		t.Fatalf("last corner = %v, want the top-right one", lastCorner)
	}
}

func islandCamera() models.Camera {
	return models.Camera{Position: graphics.Vec4{0, 600, 900, 1}, Target: graphics.Vec4{0, 0, 0, 1}, Up: graphics.Vec4{0, 1, 0, 0}, Far: 20000}
}

func TestTheSceneRendererDrawsTheSampleIsland(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	world := sample.Island()
	renderer.Draw(world, islandCamera(), 1280, 800)
	// the ground, the water and the other meshes, and the forest in a few batches
	if len(fake.draws) < 5 || len(fake.draws) > 60 {
		t.Fatalf("draw calls = %d, want the meshes and a call per batch of trees", len(fake.draws))
	}
	if fake.bufferUploads[ElementArrayBuffer] == 0 {
		t.Fatal("no index buffer was uploaded")
	}
}

func TestMeshesAreUploadedOnce(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	world := sample.Island()
	renderer.Draw(world, islandCamera(), 1280, 800)
	fake.forget()
	renderer.Draw(world, islandCamera(), 1280, 800)
	if uploads := fake.bufferUploads[ElementArrayBuffer]; uploads != 0 {
		t.Fatalf("index uploads on the second frame = %d, want 0", uploads)
	}
	if len(fake.draws) == 0 {
		t.Fatal("the second frame drew nothing")
	}
}

func TestADirtyMeshIsUploadedAgain(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	world := sample.Island()
	renderer.Draw(world, islandCamera(), 1280, 800)
	world.Nodes[0].Mesh.Dirty = true
	fake.forget()
	renderer.Draw(world, islandCamera(), 1280, 800)
	if fake.bufferUploads[ElementArrayBuffer] != 1 {
		t.Fatalf("index uploads = %d, want the changed mesh once", fake.bufferUploads[ElementArrayBuffer])
	}
}

func TestInstanceStoreRewritesInPlace(t *testing.T) {
	store := instanceStore{}
	packed := make([]float32, 2*20)
	packed[20] = 7
	handle := store.FillBatch(nil, nil, packed)
	replacement := make([]float32, 20)
	replacement[0] = 9
	store.RewriteInstances(handle, 1, replacement)
	batch := handle.(*instanceBatch)
	if batch.packed[20] != 9 || len(batch.packed) != 40 {
		t.Fatalf("second instance starts with %v (length %d), want 9 (40)", batch.packed[20], len(batch.packed))
	}
}

func TestAttributesAreBoundBeforeLinking(t *testing.T) {
	fake := newRecorder()
	if _, err := NewSceneRenderer(fake); err != nil {
		t.Fatal(err)
	}
	want := append(append([]string{}, sceneAttributes...), quadAttributes...)
	if strings.Join(fake.attributes, ",") != strings.Join(want, ",") {
		t.Fatalf("attributes bound = %v, want %v", fake.attributes, want)
	}
}

func TestTheNormalMatrixUndoesAStretch(t *testing.T) {
	stretch := graphics.Identity()
	stretch[0] = 2 // twice as wide: normals lean the other way, so x is halved
	matrix := normalMatrix(stretch)
	if matrix[0] != 0.5 || matrix[4] != 1 || matrix[8] != 1 {
		t.Fatalf("normal matrix diagonal = %v, %v, %v; want 0.5, 1, 1", matrix[0], matrix[4], matrix[8])
	}
}

func TestInstancesAreMergedIntoOneMesh(t *testing.T) {
	triangle := &scene.Mesh{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}, Normals: []float32{0, 0, 1, 0, 0, 1, 0, 0, 1},
		Indices: []uint32{0, 1, 2}}
	packed := make([]float32, 2*sceneplan.FloatsPerInstance)
	first, second := graphics.Identity(), graphics.Translate(graphics.Vec4{10, 0, 0, 0})
	copy(packed[0:16], first[:])
	copy(packed[16:20], []float32{1, 0, 0, 1}) // red
	copy(packed[20:36], second[:])
	copy(packed[36:40], []float32{0, 1, 0, 1}) // green
	merged := &scene.Mesh{}
	mergeInstances(merged, triangle, packed, 2)
	if merged.VertexCount() != 6 || len(merged.Indices) != 6 || merged.Indices[3] != 3 {
		t.Fatalf("vertices %d, indices %v: want two triangles, the second numbered after the first", merged.VertexCount(), merged.Indices)
	}
	if merged.Positions[9] != 10 || merged.Colors[0] != 1 || merged.Colors[13] != 1 {
		t.Fatalf("second copy starts at x %v; colours %v: want it moved by 10, red then green", merged.Positions[9], merged.Colors)
	}
}

func TestAMergedBatchIsMadeAgainOnlyWhenItChanges(t *testing.T) {
	triangle := &scene.Mesh{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}}
	identity := graphics.Identity()
	packed := append(append([]float32{}, identity[:]...), 1, 1, 1, 1)
	store := instanceStore{}
	batch := store.FillBatch(nil, triangle, packed).(*instanceBatch)
	merged := batch.mergedFor(triangle, 1)
	merged.Dirty = false // as if uploaded
	if batch.mergedFor(triangle, 1).Dirty {
		t.Fatal("an unchanged batch was made again")
	}
	store.RewriteInstances(batch, 0, packed)
	if !batch.mergedFor(triangle, 1).Dirty {
		t.Fatal("a rewritten batch should be made and uploaded again")
	}
}

func TestANewRendererUploadsTheMeshesAgain(t *testing.T) {
	world := sample.Island()
	first, err := NewSceneRenderer(newRecorder())
	if err != nil {
		t.Fatal(err)
	}
	first.Draw(world, islandCamera(), 1280, 800)
	// a phone came back from the background: a new graphics context, a new renderer
	fake := newRecorder()
	second, err := NewSceneRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	second.Draw(world, islandCamera(), 1280, 800)
	if fake.bufferUploads[ElementArrayBuffer] == 0 {
		t.Fatal("the new renderer drew the ground with the old context's buffers")
	}
}

func TestASecondFrameSendsOnlyWhatChanged(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	world := sample.Island()
	renderer.Draw(world, islandCamera(), 1280, 800)
	firstValues := fake.uniformValues
	fake.forget()
	renderer.Draw(world, islandCamera(), 1280, 800)
	// every call is a trip through the bindings: x/mobile's crosses threads every three calls. What
	// is left changes from one draw to the next (a material's colour and kind)
	if fake.uniformValues >= firstValues {
		t.Fatalf("uniform values sent: %d on the first frame, %d on the same frame again; want fewer", firstValues, fake.uniformValues)
	}
	// the three both programs read stay on; the quads' own two (pictures and backdrop) go on for
	// the overlay and off after
	if fake.enables > 2 {
		t.Fatalf("vertex attributes switched on %d times again; want only the quads' own two", fake.enables)
	}
}

func TestAnImageIsDrawnInTheSameCallAndOnlyItsChangesAreSent(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	photo := draw2d.NewBlankImage(64, 48)
	list := card()
	list.Image(photo, draw2d.Rect{X: 20, Y: 100, Width: 120, Height: 90}, 12, draw2d.Hex("#ffffff"))
	renderer.Draw(list, draw2d.Hex("#eef1f5"), 640, 480, 1)
	if len(fake.draws) != 1 {
		t.Fatalf("draw calls = %d with an image, want still 1", len(fake.draws))
	}
	if fake.textureUploads != 2 {
		t.Fatalf("whole uploads = %d on the first frame, want the atlas and the pictures", fake.textureUploads)
	}
	fake.forget()
	photo.Changed()
	renderer.Draw(list, draw2d.Hex("#eef1f5"), 640, 480, 1)
	if fake.textureUploads != 0 || fake.textureAreaUploads != 1 {
		t.Fatalf("whole uploads %d, area uploads %d after the picture changed; want only its area",
			fake.textureUploads, fake.textureAreaUploads)
	}
}

func TestAFlatListWithoutBackdropsIsOneDrawAndNoSnapshot(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		t.Fatal(err)
	}
	var list draw2d.List
	list.Rectangle(draw2d.Rect{X: 10, Y: 10, Width: 200, Height: 100}, 12, draw2d.Hex("#ffffff20"), draw2d.Transparent, 0)
	renderer.Draw(&list, draw2d.Hex("#202020"), 800, 600, 1)
	if len(fake.draws) != 1 || fake.snapshots != 0 {
		t.Fatalf("no backdrop: one draw and no copy of the frame, got %d draws and %d copies", len(fake.draws), fake.snapshots)
	}
}

func TestPanelsOfOneLayerShareOneSnapshot(t *testing.T) {
	fake := newRecorder()
	renderer, err := NewFlatRenderer(fake)
	if err != nil {
		t.Fatal(err)
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
		t.Fatalf("four panels of one layer copy the frame once, got %d", fake.snapshots)
	}
	// the ground; then the halvings and two blur passes; then the rest of the quads
	halvings := graphics.BackdropHalvings(24)
	if len(fake.draws) != 1+halvings+2+1 {
		t.Fatalf("got %d draws, wanted %d", len(fake.draws), 1+halvings+2+1)
	}
}
