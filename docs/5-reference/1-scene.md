# scene

```go
import "github.com/akrck02/go-renderer/scene"
```

Package scene is a retained, backend-independent description of a 3D scene:
nodes with transforms, meshes with per-vertex normals and colors, materials,
GPU instancing and the environment (sun, sky, fog, water).

Backends upload meshes once and draw them every frame. Simulations can change
geometry or instances later and mark them Dirty so the backend uploads them
again.

### Constants and variables

```go
const DefaultMaximumNotices = 5
```

DefaultMaximumNotices is how many notices are shown at once when the overlay does not say.

```go
const DefaultNoticeSeconds = 6.0
```

DefaultNoticeSeconds is how long a notice stays, in seconds, when the overlay does not say.

### Clouds

Clouds is a procedural cloud layer.

```go
type Clouds struct {
	Coverage float32 // share of the sky covered (0 = clear)
	Color    graphics.Vector
	Speed    float32         // drift in cloud-texture units per second
	Scale    float32         // size of the cloud pattern (larger = bigger clouds)
	Offset   graphics.Vector // how far the clouds have drifted (x, z), moved by a simulation
}
```

### Constellation

Constellation is a figure of stars joined by lines, fixed to the night sky.

```go
type Constellation struct {
	Name  string
	Stars []graphics.Vector // directions as seen at midnight (the sun at the lowest point of its circle)
	Lines [][2]int          // pairs of indices into Stars
	Color graphics.Vector
}
```

### Daylight

Daylight is the state of the sky at the current sun height: the colors to paint and the single directional light (sun or moon) that lights and shadows the scene. Backends compute it once per frame with Environment.Daylight.

```go
type Daylight struct {
	DayAmount      float32 // 1 in full day, 0 at night
	TwilightAmount float32 // 1 with the sun on the horizon, 0 far from it
	Zenith         graphics.Vector
	Horizon        graphics.Vector
	Fog            graphics.Vector
	SkyLight       graphics.Vector // ambient light from the sky
	GroundLight    graphics.Vector // ambient light bounced from the ground
	SunVisible     float32         // how much of the sun disc is above the horizon
	LightDirection graphics.Vector // towards the light that shades the scene
	LightColor     graphics.Vector
	MoonDirection  graphics.Vector
	StarRotation   graphics.Matrix // turns world directions into celestial directions
	Brightness     float32         // overall light level (1 day, low at night) for materials that are not lit
}
```

### DetailLevel

DetailLevel is a simpler version of a mesh. It replaces the finer versions when the model looks smaller on screen than ScreenSize, a fraction of the screen height (0.05 = a twentieth of it).

```go
type DetailLevel struct {
	Mesh       *Mesh
	ScreenSize float32
}
```

### Environment

Environment holds the lighting and atmosphere.

```go
type Environment struct {
	SunDirection  graphics.Vector // towards the sun, normalized
	SunColor      graphics.Vector
	SkyZenith     graphics.Vector
	SkyHorizon    graphics.Vector
	GroundAmbient graphics.Vector
	Ambient       float32
	FogColor      graphics.Vector
	FogNear       float32
	FogFar        float32 // 0 = no fog
	Water         *Water
	VerticalScale float32 // vertical exaggeration applied to the whole scene (0 = 1)
	Sky           *Sky    // sky over the day and night (nil = DefaultSky)
	Wind          Wind
	Snow          *Snow           // snow on the ground above a height (nil = none)
	FoliageTint   graphics.Vector // colour foliage materials take (alpha = how much; zero = none): the season's
}
```

| Function or method | |
|---|---|
| `func DefaultEnvironment() Environment` | DefaultEnvironment returns a daylight environment. |
| `func (environment Environment) Daylight() Daylight` | Daylight computes the sky and the lighting for the current sun direction. |
| `func (environment Environment) EffectiveSky() *Sky` | EffectiveSky returns the sky, or the default sky when unset. |
| `func (environment Environment) EffectiveVerticalScale() float32` | EffectiveVerticalScale returns the vertical scale, treating 0 as 1. |

### HeaderLine

HeaderLine is one line of the header in the top-left corner of the screen.

```go
type HeaderLine struct {
	Icon string
	Text string
	Tint graphics.Vector // colour of the icon (alpha 0 = the default text colour)
}
```

### Instances

Instances draws a mesh many times with a transform and a color per instance. After changing instances, a simulation either sets Dirty (instances were added or removed, or most of them changed: everything is prepared again) or lists the changed ones with MarkChanged (only those are uploaded). The backend clears both.

