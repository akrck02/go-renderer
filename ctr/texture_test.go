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

func TestQuadsBecomeTwoTrianglesWithTheAtlasInItsPixels(t *testing.T) {
	quads := []overlay.Quad{
		{Screen: [4]float32{10, 20, 30, 40}, Atlas: [4]float32{0, 0, 256, 128}, Color: [4]float32{1, 0.5, 0, 1}},
		{Screen: [4]float32{0, 0, 1, 1}, Pictured: true},
	}
	vertices := appendQuadVertices(nil, quads, 512)
	if len(vertices) != 6*floatsPerQuadVertex {
		t.Fatalf("%d floats, want one rectangle of six corners (pictures left out)", len(vertices))
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
}
