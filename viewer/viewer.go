// Package viewer is an interactive explorer for a scene.Scene: orbit and walking cameras, sun,
// fog, shadows and vertical exaggeration controls, headless captures and benchmarks. Programs pass
// an optional update function, called at every fixed step, to change the scene while it is shown
// (a simulation, for example); the viewer itself knows nothing about what changes it.
//
// Controls: left drag rotates, right drag pans, scroll zooms (orbit camera). Tab switches to
// walking on the ground: WASD or arrows move, mouse looks, shift runs, [ and ] change the walking
// speed. J/L turn the sun, I/K raise or lower it (below the horizon it is night), N jumps between
// day and night, Z/X change the vertical exaggeration, F toggles fog, O toggles shadows. M switches
// to a map: the ground from straight above without perspective, then an isometric view, then back;
// on the map any drag moves it, scroll zooms and Q/E turn it a quarter.
package viewer

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/metal"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl"
	"github.com/akrck02/go-renderer/scene"
)

// viewer holds the state of an exploration session.
type viewer struct {
	world          *scene.Scene
	renderer       sceneDrawer
	makeRenderer   func() (sceneDrawer, error) // set by the backend the session runs on
	orbit          *camera.Orbit
	walk           *camera.Walk
	walking        bool
	plan           *camera.Plan
	planning       bool       // showing the map (top or isometric view) instead of the orbit
	fogBeforePlan  [2]float32 // fog near and far to restore when leaving the map
	sceneSize      float32
	walkLimits     [2]float32 // slowest and fastest walking speed [ and ] reach
	sunAzimuth     float64
	sunElevation   float64
	started        time.Time
	baseFogNear    float32
	baseFogFar     float32
	savedScale     float32 // vertical exaggeration to restore when leaving walk mode
	shadowsEnabled bool
	benchmark      *benchmark
	update         UpdateFunc
	edit           EditFunc
	lowest         float32 // the scene's lowest and highest true heights, for finding the ground
	highest        float32
	dayElevation   float64     // sun elevation to return to when leaving the night
	fixedTime      float64     // seconds every frame is drawn at; NaN = the real time
	shown          *drawnFrame // what the window shows, when drawing on demand
	framing        *[4]float32 // a rectangle of ground the program asked to show (View.FrameGround)
	zooming        float64     // how much nearer the program asked the cameras to come (View.ZoomBy); 0 = no change
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
	session.update = update
	return session.run(options, title, loadingTime)
}

// EditFunc changes the scene in answer to the user, at every step the viewer runs, before the
// viewer's own controls; view tells where the camera that drew the last frame was, so the program
// can find what the pointer is on (View.PointOnPlane, View.PointOnGround).
type EditFunc func(world *scene.Scene, state *input.State, step float64, view View)

// Edit shows the scene like Run, for a program that edits it (a level editor, a floor plan
// designer). Unlike a simulation it changes only when the user does something, so the viewer
// still draws on demand: when input arrives, while a key is held or the camera moves, and when the
// program asks for a frame with input.State.RequestFrame. A program that claims the mouse
// (input.State.ClaimMouse) keeps it from the camera for that step. It must be called from the main
// goroutine.
func Edit(world *scene.Scene, options Options, title string, edit EditFunc) error {
	session := newViewer(world, false)
	session.edit = edit
	return session.run(options, title, 0)
}

// run applies the options and shows the session until the window closes.
func (session *viewer) run(options Options, title string, loadingTime time.Duration) error {
	if options.LookAt != nil {
		session.orbit.Target[0], session.orbit.Target[2] = options.LookAt[0], options.LookAt[1]
	}
	session.plan.Target = session.orbit.Target
	if options.StartWalking {
		session.toggleWalking()
		session.placeWalker(options)
	}
	if options.StartMap || options.Isometric {
		session.enterPlan(options.Isometric)
		session.plan.Span *= zoomOrOne(options.Zoom)
	}
	session.fixedTime = options.Time
	session.shadowsEnabled = !options.NoShadows
	session.orbit.Distance *= zoomOrOne(options.Zoom)
	session.placeSun(options.SunAzimuth, options.SunElevation)
	if options.Night {
		session.toggleNight()
	}
	if options.Benchmark > 0 {
		session.benchmark = newBenchmark(options.Benchmark, loadingTime)
	}
	app := newApplication(session, options, title)
	if options.Backend.usesMetal() {
		err := runMetal(session, app)
		if !errors.Is(err, metal.ErrUnavailable) || options.Backend == Metal {
			return err
		}
		log.Printf("viewer: %v; drawing through OpenGL", err)
		app.Input = input.NewState()
	}
	return runOpenGL(session, app)
}

