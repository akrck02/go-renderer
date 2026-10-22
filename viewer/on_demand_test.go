package viewer

import (
	"testing"

	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
	"github.com/akrck02/go-renderer/window"
)

func newOnDemandSession(test *testing.T) (*viewer, *window.Application) {
	test.Helper()
	session := newViewer(sample.Island(), false)
	app := newApplication(session, DefaultOptions(), "test")
	if !app.DrawOnDemand || app.Unchanged == nil {
		test.Fatal("a viewer with no update function draws on demand")
	}
	app.Input.FramebufferWidth, app.Input.FramebufferHeight = 800, 500
	return session, app
}

// step runs one update step and says whether it asked for another frame at once.
func step(test *testing.T, session *viewer, app *window.Application) bool {
	test.Helper()
	if err := session.advance(app, 1.0/60); err != nil {
		test.Fatal(err)
	}
	app.Input.EndFrame()
	_, wanted := app.Input.TakeFrameRequest()
	return wanted
}

func TestAStillViewAsksForNoFrames(test *testing.T) {
	session, app := newOnDemandSession(test)
	step(test, session, app) // the camera settles into place
	if step(test, session, app) {
		test.Error("nothing moved, yet another frame was asked for")
	}
	if app.Unchanged(app) {
		test.Error("the first frame must be drawn")
	}
	if !app.Unchanged(app) {
		test.Error("the same view again should be skipped")
	}
}

func TestHoldingAKeyKeepsDrawing(test *testing.T) {
	session, app := newOnDemandSession(test)
	step(test, session, app)
	app.Input.SetKey(input.KeyL, true) // turns the sun
	for frame := 0; frame < 3; frame++ {
		if !step(test, session, app) {
			test.Fatalf("frame %d: a held key should ask for the next frame", frame)
		}
	}
	app.Input.SetKey(input.KeyL, false)
	step(test, session, app)
	if step(test, session, app) {
		test.Error("once the key is let go and the sun still, frames should stop")
	}
}

func TestASimulationOrABenchmarkDrawsEveryFrame(test *testing.T) {
	session := newViewer(sample.Island(), false)
	session.update = func(*scene.Scene, *input.State, float64) {}
	if newApplication(session, DefaultOptions(), "test").DrawOnDemand {
		test.Error("a simulation keeps drawing")
	}
	options := DefaultOptions()
	options.Benchmark = 10
	if newApplication(newViewer(sample.Island(), false), options, "test").DrawOnDemand {
		test.Error("a benchmark keeps drawing")
	}
	options = DefaultOptions()
	options.Continuous = true
	if newApplication(newViewer(sample.Island(), false), options, "test").DrawOnDemand {
		test.Error("-continuous keeps drawing")
	}
}

func TestAnEditorStillDrawsOnDemand(test *testing.T) {
	session := newViewer(sample.Island(), false)
	session.edit = func(*scene.Scene, *input.State, float64, View) {}
	app := newApplication(session, DefaultOptions(), "test")
	if !app.DrawOnDemand {
		test.Error("an editor changes only with input, so it draws on demand")
	}
	if app.Unchanged != nil {
		test.Error("an editor changes its own drawing, which the viewer cannot compare")
	}
}

// The editor sees the camera that drew the frame, and the middle of the window lies where that
// camera looks.
func TestAnEditorFindsThePointUnderThePointer(test *testing.T) {
	session := newViewer(sample.Island(), false)
	session.enterPlan(false)
	var seen View
	session.edit = func(_ *scene.Scene, _ *input.State, _ float64, view View) { seen = view }
	app := newApplication(session, DefaultOptions(), "test")
	app.Input.WindowWidth, app.Input.WindowHeight = 800, 500
	for step := 0; step < 2; step++ {
		if err := session.advance(app, 1.0/60); err != nil {
			test.Fatal(err)
		}
	}
	point, hit := seen.PointOnPlane(400, 250, 0)
	target := app.Camera.Target
	if !hit || abs32(point[0]-target[0]) > 0.01 || abs32(point[2]-target[2]) > 0.01 {
		test.Errorf("the middle of the map is %v, want the camera's target %v", point, target)
	}
	if ground, onGround := seen.PointOnGround(400, 250); !onGround || abs32(ground[0]-target[0]) > 0.5 {
		test.Errorf("the ground under the middle of the map is %v (%v), want near %v", ground, onGround, target)
	}
}

func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

// An editing program can ask for a rectangle of ground to be shown: the cameras aim at its middle,
// the map fits it, and a frame follows.
func TestAProgramFramesARectangleOfGround(test *testing.T) {
	session := newViewer(sample.Island(), false)
	asked := false
	session.edit = func(_ *scene.Scene, _ *input.State, _ float64, view View) {
		if !asked {
			view.FrameGround(100, 20, 300, 60)
			asked = true
		}
	}
	app := newApplication(session, DefaultOptions(), "test")
	app.Input.WindowWidth, app.Input.WindowHeight = 800, 400
	if !step(test, session, app) {
		test.Error("framing asks for a frame")
	}
	if session.plan.Target[0] != 200 || session.plan.Target[2] != 40 || session.orbit.Target[0] != 200 {
		test.Errorf("aimed at %v / %v", session.plan.Target, session.orbit.Target)
	}
	if session.plan.Span != 100 { // 200 wide in a window twice as wide as high
		test.Errorf("span %v", session.plan.Span)
	}
}

// A program can zoom the cameras, as a zoom button does, and read how much the map shows.
func TestAProgramZoomsTheCameras(test *testing.T) {
	session := newViewer(sample.Island(), false)
	var shown float64
	steps := 0
	session.edit = func(_ *scene.Scene, _ *input.State, _ float64, view View) {
		steps++
		shown = view.Span()
		if steps == 1 {
			view.ZoomBy(2)
		}
	}
	app := newApplication(session, DefaultOptions(), "test")
	app.Input.WindowWidth, app.Input.WindowHeight = 800, 400
	before := session.plan.Span
	step(test, session, app)
	if session.plan.Span != before/2 {
		test.Errorf("span %v after zooming by 2 from %v", session.plan.Span, before)
	}
	step(test, session, app)
	if shown != before/2 {
		test.Errorf("the view tells the span: %v", shown)
	}
}