```go
type Instances struct {
	Transforms []graphics.Matrix
	Colors     []graphics.Vector
	Dirty      bool
	Changed    []int // indices changed since the last draw
	GPU        any
}
```

| Function or method | |
|---|---|
| `func (instances *Instances) MarkChanged(indices ...int)` | MarkChanged records that these instances changed their transform or color. |

### Label

Label is a small tag drawn above something in the world: an icon, a text and an optional badge under them. It keeps the same size on screen however far the camera is.

```go
type Label struct {
	// Node anchors the label to a node: it follows the node's world transform and hides with it.
	// When nil, the label stands at Point.
	Node  *Node
	Point graphics.Vector // world position of the anchor when there is no Node
	// Height lifts the label above its anchor, in world units (the top of a box, for example).
	Height float32
	Text   string
	Icon   string // name of a Lucide icon (https://lucide.dev), drawn before the text or alone
	Badge  string // a short text in a small pill under the label
	// Tint colours the icon and the badge, as a screen colour (see draw2d.ParseHex); alpha 0 = the
	// default text colour. The text itself stays dark on a light pill, so it always reads well.
	Tint   graphics.Vector
	Hidden bool
}
```

| Function or method | |
|---|---|
| `func (label *Label) IsEmpty() bool` | IsEmpty tells whether the label has nothing to draw. |

### LevelVariation

LevelVariation raises and lowers a water surface with one harmonic that differs from place to place: at each point the offset is InPhase·cos(Angle) + InQuadrature·sin(Angle), which is Amplitude·cos(Angle − Phase) with InPhase = Amplitude·cos(Phase) and InQuadrature = Amplitude·sin(Phase). A simulation moves Angle (tides advance it a full turn per period); the grids stay the same. Outside the area the values of the nearest edge apply.

```go
type LevelVariation struct {
	MinimumX, MinimumZ float32 // corner of the area (world units)
	MaximumX, MaximumZ float32 // opposite corner; grid values sit at cell centres
	Columns, Rows      int
	InPhase            []float32 // Rows×Columns, row by row from MinimumZ; heights in world units
	InQuadrature       []float32
	Angle              float32 // radians
	Dirty              bool    // the grids changed: backends upload them again
	GPU                any
}
```

| Function or method | |
|---|---|
| `func (variation *LevelVariation) OffsetAt(x, z float32) float32` | OffsetAt returns the height of the surface above the water level at a point. |

### Material

Material describes how a mesh is shaded.

```go
type Material struct {
	Name        string
	Kind        MaterialKind
	BaseColor   graphics.Vector // linear RGBA multiplied with vertex colors
	DoubleSided bool
	Transparent bool
	// Additive adds the surface's light to what is behind it instead of covering it: beams, sparks,
	// glows. It never darkens, and is drawn with the transparent surfaces (its alpha scales it).
	Additive  bool
	Sway      float32 // how much the wind bends the mesh (0 = rigid); for plants modelled with height 1
	DepthBias float32 // pulls the surface towards the camera by this share of its distance, so thin surfaces
	// lying on terrain (rivers, roads) stay visible over the terrain's simpler far versions; 0 = none
	Pattern      Pattern // procedural surface pattern of lit and glowing materials
	PatternScale float32 // size of one pattern cell (a stone, a course, a shingle) in world units
	Foliage      bool    // takes the environment's FoliageTint (leaves that change with the seasons)
	// Texture lays a picture over the surface (nil = none): its colour is BaseColor × the vertex
	// colour × the texture at (UV × TextureScale + TextureOffset), lit as before.
	Texture       *Texture
	TextureScale  [2]float32 // repeats of the texture for each unit of UV, along u and v; 0 = 1
	TextureOffset [2]float32 // moves the texture along u and v, in repeats
}
```

| Function or method | |
|---|---|
| `func (material *Material) EffectiveTextureScale() [2]float32` | EffectiveTextureScale is how many times the texture repeats for each unit of UV, along each side, with 0 meaning 1. |

### MaterialKind

MaterialKind selects the shading model.

```go
type MaterialKind int
```

```go
const (
	LitKind       MaterialKind = iota // diffuse sun + sky ambient + fog
	UnlitKind                         // flat color (+ fog)
	WaterKind                         // animated water surface
	WaterfallKind                     // animated, translucent falling water
	GlowKind                          // lit by day; shines with its own colour in the dark (glowing flowers, lamps)
)
```

### Mesh

Mesh is triangle geometry. Positions and normals are xyz triples, colors rgba quads (linear), UVs (where on a texture each vertex lies) uv pairs. Indices are optional: without them every three vertices form a triangle.

