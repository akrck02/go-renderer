package scene

import "github.com/akrck02/go-renderer/graphics"

// DetailLevel is a simpler version of a mesh. It replaces the finer versions when the model looks
// smaller on screen than ScreenSize, a fraction of the screen height (0.05 = a twentieth of it).
type DetailLevel struct {
	Mesh       *Mesh
	ScreenSize float32
}

// LevelCount returns how many versions the mesh has, itself included.
func (mesh *Mesh) LevelCount() int { return 1 + len(mesh.Details) }

// Level returns a version of the mesh: 0 is the mesh itself, 1 the first detail level, and so on.
func (mesh *Mesh) Level(index int) *Mesh {
	if index <= 0 || len(mesh.Details) == 0 {
		return mesh
	}
	if index > len(mesh.Details) {
		index = len(mesh.Details)
	}
	return mesh.Details[index-1].Mesh
}

// LevelForScreenSize returns the index of the version to draw for a model of this apparent size:
// the coarsest level whose ScreenSize is still larger than it.
func (mesh *Mesh) LevelForScreenSize(screenSize float32) int {
	level := 0
	for index, detail := range mesh.Details {
		if screenSize < detail.ScreenSize {
			level = index + 1
		}
	}
	return level
}

// Bounds returns the box around the mesh positions in its own coordinates.
func (mesh *Mesh) Bounds() (minimum, maximum graphics.Vec4) {
	box := newBoundingBox()
	for index := uint32(0); int(index) < mesh.VertexCount(); index++ {
		box.include(mesh.Vertex(index))
	}
	return box.minimum, box.maximum
}
