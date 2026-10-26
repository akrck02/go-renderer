package openxr

import "testing"

// play sends states to a lifecycle as a runtime would, doing each step it asks, and returns the
// steps.
func play(life *lifecycle, states ...sessionState) []lifecycleStep {
	var steps []lifecycleStep
	for _, state := range states {
		step := life.change(state)
		switch step {
		case stepBegin:
			life.began()
		case stepEnd:
			life.ended()
		}
		steps = append(steps, step)
	}
	return steps
}

func TestReadyBeginsAndStoppingEnds(test *testing.T) {
	var life lifecycle
	steps := play(&life, stateIdle, stateReady, stateSynchronized, stateVisible, stateFocused)
	want := []lifecycleStep{stepNothing, stepBegin, stepNothing, stepNothing, stepNothing}
	for index := range want {
		if steps[index] != want[index] {
			test.Fatalf("step %d is %d, want %d", index, steps[index], want[index])
		}
	}
	if !life.running || !life.focused() || life.exiting {
		test.Fatalf("after focus: running %v focused %v exiting %v", life.running, life.focused(), life.exiting)
	}
	steps = play(&life, stateVisible, stateSynchronized, stateStopping, stateIdle)
	if steps[2] != stepEnd || life.running || life.focused() {
		test.Fatalf("stopping did not end the session: %v, running %v", steps, life.running)
	}
}

func TestOnlyFocusGivesTheControls(test *testing.T) {
	var life lifecycle
	play(&life, stateReady, stateSynchronized, stateVisible)
	if life.focused() {
		test.Fatal("a visible session has the controls")
	}
}

func TestExitingAndLossAskTheProgramToEnd(test *testing.T) {
	for _, last := range []sessionState{stateExiting, stateLossPending} {
		var life lifecycle
		play(&life, stateReady, stateFocused, stateVisible, stateSynchronized, stateStopping, stateIdle, last)
		if !life.exiting || life.running {
			test.Fatalf("state %d: exiting %v running %v", last, life.exiting, life.running)
		}
	}
}

func TestARepeatedStateAsksNothingTwice(test *testing.T) {
	var life lifecycle
	steps := play(&life, stateReady, stateReady, stateStopping, stateStopping)
	want := []lifecycleStep{stepBegin, stepNothing, stepEnd, stepNothing}
	for index := range want {
		if steps[index] != want[index] {
			test.Fatalf("steps %v, want %v", steps, want)
		}
	}
}

func TestTheSessionCanRunAgainAfterStopping(test *testing.T) {
	var life lifecycle
	play(&life, stateReady, stateStopping, stateIdle)
	if step := life.change(stateReady); step != stepBegin {
		test.Fatalf("ready again asked %d", step)
	}
}

func TestLoseAsksTheProgramToEnd(test *testing.T) {
	var life lifecycle
	life.lose()
	if !life.exiting {
		test.Fatal("lose did not ask the program to end")
	}
}

func TestARequestedExitFinishesOnceTheSessionEnds(test *testing.T) {
	var life lifecycle
	play(&life, stateReady, stateSynchronized, stateVisible, stateFocused)
	life.requestExit()
	if life.finished() {
		test.Fatal("finished while the session still runs")
	}
	play(&life, stateVisible, stateSynchronized)
	if life.finished() {
		test.Fatal("finished before the runtime stopped the session")
	}
	steps := play(&life, stateStopping)
	if steps[0] != stepEnd || !life.finished() {
		test.Fatalf("stopping after the request: step %d, finished %v", steps[0], life.finished())
	}
	play(&life, stateIdle, stateExiting)
	if !life.finished() || !life.exiting {
		test.Fatal("exiting after the request did not finish")
	}
}

func TestAStoppedSessionIsNotFinishedWithoutARequest(test *testing.T) {
	var life lifecycle
	play(&life, stateReady, stateFocused, stateStopping, stateIdle)
	if life.finished() {
		test.Fatal("a session the runtime merely paused counts as finished")
	}
}

func TestExitingFinishesWithoutARequest(test *testing.T) {
	var life lifecycle
	play(&life, stateReady, stateFocused, stateStopping, stateIdle, stateExiting)
	if !life.finished() {
		test.Fatal("the runtime asked the program to leave, yet it is not finished")
	}
}
