// Command visor opens a glTF 2.0 scene (.glb or .gltf) and lets you explore it.
//
//	go run ./cmd/visor scene.glb
//	go run ./cmd/visor -capture frame.png scene.glb   (renders one frame to a PNG, no visible window)
//
// Controls: left drag rotates, right drag pans, scroll zooms (orbit camera). Tab switches to
// walking on the ground: WASD or arrows move, mouse looks, shift runs, [ and ] change the walking speed. J/L turn the sun,
// I/K raise or lower it, Z/X change the vertical exaggeration, F toggles fog, O toggles shadows.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/loaders"
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
}

type options struct {
	width, height int
	capturePath   string
	startWalking  bool
	noShadows     bool
	zoom          float64
}

func main() {
	runtime.LockOSThread()
	settings, scenePath := parseOptions()
	world, err := loaders.LoadScene(scenePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot load scene:", err)
		os.Exit(1)
	}
	session := newViewer(world, settings.startWalking)
	session.shadowsEnabled = !settings.noShadows
	session.orbit.Distance *= settings.zoom
	app := newApplication(session, settings, scenePath)
	if err := (&opengl.OpenGL{}).StartLoop(app); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseOptions() (options, string) {
	var settings options
	flag.IntVar(&settings.width, "width", 1280, "window width")
	flag.IntVar(&settings.height, "height", 800, "window height")
	flag.StringVar(&settings.capturePath, "capture", "", "render one frame to this PNG and exit")
	flag.BoolVar(&settings.startWalking, "walk", false, "start walking on the ground")
	flag.BoolVar(&settings.noShadows, "no-shadows", false, "start without sun shadows")
	flag.Float64Var(&settings.zoom, "zoom", 1, "initial orbit distance as a fraction of the default (0.1 = ten times closer)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: visor [-width W] [-height H] [-capture frame.png] [-walk] [-no-shadows] [-zoom F] scene.glb")
		os.Exit(2)
	}
	return settings, flag.Arg(0)
}

func newApplication(session *viewer, settings options, scenePath string) *models.Application {
	app := &models.Application{
		Type:        models.WindowApplication,
		Title:       "Visor · " + scenePath,
		Version:     "",
		Width:       settings.width,
		Height:      settings.height,
		Input:       input.NewState(),
		ModelMatrix: graphics.Identity(),
		Renderer:    &opengl.OpenGLRenderer{},
		Update:      session.update,
		Draw:        session.draw,
	}
	if settings.capturePath != "" {
		app.Type, app.CapturePath = models.HeadlessApplication, settings.capturePath
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
	session := &viewer{world: world, sceneSize: size, started: time.Now(),
		baseFogNear: world.Environment.FogNear, baseFogFar: world.Environment.FogFar}
	session.sunAzimuth, session.sunElevation = sunAnglesFromDirection(world.Environment.SunDirection)
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
	return world.Environment.Water != nil && height <= world.Environment.Water.Level
}

func (session *viewer) configureFrustum(view *models.Camera) {
	view.FieldOfView = 0.73
	view.Far = session.sceneSize * 12
	view.Near = session.sceneSize * 0.0002
}

func (session *viewer) update(app *models.Application, seconds float64) error {
	state := app.Input
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

func (session *viewer) adjustSun(state *input.State, seconds float64) {
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
		session.sunElevation = math.Max(session.sunElevation-step, 0.02)
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
	session.renderer.Draw(session.world, app.Camera, width, height, time.Since(session.started).Seconds())
	return nil
}
