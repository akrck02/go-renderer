package opengl

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// seabedMap is the depth of the ground seen straight from above, over the whole scene. The water
// shader reads it to know how deep the water is under each point and colors it accordingly.
// It is rendered once per scene and again when a ground mesh changes.
type seabedMap struct {
	framebuffer, depthTexture uint32
	resolution                int32
	program                   *shaderProgram
	region                    seabedRegion
	renderedFor               *scene.Scene
	available                 bool
}

// seabedRegion is the box seen from above: its projection and the heights at depth 0 and 1.
type seabedRegion struct {
	viewProjection graphics.Mat4
	top, bottom    float32
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

// update renders the seabed map when the scene is new or its ground changed.
func (seabed *seabedMap) update(world *scene.Scene, commands []drawCommand) {
	groundCommands := seabedCasters(commands)
	if seabed.renderedFor == world && !anyMeshDirty(groundCommands) {
		return
	}
	seabed.renderedFor = world
	seabed.available = len(groundCommands) > 0
	if !seabed.available {
		return
	}
	seabed.region = fitSeabedRegion(boundsOfCommands(groundCommands))
	seabed.render(groundCommands)
}

// seabedCasters returns the walkable ground, or every opaque lit mesh when nothing is marked
// as ground.
func seabedCasters(commands []drawCommand) []drawCommand {
	var ground, opaque []drawCommand
	for _, command := range commands {
		if command.transparent || command.material.Kind != scene.KindLit {
			continue
		}
		opaque = append(opaque, command)
		if command.node.Mesh.Ground {
			ground = append(ground, command)
		}
	}
	if len(ground) > 0 {
		return ground
	}
	return opaque
}

func anyMeshDirty(commands []drawCommand) bool {
	for _, command := range commands {
		if command.node.Mesh.Dirty || (command.node.Instances != nil && command.node.Instances.Dirty) {
			return true
		}
	}
	return false
}

// boundsOfCommands returns the world bounding box of the commands' meshes.
func boundsOfCommands(commands []drawCommand) (minimum, maximum graphics.Vec4) {
	partial := &scene.Scene{}
	for _, command := range commands {
		partial.Nodes = append(partial.Nodes, &scene.Node{Transform: command.world, Mesh: command.node.Mesh, Instances: command.node.Instances})
	}
	return partial.Bounds()
}

// fitSeabedRegion looks straight down on the bounding box, with a small margin on every side.
func fitSeabedRegion(minimum, maximum graphics.Vec4) seabedRegion {
	margin := float32(math.Max(float64(maximum[0]-minimum[0]), float64(maximum[2]-minimum[2]))) * 0.01
	top, bottom := maximum[1]+margin, minimum[1]-margin
	centerX, centerZ := (minimum[0]+maximum[0])/2, (minimum[2]+maximum[2])/2
	halfWidth := (maximum[0]-minimum[0])/2 + margin
	halfDepth := (maximum[2]-minimum[2])/2 + margin
	view := graphics.LookAt(graphics.Vec4{centerX, top, centerZ, 1}, graphics.Vec4{centerX, bottom, centerZ, 1}, graphics.Vec4{0, 0, -1, 0})
	projection := graphics.OrthographicOpenGL(-halfWidth, halfWidth, -halfDepth, halfDepth, 0, top-bottom)
	return seabedRegion{viewProjection: projection.Multiply(view), top: top, bottom: bottom}
}

// render draws the ground depth from above, without vertical exaggeration.
func (seabed *seabedMap) render(commands []drawCommand) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, seabed.framebuffer)
	gl.Viewport(0, 0, seabed.resolution, seabed.resolution)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.DepthMask(true)
	program := seabed.program
	gl.UseProgram(program.handle)
	program.setMatrix("projection", seabed.region.viewProjection)
	program.setMatrix("view", graphics.Identity())
	program.setFloat("verticalScale", 1)
	for _, command := range commands {
		drawWholeMesh(program, command)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

// bindForWater gives the lighting program the seabed map (texture unit 2) and the water colors.
func (seabed *seabedMap) bindForWater(program *shaderProgram, water *scene.Water) {
	enabled := water != nil && seabed.available
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
	program.setMatrix("seabedViewProjection", seabed.region.viewProjection)
	gl.Uniform2f(program.location("seabedHeightRange"), seabed.region.top, seabed.region.bottom)
	program.setFloat("waterLevel", water.Level)
	program.setVector4("waterShallowColor", shallowColorOf(water))
	program.setFloat("waterColorDepth", water.EffectiveColorDepth())
}

func shallowColorOf(water *scene.Water) graphics.Vec4 {
	if water.Shallow == (graphics.Vec4{}) {
		return graphics.Vec4{0.3, 0.62, 0.6, 0.5}
	}
	return water.Shallow
}

// drawWholeMesh draws a command's full mesh (or all its instances) without culling.
func drawWholeMesh(program *shaderProgram, command drawCommand) {
	program.setMatrix("model", command.world)
	program.setFloat("sway", command.material.Sway)
	buffers := uploadMesh(command.node.Mesh)
	if command.node.Instances != nil {
		drawInstanced(program, buffers, uploadInstances(command.node.Instances, buffers))
		return
	}
	drawSingle(program, buffers)
}
