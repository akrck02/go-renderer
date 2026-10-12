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
