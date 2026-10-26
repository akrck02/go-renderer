# camera

```go
import "github.com/akrck02/go-renderer/camera"
```

Package camera is where a scene is seen from (a Lens) and the controllers that
move it.

Controllers are independent of the graphics backend: they read an input.State
and write a Lens. A GroundFunc lets them follow a terrain without knowing about
it.

### Constants and variables

```go
const (
	DefaultFieldOfViewRadians = math.Pi / 4
	DefaultNear               = 0.1
	DefaultFar                = 1000
)
```

Default projection values, used where a Lens's fields are zero.

```go
const (
	DefaultRigWalkSpeedMetres = 1.5         // a calm walk
	DefaultRigTurnRadians     = math.Pi / 6 // twelve steps make a whole turn
)
```

Defaults of a RigWalk, used where its fields are zero.

### Controller

Controller moves a camera from input.

```go
type Controller interface {
	Update(state *input.State, seconds float64, camera *Lens)
}
```

### GroundFunc

GroundFunc returns the ground height at (x, z) in world units, and false when there is no ground there.

```go
type GroundFunc func(x, z float32) (float32, bool)
```

### Lens

Lens is where a scene is seen from: a position looking at a target, with up saying which way is up, and how the view is projected onto the screen.

```go
type Lens struct {
	Position graphics.Vector
	Target   graphics.Vector
	Up       graphics.Vector
	// FieldOfViewRadians is the vertical field of view; 0 is DefaultFieldOfViewRadians.
	FieldOfViewRadians float32
	// Near and Far are the distances of the clipping planes; 0 is DefaultNear and DefaultFar.
	Near float32
	Far  float32
	// OrthographicHeight, when above 0, makes the camera orthographic: no perspective, and this
	// many world units fit between the bottom and the top of the screen. Maps and isometric views
	// use it; 0 keeps the perspective of FieldOfViewRadians.
	OrthographicHeight float32
}
```

| Function or method | |
|---|---|
| `func CoveringLens(rig Rig, eyes []graphics.Pose, fields []graphics.FieldOfView, near, far float32) (lens Lens, aspect float32)` | CoveringLens is one symmetric camera that sees everything the eyes see, for planning a frame (culling, levels of detail, the order of translucent surfaces) once for all of them; aspect is the width over the height of its picture. |
| `func (camera Lens) IsOrthographic() bool` | IsOrthographic tells whether the camera has no perspective. |
| `func (camera Lens) Projection() (fieldOfViewRadians, near, far float32)` | Projection returns the camera's field of view (radians) and clipping planes, with the defaults where its fields are zero. |
| `func (camera Lens) ProjectionMatrix(aspect float64) graphics.Matrix` | ProjectionMatrix returns the camera's projection for OpenGL, perspective or orthographic, for a screen of this width over height. |

### Map

Map looks at the ground like a map, without perspective: from straight above, with north up, or isometric, from 35.26° up and at 45° to the axes, like the drawings of a town in an atlas. Any drag moves the map with the cursor, scroll and +/- zoom, arrows move it too, and Q and E turn it a quarter.

```go
type Map struct {
	Target        graphics.Vector
	Span          float64 // world units from the bottom to the top of the screen
	Isometric     bool
	Turns         int     // quarter turns: 0 has north up in the top view
	Distance      float64 // how far the camera stands from the target, so the whole scene is in front of it
	MinimumSpan   float64
	MaximumSpan   float64
	ZoomSpeed     float64    // fraction of the span per scroll step
	TargetLimits  [4]float32 // optional: minimum x, maximum x, minimum z, maximum z (all zero = none)
	Ground        GroundFunc
	VerticalScale float32 // scale applied to ground heights (vertical exaggeration); 0 = 1
}
```

| Function or method | |
|---|---|
| `func NewMap(target graphics.Vector, sceneSize float32) *Map` | NewMap returns a map of a scene of the given size (in world units), showing all of it. |
| `func (view *Map) Update(state *input.State, seconds float64, camera *Lens)` | Update applies this frame's input and places the camera. |

### Orbit

Orbit rotates around a target: left drag rotates, right drag (or shift + left drag) pans, scroll zooms. Arrow keys rotate and +/- zoom from the keyboard.

```go
type Orbit struct {
	Target          graphics.Vector
	Distance        float64
	Yaw             float64 // radians around the vertical axis
	Pitch           float64 // radians; 0 looks from the horizon, pi/2 from straight above
	MinimumDistance float64
	MaximumDistance float64
	RotationSpeed   float64 // radians per cursor unit
	ZoomSpeed       float64 // fraction of the distance per scroll step
	Ground          GroundFunc
	GroundClearance float32    // minimum camera height above the ground
	TargetLimits    [4]float32 // optional: minimum x, maximum x, minimum z, maximum z (all zero = none)
	VerticalScale   float32    // scale applied to ground heights (vertical exaggeration); 0 = 1
}
```

