package opengl

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// instancesPerCell is roughly how many instances each cell of an instance grid holds.
const instancesPerCell = 64

// maximumCellsPerSide limits the grid of very large instance sets.
const maximumCellsPerSide = 128

// instanceGrid sorts the instances of a node into square cells over the ground plane, so each
// frame the renderer can skip the cells outside the view and choose a level of detail per cell.
// Everything is in the node's own coordinates; the node may move.
type instanceGrid struct {
	cells  []instanceCell
	packed []float32 // floatsPerInstance values per instance, ordered by cell
	slots  []int     // position of each instance in packed
	cellOf []int     // cell of each instance
}

// instanceCell is a group of nearby instances.
type instanceCell struct {
	bounds  box     // around the instances' meshes
	first   int     // first instance of the cell in packed
	count   int     // instances in the cell
	largest float32 // size (box diagonal) of the largest instance
}

// buildInstanceGrid groups the instances by cell; meshBounds is the box of the instanced mesh.
func buildInstanceGrid(instances *scene.Instances, meshBounds box) *instanceGrid {
	count := len(instances.Transforms)
	perSide := cellsPerSide(count)
	area := originsArea(instances.Transforms)
	cellOf := make([]int, count)
	cells := make([]instanceCell, perSide*perSide)
	for instance, transform := range instances.Transforms {
		cellOf[instance] = cellIndex(transform, area, perSide)
		cells[cellOf[instance]].count++
	}
	assignFirstInstances(cells)
	grid := &instanceGrid{packed: make([]float32, count*floatsPerInstance), slots: make([]int, count)}
	filled := make([]int, len(cells))
	for instance, cell := range cellOf {
		position := cells[cell].first + filled[cell]
		filled[cell]++
		grid.slots[instance] = position
		packInstance(instances, instance, grid.packed[position*floatsPerInstance:])
	}
	measureCells(cells, grid.packed, meshBounds)
	var renumbered []int
	grid.cells, renumbered = nonEmptyCells(cells)
	grid.cellOf = cellOf
	for instance, cell := range cellOf {
		grid.cellOf[instance] = renumbered[cell]
	}
	return grid
}

// updateInstance writes a changed instance in its place and widens its cell to contain it (an
// instance keeps its cell even if it moves away; the cell box grows). It returns the cell.
func (grid *instanceGrid) updateInstance(instances *scene.Instances, instance int, meshBounds box) int {
	slot := grid.slots[instance]
	packInstance(instances, instance, grid.packed[slot*floatsPerInstance:])
	cell := &grid.cells[grid.cellOf[instance]]
	instanceBounds := meshBounds.transformed(instances.Transforms[instance], 1)
	cell.bounds.merge(instanceBounds)
	cell.largest = float32(math.Max(float64(cell.largest), float64(instanceBounds.diagonal())))
	return grid.cellOf[instance]
}

func cellsPerSide(count int) int {
	perSide := int(math.Ceil(math.Sqrt(float64(count) / instancesPerCell)))
	return int(math.Max(1, math.Min(maximumCellsPerSide, float64(perSide))))
}

// originsArea returns the box around the instance positions.
func originsArea(transforms []graphics.Mat4) box {
	area := emptyBox()
	for _, transform := range transforms {
		area.include(graphics.Vec4{transform[12], transform[13], transform[14], 1})
	}
	return area
}

// cellIndex returns the cell of an instance from its position on the ground plane.
func cellIndex(transform graphics.Mat4, area box, perSide int) int {
	column := cellCoordinate(transform[12], area.minimum[0], area.maximum[0], perSide)
	row := cellCoordinate(transform[14], area.minimum[2], area.maximum[2], perSide)
	return row*perSide + column
}

func cellCoordinate(value, minimum, maximum float32, perSide int) int {
	extent := maximum - minimum
	if extent <= 0 {
		return 0
	}
	coordinate := int(float32(perSide) * (value - minimum) / extent)
	return int(math.Max(0, math.Min(float64(perSide-1), float64(coordinate))))
}

// assignFirstInstances gives each cell the start of its range, in cell order.
func assignFirstInstances(cells []instanceCell) {
	next := 0
	for index := range cells {
		cells[index].first = next
		next += cells[index].count
	}
}

// measureCells computes the box and the largest instance of every cell.
func measureCells(cells []instanceCell, packed []float32, meshBounds box) {
	for index := range cells {
		cell := &cells[index]
		cell.bounds = emptyBox()
		for instance := cell.first; instance < cell.first+cell.count; instance++ {
			instanceBounds := meshBounds.transformed(unpackTransform(packed, instance), 1)
			cell.bounds.merge(instanceBounds)
			cell.largest = float32(math.Max(float64(cell.largest), float64(instanceBounds.diagonal())))
		}
	}
}

func unpackTransform(packed []float32, instance int) graphics.Mat4 {
	var transform graphics.Mat4
	copy(transform[:], packed[instance*floatsPerInstance:instance*floatsPerInstance+16])
	return transform
}

// nonEmptyCells drops the empty cells and returns the new number of every old cell.
func nonEmptyCells(cells []instanceCell) ([]instanceCell, []int) {
	result := make([]instanceCell, 0, len(cells))
	renumbered := make([]int, len(cells))
	for index, cell := range cells {
		renumbered[index] = len(result)
		if cell.count > 0 {
			result = append(result, cell)
		}
	}
	return result, renumbered
}

// packInstance writes the matrix and the color of one instance (white when missing).
func packInstance(instances *scene.Instances, instance int, destination []float32) {
	transform := instances.Transforms[instance]
	copy(destination[0:16], transform[:])
	color := graphics.Vec4{1, 1, 1, 1}
	if instance < len(instances.Colors) {
		color = instances.Colors[instance]
	}
	copy(destination[16:20], color[:])
}