```go
type Mesh struct {
	Name      string
	Positions []float32
	Normals   []float32
	Colors    []float32
	UVs       []float32 // optional; without them a texture is projected from the positions (SurfaceUVs)
	Indices   []uint32
	Material  *Material
	Details   []DetailLevel // simpler versions for when the mesh looks small, from finer to coarser
	Ground    bool          // walkable surface (used by GroundHeight)
	Dirty     bool          // geometry changed: backends upload it again
	GPU       any           // backend handle
}
```

| Function or method | |
|---|---|
| `func (mesh *Mesh) Bounds() (minimum, maximum graphics.Vector)` | Bounds returns the box around the mesh positions in its own coordinates. |
| `func (mesh *Mesh) ComputeNormals()` | ComputeNormals fills smooth vertex normals from the triangles (weighted by triangle area). |
| `func (mesh *Mesh) ForEachTriangle(visit func(first, second, third uint32))` | ForEachTriangle calls visit with the vertex indices of every triangle. |
| `func (mesh *Mesh) HasUVs() bool` | HasUVs tells whether the mesh carries its own UVs, two for each vertex. |
| `func (mesh *Mesh) Level(index int) *Mesh` | Level returns a version of the mesh: 0 is the mesh itself, 1 the first detail level, and so on. |
| `func (mesh *Mesh) LevelCount() int` | LevelCount returns how many versions the mesh has, itself included. |
| `func (mesh *Mesh) LevelForScreenSize(screenSize float32) int` | LevelForScreenSize returns the index of the version to draw for a model of this apparent size: the coarsest level whose ScreenSize is still larger than it. |
| `func (mesh *Mesh) SurfaceUVs() []float32` | SurfaceUVs returns the mesh's UVs, or, when it has none, UVs projected from where its vertices are, one unit of the model a repeat: x and z on what faces up or down, along the wall and up on what faces sideways (its facing taken from its normals; up when it has none). |
| `func (mesh *Mesh) Top() float32` | Top returns the highest y of the mesh's vertices (0 for an empty mesh), to lift a label above it. |
| `func (mesh *Mesh) Vertex(index uint32) graphics.Vector` | Vertex returns the position of a vertex. |
| `func (mesh *Mesh) VertexCount() int` | VertexCount returns the number of vertices. |

### Moon

Moon is a disc with phases that also lights the scene at night.

```go
type Moon struct {
	Direction      graphics.Vector // towards the moon; zero = opposite the sun
	AngularSize    float32         // apparent diameter in radians
	Color          graphics.Vector
	Phase          float32 // 0 new, 0.25 first quarter, 0.5 full, 0.75 last quarter
	LightIntensity float32 // moonlight on the scene at full moon, relative to the sun
}
```

### Node

Node places a mesh (optionally instanced) in the world; children inherit the transform.

```go
type Node struct {
	Name      string
	Transform graphics.Matrix
	Mesh      *Mesh
	Instances *Instances
	Children  []*Node
	Hidden    bool
	// Moving says the node's transform changes often, turning as well as moving: a ship, a
	// vehicle. Backends that work a mesh's light out once for each way it is turned (nintendo3ds, on the
	// 3DS) light a moving node on the GPU every frame instead.
	Moving bool
	Tags   map[string]any // free data from the file (glTF extras) for the application
}
```

### Notice

Notice is a short message shown under the header for a few seconds, newest first.

```go
type Notice struct {
	Icon  string
	Text  string
	Tint  graphics.Vector // colour of the icon (alpha 0 = the default text colour)
	Added float64         // the overlay's Clock when the notice was added
}
```

### Overlay

Overlay is what is drawn over the 3D view, in screen space: labels that float above things in the world, a header pinned to the top-left corner and short notices under it, and a free drawing on top of everything (Drawing). The overlay itself handles no input: an interface library fills Drawing with its panels and buttons and reads the mouse and keyboard on its own. A program changes it from the viewer's update function, which runs on the same goroutine as the drawing, so no locking is needed. The zero value is an empty overlay ready to use.

```go
type Overlay struct {
	Labels  []*Label
	Header  []HeaderLine
	Notices []Notice
	// Drawing is drawn last, over the labels, header and notices: flat shapes, text and icons in
	// screen space, filled every frame by a program or an interface library. nil = nothing.
	Drawing *draw2d.List
	// Clock counts the seconds the overlay has been shown; the viewer advances it at every step.
	// Notices remember the clock reading when they were added, so they fade on the viewer's time.
	Clock float64
	// NoticeDuration is how long a notice stays, in seconds, fading at the end; 0 = DefaultNoticeSeconds.
	NoticeDuration float64
	// MaximumNotices is how many notices are shown at once, the newest; 0 = DefaultMaximumNotices.
	MaximumNotices int
}
```

