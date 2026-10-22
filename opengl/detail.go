package opengl

import (
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// drawGeometry draws a command's mesh as seen from a viewpoint: culled against it and at the
// level of detail its apparent size calls for (sceneplan chooses). Materials must already be set.
func (renderer *SceneRenderer) drawGeometry(program *shaderProgram, command sceneplan.Command, view sceneplan.Viewpoint, which sceneplan.Pass) {
	program.setMatrix("model", command.World)
	program.setFloat("sway", command.Material.Sway)
	program.setFloat("depthBias", command.Material.DepthBias)
	if command.Node.Instances != nil {
		renderer.drawVisibleInstances(program, command, view, which)
		return
	}
	mesh, visible := renderer.planner.MeshToDraw(command, view)
	if !visible {
		return
	}
	drawSingle(program, uploadMesh(mesh))
	renderer.planner.Count(which, mesh, 0)
}

// drawVisibleInstances draws the visible cells of an instanced node, one call per level of detail.
func (renderer *SceneRenderer) drawVisibleInstances(program *shaderProgram, command sceneplan.Command, view sceneplan.Viewpoint, which sceneplan.Pass) {
	for _, batch := range renderer.planner.InstanceBatches(command, view, which) {
		buffers, _ := batch.Handle.(*instanceBatch)
		if buffers == nil || batch.Count == 0 {
			continue
		}
		drawInstanced(program, buffers.mesh, &instanceBuffers{vertexArray: buffers.vertexArray, instanceCount: int32(batch.Count)})
		renderer.planner.Count(which, batch.Mesh, batch.Count)
	}
}

// instanceBatch is an instance buffer drawn with one version of the mesh.
type instanceBatch struct {
	vertexArray, instanceBuffer uint32
	mesh                        *meshBuffers
}

// instanceBatches keeps the planner's instance batches in OpenGL buffers.
type instanceBatches struct{}

// FillBatch uploads a batch's instances, making a new vertex array when the mesh version changed.
func (instanceBatches) FillBatch(handle any, mesh *scene.Mesh, packed []float32) any {
	buffers := uploadMesh(mesh)
	batch, _ := handle.(*instanceBatch)
	if batch == nil || batch.mesh != buffers {
		batch = newInstanceBatch(buffers)
	}
	if len(packed) > 0 {
		fillInstanceBuffer(batch.instanceBuffer, packed)
	}
	return batch
}

// RewriteInstances replaces some instances of a batch in place.
func (instanceBatches) RewriteInstances(handle any, position int, packed []float32) {
	batch := handle.(*instanceBatch)
	gl.BindBuffer(gl.ARRAY_BUFFER, batch.instanceBuffer)
	gl.BufferSubData(gl.ARRAY_BUFFER, position*sceneplan.FloatsPerInstance*bytesPerFloat, len(packed)*bytesPerFloat, firstFloat(packed))
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
