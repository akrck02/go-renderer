package openxr

// sessionState is a session's state as the runtime reports it, numbered as OpenXR numbers
// XrSessionState, so the value from an event is used as it comes.
type sessionState int

const (
	stateUnknown      sessionState = 0
	stateIdle         sessionState = 1
	stateReady        sessionState = 2 // the runtime waits for the program to begin the session
	stateSynchronized sessionState = 3 // frames are paced, nothing is shown yet
	stateVisible      sessionState = 4 // the frames are shown, something else has the controls
	stateFocused      sessionState = 5 // the frames are shown and the program has the controls
	stateStopping     sessionState = 6 // the runtime waits for the program to end the session
	stateLossPending  sessionState = 7 // the runtime is going away
	stateExiting      sessionState = 8 // the runtime asks the program to end
)

// lifecycleStep is what a new state asks of the program.
type lifecycleStep int

const (
	stepNothing lifecycleStep = iota
	stepBegin                 // xrBeginSession, then began
	stepEnd                   // xrEndSession, then ended
)

// lifecycle follows a session through the runtime's states. It is plain Go, so the order of
// states a runtime sends can be played in a test without one: the session calls the runtime when
// a step asks for it.
type lifecycle struct {
	state         sessionState
	running       bool // begun and not yet ended: frames are to be submitted
	exiting       bool
	exitRequested bool // the program asked the runtime to stop the session
}

// change takes the runtime's new state and tells what the program must do now.
func (life *lifecycle) change(state sessionState) lifecycleStep {
	life.state = state
	switch state {
	case stateReady:
		if life.running {
			return stepNothing
		}
		return stepBegin
	case stateStopping:
		if !life.running {
			return stepNothing
		}
		return stepEnd
	case stateExiting, stateLossPending:
		life.exiting = true
	}
	return stepNothing
}

// began records that the session began: frames are submitted from now on.
func (life *lifecycle) began() { life.running = true }

// ended records that the session ended: no frames until it is ready again.
func (life *lifecycle) ended() { life.running = false }

// lose records the runtime going away (the instance is about to be lost) or a step that failed:
// the program should end.
func (life *lifecycle) lose() { life.exiting = true }

// requestExit records that the program asked the runtime to stop the session: the runtime answers
// with STOPPING, which ends it, and usually EXITING.
func (life *lifecycle) requestExit() { life.exitRequested = true }

// finished tells whether the program may close the session: the runtime asked it to leave, or the
// session it asked to stop has ended. A runtime that never says EXITING after a requested stop
// still lets the program go once the session is no longer running.
func (life *lifecycle) finished() bool {
	return life.exiting || (life.exitRequested && !life.running)
}

// focused tells whether the program has the controls.
func (life *lifecycle) focused() bool { return life.state == stateFocused }
