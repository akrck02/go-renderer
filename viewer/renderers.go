package viewer

import (
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/sceneplan"
)

// sceneDrawer is the scene renderer a session draws with: OpenGL's or Metal's.
type sceneDrawer interface {
	draw(world *scene.Scene, camera models.Camera, width, height int, seconds float64, settings drawSettings) sceneplan.FrameStatistics
}

// drawSettings are what the session changes on its renderer before each frame.
type drawSettings struct {
	shadows      bool
	overlayScale float32
	profiling    bool
}

type openGLScene struct{ renderer *opengl.SceneRenderer }

func (scene openGLScene) draw(world *scene.Scene, camera models.Camera, width, height int, seconds float64, settings drawSettings) sceneplan.FrameStatistics {
	renderer := scene.renderer
	renderer.Shadows.Enabled, renderer.OverlayScale, renderer.Profiling = settings.shadows, settings.overlayScale, settings.profiling
	renderer.Draw(world, camera, width, height, seconds)
	return renderer.Statistics
}
