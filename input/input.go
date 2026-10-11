// Package input holds a backend-independent snapshot of keyboard and mouse state.
//
// Window backends (GLFW for OpenGL today, others later) feed events into a State;
// applications, camera controllers and simulations only read from it.
package input

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

// State is the input state for the current frame.
type State struct {
	keysHeld            [keyCount]bool
	keysPressedNow      [keyCount]bool
	buttonsHeld         [mouseButtonCount]bool
	CursorX, CursorY    float64 // cursor position in window coordinates
	CursorDeltaX        float64 // cursor movement since the previous frame
	CursorDeltaY        float64
	ScrollDelta         float64 // vertical scroll since the previous frame
	hasCursorPosition   bool
	cursorLockRequested bool
	FramebufferWidth    int // updated by the backend
	FramebufferHeight   int
	WindowWidth         int // window size in cursor coordinates
	WindowHeight        int
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
	}
	state.keysHeld[key] = isDown
}

// SetButton records a mouse button transition (called by the backend).
func (state *State) SetButton(button MouseButton, isDown bool) {
	if isValidButton(button) {
		state.buttonsHeld[button] = isDown
	}
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

// Down reports whether a key is held.
func (state *State) Down(key Key) bool { return isValidKey(key) && state.keysHeld[key] }

// Pressed reports whether a key went down during this frame.
func (state *State) Pressed(key Key) bool { return isValidKey(key) && state.keysPressedNow[key] }

// Button reports whether a mouse button is held.
func (state *State) Button(button MouseButton) bool {
	return isValidButton(button) && state.buttonsHeld[button]
}

// LockCursor asks the backend to hide and capture the cursor (first-person look) or release it.
func (state *State) LockCursor(lock bool) { state.cursorLockRequested = lock }

// CursorLocked reports whether the application asked for a captured cursor.
func (state *State) CursorLocked() bool { return state.cursorLockRequested }

// EndFrame clears per-frame movement and presses; the backend calls it after each update step.
func (state *State) EndFrame() {
	state.CursorDeltaX, state.CursorDeltaY, state.ScrollDelta = 0, 0, 0
	state.keysPressedNow = [keyCount]bool{}
}
