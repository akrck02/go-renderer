# input

```go
import "github.com/akrck02/go-renderer/input"
```

Package input holds a backend-independent snapshot of keyboard and mouse state.

Window backends (GLFW for OpenGL today, others later) feed events into a State;
applications, camera controllers and simulations only read from it.

Besides what is held, a State remembers what happened since the last update
step: keys and buttons that went down or up, keys repeated by the system,
the characters typed and how many clicks a press makes (a double click).
An interface drawn over the view can claim the mouse or the keyboard for a step,
so the camera does not move while a slider is dragged or a name typed.

### Constants and variables

```go
const (
	MultiClickInterval = 400 * time.Millisecond
	MultiClickDistance = 4.0
)
```

MultiClickInterval is the longest pause between the presses of a double click, and MultiClickDistance how far, in window coordinates, the pointer may move between them.

### Clipboard

Clipboard is the system clipboard, given by the backend.

```go
type Clipboard interface {
	ClipboardText() string
	SetClipboardText(text string)
}
```

### CursorShape

CursorShape is the look of the mouse pointer.

```go
type CursorShape int
```

```go
const (
	CursorArrow            CursorShape = iota // the default pointer
	CursorText                                // an I-beam, over text that can be edited
	CursorHand                                // a pointing hand, over something to click
	CursorResizeHorizontal                    // a left-right arrow, over an edge to drag sideways
	CursorResizeVertical                      // an up-down arrow, over an edge to drag up or down
	CursorMove                                // four arrows, over something to drag around
)
```

### Gesture

Gesture is what two fingers did since the last step.

```go
type Gesture struct {
	Spread float64 // the distance between them now over before: above 1 they moved apart
	Turn   float64 // radians they turned around their middle, counterclockwise on screen
	PanX   float64 // how far their middle moved, in layout pixels
	PanY   float64
	// CenterX and CenterY are where their middle is now, in layout pixels: what a pinch comes
	// closer to
	CenterX, CenterY float64
}
```

### Key

Key is a backend-independent key code.

```go
type Key int
```

```go
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
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)
```

### MouseButton

MouseButton is a backend-independent mouse button.

```go
type MouseButton int
```

```go
const (
	MouseLeft MouseButton = iota
	MouseRight
	MouseMiddle
)
```

### PadButton

PadButton is a game pad's button.

```go
type PadButton int
```

```go
const (
	PadA PadButton = iota
	PadB
	PadX
	PadY
	PadLeftShoulder
	PadRightShoulder
	PadBack
	PadStart
	PadGuide
	PadLeftThumb // pressing the left stick in
	PadRightThumb
	PadUp
	PadRight
	PadDown
	PadLeft
)
```

### State

State is the input state for the current frame.

