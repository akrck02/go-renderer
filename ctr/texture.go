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
	first, count int // in rectangles
	pictured     bool
}

// writeQuadVertices writes four corners per rectangle into vertices, which holds
// len(quads)·4·floatsPerQuadVertex floats (linear memory, on the console), and lists the runs
// they make. The corners go top-left, bottom-left, bottom-right, top-right: quadIndices make them
// two triangles. Written in place and multiplied rather than divided: on the console's processor
// this is most of what painting an interface costs.
func writeQuadVertices(vertices []float32, runs []quadRun, quads []overlay.Quad, atlasSize, picturesSize int) []quadRun {
	toAtlas, toPictures := 1/float32(atlasSize), 1/float32(max(picturesSize, 1))
	_ = vertices[len(quads)*4*floatsPerQuadVertex-1] // one bounds check for them all
	for index := range quads {
		quad := &quads[index]
		if len(runs) == 0 || runs[len(runs)-1].pictured != quad.Pictured {
			runs = append(runs, quadRun{first: index, pictured: quad.Pictured})
		}
		runs[len(runs)-1].count++
		left, top, right, bottom := quad.Screen[0], quad.Screen[1], quad.Screen[2], quad.Screen[3]
		atlasLeft, atlasTop, atlasRight, atlasBottom := quad.Atlas[0]*toAtlas, quad.Atlas[1]*toAtlas, quad.Atlas[2]*toAtlas, quad.Atlas[3]*toAtlas
		var pictureLeft, pictureTop, pictureRight, pictureBottom float32
		if quad.Pictured {
			pictureLeft, pictureTop, pictureRight, pictureBottom = quad.Picture[0]*toPictures, quad.Picture[1]*toPictures, quad.Picture[2]*toPictures, quad.Picture[3]*toPictures
		}
		corners := vertices[index*4*floatsPerQuadVertex : (index+1)*4*floatsPerQuadVertex]
		writeCorner(corners[0:floatsPerQuadVertex], left, top, atlasLeft, atlasTop, &quad.Color, pictureLeft, pictureTop)
		writeCorner(corners[floatsPerQuadVertex:2*floatsPerQuadVertex], left, bottom, atlasLeft, atlasBottom, &quad.Color, pictureLeft, pictureBottom)
		writeCorner(corners[2*floatsPerQuadVertex:3*floatsPerQuadVertex], right, bottom, atlasRight, atlasBottom, &quad.Color, pictureRight, pictureBottom)
		writeCorner(corners[3*floatsPerQuadVertex:4*floatsPerQuadVertex], right, top, atlasRight, atlasTop, &quad.Color, pictureRight, pictureTop)
	}
	return runs
}

func writeCorner(corner []float32, x, y, atlasX, atlasY float32, colour *[4]float32, pictureX, pictureY float32) {
	_ = corner[floatsPerQuadVertex-1]
	corner[0], corner[1], corner[2], corner[3] = x, y, atlasX, atlasY
	corner[4], corner[5], corner[6], corner[7] = colour[0], colour[1], colour[2], colour[3]
	corner[8], corner[9] = pictureX, pictureY
}

// maxQuadsPerList is how many rectangles one painted list can have: their corners are numbered
// with 16 bits. More are left out.
const maxQuadsPerList = 65536 / 4

// largestTexture is the PICA200's largest texture side.
const largestTexture = 1024
