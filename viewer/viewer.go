// Package viewer is an interactive explorer for a scene.Scene: orbit and walking cameras, sun,
// fog, shadows and vertical exaggeration controls, headless captures and benchmarks. Programs pass
// an optional update function, called at every fixed step, to change the scene while it is shown
// (a simulation, for example); the viewer itself knows nothing about what changes it.
//
// Controls: left drag rotates, right drag pans, scroll zooms (orbit camera). Tab switches to
// walking on the ground: WASD or arrows move, mouse looks, shift runs, [ and ] change the walking
// speed. J/L turn the sun, I/K raise or lower it (below the horizon it is night), N jumps between
// day and night, Z/X change the vertical exaggeration, F toggles fog, O toggles shadows.
package viewer

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl"
	"github.com/akrck02/go-renderer/scene"
)

// viewer holds the state of an exploration session.
type viewer struct {
	world          *scene.Scene
	renderer       *opengl.SceneRenderer
	orbit          *camera.Orbit
	walk           *camera.Walk
	walking        bool
	sceneSize      float32
	sunAzimuth     float64
	sunElevation   float64
	started        time.Time
	baseFogNear    float32
	baseFogFar     float32
	savedScale     float32 // vertical exaggeration to restore when leaving walk mode
	shadowsEnabled bool
	benchmark      *benchmark
	update         UpdateFunc
	dayElevation   float64 // sun elevation to return to when leaving the night
}

// init keeps the main goroutine on the main thread, where the window system must run; it has to
// happen before main starts, since the goroutine may move to another thread later.
func init() { runtime.LockOSThread() }

// UpdateFunc changes the scene at every fixed step (seconds), before the viewer's own controls;
// state is the keyboard and mouse, for programs that add their own keys.
type UpdateFunc func(world *scene.Scene, state *input.State, step float64)

// Run shows the scene until the window closes (or, for a capture or a benchmark, until it is
// done). It must be called from the main goroutine. loadingTime only appears in benchmarks.
func Run(world *scene.Scene, options Options, title string, update UpdateFunc, loadingTime time.Duration) error {
	session := newViewer(world, false)
	if options.LookAt != nil {
		session.orbit.Target[0], session.orbit.Target[2] = options.LookAt[0], options.LookAt[1]
	}
	if options.StartWalking {
		session.toggleWalking()
		session.placeWalker(options)
	}
	session.update = update
	session.shadowsEnabled = !options.NoShadows
	session.orbit.Distance *= zoomOrOne(options.Zoom)
	session.placeSun(options.SunAzimuth, options.SunElevation)
	if options.Night {
		session.toggleNight()
	}
	if options.Benchmark > 0 {
		session.benchmark = newBenchmark(options.Benchmark, loadingTime)
	}
	return (&opengl.OpenGL{}).StartLoop(newApplication(session, options, title))
}

func zoomOrOne(zoom float64) float64 {
	if zoom <= 0 {
		return 1
	}
	return zoom
}

func newApplication(session *viewer, options Options, title string) *models.Application {
	app := &models.Application{
		Type:        models.WindowApplication,
		Title:       title,
		Version:     "",
		Width:       options.Width,
		Height:      options.Height,
		Input:       input.NewState(),
		ModelMatrix: graphics.Identity(),
		Renderer:    &opengl.OpenGLRenderer{},
		Update:      session.advance,
		Draw:        session.draw,
	}
	if options.CapturePath != "" {
		app.Type, app.CapturePath = models.HeadlessApplication, options.CapturePath
	}
	// without waiting for the display the GPU keeps its speed, so measurements are steady
	app.DisableVsync = options.Benchmark > 0 || options.NoVsync
	session.configureFrustum(&app.Camera)
	return app
}

func newViewer(world *scene.Scene, startWalking bool) *viewer {
	minimum, maximum := world.Bounds()
	size := float32(math.Max(float64(maximum[0]-minimum[0]), float64(maximum[2]-minimum[2])))
	if size <= 0 || math.IsInf(float64(size), 0) {
		size = 10
	}
	center := graphics.Vec4{(minimum[0] + maximum[0]) / 2, 0, (minimum[2] + maximum[2]) / 2, 1}
	session := &viewer{world: world, sceneSize: size, started: time.Now(),
		baseFogNear: world.Environment.FogNear, baseFogFar: world.Environment.FogFar}
	session.sunAzimuth, session.sunElevation = sunAnglesFromDirection(world.Environment.SunDirection)
	session.dayElevation = math.Max(session.sunElevation, 0.3)
	session.orbit = newOrbitFor(world, center, size)
	session.walk = newWalkFor(world, center, size)
	if startWalking {
		session.toggleWalking()
	}
	return session
}

