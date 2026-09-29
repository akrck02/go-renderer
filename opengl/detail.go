package opengl

import (
	"math"
	"slices"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// pass identifies the passes that choose their own visible parts and levels of detail.
type pass int

const (
	mainPass pass = iota
	shadowPass
	passCount
)

// drawCounter adds up what a pass draws.
type drawCounter struct {
	drawCalls, triangles, instances int
}

func (counter *drawCounter) add(buffers *meshBuffers, instances int) {
	counter.drawCalls++
	perCopy := int(buffers.elementCount) / 3
	if instances == 0 {
		counter.triangles += perCopy
		return
	}
	counter.triangles += perCopy * instances
	counter.instances += instances
}

// instancedDetail is what the renderer keeps for an instanced node: its grid and, per pass, the
// cells it chose and their GPU buffers.
type instancedDetail struct {
	grid       *instanceGrid
	mesh       *scene.Mesh
	selections [passCount]*instanceSelection
}

// instanceSelection holds the visible cells of one pass, grouped by level of detail, and one GPU
// batch per level. Batches are uploaded again only when the chosen cells change.
type instanceSelection struct {
	cellsByLevel [][]int
	chosen       [][]int
	batches      []*instanceBatch
	staging      []float32
	cellOffsets  []map[int]int // per level: where each chosen cell starts in the batch (in instances)
}

// instanceBatch is an instance buffer drawn with one version of the mesh.
type instanceBatch struct {
	vertexArray, instanceBuffer uint32
	count                       int32
	mesh                        *meshBuffers
}

// meshMeasure caches the box of a mesh.
type meshMeasure struct {
	bounds box
}

// drawGeometry draws a command's mesh as seen from a viewpoint: culled against it and at the
// level of detail its apparent size calls for. Materials must already be set.
func (renderer *SceneRenderer) drawGeometry(program *shaderProgram, command drawCommand, view viewpoint, which pass, counter *drawCounter) {
	program.setMatrix("model", command.world)
	program.setFloat("sway", command.material.Sway)
	if command.node.Instances != nil {
		renderer.drawVisibleInstances(program, command, view, which, counter)
		return
	}
	renderer.drawVisibleMesh(program, command, view, counter)
}

// drawVisibleMesh draws a single mesh unless it is outside the view, at its level of detail.
func (renderer *SceneRenderer) drawVisibleMesh(program *shaderProgram, command drawCommand, view viewpoint, counter *drawCounter) {
	mesh := command.node.Mesh
	worldBounds := renderer.boundsOf(mesh).transformed(command.world, renderer.verticalScale)
	if !view.volume.containsBox(worldBounds) {
		return
	}
	apparent := view.apparentSize(worldBounds.diagonal(), worldBounds.distanceTo(view.eye))
	level := coarserLevel(mesh, mesh.LevelForScreenSize(apparent), view.levelBias)
	buffers := uploadMesh(mesh.Level(level))
	drawSingle(program, buffers)
	counter.add(buffers, 0)
}

// boundsOf returns the cached box of a mesh, measured again when the mesh changed.
func (renderer *SceneRenderer) boundsOf(mesh *scene.Mesh) box {
	measure, found := renderer.meshMeasures[mesh]
	if !found || mesh.Dirty {
		minimum, maximum := mesh.Bounds()
		measure = &meshMeasure{bounds: box{minimum, maximum}}
		renderer.meshMeasures[mesh] = measure
	}
	return measure.bounds
}

func coarserLevel(mesh *scene.Mesh, level, bias int) int {
	return int(math.Min(float64(level+bias), float64(mesh.LevelCount()-1)))
}

// drawVisibleInstances draws the visible cells of an instanced node, one call per level of detail.
func (renderer *SceneRenderer) drawVisibleInstances(program *shaderProgram, command drawCommand, view viewpoint, which pass, counter *drawCounter) {
	detail := renderer.instancedDetailFor(command.node)
	selection := detail.selections[which]
	selection.choose(detail, command.world, renderer.verticalScale, view)
	for _, batch := range selection.batches {
		if batch == nil || batch.count == 0 {
			continue
		}
		drawInstanced(program, batch.mesh, &instanceBuffers{vertexArray: batch.vertexArray, instanceCount: batch.count})
		counter.add(batch.mesh, int(batch.count))
	}
}

// instancedDetailFor returns the grid of a node's instances, building it when they are new or changed.
func (renderer *SceneRenderer) instancedDetailFor(node *scene.Node) *instancedDetail {
	detail, found := renderer.instancedDetails[node.Instances]
	if found && !node.Instances.Dirty && detail.mesh == node.Mesh {
		detail.applyChanges(node.Instances, renderer.boundsOf(node.Mesh))
		return detail
	}
	grid := buildInstanceGrid(node.Instances, renderer.boundsOf(node.Mesh))
	detail = &instancedDetail{grid: grid, mesh: node.Mesh}
	for which := range detail.selections {
		detail.selections[which] = &instanceSelection{}
	}
	renderer.instancedDetails[node.Instances] = detail
	node.Instances.Dirty, node.Instances.Changed = false, nil
	return detail
}

// applyChanges updates the instances a simulation marked as changed, in the grid and in the
// batches that hold them, uploading only their part of each buffer.
func (detail *instancedDetail) applyChanges(instances *scene.Instances, meshBounds box) {
	for _, instance := range instances.Changed {
		if instance < 0 || instance >= len(detail.grid.slots) {
			continue
		}
		cell := detail.grid.updateInstance(instances, instance, meshBounds)
		for _, selection := range detail.selections {
			selection.uploadInstance(detail.grid, instance, cell)
		}
	}
	instances.Changed = instances.Changed[:0]
}

// uploadInstance rewrites one instance in the batch of the level that holds its cell, if any.
func (selection *instanceSelection) uploadInstance(grid *instanceGrid, instance, cell int) {
	for level, offsets := range selection.cellOffsets {
		start, chosen := offsets[cell]
		batch := selection.batches[level]
		if !chosen || batch == nil {
			continue
		}
		slot := grid.slots[instance]
		position := start + slot - grid.cells[cell].first
		values := grid.packed[slot*floatsPerInstance : (slot+1)*floatsPerInstance]
		gl.BindBuffer(gl.ARRAY_BUFFER, batch.instanceBuffer)
		gl.BufferSubData(gl.ARRAY_BUFFER, position*floatsPerInstance*bytesPerFloat, len(values)*bytesPerFloat, gl.Ptr(values))
	}
}

// choose picks the visible cells and their levels, and uploads the batches that changed.
func (selection *instanceSelection) choose(detail *instancedDetail, world graphics.Mat4, verticalScale float32, view viewpoint) {
	levels := detail.mesh.LevelCount()
	selection.resetLevels(levels)
	worldScale := largestScale(world)
	for index, cell := range detail.grid.cells {
		cellBounds := cell.bounds.transformed(world, verticalScale).grown(cell.largest * worldScale * 0.05)
		if !view.volume.containsBox(cellBounds) {
			continue
		}
		apparent := view.apparentSize(cell.largest*worldScale, cellBounds.distanceTo(view.eye))
		level := coarserLevel(detail.mesh, detail.mesh.LevelForScreenSize(apparent), view.levelBias)
		selection.cellsByLevel[level] = append(selection.cellsByLevel[level], index)
	}
	for level := 0; level < levels; level++ {
		if !slices.Equal(selection.cellsByLevel[level], selection.chosen[level]) || selection.batches[level] == nil {
			selection.upload(detail, level)
		}
	}
}

// resetLevels empties this frame's lists, keeping the previous choice to compare with.
func (selection *instanceSelection) resetLevels(levels int) {
	if len(selection.cellsByLevel) != levels {
		selection.cellsByLevel = make([][]int, levels)
		selection.chosen = make([][]int, levels)
		selection.batches = make([]*instanceBatch, levels)
		selection.cellOffsets = make([]map[int]int, levels)
	}
	for level := range selection.cellsByLevel {
		selection.cellsByLevel[level] = selection.cellsByLevel[level][:0]
	}
}

// upload copies the instances of the chosen cells of a level into its batch.
func (selection *instanceSelection) upload(detail *instancedDetail, level int) {
	cells := selection.cellsByLevel[level]
	selection.chosen[level] = append(selection.chosen[level][:0], cells...)
	selection.staging = selection.staging[:0]
	offsets := make(map[int]int, len(cells))
	selection.cellOffsets[level] = offsets
	for _, index := range cells {
		cell := detail.grid.cells[index]
		offsets[index] = len(selection.staging) / floatsPerInstance
		selection.staging = append(selection.staging, detail.grid.packed[cell.first*floatsPerInstance:(cell.first+cell.count)*floatsPerInstance]...)
	}
	mesh := uploadMesh(detail.mesh.Level(level))
	if selection.batches[level] == nil || selection.batches[level].mesh != mesh {
		selection.batches[level] = newInstanceBatch(mesh)
	}
	batch := selection.batches[level]
	batch.count = int32(len(selection.staging) / floatsPerInstance)
	if batch.count > 0 {
		fillInstanceBuffer(batch.instanceBuffer, selection.staging)
	}
}

// newInstanceBatch creates a vertex array that combines a mesh with its own instance buffer.
func newInstanceBatch(mesh *meshBuffers) *instanceBatch {
	batch := &instanceBatch{mesh: mesh}
	gl.GenVertexArrays(1, &batch.vertexArray)
	gl.GenBuffers(1, &batch.instanceBuffer)
	gl.BindVertexArray(batch.vertexArray)
	gl.BindBuffer(gl.ARRAY_BUFFER, mesh.vertexBuffer)
	describeVertexLayout()
	if mesh.indexed {
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, mesh.indexBuffer)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, batch.instanceBuffer)
	describeInstanceLayout()
	gl.BindVertexArray(0)
	return batch
}

// largestScale returns the largest axis scale of a transform.
func largestScale(transform graphics.Mat4) float32 {
	largest := 0.0
	for column := 0; column < 3; column++ {
		axis := graphics.Vec4{transform[column*4], transform[column*4+1], transform[column*4+2], 0}
		largest = math.Max(largest, math.Sqrt(float64(axis.Dot(axis))))
	}
	return float32(largest)
}
