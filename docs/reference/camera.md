# camera

```go
import "github.com/akrck02/go-renderer/camera"
```

Package camera provides camera controllers that turn input into camera
placement.

Controllers are independent of the graphics backend: they read an input.State
and write a models.Camera. A GroundFunc lets them follow a terrain without
knowing about it.

### Controller

Controller moves a camera from input.

```go
type Controller interface {
	Update(state *input.State, seconds float64, camera *models.Camera)
}
```

### GroundFunc

GroundFunc returns the ground height at (x, z) in world units, and false when there is no ground there.

```go
type GroundFunc func(x, z float32) (float32, bool)
```

### Orbit

Orbit rotates around a target: left drag rotates, right drag (or shift + left drag) pans, scroll zooms. Arrow keys rotate and +/- zoom from the keyboard.

```go
type Orbit struct {
	Target          graphics.Vec4
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
| `func NewOrbit(target graphics.Vec4, sceneSize float64) *Orbit` | NewOrbit returns an orbit controller with sensible defaults for a scene of the given size. |
| `func (orbit *Orbit) Gesture(gesture input.Gesture, windowWidth, windowHeight int)` | Gesture applies what two fingers did (input.TwoFingers), as a map does: the ground that was between the fingers stays between them while they spread (closer), turn (the view turns with them) and move together. |
| `func (orbit *Orbit) Update(state *input.State, seconds float64, camera *models.Camera)` | Update applies this frame's input and places the camera. |

### Plan

Plan looks at the ground like a map, without perspective: from straight above, with north up, or isometric, from 35.26° up and at 45° to the axes, like the drawings of a town in an atlas. Any drag moves the map with the cursor, scroll and +/- zoom, arrows move it too, and Q and E turn it a quarter.

```go
type Plan struct {
	Target        graphics.Vec4
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
| `func NewPlan(target graphics.Vec4, sceneSize float64) *Plan` | NewPlan returns a plan view of a scene of the given size, showing all of it. |
| `func (plan *Plan) Update(state *input.State, seconds float64, camera *models.Camera)` | Update applies this frame's input and places the camera. |

### Ray

Ray is a half-line in the scene as it is drawn: from Origin along Direction (of length 1). Heights are drawn heights, already multiplied by the scene's vertical exaggeration.

```go
type Ray struct {
	Origin, Direction graphics.Vec4
}
```

| Function or method | |
|---|---|
| `func RayThrough(view models.Camera, width, height int, x, y float64) (Ray, bool)` | RayThrough returns the ray from the camera through a point of the window: x and y in window pixels from the top-left corner of a window width × height pixels (where the pointer is). |
| `func (ray Ray) At(distance float32) graphics.Vec4` | At returns the point at a distance along the ray. |
| `func (ray Ray) PointAtHeight(height float32) (graphics.Vec4, bool)` | PointAtHeight returns where the ray meets the horizontal plane at a drawn height: a flat floor, a table top. |
| `func (ray Ray) PointOnGround(ground func(x, z float32) (float32, bool), verticalScale, lowest, highest float32) (graphics.Vec4, bool)` | PointOnGround returns the first point where the ray meets a relief: ground gives the height of the ground at a point (in true heights, as scene.Scene.GroundHeight does, false where there is none), verticalScale is the exaggeration it is drawn with, and lowest and highest bound the ground's true heights (the scene's bounds), so the search only covers where it can be. |

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
| `func (walk *Walk) Update(state *input.State, seconds float64, camera *models.Camera)` | Update applies this frame's input and places the camera at eye height. |

---

[← Reference](README.md) · generated from the doc comments by `generate.py`
