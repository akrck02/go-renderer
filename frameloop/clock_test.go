package frameloop

import (
	"testing"
	"time"

	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
)

func countingApplication(updates *int) *models.Application {
	return &models.Application{Input: input.NewState(), Update: func(*models.Application, float64) error {
		*updates++
		return nil
	}}
}

func TestStepOnceRunsExactlyOneStep(test *testing.T) {
	updates := 0
	app := countingApplication(&updates)
	clock := NewClock(0)
	clock.previous = time.Now().Add(-time.Second) // a long sleep must not be caught up
	if err := clock.StepOnce(app); err != nil {
		test.Fatal(err)
	}
	if updates != 1 || clock.Steps() != 1 {
		test.Fatalf("updates %d, steps %d, want 1 and 1", updates, clock.Steps())
	}
}

func TestAdvanceCatchesUpAtMostAQuarterOfASecond(test *testing.T) {
	updates := 0
	app := countingApplication(&updates)
	clock := NewClock(0.05)
	clock.previous = time.Now().Add(-10 * time.Second)
	if err := clock.Advance(app); err != nil {
		test.Fatal(err)
	}
	if updates != 5 {
		test.Fatalf("updates = %d, want 5 steps of 0.05 s in a quarter of a second", updates)
	}
}

func TestAStepClearsTheStepsInput(test *testing.T) {
	updates := 0
	app := countingApplication(&updates)
	app.Input.MoveCursor(10, 10)
	app.Input.MoveCursor(20, 10)
	if err := RunUpdateStep(app, 0.1); err != nil {
		test.Fatal(err)
	}
	if app.Input.HasInput() {
		test.Fatal("the cursor's movement is still there after the step")
	}
}

func TestUntilNextStepIsNeverNegative(test *testing.T) {
	clock := NewClock(0.01)
	clock.previous = time.Now().Add(-time.Second)
	if wait := clock.UntilNextStep(); wait != 0 {
		test.Fatalf("wait = %v, want 0 for an overdue step", wait)
	}
}
