package ctr

// The PICA200 reads indices of 16 bits at most, so a mesh with more vertices is drawn in pieces.
const largestPieceVertices = 65535

// meshPiece is part of a mesh with its own vertices (interleaved, floatsPerVertex each) and the
// triangles that use them, numbered from 0.
type meshPiece struct {
	vertices []float32
	indices  []uint16
}

// splitForShortIndices cuts interleaved vertices and their triangles into pieces of at most
// largestPieceVertices vertices each. A mesh with no indices is a list of triangles. A mesh that
// needs no cutting gets its indices in reused's buffer (nil makes a new one).
func splitForShortIndices(vertices []float32, indices []uint32, floatsPerVertex int, reused *[]uint16) []meshPiece {
	vertexCount := len(vertices) / floatsPerVertex
	if len(indices) == 0 && vertexCount <= largestPieceVertices {
		short := sequence(reused, vertexCount)
		return []meshPiece{{vertices: vertices, indices: short}}
	}
	if len(indices) == 0 {
		indices = make([]uint32, vertexCount)
		for index := range indices {
			indices[index] = uint32(index)
		}
	}
	if vertexCount <= largestPieceVertices {
		return []meshPiece{{vertices: vertices, indices: shortIndices(reused, indices)}}
	}
	var pieces []meshPiece
	current := newPieceBuilder()
	for first := 0; first+2 < len(indices); first += 3 {
		triangle := indices[first : first+3]
		if current.wouldOverflow(triangle) {
			pieces = append(pieces, current.piece)
			current = newPieceBuilder()
		}
		current.addTriangle(triangle, vertices, floatsPerVertex)
	}
	if len(current.piece.indices) > 0 {
		pieces = append(pieces, current.piece)
	}
	return pieces
}

func shortIndices(reused *[]uint16, indices []uint32) []uint16 {
	short := reusedIndices(reused, len(indices))
	for position, index := range indices {
		short[position] = uint16(index)
	}
	return short
}

// sequence is the indices 0, 1, 2… of a list of triangles.
func sequence(reused *[]uint16, count int) []uint16 {
	short := reusedIndices(reused, count)
	for position := range short {
		short[position] = uint16(position)
	}
	return short
}

func reusedIndices(reused *[]uint16, count int) []uint16 {
	if reused == nil {
		return make([]uint16, count)
	}
	*reused = grownTo(*reused, count)
	return *reused
}

// pieceBuilder gathers triangles into a piece, numbering each vertex the first time it is used.
type pieceBuilder struct {
	piece    meshPiece
	renumber map[uint32]uint16
}

func newPieceBuilder() *pieceBuilder {
	return &pieceBuilder{renumber: map[uint32]uint16{}}
}

// wouldOverflow tells whether the triangle's new vertices no longer fit in the piece.
func (builder *pieceBuilder) wouldOverflow(triangle []uint32) bool {
	newVertices := 0
	for _, index := range triangle {
		if _, known := builder.renumber[index]; !known {
			newVertices++
		}
	}
	return len(builder.renumber)+newVertices > largestPieceVertices
}

func (builder *pieceBuilder) addTriangle(triangle []uint32, vertices []float32, floatsPerVertex int) {
	for _, index := range triangle {
		local, known := builder.renumber[index]
		if !known {
			local = uint16(len(builder.renumber))
			builder.renumber[index] = local
			start := int(index) * floatsPerVertex
			builder.piece.vertices = append(builder.piece.vertices, vertices[start:start+floatsPerVertex]...)
		}
		builder.piece.indices = append(builder.piece.indices, local)
	}
}
