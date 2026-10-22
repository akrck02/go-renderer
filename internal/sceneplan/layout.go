package sceneplan

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// FloatsPerVertex is the size of a vertex on the GPU: position 3, normal 3, colour 4. Every
// backend lays meshes out this way.
const FloatsPerVertex = 10

// FloatsPerInstance is the size of an instance on the GPU: its matrix 16 (column by column) and
// its colour 4.
const FloatsPerInstance = 20

// InterleaveVertices packs position, normal and colour per vertex (white when there are no
// colours), computing the normals first when the mesh has none.
func InterleaveVertices(mesh *scene.Mesh) []float32 {
	if len(mesh.Normals) != len(mesh.Positions) {
		mesh.ComputeNormals()
	}
	vertexCount := mesh.VertexCount()
	data := make([]float32, vertexCount*FloatsPerVertex)
	for vertex := 0; vertex < vertexCount; vertex++ {
		offset := vertex * FloatsPerVertex
		copy(data[offset:offset+3], mesh.Positions[3*vertex:3*vertex+3])
		copy(data[offset+3:offset+6], mesh.Normals[3*vertex:3*vertex+3])
		if len(mesh.Colors) >= 4*(vertex+1) {
			copy(data[offset+6:offset+10], mesh.Colors[4*vertex:4*vertex+4])
		} else {
			copy(data[offset+6:offset+10], []float32{1, 1, 1, 1})
		}
	}
	return data
}

// PackInstances writes the matrix and the colour of every instance (white when missing).
func PackInstances(instances *scene.Instances) []float32 {
	data := make([]float32, len(instances.Transforms)*FloatsPerInstance)
	for instance := range instances.Transforms {
		packInstance(instances, instance, data[instance*FloatsPerInstance:])
	}
	return data
}

// ElementCount is how many vertices a mesh's draw call reads: its indices, or its vertices when it
// has none.
func ElementCount(mesh *scene.Mesh) int {
	if len(mesh.Indices) > 0 {
		return len(mesh.Indices)
	}
	return mesh.VertexCount()
}

// InterleaveHarmonics packs the in-phase and in-quadrature grids of a water level variation as two
// channels of one texture.
func InterleaveHarmonics(variation *scene.LevelVariation) []float32 {
	cells := variation.Columns * variation.Rows
	data := make([]float32, 2*cells)
	for cell := 0; cell < cells && cell < len(variation.InPhase) && cell < len(variation.InQuadrature); cell++ {
		data[2*cell], data[2*cell+1] = variation.InPhase[cell], variation.InQuadrature[cell]
	}
	return data
}

// ColorOrWhite returns the colour, or white when it is unset.
func ColorOrWhite(color graphics.Vector) graphics.Vector {
	if color == (graphics.Vector{}) {
		return graphics.Vector{1, 1, 1, 1}
	}
	return color
}

// ScaleOrOne returns the value, or 1 when it is not positive.
func ScaleOrOne(value float32) float32 {
	if value <= 0 {
		return 1
	}
	return value
}