| Function or method | |
|---|---|
| `func (overlay *Overlay) AddLabel(label *Label)` | AddLabel shows a label; the caller keeps the pointer to change or remove it later. |
| `func (overlay *Overlay) AddNotice(icon, text string, tint graphics.Vector)` | AddNotice shows a message now; it fades after the notice duration. |
| `func (overlay *Overlay) Advance(seconds float64)` | Advance moves the overlay's clock on and forgets the notices that have faded out. |
| `func (overlay *Overlay) IsEmpty() bool` | IsEmpty tells whether there is nothing to draw on the overlay. |
| `func (overlay *Overlay) RemoveLabel(label *Label)` | RemoveLabel stops showing a label. |
| `func (overlay *Overlay) VisibleNotices() []VisibleNotice` | VisibleNotices returns the notices to draw now, newest first, at most MaximumNotices. |

### Pattern

Pattern is a procedural surface pattern that modulates a lit material's colour, computed from the world position (no texture): stones, blocks, shingles… It fades out where its cells get too small on screen.

```go
type Pattern int
```

```go
const (
	PatternNone     Pattern = iota
	PatternCobbles          // irregular paving stones with joints (horizontal surfaces)
	PatternMasonry          // courses of stone blocks with mortar (walls)
	PatternPlaster          // mottled render
	PatternShingles         // overlapping rows of shingles or slates (roofs)
	PatternPlanks           // wooden planks
	PatternGrass            // mottled grass
)
```

The patterns; PatternNone leaves the material's colour as it is.

| Function or method | |
|---|---|
| `func PatternNamed(name string) Pattern` | PatternNamed returns the pattern with a name (cobbles, masonry, plaster, shingles, planks, grass); PatternNone for any other name. |

### PointLight

PointLight is a light that shines from a point all around, as a lamp: full within Inner of it and fading smoothly to nothing at Outer (world units), the same fade as draw2d's point lights. It lights the lit materials it reaches, besides the sun and the sky; each draw takes the few that reach it (see sceneplan.MaxLightsPerDraw).

```go
type PointLight struct {
	Position graphics.Vector // in the world, before the vertical scale
	// Color is linear, like the materials'; its alpha is the intensity (1 a plain lamp, up to a few).
	Color        graphics.Vector
	Inner, Outer float32
}
```

### Scene

Scene is the root of a scene.

```go
type Scene struct {
	Nodes       []*Node
	Environment Environment
	Tags        map[string]any
	Overlay     Overlay // labels, header and notices drawn over the view
	// Lights are the point lights besides the sun; a program changes them between frames as its
	// lamps move.
	Lights []PointLight

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func New() *Scene` | New returns an empty scene with the default environment. |
| `func (scene *Scene) Bounds() (minimum, maximum graphics.Vector)` | Bounds returns the world-space bounding box of all meshes (instances counted by their origin). |
| `func (scene *Scene) GroundHeight(x, z float32) (float32, bool)` | GroundHeight returns the highest walkable surface at (x, z), before vertical scaling. |
| `func (scene *Scene) InvalidateGround()` | InvalidateGround drops the ground index; call it after changing walkable meshes. |
| `func (scene *Scene) Walk(visit func(node *Node, world graphics.Matrix))` | Walk visits every visible node with its world transform. |
| `func (scene *Scene) WorldTransforms(wanted map[*Node]bool) map[*Node]graphics.Matrix` | WorldTransforms returns the world transform of every visible node, for anchoring labels. |

### Sky

Sky describes what is drawn behind the scene over a whole day and how it lights the scene at night. The height of the sun decides the time of day: above the horizon it is day, near it twilight, below it night. By default the sky is procedural: the day gradient of the environment, a twilight glow, and a night with stars, a moon and constellations; clouds float over both. Images can replace the day or the night (a skybox): an equirectangular panorama or six cube faces.

```go
type Sky struct {
	NightZenith   graphics.Vector // sky color straight up at night (linear)
	NightHorizon  graphics.Vector // sky color at the horizon at night
	NightAmbient  graphics.Vector // sky light that reaches the scene at night
	TwilightColor graphics.Vector // horizon glow at dawn and dusk
	CelestialPole graphics.Vector // axis the night sky turns around (towards the visible pole)

	Stars          Stars
	Moon           *Moon // nil = no moon
	Clouds         Clouds
	Constellations []Constellation

	DayImage   *SkyImage // replaces the procedural day sky
	NightImage *SkyImage // replaces the procedural night sky; it turns with the stars
}
```

