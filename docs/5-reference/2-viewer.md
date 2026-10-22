# viewer

```go
import "github.com/akrck02/go-renderer/viewer"
```

Package viewer is an interactive explorer for a scene.Scene: orbit and walking
cameras, sun, fog, shadows and vertical exaggeration controls, headless captures
and benchmarks. Programs pass an optional update function, called at every
fixed step, to change the scene while it is shown (a simulation, for example);
the viewer itself knows nothing about what changes it.

Controls: left drag rotates, right drag pans, scroll zooms (orbit camera).
Tab switches to walking on the ground: WASD or arrows move, mouse looks, shift
runs, [ and ] change the walking speed. J/L turn the sun, I/K raise or lower it
(below the horizon it is night), N jumps between day and night, Z/X change the
vertical exaggeration, F toggles fog, O toggles shadows. M switches to a map:
the ground from straight above without perspective, then an isometric view,
then back; on the map any drag moves it, scroll zooms and Q/E turn it a quarter.

### Functions

| Function | |
|---|---|
| `func Edit(world *scene.Scene, options Options, title string, edit EditFunc) error` | Edit shows the scene like Run, for a program that edits it (a level editor, a floor plan designer). |
| `func Run(world *scene.Scene, options Options, title string, update UpdateFunc, loadingTime time.Duration) error` | Run shows the scene until the window closes (or, for a capture or a benchmark, until it is done). |

### Constants and variables

```go
const Usage = "[-width W] [-height H] [-capture frame.png] [-walk] [-no-shadows] [-night] [-sun-elevation D] [-sun-azimuth D] [-zoom F] [-benchmark N] [-no-vsync] [-look-at x,z] [-heading D] [-pitch D] [-eye-height H] [-walk-speed S] [-time S] [-backend B] [-continuous]"
```

Usage lists the viewer flags for a usage line.

```go
var (
	// InvalidPoint is a point flag (-look-at) not written as two numbers, "x,z".
	InvalidPoint = failure.Define("viewer.invalid-point", "a point is written as two numbers, x,z")
	// UnknownBackend is a -backend that is not automatic, metal or opengl.
	UnknownBackend = failure.Define("viewer.unknown-backend", "the backend is automatic, metal or opengl")
)
```

Codes of the errors the viewer's flags return.

### Backend

Backend is the graphics interface a session draws through.

```go
type Backend string
```

```go
const (
	Automatic Backend = ""       // Metal on macOS, falling back to OpenGL if it cannot start; OpenGL elsewhere
	Metal     Backend = "metal"  // only on macOS
	OpenGL    Backend = "opengl" // OpenGL 4.1
)
```

| Function or method | |
|---|---|
| `func (backend *Backend) Set(text string) error` | Set reads a backend from a flag: automatic, metal or opengl. |
| `func (backend *Backend) String() string` | String is the backend as its flag writes it: automatic, metal or opengl. |

### EditFunc

EditFunc changes the scene in answer to the user, at every step the viewer runs, before the viewer's own controls; view tells where the camera that drew the last frame was, so the program can find what the pointer is on (View.PointOnPlane, View.PointOnGround).

```go
type EditFunc func(world *scene.Scene, state *input.State, step float64, view View)
```

### Options

Options configure a viewing session.

```go
type Options struct {
	Width, Height int
	CapturePath   string // render one frame to this PNG (hidden window) instead of exploring
	StartWalking  bool
	StartMap      bool // start on the map: the ground from straight above, without perspective
	Isometric     bool // start on the map's isometric view
	NoShadows     bool
	Night         bool
	SunElevation  float64     // degrees; NaN keeps the scene's sun
	SunAzimuth    float64     // degrees clockwise from north; NaN keeps the scene's sun
	Zoom          float64     // initial orbit distance as a fraction of the default (0 = 1)
	Benchmark     int         // frames to measure before printing times and exiting; 0 = explore
	NoVsync       bool        // draw as fast as possible instead of waiting for the display
	LookAt        *[2]float32 // point (x, z) the view starts centred on, and where walking starts; nil = the scene centre
	Heading       float64     // walking: degrees clockwise from north (-z) the walker starts facing; NaN keeps the default
	Pitch         float64     // walking: degrees the walker starts looking up (negative = down)
	EyeHeight     float64     // walking: eye height in scene units; 0 keeps the default (a share of the scene size)
	WalkSpeed     float64     // walking: speed in scene units per second; 0 keeps the default (a share of the scene size)
	// Continuous draws every frame, so water, clouds and plants in the wind keep moving while
	// nothing else happens. By default the viewer draws only when something changes (input, the
	// camera, the sun, notices), unless it runs a program's update function (a simulation).
	Continuous bool
	// Backend is what draws: Automatic (Metal on macOS, OpenGL elsewhere), Metal or OpenGL.
	Backend Backend
	// Time, when not NaN, draws every frame as if this many seconds had passed (water, clouds and
	// waterfalls stand still): captures that must come out the same, to compare them.
	Time float64
}
```

| Function or method | |
|---|---|
| `func DefaultOptions() Options` | DefaultOptions returns a 1280×800 window with the scene's own sun. |
| `func (options *Options) RegisterFlags(flags *flag.FlagSet)` | RegisterFlags declares the viewer options as command-line flags, starting from the defaults. |

### UpdateFunc

UpdateFunc changes the scene at every fixed step (seconds), before the viewer's own controls; state is the keyboard and mouse, for programs that add their own keys.

```go
type UpdateFunc func(world *scene.Scene, state *input.State, step float64)
```

### View

View is what an editing program knows about the view at a step: the camera that drew the frame the user sees, and the window's size. The pointer's position (input.State.CursorX and CursorY) is in the same window pixels.

```go
type View struct {
	Camera        models.Camera
	Width, Height int // window pixels

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (view View) FrameGround(minimumX, minimumZ, maximumX, maximumZ float32)` | FrameGround asks the cameras to show a rectangle of ground whole (x and z from minimum to maximum): they aim at its middle and the map's span fits it in the window. |
| `func (view View) PointOnGround(x, y float64) (graphics.Vec4, bool)` | PointOnGround returns the point of the window (x, y in window pixels) on the scene's walkable ground (meshes marked Ground), with its true height. |
| `func (view View) PointOnPlane(x, y float64, height float32) (graphics.Vec4, bool)` | PointOnPlane returns the point of the window (x, y in window pixels) on the horizontal plane at a true height: a flat floor. |
| `func (view View) Ray(x, y float64) (camera.Ray, bool)` | Ray returns the ray from the camera through a point of the window, in drawn heights. |
| `func (view View) Span() float64` | Span is how much ground the map shows from the bottom of the window to its top, in world units: a program shows its zoom from it. |
| `func (view View) ZoomBy(factor float64)` | ZoomBy asks the cameras to come nearer by a factor (2 shows half as much, 0.5 twice as much), keeping what they aim at, as a zoom button does. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
