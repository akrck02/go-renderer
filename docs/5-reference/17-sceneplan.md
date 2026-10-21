# sceneplan

```go
import "github.com/akrck02/go-renderer/sceneplan"
```

Package sceneplan decides what a frame of a scene.Scene draws, for any graphics
backend: the draw commands in order, what the camera and the sun see, the level
of detail of every mesh and group of instances, where the shadow map and the
seabed map look. It makes no graphics calls; a backend (OpenGL, Metal) draws the
plan, so both draw the same thing.

### Functions

| Function | |
|---|---|
| `func BaseColorOf(material *scene.Material) graphics.Vec4` | BaseColorOf returns a material's colour, white when it is unset. |
| `func CastsShadow(command Command) bool` | CastsShadow tells whether a command is drawn into the shadow map. |
| `func ColorOrWhite(color graphics.Vec4) graphics.Vec4` | ColorOrWhite returns the colour, or white when it is unset. |
| `func ConstellationVisibility(sky *scene.Sky, light scene.Daylight) float32` | ConstellationVisibility is how much the constellations show: fully at night, faintly by day as much as the stars do. |
| `func ElementCount(mesh *scene.Mesh) int` | ElementCount is how many vertices a mesh's draw call reads: its indices, or its vertices when it has none. |
| `func InterleaveHarmonics(variation *scene.LevelVariation) []float32` | InterleaveHarmonics packs the in-phase and in-quadrature grids of a water level variation as two channels of one texture. |
| `func InterleaveVertices(mesh *scene.Mesh) []float32` | InterleaveVertices packs position, normal and colour per vertex (white when there are no colours), computing the normals first when the mesh has none. |
| `func IsTransparent(material *scene.Material) bool` | IsTransparent tells whether a material is blended over what is behind it. |
| `func LineColor(color graphics.Vec4) graphics.Vec4` | LineColor is the colour a constellation's lines are drawn with: its own, more transparent. |
| `func MergeInstances(into, mesh *scene.Mesh, packed []float32, count int)` | MergeInstances fills into with count copies of mesh, each placed by its instance's matrix, its normals turned with it and its colours tinted by the instance's colour. |
| `func NormalMatrix(model graphics.Mat4) [9]float32` | NormalMatrix is the inverse transpose of a model matrix's rotation and scale: what turns normals so they stay perpendicular to surfaces scaled differently along each axis. |
| `func PackInstances(instances *scene.Instances) []float32` | PackInstances writes the matrix and the colour of every instance (white when missing). |
| `func ScaleOrOne(value float32) float32` | ScaleOrOne returns the value, or 1 when it is not positive. |
| `func ShallowColorOf(water *scene.Water) graphics.Vec4` | ShallowColorOf is the colour of shallow water, with a default when it is unset. |
| `func WaveLengthFor(water *scene.Water) float32` | WaveLengthFor is the length of the sea's main wave, 1 without water. |
| `func WindDirection(wind scene.Wind) (float32, float32)` | WindDirection is the horizontal direction of the wind, normalised, as x and z. |

### Constants and variables

```go
const FloatsPerInstance = 20
```

FloatsPerInstance is the size of an instance on the GPU: its matrix 16 (column by column) and its colour 4.

```go
const FloatsPerVertex = 10
```

FloatsPerVertex is the size of a vertex on the GPU: position 3, normal 3, colour 4.

```go
const MaxLightsPerDraw = 4
```

MaxLightsPerDraw is how many point lights one draw takes at most: the nearest whose reach meets what it draws.

### Batch

Batch is one draw call of an instanced node: a version of its mesh and the instances drawn with it.

```go
type Batch struct {
	Mesh   *scene.Mesh
	Count  int
	Handle any // what the backend's BatchStore returned
}
```

### BatchStore

BatchStore keeps the GPU buffers of instance batches for a backend. The planner decides which instances go in each batch; the store holds them where the GPU reads them.

```go
type BatchStore interface {
	// FillBatch puts packed instances (FloatsPerInstance values each) in a batch drawn with mesh,
	// a version of the instanced mesh, and returns the batch. handle is the batch it had for this
	// level before (nil the first time); the store may reuse it or make a new one.
	FillBatch(handle any, mesh *scene.Mesh, packed []float32) any
	// RewriteInstances replaces instances of a batch, starting at position (counted in instances).
	RewriteInstances(handle any, position int, packed []float32)
}
```

