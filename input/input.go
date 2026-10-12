// Package input holds a backend-independent snapshot of keyboard and mouse state.
//
// Window backends (GLFW for OpenGL today, others later) feed events into a State;
// applications, camera controllers and simulations only read from it.
//
// Besides what is held, a State remembers what happened since the last update step: keys and
// buttons that went down or up, keys repeated by the system, the characters typed and how many
// clicks a press makes (a double click). An interface drawn over the view can claim the mouse or
// the keyboard for a step, so the camera does not move while a slider is dragged or a name typed.
package input

import "time"

// Key is a backend-independent key code.
type Key int

const (
	KeyUnknown Key = iota
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeySpace
	KeyTab
	KeyEnter
	KeyEscape
	KeyShift
	KeyControl
	KeyAlt
	KeyPlus
	KeyMinus
	KeyPageUp
	KeyPageDown
	KeyLeftBracket
	KeyRightBracket
	KeyComma
	KeyPeriod
	KeyBackspace
	KeyDelete
	KeyHome
	KeyEnd
	KeySuper // the Command key on a Mac, the Windows key elsewhere
	keyCount
)

// MouseButton is a backend-independent mouse button.
type MouseButton int

const (
	MouseLeft MouseButton = iota
	MouseRight
	MouseMiddle
	mouseButtonCount
)

// CursorShape is the look of the mouse pointer.
type CursorShape int

const (
	CursorArrow            CursorShape = iota // the default pointer
	CursorText                                // an I-beam, over text that can be edited
	CursorHand                                // a pointing hand, over something to click
	CursorResizeHorizontal                    // a left-right arrow, over an edge to drag sideways
	CursorResizeVertical                      // an up-down arrow, over an edge to drag up or down
	CursorMove                                // four arrows, over something to drag around
)

// Clipboard is the system clipboard, given by the backend.
type Clipboard interface {
	ClipboardText() string
	SetClipboardText(text string)
}

// MultiClickInterval is the longest pause between the presses of a double click, and
// MultiClickDistance how far, in window coordinates, the pointer may move between them.
const (
	MultiClickInterval = 400 * time.Millisecond
	MultiClickDistance = 4.0
)

// State is the input state for the current frame.
type State struct {
	keysHeld            [keyCount]bool
	keysPressedNow      [keyCount]bool
	keysTypedNow        [keyCount]bool // pressed or repeated by the system
	buttonsHeld         [mouseButtonCount]bool
	buttonsPressedNow   [mouseButtonCount]bool
	buttonsReleasedNow  [mouseButtonCount]bool
	clicks              [mouseButtonCount]int // clicks of the last press this step: 1, 2 (double), 3…
	lastPress           pressRecord
	characters          []rune
	CursorX, CursorY    float64 // cursor position in window coordinates
	CursorDeltaX        float64 // cursor movement since the previous frame
	CursorDeltaY        float64
	ScrollDelta         float64 // vertical scroll since the previous frame
	ScrollDeltaX        float64 // horizontal scroll since the previous frame (trackpads, tilting wheels)
	hasCursorPosition   bool
	cursorLockRequested bool
	cursorShape         CursorShape
	mouseClaimed        bool
	keyboardClaimed     bool
	FramebufferWidth    int // updated by the backend
	FramebufferHeight   int
	WindowWidth         int // window size in cursor coordinates
	WindowHeight        int
	// Clipboard is set by the backend; nil when there is no system clipboard (tests, headless).
	Clipboard Clipboard
	// Now tells the time, for double clicks; nil = time.Now. Tests set it to a fake clock.
	Now func() time.Time
	// frameWanted is the earliest time a frame was asked for (RequestFrame); zero = none.
	frameWanted time.Time
}

// pressRecord is the last mouse press, to recognise the next one as part of a double click.
type pressRecord struct {
	button MouseButton
	at     time.Time
	x, y   float64
	count  int
}

// NewState returns an empty input state.
func NewState() *State { return &State{} }

func isValidKey(key Key) bool { return key > KeyUnknown && key < keyCount }

func isValidButton(button MouseButton) bool { return button >= 0 && button < mouseButtonCount }

// SetKey records a key transition (called by the backend).
func (state *State) SetKey(key Key, isDown bool) {
	if !isValidKey(key) {
		return
	}
	if isDown && !state.keysHeld[key] {
		state.keysPressedNow[key] = true
		state.keysTypedNow[key] = true
	}
	state.keysHeld[key] = isDown
}