func newOrbitFor(world *scene.Scene, center graphics.Vec4, size float32) *camera.Orbit {
	orbit := camera.NewOrbit(center, float64(size)*1.1)
	orbit.MaximumDistance = float64(size) * 3
	orbit.Ground = world.GroundHeight
	return orbit
}

func newWalkFor(world *scene.Scene, center graphics.Vec4, size float32) *camera.Walk {
	walk := camera.NewWalk(center[0], center[2], size*0.00004, size*0.00075)
	walk.Ground = world.GroundHeight
	walk.Blocked = func(x, z float32) bool { return isUnderWater(world, x, z) }
	return walk
}

// isUnderWater reports whether a point has no ground or lies below the water level.
func isUnderWater(world *scene.Scene, x, z float32) bool {
	height, found := world.GroundHeight(x, z)
	if !found {
		return true
	}
	return world.Environment.Water != nil && height <= world.Environment.Water.LevelAt(x, z)
}

func (session *viewer) configureFrustum(view *models.Camera) {
	view.FieldOfView = 0.73
	view.Far = session.sceneSize * 12
	view.Near = session.sceneSize * 0.0002
}

// advance runs the program's update and then the viewer's controls.
func (session *viewer) advance(app *models.Application, seconds float64) error {
	state := app.Input
	if session.update != nil {
		session.update(session.world, state, seconds)
	}
	session.handleShortcuts(state, seconds)
	session.syncVerticalScale()
	if session.walking {
		session.walk.Update(state, seconds, &app.Camera)
		app.Camera.Near = session.walk.EyeHeight * 0.1
	} else {
		session.orbit.Update(state, seconds, &app.Camera)
		app.Camera.Near = float32(math.Max(float64(session.sceneSize)*0.00002, session.orbit.Distance*0.002))
	}
	return nil
}

func (session *viewer) handleShortcuts(state *input.State, seconds float64) {
	if state.Pressed(input.KeyTab) {
		session.toggleWalking()
	}
	if state.Pressed(input.KeyF) {
		session.toggleFog()
	}
	if state.Pressed(input.KeyO) {
		session.shadowsEnabled = !session.shadowsEnabled
	}
	if state.Pressed(input.KeyN) {
		session.toggleNight()
	}
	session.adjustWalkingSpeed(state)
	session.adjustSun(state, seconds)
	session.adjustVerticalScale(state, seconds)
}

// toggleWalking switches between the orbit camera and walking; walking uses true heights.
func (session *viewer) toggleWalking() {
	environment := &session.world.Environment
	session.walking = !session.walking
	if !session.walking {
		environment.VerticalScale = session.savedScale
		return
	}
	session.savedScale = environment.EffectiveVerticalScale()
	environment.VerticalScale = 1
	session.walk.X, session.walk.Z = session.orbit.Target[0], session.orbit.Target[2]
	session.walk.Yaw = math.Pi - session.orbit.Yaw
}

// placeWalker applies the walking options: where the walker faces, looks and how tall it is.
func (session *viewer) placeWalker(options Options) {
	if !math.IsNaN(options.Heading) {
		session.walk.Yaw = options.Heading * math.Pi / 180
	}
	session.walk.Pitch = options.Pitch * math.Pi / 180
	if options.EyeHeight > 0 {
		session.walk.EyeHeight = float32(options.EyeHeight)
	}
}

// adjustWalkingSpeed makes walking faster with ] and slower with [, by a factor of 1.5 per press.
func (session *viewer) adjustWalkingSpeed(state *input.State) {
	factor := float32(1)
	if state.Pressed(input.KeyRightBracket) {
		factor = 1.5
	}
	if state.Pressed(input.KeyLeftBracket) {
		factor = 1 / 1.5
	}
	if factor == 1 {
		return
	}
	minimum, maximum := session.sceneSize*0.00002, session.sceneSize*0.02
	session.walk.Speed = float32(math.Max(float64(minimum), math.Min(float64(maximum), float64(session.walk.Speed*factor))))
	fmt.Printf("walking speed: %.4g units per second (shift: x%.0f)\n", session.walk.Speed, session.walk.RunFactor)
}

func (session *viewer) toggleFog() {
	environment := &session.world.Environment
	if environment.FogFar > 0 {
		environment.FogNear, environment.FogFar = 0, 0
		return
	}
	environment.FogNear, environment.FogFar = session.baseFogNear, session.baseFogFar
	if environment.FogFar == 0 {
		environment.FogNear, environment.FogFar = session.sceneSize*1.1, session.sceneSize*3
	}
}