### Blending

Blending is how a surface is mixed with what is behind it.

```go
type Blending int
```

```go
const (
	BlendNone     Blending = iota // opaque: it covers what is behind, and writes depth
	BlendAlpha                    // translucent: its alpha mixes it over what is behind
	BlendAdditive                 // its light, scaled by its alpha, adds to what is behind
)
```

| Function or method | |
|---|---|
| `func BlendingOf(command Command) Blending` | BlendingOf is how a command's surface is mixed with what is behind it. |

### CameraMatrices

CameraMatrices are the matrices of one frame, in OpenGL's clip space (depth from -1 to 1); a backend with another clip space converts them.

```go
type CameraMatrices struct {
	Projection, View, ViewProjection graphics.Mat4
}
```

| Function or method | |
|---|---|
| `func ComputeCameraMatrices(camera models.Camera, width, height int) CameraMatrices` | ComputeCameraMatrices returns the camera's matrices for a framebuffer of this size. |

### Command

Command is one node to draw this frame.

```go
type Command struct {
	Node                  *scene.Node
	World                 graphics.Mat4
	Material              *scene.Material
	SquaredCameraDistance float32
	Transparent           bool
	SeaSurface            bool // the environment's water plane: the only surface the level variation moves
	// Lights are the frame's point lights this draw takes (see Frame.Lights); none for the sea,
	// whose look is its own.
	Lights DrawLights
}
```

### DrawLights

DrawLights are the lights a command takes: indices into the frame's Lights.

```go
type DrawLights struct {
	Indices [MaxLightsPerDraw]uint16
	Count   int
}
```

### Frame

Frame is the plan of one frame.

```go
type Frame struct {
	Matrices    CameraMatrices
	Light       scene.Daylight
	View        Viewpoint    // what the camera sees
	Opaque      []Command    // drawn first, in order
	Transparent []Command    // drawn after the sky, farthest first
	Commands    []Command    // all of them, opaque first
	Lights      []FrameLight // the scene's point lights; each command says which it takes
	Shadow      ShadowPlan
	Seabed      SeabedPlan
}
```

### FrameLight

FrameLight is a scene's point light as the shaders take it: its position with the vertical scale, its colour times its intensity, and its radii.

```go
type FrameLight struct {
	Position     graphics.Vec4
	Color        graphics.Vec4 // linear, times the intensity
	Inner, Outer float32
}
```

### FrameStatistics

FrameStatistics describes the last frame drawn by a scene renderer. Counts are always filled; times only when the renderer is profiling.

```go
type FrameStatistics struct {
	DrawCalls       int // main pass
	ShadowDrawCalls int
	Triangles       int // main pass, instances included
	ShadowTriangles int
	Instances       int // instances drawn in the main pass

	CPU time.Duration // time spent drawing on the processor (without waiting for the GPU)
	GPU PassTimes     // time spent by the GPU on each pass
}
```

### LineRange

LineRange is the part of the constellation buffer that holds one constellation's lines, with its colour.

```go
type LineRange struct {
	First, Count int32
	Color        graphics.Vec4
}
```

| Function or method | |
|---|---|
| `func ConstellationVertices(constellations []scene.Constellation) (vertices []float32, pointCount int32, lineRanges []LineRange)` | ConstellationVertices lays out every star first (drawn as points) and then the line segments of each constellation, three floats (a direction) per vertex. |

### MergedBatch

MergedBatch keeps a batch's packed instances, and the batch as one mesh, for GPUs that cannot draw instances (OpenGL ES 2, the 3DS's PICA200): a draw call per copy is far too slow (a forest is thousands of calls). The mesh is made again only when the batch changes.

```go
type MergedBatch struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (batch *MergedBatch) MergedFor(mesh *scene.Mesh, count int) *scene.Mesh` | MergedFor returns the batch's first count instances of mesh as one mesh, making it when needed. |

### MergingStore

MergingStore keeps the planner's instance batches as MergedBatch.

```go
type MergingStore struct{}
```

| Function or method | |
|---|---|
| `func (MergingStore) FillBatch(handle any, mesh *scene.Mesh, packed []float32) any` |  |
| `func (MergingStore) RewriteInstances(handle any, position int, packed []float32)` |  |

### Pass

Pass identifies the passes that choose their own visible parts and levels of detail.

```go
type Pass int
```

```go
const (
	MainPass Pass = iota
	ShadowPass
)
```