// RepeatKey records that the system repeated a held key (called by the backend).
func (state *State) RepeatKey(key Key) {
	if isValidKey(key) {
		state.keysTypedNow[key] = true
	}
}

// AddCharacter records a character typed, after the keyboard layout and dead keys (called by the
// backend): "ñ" is one character, whatever keys made it.
func (state *State) AddCharacter(character rune) {
	state.characters = append(state.characters, character)
}

// SetButton records a mouse button transition (called by the backend).
func (state *State) SetButton(button MouseButton, isDown bool) {
	if !isValidButton(button) {
		return
	}
	if isDown && !state.buttonsHeld[button] {
		state.buttonsPressedNow[button] = true
		state.clicks[button] = state.countClicks(button)
	}
	if !isDown && state.buttonsHeld[button] {
		state.buttonsReleasedNow[button] = true
	}
	state.buttonsHeld[button] = isDown
}

// countClicks numbers a press: one more than the previous press when it was the same button,
// soon enough and close enough, else 1.
func (state *State) countClicks(button MouseButton) int {
	now := state.now()
	previous := state.lastPress
	isRepeat := previous.count > 0 && previous.button == button &&
		now.Sub(previous.at) <= MultiClickInterval &&
		abs(state.CursorX-previous.x) <= MultiClickDistance && abs(state.CursorY-previous.y) <= MultiClickDistance
	count := 1
	if isRepeat {
		count = previous.count + 1
	}
	state.lastPress = pressRecord{button: button, at: now, x: state.CursorX, y: state.CursorY, count: count}
	return count
}

