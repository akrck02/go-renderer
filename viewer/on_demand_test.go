package viewer

import (
	"testing"

	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

func newOnDemandSession(t *testing.T) (*viewer, *models.Application) {
	t.Helper()
	session := newViewer(sample.Island(), false)
	app := newApplication(session, DefaultOptions(), "test")
	if !app.DrawOnDemand || app.Unchanged == nil {
		t.Fatal("a viewer with no update function draws on demand")
	}
	app.Input.FramebufferWidth, app.Input.FramebufferHeight = 800, 500
	return session, app
}

// step runs one update step and says whether it asked for another frame at once.
func step(t *testing.T, session *viewer, app *models.Application) bool {
	t.Helper()
	if err := session.advance(app, 1.0/60); err != nil {
		t.Fatal(err)
	}
	app.Input.EndFrame()
	_, wanted := app.Input.TakeFrameRequest()
	return wanted
}

func TestAStillViewAsksForNoFrames(t *testing.T) {
	session, app := newOnDemandSession(t)
	step(t, session, app) // the camera settles into place
	if step(t, session, app) {
		t.Error("nothing moved, yet another frame was asked for")
	}
	if app.Unchanged(app) {
		t.Error("the first frame must be drawn")
	}
	if !app.Unchanged(app) {
		t.Error("the same view again should be skipped")
	}
}

func TestHoldingAKeyKeepsDrawing(t *testing.T) {
	session, app := newOnDemandSession(t)
	step(t, session, app)
	app.Input.SetKey(input.KeyL, true) // turns the sun
	for frame := 0; frame < 3; frame++ {
		if !step(t, session, app) {
			t.Fatalf("frame %d: a held key should ask for the next frame", frame)
		}
	}
	app.Input.SetKey(input.KeyL, false)
	step(t, session, app)
	if step(t, session, app) {
		t.Error("once the key is let go and the sun still, frames should stop")
	}
}

func TestASimulationOrABenchmarkDrawsEveryFrame(t *testing.T) {
	session := newViewer(sample.Island(), false)
	session.update = func(*scene.Scene, *input.State, float64) {}
	if newApplication(session, DefaultOptions(), "test").DrawOnDemand {
		t.Error("a simulation keeps drawing")
	}
	options := DefaultOptions()
	options.Benchmark = 10
	if newApplication(newViewer(sample.Island(), false), options, "test").DrawOnDemand {
		t.Error("a benchmark keeps drawing")
	}
	options = DefaultOptions()
	options.Continuous = true
	if newApplication(newViewer(sample.Island(), false), options, "test").DrawOnDemand {
		t.Error("-continuous keeps drawing")
	}
}

func TestAnEditorStillDrawsOnDemand(t *testing.T) {
	session := newViewer(sample.Island(), false)
	session.edit = func(*scene.Scene, *input.State, float64, View) {}
	app := newApplication(session, DefaultOptions(), "test")
	if !app.DrawOnDemand {
		t.Error("an editor changes only with input, so it draws on demand")
	}
	if app.Unchanged != nil {
		t.Error("an editor changes its own drawing, which the viewer cannot compare")
	}
}

// The editor sees the camera that drew the frame, and the middle of the window lies where that
// camera looks.
func TestAnEditorFindsThePointUnderThePointer(t *testing.T) {
	session := newViewer(sample.Island(), false)
	session.enterPlan(false)
	var seen View
	session.edit = func(_ *scene.Scene, _ *input.State, _ float64, view View) { seen = view }
	app := newApplication(session, DefaultOptions(), "test")
	app.Input.WindowWidth, app.Input.WindowHeight = 800, 500
	for step := 0; step < 2; step++ {
		if err := session.advance(app, 1.0/60); err != nil {
			t.Fatal(err)
		}
	}
	point, hit := seen.PointOnPlane(400, 250, 0)
	target := app.Camera.Target
	if !hit || abs32(point[0]-target[0]) > 0.01 || abs32(point[2]-target[2]) > 0.01 {
		t.Errorf("the middle of the map is %v, want the camera's target %v", point, target)
	}
	if ground, onGround := seen.PointOnGround(400, 250); !onGround || abs32(ground[0]-target[0]) > 0.5 {
		t.Errorf("the ground under the middle of the map is %v (%v), want near %v", ground, onGround, target)
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
func TestAProgramFramesARectangleOfGround(t *testing.T) {
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
	if !step(t, session, app) {
		t.Error("framing asks for a frame")
	}
	if session.plan.Target[0] != 200 || session.plan.Target[2] != 40 || session.orbit.Target[0] != 200 {
		t.Errorf("aimed at %v / %v", session.plan.Target, session.orbit.Target)
	}
	if session.plan.Span != 100 { // 200 wide in a window twice as wide as high
		t.Errorf("span %v", session.plan.Span)
	}
}