### PassTimes

PassTimes are GPU durations of the passes of one frame.

```go
type PassTimes struct {
	Seabed, Shadows, Sky, Scene time.Duration
}
```

| Function or method | |
|---|---|
| `func (times PassTimes) Total() time.Duration` | Total returns the sum of the pass durations. |

### Planner

Planner plans the frames of one renderer. It remembers what does not change between frames: the boxes of meshes, the grids of instances and the batches the backend keeps for them, the water meshes, the size of the scene.

```go
type Planner struct {
	// WaterGridDivisions is the number of cells per side of the water surface (0 = 256). Fewer
	// suit GPUs that draw few triangles and no waves, such as the 3DS's.
	WaterGridDivisions int

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewPlanner(batches BatchStore) *Planner` | NewPlanner makes a planner whose instance batches live in batches. |
| `func (planner *Planner) Count(which Pass, mesh *scene.Mesh, instances int)` | Count records a draw call of a mesh in a pass, with this many instances (0 for a single copy). |
| `func (planner *Planner) InstanceBatches(command Command, view Viewpoint, which Pass) []Batch` | InstanceBatches returns the batches to draw for an instanced node from a viewpoint: its visible cells, one batch per level of detail. |
| `func (planner *Planner) MeshToDraw(command Command, view Viewpoint) (*scene.Mesh, bool)` | MeshToDraw returns the version of a single (not instanced) mesh to draw from a viewpoint, at the level of detail its apparent size calls for, and false when it is outside the view. |
| `func (planner *Planner) Plan(world *scene.Scene, camera models.Camera, width, height int, shadows ShadowSettings) Frame` | Plan plans a frame of the scene from the camera, for a framebuffer of this size. |
| `func (planner *Planner) Statistics(processorTime time.Duration) FrameStatistics` | Statistics returns the counts of the frame planned last, with the processor time it took. |

### SeabedPlan

SeabedPlan says whether the seabed map, the depth of the ground seen from above over the whole scene, must be drawn again this frame. The water shader reads it to know how deep the water is.

```go
type SeabedPlan struct {
	Redraw    bool // the scene is new or its ground changed
	Available bool // there is ground to draw
	Region    SeabedRegion
	Casters   []Command // what to draw into the map, whole and without culling
}
```

### SeabedRegion

SeabedRegion is the box the seabed map sees from straight above: its projection (OpenGL clip space) and the heights at depth 0 and 1.

```go
type SeabedRegion struct {
	ViewProjection graphics.Mat4
	Top, Bottom    float32
}
```

### ShadowPlan

ShadowPlan is where the shadow map looks this frame.

```go
type ShadowPlan struct {
	Enabled bool
	Region  ShadowRegion
	View    Viewpoint // what the sun sees, at one level of detail coarser
}
```

### ShadowRegion

ShadowRegion is what the lighting pass needs to look up the shadow map.

```go
type ShadowRegion struct {
	LightView, LightProjection graphics.Mat4 // OpenGL clip space
	TexelSize                  float32       // in shadow map coordinates
	NormalOffset               float32       // in world units
}
```

| Function or method | |
|---|---|
| `func (region ShadowRegion) LightViewProjection() graphics.Mat4` | LightViewProjection is the light's projection times its view. |

### ShadowSettings

ShadowSettings controls the sun shadows. A single shadow map covers a square region around the camera target. The region grows with the distance between the camera and its target, so a far view shadows the whole scene coarsely and a close (or walking) view gets sharp shadows nearby.

```go
type ShadowSettings struct {
	Enabled                 bool
	Resolution              int32   // texels per side of the shadow map (0 = 4096)
	MinimumExtent           float32 // smallest half-size of the region in world units (0 = 1% of the scene)
	MaximumExtent           float32 // largest half-size (0 = the whole scene)
	ExtentPerCameraDistance float32 // half-size per unit of camera distance (0 = 1.2)
}
```

| Function or method | |
|---|---|
| `func DefaultShadowSettings() ShadowSettings` | DefaultShadowSettings returns enabled shadows with automatic sizes. |
| `func (settings ShadowSettings) EffectiveResolution() int32` | EffectiveResolution is the shadow map's texels per side. |

### Viewpoint

Viewpoint is what decides which parts of the scene are drawn and how detailed they are.

```go
type Viewpoint struct {
	// Has unexported fields.
}
```

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