func (state *State) now() time.Time {
	if state.Now != nil {
		return state.Now()
	}
	return time.Now()
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

// MoveCursor records a new cursor position (called by the backend).
func (state *State) MoveCursor(x, y float64) {
	if state.hasCursorPosition {
		state.CursorDeltaX += x - state.CursorX
		state.CursorDeltaY += y - state.CursorY
	}
	state.CursorX, state.CursorY, state.hasCursorPosition = x, y, true
}

// AddScroll records scroll movement (called by the backend).
func (state *State) AddScroll(delta float64) { state.ScrollDelta += delta }

// AddHorizontalScroll records sideways scroll movement (called by the backend).
func (state *State) AddHorizontalScroll(delta float64) { state.ScrollDeltaX += delta }

// Down reports whether a key is held.
func (state *State) Down(key Key) bool { return isValidKey(key) && state.keysHeld[key] }

// Pressed reports whether a key went down during this frame.
func (state *State) Pressed(key Key) bool { return isValidKey(key) && state.keysPressedNow[key] }

// Typed reports whether a key went down or was repeated by the system during this step: what
// text fields use for Backspace or the arrows, so holding them keeps acting.
func (state *State) Typed(key Key) bool { return isValidKey(key) && state.keysTypedNow[key] }

// TypedText returns the characters typed during this step, in order.
func (state *State) TypedText() string { return string(state.characters) }

// Button reports whether a mouse button is held.
func (state *State) Button(button MouseButton) bool {
	return isValidButton(button) && state.buttonsHeld[button]
}

// ButtonPressed reports whether a mouse button went down during this step.
func (state *State) ButtonPressed(button MouseButton) bool {
	return isValidButton(button) && state.buttonsPressedNow[button]
}

// ButtonReleased reports whether a mouse button went up during this step.
func (state *State) ButtonReleased(button MouseButton) bool {
	return isValidButton(button) && state.buttonsReleasedNow[button]
}

// Clicks says which click a press during this step was: 1 for a single click, 2 for the second
// press of a double click, 3 for a triple click; 0 when the button was not pressed.
func (state *State) Clicks(button MouseButton) int {
	if !state.ButtonPressed(button) {
		return 0
	}
	return state.clicks[button]
}

// ClipboardText returns the text on the system clipboard, or "" without one.
func (state *State) ClipboardText() string {
	if state.Clipboard == nil {
		return ""
	}
	return state.Clipboard.ClipboardText()
}

// SetClipboardText puts a text on the system clipboard, when there is one.
func (state *State) SetClipboardText(text string) {
	if state.Clipboard != nil {
		state.Clipboard.SetClipboardText(text)
	}
}

// SetCursorShape asks the backend to show the pointer with a shape; it stays until changed.
func (state *State) SetCursorShape(shape CursorShape) { state.cursorShape = shape }

// CursorShape is the pointer shape the application asked for.
func (state *State) CursorShape() CursorShape { return state.cursorShape }

// LockCursor asks the backend to hide and capture the cursor (first-person look) or release it.
func (state *State) LockCursor(lock bool) { state.cursorLockRequested = lock }

// CursorLocked reports whether the application asked for a captured cursor.
func (state *State) CursorLocked() bool { return state.cursorLockRequested }

// ClaimMouse marks the mouse as used by an interface during this step (the pointer is over a
// panel, or dragging a slider): cameras and shortcuts should leave it alone. See Unclaimed.
func (state *State) ClaimMouse() { state.mouseClaimed = true }

// ClaimKeyboard marks the keyboard as used by an interface during this step (a text field has
// the focus).
func (state *State) ClaimKeyboard() { state.keyboardClaimed = true }

// MouseClaimed tells whether an interface claimed the mouse during this step.
func (state *State) MouseClaimed() bool { return state.mouseClaimed }

// KeyboardClaimed tells whether an interface claimed the keyboard during this step.
func (state *State) KeyboardClaimed() bool { return state.keyboardClaimed }

// Unclaimed returns a copy of the state without what an interface claimed: no buttons, pointer
// movement or scroll when it has the mouse, no keys or characters when it has the keyboard. The
// viewer gives this copy to its cameras and shortcuts.
func (state *State) Unclaimed() State {
	copied := *state
	copied.characters = nil
	if state.mouseClaimed {
		copied.buttonsHeld, copied.buttonsPressedNow, copied.buttonsReleasedNow = [mouseButtonCount]bool{}, [mouseButtonCount]bool{}, [mouseButtonCount]bool{}
		copied.CursorDeltaX, copied.CursorDeltaY, copied.ScrollDelta, copied.ScrollDeltaX = 0, 0, 0, 0
	}
	if state.keyboardClaimed {
		copied.keysHeld, copied.keysPressedNow, copied.keysTypedNow = [keyCount]bool{}, [keyCount]bool{}, [keyCount]bool{}
	} else {
		copied.characters = state.characters
	}
	return copied
}

// Time is the time now, by Now when it is set (a test's fake clock).
func (state *State) Time() time.Time { return state.now() }

// RequestFrame asks the backend for another frame within a delay, even if no input arrives: the
// next step of an animation, a caret's blink, a tooltip due. Backends that draw only on demand
// sleep until input arrives or the earliest of these comes; the others draw every frame anyway.
func (state *State) RequestFrame(delay time.Duration) {
	at := state.now().Add(max(delay, 0))
	if state.frameWanted.IsZero() || at.Before(state.frameWanted) {
		state.frameWanted = at
	}
}

// TakeFrameRequest returns how long until the frame asked for with RequestFrame (0 when it is
// due) and forgets the request; false when none was made (called by the backend).
func (state *State) TakeFrameRequest() (time.Duration, bool) {
	if state.frameWanted.IsZero() {
		return 0, false
	}
	delay := max(state.frameWanted.Sub(state.now()), 0)
	state.frameWanted = time.Time{}
	return delay, true
}

// HasInput tells whether anything arrived during this step: keys, characters, buttons, pointer
// movement or scrolling.
func (state *State) HasInput() bool {
	return state.keysTypedNow != [keyCount]bool{} || len(state.characters) > 0 ||
		state.buttonsPressedNow != [mouseButtonCount]bool{} || state.buttonsReleasedNow != [mouseButtonCount]bool{} ||
		state.CursorDeltaX != 0 || state.CursorDeltaY != 0 || state.ScrollDelta != 0 || state.ScrollDeltaX != 0
}

// EndFrame clears per-frame movement, presses, typed characters and claims; the backend calls it
// after each update step.
func (state *State) EndFrame() {
	state.CursorDeltaX, state.CursorDeltaY, state.ScrollDelta, state.ScrollDeltaX = 0, 0, 0, 0
	state.keysPressedNow = [keyCount]bool{}
	state.keysTypedNow = [keyCount]bool{}
	state.buttonsPressedNow = [mouseButtonCount]bool{}
	state.buttonsReleasedNow = [mouseButtonCount]bool{}
	state.characters = state.characters[:0]
	state.mouseClaimed, state.keyboardClaimed = false, false
}
