package sceneplan

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// MergedBatch keeps a batch's packed instances, and the batch as one mesh, for GPUs that cannot
// draw instances (OpenGL ES 2, the 3DS's PICA200): a draw call per copy is far too slow (a forest
// is thousands of calls). The mesh is made again only when the batch changes.
type MergedBatch struct {
	packed []float32
	merged *scene.Mesh // every copy placed and tinted; nil or stale when the batch changed
	stale  bool
	// mesh and count are what merged was made from
	mesh  *scene.Mesh
	count int
}

// MergedFor returns the batch's first count instances of mesh as one mesh, making it when needed.
func (batch *MergedBatch) MergedFor(mesh *scene.Mesh, count int) *scene.Mesh {
	if batch.merged != nil && !batch.stale && batch.mesh == mesh && batch.count == count {
		return batch.merged
	}
	if batch.merged == nil {
		batch.merged = &scene.Mesh{Name: mesh.Name + " (instances)"}
	}
	MergeInstances(batch.merged, mesh, batch.packed, count)
	batch.merged.Dirty = true // upload it again, into the same buffers
	batch.stale, batch.mesh, batch.count = false, mesh, count
	return batch.merged
}

// MergeInstances fills into with count copies of mesh, each placed by its instance's matrix,
// its normals turned with it and its colours tinted by the instance's colour.
func MergeInstances(into, mesh *scene.Mesh, packed []float32, count int) {
	if len(mesh.Normals) != len(mesh.Positions) {
		mesh.ComputeNormals()
	}
	vertices := mesh.VertexCount()
	into.Positions, into.Normals = into.Positions[:0], into.Normals[:0]
	into.Colors, into.Indices = into.Colors[:0], into.Indices[:0]
	for instance := range count {
		values := packed[instance*FloatsPerInstance:]
		placement, tint := graphics.Matrix(values[0:16]), graphics.Vector(values[16:20])
		turn := NormalMatrix(placement)
		for vertex := range vertices {
			into.Positions = appendPoint(into.Positions, placement.Apply(mesh.Vertex(uint32(vertex))))
			into.Normals = appendNormal(into.Normals, turn, mesh.Normals[3*vertex:3*vertex+3])
			into.Colors = appendTinted(into.Colors, mesh.Colors, vertex, tint)
		}
		first := uint32(instance * vertices)
		for _, index := range mesh.Indices {
			into.Indices = append(into.Indices, first+index)
		}
	}
}

func appendPoint(values []float32, point graphics.Vector) []float32 {
	return append(values, point[0], point[1], point[2])
}

// appendNormal turns a normal by a normal matrix (column by column) and keeps it of unit length.
func appendNormal(values []float32, turn [9]float32, normal []float32) []float32 {
	x := turn[0]*normal[0] + turn[3]*normal[1] + turn[6]*normal[2]
	y := turn[1]*normal[0] + turn[4]*normal[1] + turn[7]*normal[2]
	z := turn[2]*normal[0] + turn[5]*normal[1] + turn[8]*normal[2]
	length := float32(math.Sqrt(float64(x*x + y*y + z*z)))
	if length == 0 {
		return append(values, 0, 1, 0)
	}
	return append(values, x/length, y/length, z/length)
}

// appendTinted is a vertex's colour (white when the mesh has none) times the instance's tint.
func appendTinted(values, colors []float32, vertex int, tint graphics.Vector) []float32 {
	color := [4]float32{1, 1, 1, 1}
	if len(colors) >= 4*(vertex+1) {
		color = [4]float32(colors[4*vertex : 4*vertex+4])
	}
	return append(values, color[0]*tint[0], color[1]*tint[1], color[2]*tint[2], color[3]*tint[3])
}

// MergingStore keeps the planner's instance batches as MergedBatch.
type MergingStore struct{}

// FillBatch keeps a batch's packed instances in a MergedBatch, reusing the one it had.
func (MergingStore) FillBatch(handle any, mesh *scene.Mesh, packed []float32) any {
	batch, _ := handle.(*MergedBatch)
	if batch == nil {
		batch = &MergedBatch{}
	}
	batch.packed = append(batch.packed[:0], packed...)
	batch.stale = true
	return batch
}

// RewriteInstances replaces some packed instances of a MergedBatch in place.
func (MergingStore) RewriteInstances(handle any, position int, packed []float32) {
	batch, _ := handle.(*MergedBatch)
	if batch == nil {
		return
	}
	copy(batch.packed[position*FloatsPerInstance:], packed)
	batch.stale = true
}

// NormalMatrix is the inverse transpose of a model matrix's rotation and scale: what turns normals
// so they stay perpendicular to surfaces scaled differently along each axis.
func NormalMatrix(model graphics.Matrix) [9]float32 {
	inverse, invertible := model.Inverse()
	if !invertible {
		inverse = graphics.Identity()
	}
	// the transpose of the inverse's upper 3 × 3, column by column
	return [9]float32{
		inverse[0], inverse[4], inverse[8],
		inverse[1], inverse[5], inverse[9],
		inverse[2], inverse[6], inverse[10],
	}
}