// adjustSun moves the sun with the keys. Otherwise the viewer follows the scene's sun, which the
// program's update may be changing.
func (session *viewer) adjustSun(state *input.State, seconds float64) {
	if !anyDown(state, input.KeyJ, input.KeyL, input.KeyI, input.KeyK) {
		session.sunAzimuth, session.sunElevation = sunAnglesFromDirection(session.world.Environment.SunDirection)
		return
	}
	step := 0.8 * seconds
	if state.Down(input.KeyJ) {
		session.sunAzimuth -= step
	}
	if state.Down(input.KeyL) {
		session.sunAzimuth += step
	}
	if state.Down(input.KeyI) {
		session.sunElevation = math.Min(session.sunElevation+step, 1.5)
	}
	if state.Down(input.KeyK) {
		session.sunElevation = math.Max(session.sunElevation-step, -1.2)
	}
	session.world.Environment.SunDirection = sunDirectionFromAngles(session.sunAzimuth, session.sunElevation)
}

func anyDown(state *input.State, keys ...input.Key) bool {
	for _, key := range keys {
		if state.Down(key) {
			return true
		}
	}
	return false
}

// placeSun sets the sun from angles in degrees; NaN keeps the current angle.
func (session *viewer) placeSun(azimuthDegrees, elevationDegrees float64) {
	if !math.IsNaN(azimuthDegrees) {
		session.sunAzimuth = azimuthDegrees * math.Pi / 180
	}
	if !math.IsNaN(elevationDegrees) {
		session.sunElevation = elevationDegrees * math.Pi / 180
	}
	session.world.Environment.SunDirection = sunDirectionFromAngles(session.sunAzimuth, session.sunElevation)
}

// toggleNight puts the sun well below the horizon, or back where it was during the day.
func (session *viewer) toggleNight() {
	if session.sunElevation > 0 {
		session.dayElevation = session.sunElevation
		session.sunElevation = -0.45
	} else {
		session.sunElevation = session.dayElevation
	}
	session.world.Environment.SunDirection = sunDirectionFromAngles(session.sunAzimuth, session.sunElevation)
}

func (session *viewer) adjustVerticalScale(state *input.State, seconds float64) {
	if session.walking {
		return
	}
	environment := &session.world.Environment
	scale := float64(environment.EffectiveVerticalScale())
	if state.Down(input.KeyX) {
		scale *= math.Exp(seconds)
	}
	if state.Down(input.KeyZ) {
		scale *= math.Exp(-seconds)
	}
	environment.VerticalScale = float32(math.Max(1, math.Min(8, scale)))
}

// syncVerticalScale tells the controllers the current exaggeration so they follow the drawn ground.
func (session *viewer) syncVerticalScale() {
	scale := session.world.Environment.EffectiveVerticalScale()
	session.orbit.VerticalScale, session.walk.VerticalScale = scale, scale
}

// sunDirectionFromAngles converts azimuth (from north, clockwise, radians) and elevation to a
// direction towards the sun, with -Z as north and +X as east.
func sunDirectionFromAngles(azimuth, elevation float64) graphics.Vec4 {
	return graphics.Vec4{
		float32(math.Sin(azimuth) * math.Cos(elevation)),
		float32(math.Sin(elevation)),
		float32(-math.Cos(azimuth) * math.Cos(elevation)),
		0,
	}
}

func sunAnglesFromDirection(direction graphics.Vec4) (azimuth, elevation float64) {
	unit := direction.Normalize()
	return math.Atan2(float64(unit[0]), float64(-unit[2])), math.Asin(float64(unit[1]))
}

func (session *viewer) draw(app *models.Application) error {
	if session.renderer == nil {
		renderer, err := opengl.NewSceneRenderer()
		if err != nil {
			return err
		}
		session.renderer = renderer
	}
	session.renderer.Shadows.Enabled = session.shadowsEnabled
	width, height := app.Input.FramebufferWidth, app.Input.FramebufferHeight
	if width == 0 || height == 0 {
		width, height = app.Width, app.Height
	}
	session.renderer.Profiling = session.benchmark != nil && session.benchmark.timing()
	session.renderer.Draw(session.world, app.Camera, width, height, time.Since(session.started).Seconds())
	if session.benchmark != nil && session.benchmark.record(session.renderer.Statistics) {
		session.benchmark.report(os.Stdout)
		app.Stop = true
	}
	return nil
}
