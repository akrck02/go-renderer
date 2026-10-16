// Package sceneplan decides what a frame of a scene.Scene draws, for any graphics backend: the
// draw commands in order, what the camera and the sun see, the level of detail of every mesh and
// group of instances, where the shadow map and the seabed map look. It makes no graphics calls; a
// backend (OpenGL, Metal) draws the plan, so both draw the same thing.
package sceneplan

import (
	"math"

	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
)

// Planner plans the frames of one renderer. It remembers what does not change between frames:
// the boxes of meshes, the grids of instances and the batches the backend keeps for them, the
// water meshes, the size of the scene.
type Planner struct {
	// WaterGridDivisions is the number of cells per side of the water surface (0 = 256). Fewer
	// suit GPUs that draw few triangles and no waves, such as the 3DS's.
	WaterGridDivisions int

	batches          BatchStore
	meshMeasures     map[*scene.Mesh]*meshMeasure
	instancedDetails map[*scene.Instances]*instancedDetail
	waterPlane       *scene.Mesh
	waterFloor       *scene.Mesh
	verticalScale    float32 // of the frame being planned
	counters         [passCount]counter
	measured         *scene.Scene
	sceneSize        float32 // horizontal size of the measured scene
	sceneHeight      float32 // vertical span of the measured scene (before vertical scaling)
	seabedFor        *scene.Scene
	seabed           SeabedPlan
}

// NewPlanner makes a planner whose instance batches live in batches.
func NewPlanner(batches BatchStore) *Planner {
	return &Planner{batches: batches,
		meshMeasures: map[*scene.Mesh]*meshMeasure{}, instancedDetails: map[*scene.Instances]*instancedDetail{}}
}

// Frame is the plan of one frame.
type Frame struct {
	Matrices    CameraMatrices
	Light       scene.Daylight
	View        Viewpoint // what the camera sees
	Opaque      []Command // drawn first, in order
	Transparent []Command // drawn after the sky, farthest first
	Commands    []Command // all of them, opaque first
	Shadow      ShadowPlan
	Seabed      SeabedPlan
}

// ShadowPlan is where the shadow map looks this frame.
type ShadowPlan struct {
	Enabled bool
	Region  ShadowRegion
	View    Viewpoint // what the sun sees, at one level of detail coarser
}

// Plan plans a frame of the scene from the camera, for a framebuffer of this size. It also resets
// the counts that Count adds to.
func (planner *Planner) Plan(world *scene.Scene, camera models.Camera, width, height int, shadows ShadowSettings) Frame {
	planner.verticalScale = world.Environment.EffectiveVerticalScale()
	planner.counters = [passCount]counter{}
	frame := Frame{Matrices: ComputeCameraMatrices(camera, width, height), Light: world.Environment.Daylight()}
	frame.View = newViewpoint(frame.Matrices.ViewProjection, camera)
	frame.Commands = planner.collectCommands(world, camera)
	sortOpaqueBeforeTransparent(frame.Commands)
	frame.Opaque, frame.Transparent = splitOpaqueAndTransparent(frame.Commands)
	frame.Seabed = planner.planSeabed(world, frame.Commands)
	if shadows.Enabled {
		frame.Shadow = planner.planShadows(world, camera, frame, shadows)
	}
	return frame
}

// planShadows fits the shadow map's region to the camera and the scene.
func (planner *Planner) planShadows(world *scene.Scene, camera models.Camera, frame Frame, settings ShadowSettings) ShadowPlan {
	planner.measureScene(world)
	verticalScale := world.Environment.EffectiveVerticalScale()
	extent := settings.extent(camera, planner.sceneSize)
	depthRange := extent*2 + planner.sceneHeight*verticalScale*2
	region := fitShadowRegion(frame.Light.LightDirection, shadowCenter(camera, extent), extent, depthRange, settings.EffectiveResolution())
	view := frame.View
	view.volume = frustumFromMatrix(region.LightViewProjection())
	view.levelBias += shadowLevelBias
	return ShadowPlan{Enabled: true, Region: region, View: view}
}

// measureScene caches the size of the scene, used to size the shadow region.
func (planner *Planner) measureScene(world *scene.Scene) {
	if planner.measured == world {
		return
	}
	minimum, maximum := world.Bounds()
	planner.sceneSize = float32(math.Max(float64(maximum[0]-minimum[0]), float64(maximum[2]-minimum[2])))
	planner.sceneHeight = maximum[1] - minimum[1]
	if planner.sceneSize <= 0 || math.IsInf(float64(planner.sceneSize), 0) {
		planner.sceneSize, planner.sceneHeight = 1, 1
	}
	planner.measured = world
}
