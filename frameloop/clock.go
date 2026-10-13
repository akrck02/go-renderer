// Package frameloop holds the parts of a window's main loop that do not depend on any window
// system: the fixed-step update clock and the update step itself. The desktop loop
// (windowloop, over GLFW) and the browser loop (browser, over a canvas) share them, so an
// application advances the same way everywhere.
package frameloop

import (
	"math"
	"time"

	"github.com/akrck02/go-renderer/models"
)

// Clock runs Application.Update at a fixed time step, independent of the frame rate, so that
// simulations advance the same way on fast and slow machines.
type Clock struct {
	step        float64
	previous    time.Time
	accumulated float64
	steps       int // update steps run so far
}

// NewClock makes a clock of step seconds (1/60 when 0).
func NewClock(step float64) *Clock {
	if step <= 0 {
		step = 1.0 / 60
	}
	return &Clock{step: step, previous: time.Now()}
}

// Step is the clock's time step in seconds.
func (clock *Clock) Step() float64 { return clock.step }

// Steps is the number of update steps run so far.
func (clock *Clock) Steps() int { return clock.steps }

// Advance runs as many update steps as the elapsed time requires (at most a quarter of a second
// worth, so a stall does not trigger a long catch-up).
func (clock *Clock) Advance(app *models.Application) error {
	elapsed := time.Since(clock.previous).Seconds()
	clock.previous = time.Now()
	clock.accumulated += math.Min(elapsed, 0.25)
	for clock.accumulated >= clock.step {
		if err := RunUpdateStep(app, clock.step); err != nil {
			return err
		}
		clock.accumulated -= clock.step
		clock.steps++
	}
	return nil
}

// StepOnce runs exactly one update step and restarts the clock from now: a loop that draws on
// demand runs one step per frame, and after a long sleep the clock would otherwise catch up in a
// burst.
func (clock *Clock) StepOnce(app *models.Application) error {
	clock.previous, clock.accumulated = time.Now(), 0
	clock.steps++
	return RunUpdateStep(app, clock.step)
}

// BorrowStep runs one update step now, taken from the time of the next one: for a frame that has
// to be laid out again (the window changed size) before a step is due.
func (clock *Clock) BorrowStep(app *models.Application) error {
	if err := RunUpdateStep(app, clock.step); err != nil {
		return err
	}
	clock.accumulated -= clock.step
	return nil
}

// UntilNextStep is how long until the next update step is due.
func (clock *Clock) UntilNextStep() time.Duration {
	elapsed := time.Since(clock.previous).Seconds()
	wait := clock.step - clock.accumulated - elapsed
	if wait <= 0 {
		return 0
	}
	return time.Duration(wait * float64(time.Second))
}

// SleepUntilNextStep waits until the next update step is due.
func (clock *Clock) SleepUntilNextStep() {
	if wait := clock.UntilNextStep(); wait > 0 {
		time.Sleep(wait)
	}
}

// RunUpdateStep calls the application update once and clears the per-step input deltas.
func RunUpdateStep(app *models.Application, step float64) error {
	if app.Update != nil {
		if err := app.Update(app, step); err != nil {
			return err
		}
	}
	if app.Input != nil {
		app.Input.EndFrame()
	}
	return nil
}