// runOpenGL shows the session through OpenGL.
func runOpenGL(session *viewer, app *models.Application) error {
	session.makeRenderer = func() (sceneDrawer, error) {
		renderer, err := opengl.NewSceneRenderer()
		if err != nil {
			return nil, err
		}
		return openGLScene{renderer}, nil
	}
	return (&opengl.OpenGL{}).StartLoop(app)
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
	// a still scene costs nothing while nobody touches it; a simulation or a benchmark draws on
	app.DrawOnDemand = !options.Continuous && session.update == nil && options.Benchmark == 0
	if app.DrawOnDemand && session.edit == nil {
		// an editor changes the scene and its own drawing, which the view state does not see
		app.Unchanged = session.unchanged
	}
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
	session := &viewer{world: world, sceneSize: size, started: time.Now(), fixedTime: math.NaN(),
		lowest: minimum[1], highest: maximum[1],
		baseFogNear: world.Environment.FogNear, baseFogFar: world.Environment.FogFar}
	session.sunAzimuth, session.sunElevation = sunAnglesFromDirection(world.Environment.SunDirection)
	session.dayElevation = math.Max(session.sunElevation, 0.3)
	session.orbit = newOrbitFor(world, center, size)
	session.walk = newWalkFor(world, center, size)
	session.plan = newPlanFor(world, center, size)
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

func newPlanFor(world *scene.Scene, center graphics.Vec4, size float32) *camera.Plan {
	plan := camera.NewPlan(center, float64(size)*1.05)
	plan.Ground = world.GroundHeight
	return plan
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

// advance runs the program's update and then the viewer's controls. The controls see the input
// without what an interface drawn by the program claimed (input.State.Unclaimed), so dragging a
// slider does not turn the camera and typing a name does not walk.
func (session *viewer) advance(app *models.Application, seconds float64) error {
	before := session.snapshot(app)
	if session.update != nil {
		session.update(session.world, app.Input, seconds)
	}
	if session.edit != nil {
		session.edit(session.world, app.Input, seconds, session.view(app))
		session.applyFraming(app)
		session.applyZoom(app)
	}
	unclaimed := app.Input.Unclaimed()
	state := &unclaimed
	session.world.Overlay.Advance(seconds)
	session.handleShortcuts(state, seconds)
	session.syncVerticalScale()
	session.moveCamera(app, state, seconds)
	if session.stillChanging(app, before) {
		app.Input.RequestFrame(0)
	}
	return nil
}

// viewState is what a frame of the viewer shows besides the scene's own animation: when it
// changes during a step, the next step follows at once.
type viewState struct {
	camera      models.Camera
	environment scene.Environment
	shadows     bool
}

func (session *viewer) snapshot(app *models.Application) viewState {
	return viewState{camera: app.Camera, environment: session.world.Environment, shadows: session.shadowsEnabled}
}

// stillChanging tells whether the view is moving, so another frame should follow even without
// input: a key or button held (walking, turning the sun), the camera or the light still changing
// (a camera easing to a stop), notices fading.
func (session *viewer) stillChanging(app *models.Application, before viewState) bool {
	return app.Input.AnyHeld() || session.snapshot(app) != before || len(session.world.Overlay.Notices) > 0
}

// drawnFrame is what the last frame shown was drawn from.
type drawnFrame struct {
	view          viewState
	width, height int
}

// unchanged tells whether a frame would look like the last one shown (the pointer moving over the
// window with no button held, say), and otherwise remembers it as the one about to be shown. Only
// for a scene nothing else changes: the water and clouds stand still between the frames drawn.
func (session *viewer) unchanged(app *models.Application) bool {
	current := drawnFrame{view: session.snapshot(app), width: app.Input.FramebufferWidth, height: app.Input.FramebufferHeight}
	if session.shown != nil && *session.shown == current && len(session.world.Overlay.Notices) == 0 {
		return true
	}
	session.shown = &current
	return false
}

// moveCamera lets the camera of the current mode follow the input.
func (session *viewer) moveCamera(app *models.Application, state *input.State, seconds float64) {
	if session.planning {
		session.plan.Update(state, seconds, &app.Camera)
		return
	}
	app.Camera.OrthographicHeight = 0
	app.Camera.Far = session.sceneSize * 12
	if session.walking {
		session.walk.Update(state, seconds, &app.Camera)
		app.Camera.Near = session.walk.EyeHeight * 0.1
	} else {
		session.orbit.Update(state, seconds, &app.Camera)
		app.Camera.Near = float32(math.Max(float64(session.sceneSize)*0.00002, session.orbit.Distance*0.002))
	}
}

func (session *viewer) handleShortcuts(state *input.State, seconds float64) {
	if state.Pressed(input.KeyTab) {
		session.leavePlan()
		session.toggleWalking()
	}
	if state.Pressed(input.KeyM) {
		session.switchMapView()
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

// switchMapView goes from the orbit (or walking) to the map seen from straight above, then to the
// isometric view, then back to the orbit.
func (session *viewer) switchMapView() {
	switch {
	case !session.planning:
		if session.walking {
			session.toggleWalking()
		}
		session.enterPlan(false)
	case !session.plan.Isometric:
		session.plan.Isometric = true
	default:
		session.leavePlan()
	}
}

// enterPlan shows the map centred where the orbit looks. Fog is lifted: seen from high above
// without perspective, it would hide the whole map.
func (session *viewer) enterPlan(isometric bool) {
	if session.planning {
		session.plan.Isometric = isometric
		return
	}
	environment := &session.world.Environment
	session.planning, session.plan.Isometric = true, isometric
	session.plan.Target = session.orbit.Target
	session.fogBeforePlan = [2]float32{environment.FogNear, environment.FogFar}
	environment.FogNear, environment.FogFar = 0, 0
}

// leavePlan goes back to the orbit, centred where the map was, with the fog it had.
func (session *viewer) leavePlan() {
	if !session.planning {
		return
	}
	environment := &session.world.Environment
	session.planning = false
	session.orbit.Target[0], session.orbit.Target[2] = session.plan.Target[0], session.plan.Target[2]
	if environment.FogFar == 0 {
		environment.FogNear, environment.FogFar = session.fogBeforePlan[0], session.fogBeforePlan[1]
	}
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
	if options.WalkSpeed > 0 {
		session.walk.Speed = float32(options.WalkSpeed)
	}
	session.walkLimits = walkingSpeedLimits(session.sceneSize, session.walk.Speed)
}

// walkingSpeedLimits keeps [ and ] within a range around both the scene's size and the speed walking starts at, so
// a scene measured in large units (a few hundred metres in kilometres) can still be walked at a person's pace.
func walkingSpeedLimits(sceneSize, start float32) [2]float32 {
	minimum := float32(math.Min(float64(sceneSize*0.00002), float64(start/20)))
	maximum := float32(math.Max(float64(sceneSize*0.02), float64(start*20)))
	return [2]float32{minimum, maximum}
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
	if session.walkLimits[1] == 0 {
		session.walkLimits = walkingSpeedLimits(session.sceneSize, session.walk.Speed)
	}
	minimum, maximum := session.walkLimits[0], session.walkLimits[1]
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
	session.orbit.VerticalScale, session.walk.VerticalScale, session.plan.VerticalScale = scale, scale, scale
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
		renderer, err := session.makeRenderer()
		if err != nil {
			return err
		}
		session.renderer = renderer
	}
	width, height := app.Input.FramebufferWidth, app.Input.FramebufferHeight
	if width == 0 || height == 0 {
		width, height = app.Width, app.Height
	}
	settings := drawSettings{shadows: session.shadowsEnabled, overlayScale: overlayScale(app.Input.WindowWidth, app.Width, width),
		profiling: session.benchmark != nil && session.benchmark.timing()}
	statistics := session.renderer.draw(session.world, app.Camera, width, height, session.seconds(), settings)
	if session.benchmark != nil && session.benchmark.record(statistics) {
		session.benchmark.report(os.Stdout)
		app.Stop = true
	}
	return nil
}

// seconds is the time the frame is drawn at: since the session started, or the fixed time.
func (session *viewer) seconds() float64 {
	if !math.IsNaN(session.fixedTime) {
		return session.fixedTime
	}
	return time.Since(session.started).Seconds()
}

// overlayScale is how many framebuffer pixels make one window pixel (2 on a high-density screen),
// so the overlay keeps the same size on any screen.
func overlayScale(windowWidth, requestedWidth, framebufferWidth int) float32 {
	if windowWidth <= 0 {
		windowWidth = requestedWidth
	}
	if windowWidth <= 0 || framebufferWidth <= 0 {
		return 1
	}
	return float32(framebufferWidth) / float32(windowWidth)
}
