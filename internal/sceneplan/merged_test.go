package sceneplan

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

func TestMergingStoreRewritesInPlace(test *testing.T) {
	store := MergingStore{}
	packed := make([]float32, 2*20)
	packed[20] = 7
	handle := store.FillBatch(nil, nil, packed)
	replacement := make([]float32, 20)
	replacement[0] = 9
	store.RewriteInstances(handle, 1, replacement)
	batch := handle.(*MergedBatch)
	if batch.packed[20] != 9 || len(batch.packed) != 40 {
		test.Fatalf("second instance starts with %v (length %d), want 9 (40)", batch.packed[20], len(batch.packed))
	}
}

func TestTheNormalMatrixUndoesAStretch(test *testing.T) {
	stretch := graphics.Identity()
	stretch[0] = 2 // twice as wide: normals lean the other way, so x is halved
	matrix := NormalMatrix(stretch)
	if matrix[0] != 0.5 || matrix[4] != 1 || matrix[8] != 1 {
		test.Fatalf("normal matrix diagonal = %v, %v, %v; want 0.5, 1, 1", matrix[0], matrix[4], matrix[8])
	}
}

func TestInstancesAreMergedIntoOneMesh(test *testing.T) {
	triangle := &scene.Mesh{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}, Normals: []float32{0, 0, 1, 0, 0, 1, 0, 0, 1},
		Indices: []uint32{0, 1, 2}}
	packed := make([]float32, 2*FloatsPerInstance)
	first, second := graphics.Identity(), graphics.Translate(graphics.Vec4{10, 0, 0, 0})
	copy(packed[0:16], first[:])
	copy(packed[16:20], []float32{1, 0, 0, 1}) // red
	copy(packed[20:36], second[:])
	copy(packed[36:40], []float32{0, 1, 0, 1}) // green
	merged := &scene.Mesh{}
	MergeInstances(merged, triangle, packed, 2)
	if merged.VertexCount() != 6 || len(merged.Indices) != 6 || merged.Indices[3] != 3 {
		test.Fatalf("vertices %d, indices %v: want two triangles, the second numbered after the first", merged.VertexCount(), merged.Indices)
	}
	if merged.Positions[9] != 10 || merged.Colors[0] != 1 || merged.Colors[13] != 1 {
		test.Fatalf("second copy starts at x %v; colours %v: want it moved by 10, red then green", merged.Positions[9], merged.Colors)
	}
}

func TestAMergedBatchIsMadeAgainOnlyWhenItChanges(test *testing.T) {
	triangle := &scene.Mesh{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}}
	identity := graphics.Identity()
	packed := append(append([]float32{}, identity[:]...), 1, 1, 1, 1)
	store := MergingStore{}
	batch := store.FillBatch(nil, triangle, packed).(*MergedBatch)
	merged := batch.MergedFor(triangle, 1)
	merged.Dirty = false // as if uploaded
	if batch.MergedFor(triangle, 1).Dirty {
		test.Fatal("an unchanged batch was made again")
	}
	store.RewriteInstances(batch, 0, packed)
	if !batch.MergedFor(triangle, 1).Dirty {
		test.Fatal("a rewritten batch should be made and uploaded again")
	}
}

func TestTheWaterGridCanBeCoarser(test *testing.T) {
	if cells := len(newWaterGrid(100, 0).Indices) / 6; cells != 256*256 {
		test.Fatalf("%d cells by default, want 256 × 256", cells)
	}
	if cells := len(newWaterGrid(100, 32).Indices) / 6; cells != 32*32 {
		test.Fatalf("%d cells, want 32 × 32", cells)
	}
}