```go
type State struct {
	CursorX, CursorY float64 // cursor position in window coordinates
	CursorDeltaX     float64 // cursor movement since the previous frame
	CursorDeltaY     float64
	ScrollDelta      float64 // vertical scroll since the previous frame
	ScrollDeltaX     float64 // horizontal scroll since the previous frame (trackpads, tilting wheels)
	// StickX and StickY are a game pad's main stick where there is one (a console's circle pad, a
	// pad's left stick), from -1 to 1 each way, right and up positive; 0 at rest or without one.
	// RightStickX and RightStickY are its second stick (a New 3DS's C-stick, a pad's right one).
	// The backend sets them every frame.
	StickX, StickY           float64
	RightStickX, RightStickY float64

	FramebufferWidth  int // updated by the backend
	FramebufferHeight int
	WindowWidth       int // window size in cursor coordinates
	WindowHeight      int
	// Clipboard is set by the backend; nil when there is no system clipboard (tests, headless).
	Clipboard Clipboard
	// Now tells the time, for double clicks; nil = time.Now. Tests set it to a fake clock.
	Now func() time.Time

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewState() *State` | NewState returns an empty input state. |
| `func (state *State) AddCharacter(character rune)` | AddCharacter records a character typed, after the keyboard layout and dead keys (called by the backend): "ñ" is one character, whatever keys made it. |
| `func (state *State) AddHorizontalScroll(delta float64)` | AddHorizontalScroll records sideways scroll movement (called by the backend). |
| `func (state *State) AddScroll(delta float64)` | AddScroll records scroll movement (called by the backend). |
| `func (state *State) AnyHeld() bool` | AnyHeld tells whether a key or a mouse button is being held down: a program that draws on demand keeps drawing while one is, since holding a key (walking forward) sends no events. |
| `func (state *State) BackUnanswered() bool` | BackUnanswered tells whether the last step had a back that nobody took, once, for the backend to leave the program. |
| `func (state *State) Button(button MouseButton) bool` | Button reports whether a mouse button is held. |
| `func (state *State) ButtonPressed(button MouseButton) bool` | ButtonPressed reports whether a mouse button went down during this step. |
| `func (state *State) ButtonReleased(button MouseButton) bool` | ButtonReleased reports whether a mouse button went up during this step. |
| `func (state *State) ClaimKeyboard()` | ClaimKeyboard marks the keyboard as used by an interface during this step (a text field has the focus). |
| `func (state *State) ClaimMouse()` | ClaimMouse marks the mouse as used by an interface during this step (the pointer is over a panel, or dragging a slider): cameras and shortcuts should leave it alone. |
| `func (state *State) ClearTouches()` | ClearTouches takes every finger off, when the system takes a gesture over (called by the backend). |
| `func (state *State) Clicks(button MouseButton) int` | Clicks says which click a press during this step was: 1 for a single click, 2 for the second press of a double click, 3 for a triple click; 0 when the button was not pressed. |
| `func (state *State) ClipboardText() string` | ClipboardText returns the text on the system clipboard, or "" without one. |
| `func (state *State) CursorLocked() bool` | CursorLocked reports whether the application asked for a captured cursor. |
| `func (state *State) CursorShape() CursorShape` | CursorShape is the pointer shape the application asked for. |
| `func (state *State) Down(key Key) bool` | Down reports whether a key is held. |
| `func (state *State) EndFrame()` | EndFrame clears per-frame movement, presses, typed characters and claims; the backend calls it after each update step. |
| `func (state *State) Fullscreen() bool` | Fullscreen reports whether the window is asked to be on the whole screen. |
| `func (state *State) HasInput() bool` | HasInput tells whether anything arrived during this step: keys, characters, buttons, pointer movement or scrolling. |
| `func (state *State) KeyboardClaimed() bool` | KeyboardClaimed tells whether an interface claimed the keyboard during this step. |
| `func (state *State) LiftTouch(finger int64)` | LiftTouch takes a finger off the screen (called by the backend). |
| `func (state *State) LockCursor(lock bool)` | LockCursor asks the backend to hide and capture the cursor (first-person look) or release it. |
| `func (state *State) MouseClaimed() bool` | MouseClaimed tells whether an interface claimed the mouse during this step. |
| `func (state *State) MoveCursor(x, y float64)` | MoveCursor records a new cursor position (called by the backend). |
| `func (state *State) PadDown(button PadButton) bool` | PadDown tells whether a game pad's button is held. |
| `func (state *State) PadPressed(button PadButton) bool` | PadPressed tells whether a game pad's button went down this frame. |
| `func (state *State) PointerLeaves()` | PointerLeaves takes the pointer off the window once this frame ends: a finger lifted from a touch screen leaves nothing under it, so no control stays lit as if a mouse rested on it. |
| `func (state *State) PressBack()` | PressBack records the system's back: a phone's back gesture or button (called by the backend). |
| `func (state *State) Pressed(key Key) bool` | Pressed reports whether a key went down during this frame. |
| `func (state *State) RepeatKey(key Key)` | RepeatKey records that the system repeated a held key (called by the backend). |
| `func (state *State) RequestFrame(delay time.Duration)` | RequestFrame asks the backend for another frame within a delay, even if no input arrives: the next step of an animation, a caret's blink, a tooltip due. |
| `func (state *State) SetButton(button MouseButton, isDown bool)` | SetButton records a mouse button transition (called by the backend). |
| `func (state *State) SetClipboardText(text string)` | SetClipboardText puts a text on the system clipboard, when there is one. |
| `func (state *State) SetCursorShape(shape CursorShape)` | SetCursorShape asks the backend to show the pointer with a shape; it stays until changed. |
| `func (state *State) SetFullscreen(on bool)` | SetFullscreen asks the backend to show the window on the whole screen, or in a window again. |
| `func (state *State) SetKey(key Key, isDown bool)` | SetKey records a key transition (called by the backend). |
| `func (state *State) SetPadButton(button PadButton, isDown bool)` | SetPadButton records a game pad's button as held or let go; going down presses it this frame. |
| `func (state *State) SetTouch(finger int64, x, y float64)` | SetTouch puts a finger down, or moves it (called by the backend). |
| `func (state *State) TakeBack() bool` | TakeBack answers the system's back in this step, and tells whether there was one to answer: the part of the program the person is in goes back (a dialog closes, a screen returns to the one before) and returns true; any other part asking after it gets false. |
| `func (state *State) TakeFrameRequest() (time.Duration, bool)` | TakeFrameRequest returns how long until the frame asked for with RequestFrame (0 when it is due) and forgets the request; false when none was made (called by the backend). |
| `func (state *State) Time() time.Time` | Time is the time now, by Now when it is set (a test's fake clock). |
| `func (state *State) Touches() []Touch` | Touches returns the fingers on the screen now: for gestures of more than one finger (pinch, twist). |
| `func (state *State) Typed(key Key) bool` | Typed reports whether a key went down or was repeated by the system during this step: what text fields use for Backspace or the arrows, so holding them keeps acting. |
| `func (state *State) TypedText() string` | TypedText returns the characters typed during this step, in order. |
| `func (state *State) Unclaimed() State` | Unclaimed returns a copy of the state without what an interface claimed: no buttons, pointer movement or scroll when it has the mouse, no keys or characters when it has the keyboard. |

### Touch

Touch is a finger on the screen: which one, and where, in layout pixels.

```go
type Touch struct {
	Finger int64
	X, Y   float64
}
```

### TouchPointer

TouchPointer turns fingers into the pointer every interface already understands: one finger down moves the cursor and presses the left button, lifting it releases the button. A second finger lets the pointer go: two fingers are a gesture (pinch, twist), not a click or a drag; the pointer comes back with the next finger down once all are lifted. Positions come in screen pixels and go to the state in layout pixels, divided by the screen's scale; every finger is also in the state's touches.

```go
type TouchPointer struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (pointer *TouchPointer) Begin(state *State, finger int64, x, y, scale float32)` | Begin puts a finger down: the first becomes the pointer, a second lets it go. |
| `func (pointer *TouchPointer) Cancel(state *State)` | Cancel lets go of every finger, when the system takes the gesture over (a swipe from the screen's edge): the button is released, so nothing stays pressed. |
| `func (pointer *TouchPointer) End(state *State, finger int64, x, y, scale float32)` | End lifts a finger: lifting the pointer's is a click where it was. |
| `func (pointer *TouchPointer) Move(state *State, finger int64, x, y, scale float32)` | Move follows a finger; the pointer follows its own. |

### TwoFingers

TwoFingers follows a gesture of two fingers from one step to the next: how much they spread (pinch), turned (twist) and moved together (pan).

```go
type TwoFingers struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (follower *TwoFingers) Step(state *State) (Gesture, bool)` | Step reads the state's touches and returns the gesture since the last step; false when there are not exactly two fingers, or they just came down (a gesture starts at the next step). |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
