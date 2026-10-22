package opengl

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// seabedMap is the depth of the ground seen straight from above, over the whole scene. The water
// shader reads it to know how deep the water is under each point and colors it accordingly.
// It is rendered once per scene and again when a ground mesh changes (sceneplan decides).
type seabedMap struct {
	framebuffer, depthTexture uint32
	resolution                int32
	program                   *shaderProgram
	plan                      sceneplan.SeabedPlan
}

const seabedResolution = 2048

func newSeabedMap(resolution int32) (*seabedMap, error) {
	program, err := newShaderProgram(graphics.SceneVertexShader, graphics.DepthOnlyFragmentShader)
	if err != nil {
		return nil, err
	}
	seabed := &seabedMap{resolution: resolution, program: program}
	seabed.depthTexture = newDepthTexture(resolution)
	if seabed.framebuffer, err = newDepthFramebuffer(seabed.depthTexture); err != nil {
		return nil, err
	}
	return seabed, nil
}

// update renders the seabed map when the plan says the ground changed.
func (seabed *seabedMap) update(plan sceneplan.SeabedPlan) {
	seabed.plan = plan
	if plan.Redraw && plan.Available {
		seabed.render(plan.Region, plan.Casters)
	}
}

// render draws the ground depth from above, without vertical exaggeration.
func (seabed *seabedMap) render(region sceneplan.SeabedRegion, commands []sceneplan.Command) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, seabed.framebuffer)
	gl.Viewport(0, 0, seabed.resolution, seabed.resolution)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.DepthMask(true)
	program := seabed.program
	gl.UseProgram(program.handle)
	program.setMatrix("projection", region.ViewProjection)
	program.setMatrix("view", graphics.Identity())
	program.setFloat("verticalScale", 1)
	for _, command := range commands {
		drawWholeMesh(program, command)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

// bindForWater gives the lighting program the seabed map (texture unit 2) and the water colors.
func (seabed *seabedMap) bindForWater(program *shaderProgram, water *scene.Water) {
	enabled := water != nil && seabed.plan.Available
	program.setInteger("seabedMap", 2)
	// bound even when unused: some drivers check every sampler on every draw
	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, seabed.depthTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	if !enabled {
		program.setInteger("seabedEnabled", 0)
		return
	}
	program.setInteger("seabedEnabled", 1)
	program.setMatrix("seabedViewProjection", seabed.plan.Region.ViewProjection)
	gl.Uniform2f(program.location("seabedHeightRange"), seabed.plan.Region.Top, seabed.plan.Region.Bottom)
	program.setVector4("waterShallowColor", sceneplan.ShallowColorOf(water))
	program.setFloat("waterColorDepth", water.EffectiveColorDepth())
	program.setFloat("shoreFadeDepth", water.EffectiveShoreFade())
}

// drawWholeMesh draws a command's full mesh (or all its instances) without culling.
func drawWholeMesh(program *shaderProgram, command sceneplan.Command) {
	program.setMatrix("model", command.World)
	program.setFloat("sway", command.Material.Sway)
	buffers := uploadMesh(command.Node.Mesh)
	if command.Node.Instances != nil {
		drawInstanced(program, buffers, uploadInstances(command.Node.Instances, buffers))
		return
	}
	drawSingle(program, buffers)
}
