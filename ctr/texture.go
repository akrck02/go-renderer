package ctr

import (
	"github.com/akrck02/go-renderer/opengl/overlay"
)

// The PICA200 reads textures in tiles of 8 × 8 pixels, the tiles row by row from the bottom of the
// picture, and the pixels inside a tile in Morton order (x and y bits interleaved). tileForPICA
// lays out a square picture of one byte per pixel (the overlay's atlas of coverage) that way.
func tileForPICA(into, pixels []byte, size int) []byte {
	if cap(into) < size*size {
		into = make([]byte, size*size)
	}
	into = into[:size*size]
	tilesPerRow := size / 8
	for y := range size {
		fromBottom := size - 1 - y
		for x := range size {
			tile := (fromBottom/8)*tilesPerRow + x/8
			into[tile*64+mortonInTile(x, fromBottom)] = pixels[y*size+x]
		}
	}
	return into
}

// mortonInTile is a pixel's place inside its tile: bits x0 y0 x1 y1 x2 y2, from the lowest.
func mortonInTile(x, y int) int {
	return x&1 | (y&1)<<1 | (x&2)<<1 | (y&2)<<2 | (x&4)<<2 | (y&4)<<3
}

// floatsPerQuadVertex is a corner of an overlay rectangle: screen position 2, atlas position 2
// (0 to 1, from the atlas's top: tileForPICA already put its rows bottom first), colour 4.
const floatsPerQuadVertex = 8

// appendQuadVertices writes six corners per rectangle, two triangles. Rectangles of pictures are
// left out: the PICA200 path has no picture atlas yet.
func appendQuadVertices(vertices []float32, quads []overlay.Quad, atlasSize int) []float32 {
	size := float32(atlasSize)
	for _, quad := range quads {
		if quad.Pictured {
			continue
		}
		for _, corner := range quadCorners {
			x, y := corner[0], corner[1] // 0 for the left or top edge, 2 for the right or bottom
			vertices = append(vertices, quad.Screen[x], quad.Screen[y+1],
				quad.Atlas[x]/size, quad.Atlas[y+1]/size,
				quad.Color[0], quad.Color[1], quad.Color[2], quad.Color[3])
		}
	}
	return vertices
}

// quadCorners are a rectangle's two triangles: for each corner, which edges it is on (0 left or
// top, 2 right or bottom), into a quad's left, top, right, bottom.
var quadCorners = [6][2]int{{0, 0}, {0, 2}, {2, 2}, {0, 0}, {2, 2}, {2, 0}}

// largestTexture is the PICA200's largest texture side.
const largestTexture = 1024