| Function or method | |
|---|---|
| `func NewOrbit(target graphics.Vector, sceneSize float32) *Orbit` | NewOrbit returns an orbit controller with sensible defaults for a scene of the given size (in world units). |
| `func (orbit *Orbit) Gesture(gesture input.Gesture, windowWidth, windowHeight int)` | Gesture applies what two fingers did (input.TwoFingers), as a map does: the ground that was between the fingers stays between them while they spread (closer), turn (the view turns with them) and move together. |
| `func (orbit *Orbit) Update(state *input.State, seconds float64, camera *Lens)` | Update applies this frame's input and places the camera. |

### Ray

Ray is a half-line in the scene as it is drawn: from Origin along Direction (of length 1). Heights are drawn heights, already multiplied by the scene's vertical exaggeration.

```go
type Ray struct {
	Origin, Direction graphics.Vector
}
```

| Function or method | |
|---|---|
| `func RayThrough(view Lens, width, height int, x, y float64) (Ray, bool)` | RayThrough returns the ray from the camera through a point of the window: x and y in window pixels from the top-left corner of a window width × height pixels (where the pointer is). |
| `func (ray Ray) At(distance float32) graphics.Vector` | At returns the point at a distance along the ray. |
| `func (ray Ray) PointAtHeight(height float32) (graphics.Vector, bool)` | PointAtHeight returns where the ray meets the horizontal plane at a drawn height: a flat floor, a table top. |
| `func (ray Ray) PointOnGround(ground func(x, z float32) (float32, bool), verticalScale, lowest, highest float32) (graphics.Vector, bool)` | PointOnGround returns the first point where the ray meets a relief: ground gives the height of the ground at a point (in true heights, as scene.Scene.GroundHeight does, false where there is none), verticalScale is the exaggeration it is drawn with, and lowest and highest bound the ground's true heights (the scene's bounds), so the search only covers where it can be. |

### Rig

Rig places a headset's tracked space in the world. A headset reports where the eyes, the head and the hands are in its own space, in metres, from a point on the floor of the room; the rig says where that point stands in the scene, which way the room is turned, and how many scene units a metre is. Walking with a stick moves the rig; walking in the room moves the poses.

```go
type Rig struct {
	// Position is where the room's origin (on its floor) is in the world.
	Position graphics.Vector
	// YawRadians turns the room around the world's up (y) axis, counter-clockwise seen from above.
	YawRadians float32
	// Scale is scene units per metre (0 = 1). Above 1 the player is a giant: a step crosses more
	// of the scene and the eyes are wider apart in it, so the world looks smaller.
	Scale float32
}
```

| Function or method | |
|---|---|
| `func (rig Rig) EyePosition(eye graphics.Pose) graphics.Vector` | EyePosition is where an eye is in the world: what lighting, fog and water reflections are worked out from. |
| `func (rig Rig) EyeView(eye graphics.Pose) graphics.Matrix` | EyeView is the view matrix of an eye (a pose in the headset's space): from the world to the eye. |
| `func (rig Rig) Matrix() graphics.Matrix` | Matrix takes the headset's space (metres) to the world: scale, turn, then move. |
| `func (rig Rig) Place(pose graphics.Pose) graphics.Matrix` | Place is the model matrix of something tracked (a hand, a controller) in the world. |

### RigWalk

RigWalk moves a headset's rig with a pad's sticks: the left stick walks the way the head looks (flat on the ground), the right stick turns in steps around the head. The head itself is never moved by the program: a view that turns smoothly without the head turning makes many people sick, a step does not.

```go
type RigWalk struct {
	// SpeedMetres is how far a full stick walks in a second, in the room's metres (0 =
	// DefaultRigWalkSpeedMetres); a giant (Rig.Scale above 1) covers as many of its own metres.
	SpeedMetres float32
	// TurnRadians is one step of a turn (0 = DefaultRigTurnRadians).
	TurnRadians float32
	// Ground, when set, keeps the room's floor on the ground under the head.
	Ground GroundFunc

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (walk *RigWalk) Update(state *input.State, seconds float64, head graphics.Pose, rig *Rig)` | Update walks and turns the rig from the input's sticks, seconds after the last update. |

### Walk

Walk is a first-person controller that stays on the ground: WASD or arrows move, the mouse looks around (while a button is held, or always when the cursor is locked), shift runs.

```go
type Walk struct {
	X, Z          float32
	Yaw           float64 // radians; 0 looks towards -Z
	Pitch         float64 // radians; positive looks up
	EyeHeight     float32
	Speed         float32 // world units per second
	RunFactor     float32
	LookSpeed     float64 // radians per cursor unit
	Ground        GroundFunc
	Blocked       func(x, z float32) bool // optional: true where the walker cannot go (water, walls)
	VerticalScale float32                 // 0 = 1
}
```

| Function or method | |
|---|---|
| `func NewWalk(x, z, eyeHeight, speed float32) *Walk` | NewWalk returns a walker at (x, z) with defaults for the given eye height and walking speed. |
| `func (walk *Walk) Steer(forward, sideways float32, seconds float64)` | Steer walks as a stick on a phone's screen says: forward and sideways from -1 to 1 (right and ahead positive), at the walking speed. |
| `func (walk *Walk) Turn(deltaX, deltaY float64)` | Turn looks around by a finger's movement, in layout pixels: right turns right, down looks down. |
| `func (walk *Walk) Update(state *input.State, seconds float64, camera *Lens)` | Update applies this frame's input and places the camera at eye height. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
