package ctr

import (
	"testing"

	"github.com/akrck02/go-renderer/opengl/overlay"
)

func TestTexturesAreTiledFromTheBottomInMortonOrder(t *testing.T) {
	const size = 16
	pixels := make([]byte, size*size)
	at := func(x, y int) *byte { return &pixels[y*size+x] }
	*at(0, size-1) = 1 // bottom-left: the first pixel
	*at(1, size-1) = 2 // its right: the second
	*at(0, size-2) = 3 // above it: the third
	*at(8, size-1) = 4 // the next tile along the bottom row
	*at(0, size-9) = 5 // the first tile of the second row of tiles
	*at(7, 7) = 6      // the top-left tile's bottom row, its last pixel
	tiled := tileForPICA(nil, pixels, size)
	checks := map[int]byte{0: 1, 1: 2, 2: 3, 64: 4, 2 * 64: 5, 2*64 + mortonInTile(7, 0): 6}
	for index, want := range checks {
		if tiled[index] != want {
			t.Errorf("tiled[%d] = %d, want %d", index, tiled[index], want)
		}
	}
}

func TestQuadsBecomeTwoTrianglesInRunsOfPicturesOrNone(t *testing.T) {
	quads := []overlay.Quad{
		{Screen: [4]float32{10, 20, 30, 40}, Atlas: [4]float32{0, 0, 256, 128}, Color: [4]float32{1, 0.5, 0, 1}},
		{Screen: [4]float32{0, 0, 1, 1}, Pictured: true, Picture: [4]float32{0, 0, 512, 256}},
		{Screen: [4]float32{0, 0, 1, 1}, Pictured: true},
		{Screen: [4]float32{0, 0, 1, 1}},
	}
	vertices, runs := appendQuadVertices(nil, nil, quads, 512, 1024)
	if len(vertices) != 4*6*floatsPerQuadVertex {
		t.Fatalf("%d floats, want four rectangles of six corners", len(vertices))
	}
	want := []quadRun{{0, 6, false}, {6, 12, true}, {18, 6, false}}
	if len(runs) != len(want) {
		t.Fatalf("runs %v, want %v", runs, want)
	}
	for index := range want {
		if runs[index] != want[index] {
			t.Fatalf("runs %v, want %v", runs, want)
		}
	}
	// The first corner is the top-left: the atlas's top-left, 0, 0.
	first := vertices[0:floatsPerQuadVertex]
	if first[0] != 10 || first[1] != 20 || first[2] != 0 || first[3] != 0 || first[4] != 1 || first[5] != 0.5 {
		t.Fatalf("first corner %v", first)
	}
	// The third corner is the bottom-right: atlas 256, 128 of 512.
	third := vertices[2*floatsPerQuadVertex : 3*floatsPerQuadVertex]
	if third[0] != 30 || third[1] != 40 || third[2] != 0.5 || third[3] != 0.25 {
		t.Fatalf("third corner %v", third)
	}
	// The picture's bottom-right corner: 512, 256 of 1024.
	pictured := vertices[(6+2)*floatsPerQuadVertex : (6+3)*floatsPerQuadVertex]
	if pictured[8] != 0.5 || pictured[9] != 0.25 {
		t.Fatalf("picture corner %v", pictured[8:10])
	}
}

func TestPicturesAreTiledAsAlphaBlueGreenRed(t *testing.T) {
	const size = 8
	pixels := make([]byte, size*size*4)
	bottomLeft := (size - 1) * size * 4
	copy(pixels[bottomLeft:], []byte{10, 20, 30, 40}) // red, green, blue, alpha
	tiled := tileRGBAForPICA(nil, pixels, size)
	if tiled[0] != 40 || tiled[1] != 30 || tiled[2] != 20 || tiled[3] != 10 {
		t.Fatalf("first pixel %v, want 40 30 20 10", tiled[0:4])
	}
}
