package viewer

import (
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/metal"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/window"
)

// runMetal shows the session through Metal.
func runMetal(session *viewer, app *window.Application) error {
	window := &metal.SceneWindow{}
	session.makeRenderer = func() (sceneDrawer, error) { return metalScene{window.Renderer()}, nil }
	return window.Run(app)
}

type metalScene struct{ renderer *metal.SceneRenderer }

func (scene metalScene) draw(world *scene.Scene, camera camera.Camera, width, height int, seconds float64, settings drawSettings) sceneplan.FrameStatistics {
	renderer := scene.renderer
	renderer.Shadows.Enabled, renderer.OverlayScale, renderer.Profiling = settings.shadows, settings.overlayScale, settings.profiling
	renderer.Draw(world, camera, width, height, seconds)
	return renderer.Statistics
}
