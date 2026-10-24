package windowloop

import (
	"testing"
	"time"

	"github.com/akrck02/go-renderer/frameloop"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/window"
)

// countingApplication is an application that counts its update steps.
func countingApplication(updates *int) *window.Application {
	return &window.Application{Callbacks: window.Callbacks{Update: func(*window.Application, float64) error {
		*updates++
		return nil
	}}}
}

// followerOf is a resize follower of a window whose framebuffer has the size in size.
func followerOf(app *window.Application, size *[2]int) *resizeFollower {
	return &resizeFollower{
		framebufferSize: func() (int, int) { return size[0], size[1] },
		app:             app,
		clock:           frameloop.NewClock(1), // one step a second: none is due in a test
	}
}

func TestAWindowThatChangedSizeIsUpdatedBeforeItsStepIsDue(test *testing.T) {
	updates := 0
	size := [2]int{800, 600}
	follower := followerOf(countingApplication(&updates), &size)
	if err := follower.advance(); err != nil {
		test.Fatal(err)
	}
	updates = 0
	size = [2]int{1024, 600}
	if err := follower.advance(); err != nil {
		test.Fatal(err)
	}
	if updates != 1 {
		test.Fatalf("a new size ran %d update steps, want one borrowed step", updates)
	}
	if err := follower.advance(); err != nil {
		test.Fatal(err)
	}
	if updates != 1 {
		test.Fatalf("the same size ran another step before one was due")
	}
}

func TestAWindowDrawingOnDemandRunsOneStepPerFrame(test *testing.T) {
	updates := 0
	app := countingApplication(&updates)
	app.DrawOnDemand, app.Input = true, input.NewState()
	size := [2]int{800, 600}
	follower := followerOf(app, &size)
	for range 3 {
		if err := follower.advance(); err != nil {
			test.Fatal(err)
		}
	}
	if updates != 3 {
		test.Fatalf("three frames ran %d steps", updates)
	}
}

func TestAFrameIsDrawnOnlyAfterAStepUnlessOnDemand(test *testing.T) {
	if frameDue(false, 4, 4) {
		test.Error("a frame without a new step would repeat the last one")
	}
	if !frameDue(false, 4, 5) || !frameDue(true, 4, 4) {
		test.Error("a step, or a loop woken on demand, draws a frame")
	}
}

func TestTheFramesPerSecondAreCountedOnceASecond(test *testing.T) {
	start := time.Unix(0, 0)
	counter := framesCounter{since: start}
	for frame := 1; frame < 60; frame++ {
		if _, due := counter.count(start.Add(time.Duration(frame) * time.Second / 60)); due {
			test.Fatalf("frame %d reported before a second passed", frame)
		}
	}
	frames, due := counter.count(start.Add(time.Second))
	if !due || frames != 60 {
		test.Fatalf("after a second: %d frames, due %v", frames, due)
	}
	if title := framesTitle("Island", "1.0", frames); title != "Island 1.0   |   60 FPS" {
		test.Fatalf("title %q", title)
	}
}

// pretendWindow records what sleepReleasingMemory asked of it.
type pretendWindow struct {
	wokenEarly, input      bool
	released, sleptForever bool
}

func (window *pretendWindow) waitAtMost(time.Duration) bool { return !window.wokenEarly }
func (window *pretendWindow) waitForever()                  { window.sleptForever = true }
func (window *pretendWindow) hadInput() bool                { return window.input }
func (window *pretendWindow) release()                      { window.released = true }

func TestAnIdleWindowGivesItsMemoryBackThenSleepsOn(test *testing.T) {
	window := &pretendWindow{}
	sleepReleasingMemory(window)
	if !window.released || !window.sleptForever {
		test.Errorf("an idle window released %v and slept on %v", window.released, window.sleptForever)
	}
}

func TestAWindowWokenBeforeItIsIdleDrawsWithoutReleasing(test *testing.T) {
	for _, window := range []*pretendWindow{{wokenEarly: true}, {input: true}} {
		sleepReleasingMemory(window)
		if window.released || window.sleptForever {
			test.Errorf("a window woken (early %v, input %v) released memory or slept on", window.wokenEarly, window.input)
		}
	}
}