| Function or method | |
|---|---|
| `func DefaultSky() *Sky` | DefaultSky returns an Earth-like sky: stars, a full moon opposite the sun and no clouds. |

### SkyImage

SkyImage is a skybox: an equirectangular panorama or six cube faces (+X, -X, +Y, -Y, +Z, -Z).

```go
type SkyImage struct {
	Equirectangular image.Image
	Faces           [6]image.Image
	GPU             any // backend handle
}
```

| Function or method | |
|---|---|
| `func (skyImage *SkyImage) IsCube() bool` | IsCube reports whether the image is made of six faces. |

### Snow

Snow covers the lit surfaces that face up above a height, blending over a band.

```go
type Snow struct {
	Level float32         // world units, before vertical scaling
	Blend float32         // world units over which the snow fades in
	Color graphics.Vector // linear
}
```

### Stars

Stars is a procedural star field.

```go
type Stars struct {
	Density           float32 // share of the sky cells that hold a star (0 = no stars, 1 = one per cell)
	Brightness        float32 // brightness multiplier
	Twinkle           float32 // amount of twinkling (0 = steady)
	DaytimeVisibility float32 // how visible the stars stay in full daylight (0 = only at night)
}
```

### Texture

Texture is a picture a material lays over its surfaces: a tiling detail (panels, stones, a weave) or a whole picture. A program makes one, keeps it, and can give it to any number of materials: backends upload its pixels once, and again only after Picture.Changed().

```go
type Texture struct {
	Picture *draw2d.Image // its pixels, the same kind of picture 2D lists draw
	Wrap    Wrap          // what the surface shows past the picture's edges
	// Sharp shows the nearest pixel, as pixel art wants, instead of blending between pixels (and
	// between smaller copies of the picture far away).
	Sharp  bool
	Format TextureFormat // how the 3DS keeps it in its memory; other targets keep RGBA8
}
```

### TextureFormat

TextureFormat is how the 3DS keeps a texture's pixels: in full, in half the memory, or in an eighth. Other targets keep every texture in RGBA8.

```go
type TextureFormat int
```

```go
const (
	RGBA8  TextureFormat = iota // 32 bits a pixel: every colour and opacity
	RGB565                      // 16 bits a pixel: no opacity, colours in steps of 1/32 (green 1/64)
	ETC1                        // 4 bits a pixel, compressed in blocks of 4×4: no opacity, small loss in detail
)
```

The texture formats.

### VisibleNotice

VisibleNotice is a notice as it should be drawn now.

```go
type VisibleNotice struct {
	Notice
	Opacity float32 // 1 while fresh, falling to 0 as the notice ends
}
```

### Water

Water is an infinite animated water plane.

```go
type Water struct {
	Level      float32
	Shallow    graphics.Vector
	Deep       graphics.Vector
	Size       float32         // extent of the plane (world units)
	WaveLength float32         // length of the main wave (world units); 0 = Size / 2000
	FloorDepth float32         // depth of an opaque floor drawn under the water; 0 = Size / 500
	ColorDepth float32         // water depth at which the color is about two thirds of the way from Shallow to Deep; 0 = FloorDepth / 8
	Variation  *LevelVariation // raises and lowers the surface over an area (tides, for example); nil = flat
	ShoreFade  float32         // depth over which the sea fades into the shore; 0 = ColorDepth / 20
}
```

| Function or method | |
|---|---|
| `func (water *Water) EffectiveColorDepth() float32` | EffectiveColorDepth returns the color depth, deriving it from the floor depth when unset. |
| `func (water *Water) EffectiveFloorDepth() float32` | EffectiveFloorDepth returns the floor depth, deriving it from the size when unset. |
| `func (water *Water) EffectiveShoreFade() float32` | EffectiveShoreFade returns the shore fade depth, deriving it from the color depth when unset. |
| `func (water *Water) EffectiveWaveLength() float32` | EffectiveWaveLength returns the wave length, deriving it from the size when unset. |
| `func (water *Water) LevelAt(x, z float32) float32` | LevelAt returns the height of the water surface at a point. |

### Wind

Wind moves clouds and bends plants (materials with Sway).

```go
type Wind struct {
	Direction graphics.Vector // horizontal direction the wind blows towards
	Strength  float32         // 0 calm, 1 strong
}
```

### Wrap

Wrap is what a surface shows where its UVs leave the picture (below 0 or above 1).

```go
type Wrap int
```

```go
const (
	Repeat Wrap = iota // the picture again, as tiles
	Clamp              // the colour at the picture's edge, stretched
	Mirror             // the picture again, flipped every other time, so its edges always meet
)
```

The ways a picture wraps.

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
