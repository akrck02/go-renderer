package ctr

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
)

func TestOpenGLMatricesBecomePICAOnes(t *testing.T) {
	openGL := graphics.OrthographicOpenGL(0, 400, 0, 240, 0, 1)
	pica := PICAViewProjection(openGL)
	want := OrthographicTilt(0, 400, 0, 240, 0, 1)
	for index := range pica {
		if !near(pica[index], want[index]) {
			t.Fatalf("element %d is %v, want %v (OrthographicTilt)", index, pica[index], want[index])
		}
	}
}

func TestASmallMeshStaysInOnePiece(t *testing.T) {
	vertices := make([]float32, 3*2)
	pieces := splitForShortIndices(vertices, []uint32{0, 1, 2}, 2)
	if len(pieces) != 1 || len(pieces[0].indices) != 3 || pieces[0].indices[2] != 2 {
		t.Fatalf("pieces %v", pieces)
	}
}

func TestALargeMeshIsCutIntoPiecesOfShortIndices(t *testing.T) {
	// 30 000 separate triangles: 90 000 vertices, more than 16-bit indices reach.
	const triangles = 30000
	vertices := make([]float32, triangles*3)
	indices := make([]uint32, triangles*3)
	for index := range indices {
		vertices[index] = float32(index) // one float per vertex: its own number
		indices[index] = uint32(index)
	}
	pieces := splitForShortIndices(vertices, indices, 1)
	if len(pieces) != 2 {
		t.Fatalf("%d pieces, want 2", len(pieces))
	}
	drawn := 0
	for _, piece := range pieces {
		if len(piece.vertices) > largestPieceVertices {
			t.Fatalf("a piece has %d vertices", len(piece.vertices))
		}
		for position, local := range piece.indices {
			if want := float32(drawn + position); piece.vertices[local] != want {
				t.Fatalf("index %d reads vertex %v, want %v", position, piece.vertices[local], want)
			}
		}
		drawn += len(piece.indices)
	}
	if drawn != triangles*3 {
		t.Fatalf("drew %d corners, want %d", drawn, triangles*3)
	}
}

func TestAMeshWithoutIndicesIsAListOfTriangles(t *testing.T) {
	pieces := splitForShortIndices(make([]float32, 6), nil, 1)
	if len(pieces) != 1 || len(pieces[0].indices) != 6 || pieces[0].indices[5] != 5 {
		t.Fatalf("pieces %v", pieces)
	}
}

func TestEachMaterialKindPicksOneShading(t *testing.T) {
	cases := map[scene.MaterialKind]graphics.Vec4{
		scene.KindLit:       {1, 0, 0, 0},
		scene.KindUnlit:     {0, 1, 0, 0},
		scene.KindWater:     {0, 0, 1, 0},
		scene.KindWaterfall: {0, 1, 0, 0},
		scene.KindGlow:      {1, 0, 0, 0.7},
	}
	for kind, want := range cases {
		if got := materialWeights(kind, 0.5); got != want {
			t.Errorf("kind %d: %v, want %v", kind, got, want)
		}
	}
}

func TestTheFogIsOffWithoutAFarDistance(t *testing.T) {
	shading := shadingFor(scene.Environment{FogNear: 10, VerticalScale: 1}, scene.Daylight{LightDirection: graphics.Vec4{0, 1, 0, 0}}, cameraAt(0, 0, 0))
	if shading.fog[1] != 0 {
		t.Fatalf("fog %v, want no fog", shading.fog)
	}
	shading = shadingFor(scene.Environment{FogNear: 10, FogFar: 20, VerticalScale: 1}, scene.Daylight{LightDirection: graphics.Vec4{0, 1, 0, 0}}, cameraAt(0, 0, 0))
	if shading.fog[0] != 10 || shading.fog[1] != 0.1 {
		t.Fatalf("fog %v, want near 10 and 1/10", shading.fog)
	}
}

func TestTheHorizonIsPackedInTheScreensGamma(t *testing.T) {
	if got := screenColour(graphics.Vec4{1, 0, 0.2195, 1}); got != 0xFF0080FF {
		t.Fatalf("%08X, want FF0080FF", got)
	}
}

func cameraAt(x, y, z float32) models.Camera {
	return models.Camera{Position: graphics.Vec4{x, y, z, 1}}
}
