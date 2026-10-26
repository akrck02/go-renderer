package gles

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene/sample"
)

// twoEyes are a left and a right view of the sample island, each in its own target, as a
// headset hands them out, and the covering lens to plan them with.
func twoEyes() ([]View, camera.Lens, float32) {
	rig := camera.Rig{Position: graphics.Vector{0, 600, 900, 1}, Scale: 100}
	field := graphics.FieldOfView{LeftRadians: -0.8, RightRadians: 0.8, UpRadians: 0.8, DownRadians: -0.8}
	var poses []graphics.Pose
	var views []View
	for index, side := range []float32{-1, 1} {
		pose := graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{side * 0.032, 1.6, 0, 1}}
		poses = append(poses, pose)
		views = append(views, View{
			Target: Framebuffer(100 + index), X: 0, Y: 0, Width: 640 + index, Height: 480,
			View: rig.EyeView(pose), Projection: graphics.PerspectiveFromFieldOfView(field, 1, 20000),
			Eye: rig.EyePosition(pose),
		})
	}
	covering, aspect := camera.CoveringLens(rig, poses, []graphics.FieldOfView{field, field}, 1, 20000)
	return views, covering, aspect
}

func TestDrawViewsPlansOnceAndDrawsEveryView(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake, overlay.Fonts{})
	if err != nil {
		test.Fatal(err)
	}
	views, covering, aspect := twoEyes()
	renderer.DrawViews(sample.Island(), covering, aspect, views)

	if renderer.plans != 1 {
		test.Fatalf("frames planned = %d, want 1 for both views", renderer.plans)
	}
	drawsIn := map[Framebuffer]int{}
	for _, draw := range fake.draws {
		drawsIn[draw.framebuffer]++
	}
	for _, view := range views {
		if drawsIn[view.Target] == 0 {
			test.Fatalf("nothing was drawn in target %d", view.Target)
		}
	}
	if drawsIn[views[0].Target] != drawsIn[views[1].Target] || len(drawsIn) != 2 {
		test.Fatalf("draws per target = %v, want the same plan in both and nothing elsewhere", drawsIn)
	}
	if len(fake.viewports) != 2 {
		test.Fatalf("viewports = %v, want one per view", fake.viewports)
	}
	for index, viewport := range fake.viewports {
		view := views[index]
		if viewport != (recordedViewport{view.Target, view.X, view.Y, view.Width, view.Height}) {
			test.Fatalf("viewport %d = %+v, want view %+v's", index, viewport, view)
		}
	}
	if len(fake.clears) != 2 {
		test.Fatalf("clears = %v, want one per target", fake.clears)
	}
}

func TestViewsSharingATargetClearItOnce(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake, overlay.Fonts{})
	if err != nil {
		test.Fatal(err)
	}
	views, covering, aspect := twoEyes()
	views[1].Target, views[1].X = views[0].Target, 640 // side by side in one target
	renderer.DrawViews(sample.Island(), covering, aspect, views)
	if len(fake.clears) != 1 {
		test.Fatalf("clears = %v, want one: a second would wipe the first view", fake.clears)
	}
}

func TestLinearOutputLeavesTheGammaOut(test *testing.T) {
	for _, linear := range []bool{false, true} {
		fake := newRecorder()
		renderer, err := NewSceneRenderer(fake, overlay.Fonts{})
		if err != nil {
			test.Fatal(err)
		}
		renderer.LinearOutput = linear
		views, covering, aspect := twoEyes()
		renderer.DrawViews(sample.Island(), covering, aspect, views)
		exponent, set := fake.float("outputExponent")
		want := float32(1 / 2.2)
		if linear {
			want = 1
		}
		if !set || math.Abs(float64(exponent-want)) > 1e-6 {
			test.Fatalf("linear %v: output exponent = %v (set %v), want %v", linear, exponent, set, want)
		}
	}
}

func TestDrawStillTurnsColoursToTheScreensGamma(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake, overlay.Fonts{})
	if err != nil {
		test.Fatal(err)
	}
	renderer.Draw(sample.Island(), islandCamera(), 1280, 800)
	if exponent, _ := fake.float("outputExponent"); math.Abs(float64(exponent-1/2.2)) > 1e-6 {
		test.Fatalf("output exponent = %v, want 1/2.2", exponent)
	}
	if len(fake.viewports) == 0 || fake.viewports[0] != (recordedViewport{0, 0, 0, 1280, 800}) {
		test.Fatalf("viewports = %v, want the whole screen first", fake.viewports)
	}
}

func TestNoViewsDrawNothing(test *testing.T) {
	fake := newRecorder()
	renderer, err := NewSceneRenderer(fake, overlay.Fonts{})
	if err != nil {
		test.Fatal(err)
	}
	renderer.DrawViews(sample.Island(), camera.Lens{}, 1, nil)
	if renderer.plans != 0 || len(fake.draws) != 0 {
		test.Fatalf("planned %d frames and drew %d calls for no views", renderer.plans, len(fake.draws))
	}
}

func TestASeeThroughSceneClearsToNothingSoTheRoomShows(test *testing.T) {
	for _, seeThrough := range []bool{false, true} {
		fake := newRecorder()
		renderer, err := NewSceneRenderer(fake, overlay.Fonts{})
		if err != nil {
			test.Fatal(err)
		}
		renderer.SeeThrough = seeThrough
		views, covering, aspect := twoEyes()
		renderer.DrawViews(sample.Island(), covering, aspect, views)
		if len(fake.clearColors) == 0 {
			test.Fatalf("see-through %v: nothing was cleared", seeThrough)
		}
		alpha := fake.clearColors[len(fake.clearColors)-1][3]
		if seeThrough && fake.clearColors[len(fake.clearColors)-1] != [4]float32{} {
			test.Errorf("see-through: clear colour %v, want transparent", fake.clearColors[len(fake.clearColors)-1])
		}
		if !seeThrough && alpha != 1 {
			test.Errorf("not see-through: clear alpha %v, want the opaque horizon", alpha)
		}
	}
}
