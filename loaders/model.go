package loaders

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/immediate"
)

// positionValues is the number of values of a vertex position: x, y and z.
const positionValues = 3

// LoadModel reads the meshes of a glTF 2.0 file (.glb, or .gltf with external buffers) into an
// immediate.Model, for the immediate-mode Renderer: each triangle primitive's positions and
// indices, in any of glTF's component types. Materials, nodes and extras are left out: LoadScene
// reads those.
func LoadModel(path string) (*immediate.Model, error) {
	file, err := readGLTFFile(path)
	if err != nil {
		return nil, err
	}
	return file.buildModel()
}

// buildModel makes a mesh of every triangle primitive that has positions.
func (file *gltfFile) buildModel() (*immediate.Model, error) {
	model := &immediate.Model{}
	for _, mesh := range file.document.Meshes {
		for _, primitive := range mesh.Primitives {
			built, found, err := file.buildModelMesh(primitive)
			if err != nil {
				return nil, err
			}
			if found {
				model.Meshes = append(model.Meshes, built)
			}
		}
	}
	return model, nil
}

// buildModelMesh reads one primitive's positions and indices; false for a primitive with no
// positions or that is not made of triangles.
func (file *gltfFile) buildModelMesh(primitive gltfPrimitiveDefinition) (immediate.Mesh, bool, error) {
	positionAccessor, hasPositions := primitive.Attributes["POSITION"]
	if !hasPositions || (primitive.Mode != nil && *primitive.Mode != trianglesMode) {
		return immediate.Mesh{}, false, nil
	}
	positions, _, err := file.readAccessor(positionAccessor)
	if err != nil {
		return immediate.Mesh{}, false, err
	}
	mesh := immediate.Mesh{Vertices: pointsFromPositions(positions)}
	if primitive.Indices != nil {
		if mesh.Indices, err = file.readIndices(*primitive.Indices); err != nil {
			return immediate.Mesh{}, false, err
		}
	}
	return mesh, true, nil
}

// pointsFromPositions turns x, y, z triples into points (w = 1).
func pointsFromPositions(positions []float32) []graphics.Vector {
	points := make([]graphics.Vector, len(positions)/positionValues)
	for index := range points {
		start := index * positionValues
		points[index] = graphics.Vector{positions[start], positions[start+1], positions[start+2], 1}
	}
	return points
}
