package sceneplan

import (
	"math"
	"slices"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// Pass identifies the passes that choose their own visible parts and levels of detail.
type Pass int

// The passes: the main one, seen from the camera, and the shadow map's.
const (
	MainPass Pass = iota
	ShadowPass
	passCount
)

// shadowLevelBias makes casters one level of detail coarser than what the camera sees: a shadow
// shows an outline, and the shadow pass covers more than the view (what is behind the camera too).
const shadowLevelBias = 1

// BatchStore keeps the GPU buffers of instance batches for a backend. The planner decides which
// instances go in each batch; the store holds them where the GPU reads them.
type BatchStore interface {
	// FillBatch puts packed instances (FloatsPerInstance values each) in a batch drawn with mesh,
	// a version of the instanced mesh, and returns the batch. handle is the batch it had for this
	// level before (nil the first time); the store may reuse it or make a new one.
	FillBatch(handle any, mesh *scene.Mesh, packed []float32) any
	// RewriteInstances replaces instances of a batch, starting at position (counted in instances).
	RewriteInstances(handle any, position int, packed []float32)
}

// Batch is one draw call of an instanced node: a version of its mesh and the instances drawn with it.
type Batch struct {
	Mesh   *scene.Mesh
	Count  int
	Handle any // what the backend's BatchStore returned
}

// counter adds up what a pass draws.
type counter struct {
	drawCalls, triangles, instances int
}

// Count records a draw call of a mesh in a pass, with this many instances (0 for a single copy).
func (planner *Planner) Count(which Pass, mesh *scene.Mesh, instances int) {
	counter := &planner.counters[which]
	counter.drawCalls++
	perCopy := ElementCount(mesh) / 3
	if instances == 0 {
		counter.triangles += perCopy
		return
	}
	counter.triangles += perCopy * instances
	counter.instances += instances
}

// instancedDetail is what the planner keeps for an instanced node: its grid and, per pass, the
// cells it chose and their batches.
type instancedDetail struct {
	grid       *instanceGrid
	mesh       *scene.Mesh
	selections [passCount]*instanceSelection
	// the cells' boxes in world units, for the world matrix and vertical scale they were made for:
	// the same every frame unless the node moves or the relief's exaggeration changes
	worldCells        []box
	worldCellsFor     graphics.Matrix
	worldCellsScale   float32
	worldCellsLargest []float32
}

// instanceSelection holds the visible cells of one pass, grouped by level of detail, and one batch
// per level. Batches are filled again only when the chosen cells change.
type instanceSelection struct {
	cellsByLevel [][]int
	chosen       [][]int
	batches      []Batch
	staging      []float32
	cellOffsets  []map[int]int // per level: where each chosen cell starts in the batch (in instances)
}

// meshMeasure caches the box of a mesh.
type meshMeasure struct {
	bounds box
}

// MeshToDraw returns the version of a single (not instanced) mesh to draw from a viewpoint, at the
// level of detail its apparent size calls for, and false when it is outside the view.
func (planner *Planner) MeshToDraw(command Command, view Viewpoint) (*scene.Mesh, bool) {
	mesh := command.Node.Mesh
	worldBounds := planner.boundsOf(mesh).transformed(command.World, planner.verticalScale)
	if !view.volume.containsBox(worldBounds) {
		return nil, false
	}
	apparent := view.apparentSize(worldBounds.diagonal(), worldBounds.distanceTo(view.eye))
	level := coarserLevel(mesh, mesh.LevelForScreenSize(apparent), view.levelBias)
	return mesh.Level(level), true
}

// boundsOf returns the cached box of a mesh, measured again when the mesh changed.
func (planner *Planner) boundsOf(mesh *scene.Mesh) box {
	measure, found := planner.meshMeasures[mesh]
	if !found || mesh.Dirty {
		minimum, maximum := mesh.Bounds()
		measure = &meshMeasure{bounds: box{minimum, maximum}}
		planner.meshMeasures[mesh] = measure
	}
	return measure.bounds
}

func coarserLevel(mesh *scene.Mesh, level, bias int) int {
	return int(math.Min(float64(level+bias), float64(mesh.LevelCount()-1)))
}

// InstanceBatches returns the batches to draw for an instanced node from a viewpoint: its visible
// cells, one batch per level of detail. Batches whose cells changed are filled again through the
// planner's BatchStore.
func (planner *Planner) InstanceBatches(command Command, view Viewpoint, which Pass) []Batch {
	detail := planner.instancedDetailFor(command.Node)
	selection := detail.selections[which]
	selection.choose(planner.batches, detail, command.World, planner.verticalScale, view)
	return selection.batches
}

// instancedDetailFor returns the grid of a node's instances, building it when they are new or changed.
func (planner *Planner) instancedDetailFor(node *scene.Node) *instancedDetail {
	detail, found := planner.instancedDetails[node.Instances]
	if found && !node.Instances.Dirty && detail.mesh == node.Mesh {
		detail.applyChanges(planner.batches, node.Instances, planner.boundsOf(node.Mesh))
		return detail
	}
	grid := buildInstanceGrid(node.Instances, planner.boundsOf(node.Mesh))
	detail = &instancedDetail{grid: grid, mesh: node.Mesh}
	for which := range detail.selections {
		detail.selections[which] = &instanceSelection{}
	}
	planner.instancedDetails[node.Instances] = detail
	node.Instances.Dirty, node.Instances.Changed = false, nil
	return detail
}

// applyChanges updates the instances a simulation marked as changed, in the grid and in the
// batches that hold them, rewriting only their part of each batch.
func (detail *instancedDetail) applyChanges(store BatchStore, instances *scene.Instances, meshBounds box) {
	for _, instance := range instances.Changed {
		if instance < 0 || instance >= len(detail.grid.slots) {
			continue
		}
		cell := detail.grid.updateInstance(instances, instance, meshBounds)
		detail.worldCells = nil // a cell's box may have changed
		for _, selection := range detail.selections {
			selection.rewriteInstance(store, detail.grid, instance, cell)
		}
	}
	instances.Changed = instances.Changed[:0]
}

// rewriteInstance rewrites one instance in the batch of the level that holds its cell, if any.
func (selection *instanceSelection) rewriteInstance(store BatchStore, grid *instanceGrid, instance, cell int) {
	for level, offsets := range selection.cellOffsets {
		start, chosen := offsets[cell]
		batch := selection.batches[level]
		if !chosen || batch.Handle == nil {
			continue
		}
		slot := grid.slots[instance]
		position := start + slot - grid.cells[cell].first
		store.RewriteInstances(batch.Handle, position, grid.packed[slot*FloatsPerInstance:(slot+1)*FloatsPerInstance])
	}
}

// choose picks the visible cells and their levels, and fills the batches that changed.
func (selection *instanceSelection) choose(store BatchStore, detail *instancedDetail, world graphics.Matrix, verticalScale float32, view Viewpoint) {
	levels := detail.mesh.LevelCount()
	selection.resetLevels(levels)
	worldCells, largest := detail.cellsInWorld(world, verticalScale)
	for index, cellBounds := range worldCells {
		if !view.volume.containsBox(cellBounds) {
			continue
		}
		apparent := view.apparentSize(largest[index], cellBounds.distanceTo(view.eye))
		level := coarserLevel(detail.mesh, detail.mesh.LevelForScreenSize(apparent), view.levelBias)
		selection.cellsByLevel[level] = append(selection.cellsByLevel[level], index)
	}
	for level := 0; level < levels; level++ {
		if !slices.Equal(selection.cellsByLevel[level], selection.chosen[level]) || selection.batches[level].Handle == nil {
			selection.fill(store, detail, level)
		}
	}
}

// cellsInWorld are the cells' boxes in world units (grown a little, as the instances sway) and the
// size of their largest instance, made again only when the node moves or the exaggeration changes.
func (detail *instancedDetail) cellsInWorld(world graphics.Matrix, verticalScale float32) ([]box, []float32) {
	if detail.worldCells != nil && detail.worldCellsFor == world && detail.worldCellsScale == verticalScale {
		return detail.worldCells, detail.worldCellsLargest
	}
	worldScale := largestScale(world)
	detail.worldCells = make([]box, len(detail.grid.cells))
	detail.worldCellsLargest = make([]float32, len(detail.grid.cells))
	for index, cell := range detail.grid.cells {
		detail.worldCells[index] = cell.bounds.transformed(world, verticalScale).grown(cell.largest * worldScale * 0.05)
		detail.worldCellsLargest[index] = cell.largest * worldScale
	}
	detail.worldCellsFor, detail.worldCellsScale = world, verticalScale
	return detail.worldCells, detail.worldCellsLargest
}

// resetLevels empties this frame's lists, keeping the previous choice to compare with.
func (selection *instanceSelection) resetLevels(levels int) {
	if len(selection.cellsByLevel) != levels {
		selection.cellsByLevel = make([][]int, levels)
		selection.chosen = make([][]int, levels)
		selection.batches = make([]Batch, levels)
		selection.cellOffsets = make([]map[int]int, levels)
	}
	for level := range selection.cellsByLevel {
		selection.cellsByLevel[level] = selection.cellsByLevel[level][:0]
	}
}

// fill copies the instances of the chosen cells of a level into its batch.
func (selection *instanceSelection) fill(store BatchStore, detail *instancedDetail, level int) {
	cells := selection.cellsByLevel[level]
	selection.chosen[level] = append(selection.chosen[level][:0], cells...)
	selection.staging = selection.staging[:0]
	offsets := make(map[int]int, len(cells))
	selection.cellOffsets[level] = offsets
	for _, index := range cells {
		cell := detail.grid.cells[index]
		offsets[index] = len(selection.staging) / FloatsPerInstance
		selection.staging = append(selection.staging, detail.grid.packed[cell.first*FloatsPerInstance:(cell.first+cell.count)*FloatsPerInstance]...)
	}
	mesh := detail.mesh.Level(level)
	batch := &selection.batches[level]
	batch.Handle = store.FillBatch(batch.Handle, mesh, selection.staging)
	batch.Mesh, batch.Count = mesh, len(selection.staging)/FloatsPerInstance
}

// largestScale returns the largest axis scale of a transform.
func largestScale(transform graphics.Matrix) float32 {
	largest := 0.0
	for column := 0; column < 3; column++ {
		axis := graphics.Vector{transform[column*4], transform[column*4+1], transform[column*4+2], 0}
		largest = math.Max(largest, math.Sqrt(float64(axis.Dot(axis))))
	}
	return float32(largest)
}
