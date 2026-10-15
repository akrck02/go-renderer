package sceneplan

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
)

// A map camera stands far above the ground whatever it shows: its shadows must follow what is on
// screen, not how far away it stands.
func TestShadowsOfAMapFollowWhatIsOnScreen(t *testing.T) {
	settings := ShadowSettings{}
	mapCamera := models.Camera{Position: graphics.Vec4{0, 400, 0, 1}, Target: graphics.Vec4{3, 0, 4, 1}, OrthographicHeight: 10}
	extent := settings.extent(mapCamera, 100)
	if extent < 9 || extent > 10 {
		t.Errorf("extent %v for 10 units on screen, want about 9.5", extent)
	}
	if center := shadowCenter(mapCamera, extent); center != mapCamera.Target {
		t.Errorf("shadow box centred on %v, want the middle of the screen %v", center, mapCamera.Target)
	}
	mapCamera.OrthographicHeight = 2
	if closer := settings.extent(mapCamera, 100); closer >= extent {
		t.Errorf("zooming in from 10 to 2 units should shrink the shadow box (%v, then %v)", extent, closer)
	}
}

func TestAMapCameraSeesEverySizeTheSameFromAnyDistance(t *testing.T) {
	camera := models.Camera{Position: graphics.Vec4{0, 400, 0, 1}, Target: graphics.Vec4{0, 0, 0, 1}, Up: graphics.Vec4{0, 0, -1, 0},
		OrthographicHeight: 20, Near: 1, Far: 1000}
	view := newViewpoint(camera.ProjectionMatrix(1.6).Multiply(graphics.LookAt(camera.Position, camera.Target, camera.Up)), camera)
	if near, far := view.apparentSize(2, 10), view.apparentSize(2, 800); near != far || near != 0.1 {
		t.Errorf("2 units on a 20-unit screen look %v near and %v far, want 0.1 both", near, far)
	}
}
