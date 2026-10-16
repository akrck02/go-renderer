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

// tileRGBAForPICA lays out a square picture of straight RGBA pixels (the overlay's pictures) as
// tileForPICA does, each pixel as the PICA200's RGBA8 reads it: one little-endian word
// 0xRRGGBBAA, so the bytes alpha, blue, green, red.
func tileRGBAForPICA(into, pixels []byte, size int) []byte {
	if cap(into) < size*size*4 {
		into = make([]byte, size*size*4)
	}
	into = into[:size*size*4]
	tilesPerRow := size / 8
	for y := range size {
		fromBottom := size - 1 - y
		for x := range size {
			tile := (fromBottom/8)*tilesPerRow + x/8
			to := (tile*64 + mortonInTile(x, fromBottom)) * 4
			from := (y*size + x) * 4
			into[to], into[to+1], into[to+2], into[to+3] = pixels[from+3], pixels[from+2], pixels[from+1], pixels[from]
		}
	}
	return into
}

// mortonInTile is a pixel's place inside its tile: bits x0 y0 x1 y1 x2 y2, from the lowest.
func mortonInTile(x, y int) int {
	return x&1 | (y&1)<<1 | (x&2)<<1 | (y&2)<<2 | (x&4)<<2 | (y&4)<<3
}

// floatsPerQuadVertex is a corner of an overlay rectangle: screen position 2, atlas position 2,
// colour 4 and picture position 2 (texture positions from 0 to 1, from the top: the tiling has
// already put the rows bottom first).
const floatsPerQuadVertex = 10

// quadRun is rectangles in a row that all show a picture, or none: each run is drawn with its own
// colour stages, in the list's order, so what is painted over what stays the same.
type quadRun struct {
	first, count int // in corners
	pictured     bool
}

// appendQuadVertices writes six corners per rectangle, two triangles, and the runs they make.
func appendQuadVertices(vertices []float32, runs []quadRun, quads []overlay.Quad, atlasSize, picturesSize int) ([]float32, []quadRun) {
	atlas, pictures := float32(atlasSize), float32(max(picturesSize, 1))
	for _, quad := range quads {
		picture := [4]float32{}
		if quad.Pictured {
			picture = [4]float32{quad.Picture[0] / pictures, quad.Picture[1] / pictures, quad.Picture[2] / pictures, quad.Picture[3] / pictures}
		}
		corner := len(vertices) / floatsPerQuadVertex
		if len(runs) == 0 || runs[len(runs)-1].pictured != quad.Pictured {
			runs = append(runs, quadRun{first: corner, pictured: quad.Pictured})
		}
		runs[len(runs)-1].count += len(quadCorners)
		for _, edges := range quadCorners {
			x, y := edges[0], edges[1] // 0 for the left or top edge, 2 for the right or bottom
			vertices = append(vertices, quad.Screen[x], quad.Screen[y+1],
				quad.Atlas[x]/atlas, quad.Atlas[y+1]/atlas,
				quad.Color[0], quad.Color[1], quad.Color[2], quad.Color[3],
				picture[x], picture[y+1])
		}
	}
	return vertices, runs
}

// quadCorners are a rectangle's two triangles: for each corner, which edges it is on (0 left or
// top, 2 right or bottom), into a quad's left, top, right, bottom.
var quadCorners = [6][2]int{{0, 0}, {0, 2}, {2, 2}, {0, 0}, {2, 2}, {2, 0}}

// largestTexture is the PICA200's largest texture side.
const largestTexture = 1024
