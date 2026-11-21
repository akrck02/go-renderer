package windowloop

import (
	"math"

	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/akrck02/go-renderer/input"
)

var keysFromGLFW = map[glfw.Key]input.Key{
	glfw.KeyA: input.KeyA, glfw.KeyB: input.KeyB, glfw.KeyC: input.KeyC, glfw.KeyD: input.KeyD, glfw.KeyE: input.KeyE,
	glfw.KeyF: input.KeyF, glfw.KeyG: input.KeyG, glfw.KeyH: input.KeyH, glfw.KeyI: input.KeyI, glfw.KeyJ: input.KeyJ,
	glfw.KeyK: input.KeyK, glfw.KeyL: input.KeyL, glfw.KeyM: input.KeyM, glfw.KeyN: input.KeyN, glfw.KeyO: input.KeyO,
	glfw.KeyP: input.KeyP, glfw.KeyQ: input.KeyQ, glfw.KeyR: input.KeyR, glfw.KeyS: input.KeyS, glfw.KeyT: input.KeyT,
	glfw.KeyU: input.KeyU, glfw.KeyV: input.KeyV, glfw.KeyW: input.KeyW, glfw.KeyX: input.KeyX, glfw.KeyY: input.KeyY,
	glfw.KeyZ: input.KeyZ,
	glfw.Key0: input.Key0, glfw.Key1: input.Key1, glfw.Key2: input.Key2, glfw.Key3: input.Key3, glfw.Key4: input.Key4,
	glfw.Key5: input.Key5, glfw.Key6: input.Key6, glfw.Key7: input.Key7, glfw.Key8: input.Key8, glfw.Key9: input.Key9,
	glfw.KeyUp: input.KeyUp, glfw.KeyDown: input.KeyDown, glfw.KeyLeft: input.KeyLeft, glfw.KeyRight: input.KeyRight,
	glfw.KeySpace: input.KeySpace, glfw.KeyTab: input.KeyTab, glfw.KeyEnter: input.KeyEnter, glfw.KeyEscape: input.KeyEscape,
	glfw.KeyLeftShift: input.KeyShift, glfw.KeyRightShift: input.KeyShift,
	glfw.KeyLeftControl: input.KeyControl, glfw.KeyRightControl: input.KeyControl,
	glfw.KeyLeftAlt: input.KeyAlt, glfw.KeyRightAlt: input.KeyAlt,
	glfw.KeyEqual: input.KeyPlus, glfw.KeyKPAdd: input.KeyPlus, glfw.KeyMinus: input.KeyMinus, glfw.KeyKPSubtract: input.KeyMinus,
	glfw.KeyPageUp: input.KeyPageUp, glfw.KeyPageDown: input.KeyPageDown,
	glfw.KeyLeftBracket: input.KeyLeftBracket, glfw.KeyRightBracket: input.KeyRightBracket,
	glfw.KeyComma: input.KeyComma, glfw.KeyPeriod: input.KeyPeriod,
	glfw.KeyBackspace: input.KeyBackspace, glfw.KeyDelete: input.KeyDelete,
	glfw.KeyHome: input.KeyHome, glfw.KeyEnd: input.KeyEnd,
	glfw.KeyLeftSuper: input.KeySuper, glfw.KeyRightSuper: input.KeySuper,
	glfw.KeyF1: input.KeyF1, glfw.KeyF2: input.KeyF2, glfw.KeyF3: input.KeyF3, glfw.KeyF4: input.KeyF4,
	glfw.KeyF5: input.KeyF5, glfw.KeyF6: input.KeyF6, glfw.KeyF7: input.KeyF7, glfw.KeyF8: input.KeyF8,
	glfw.KeyF9: input.KeyF9, glfw.KeyF10: input.KeyF10, glfw.KeyF11: input.KeyF11, glfw.KeyF12: input.KeyF12,
}

// keysByCharacter are keys known by the character they write on the keyboard's own layout, not by
// where they sit: GLFW names keys after the US layout, where + is on the = key, but on a Spanish
// keyboard + has a key of its own, where the US layout has ], and - sits where it has /.
var keysByCharacter = map[string]input.Key{"+": input.KeyPlus, "=": input.KeyPlus, "-": input.KeyMinus}

// keyOf is the input key a GLFW key stands for: by the character it writes on the keyboard's layout
// (name, as glfw.GetKeyName gives it) when that is one of keysByCharacter, or else by its place.
func keyOf(key glfw.Key, name string) (input.Key, bool) {
	if mapped, found := keysByCharacter[name]; found {
		return mapped, true
	}
	mapped, known := keysFromGLFW[key]
	return mapped, known
}

var cursorShapesToGLFW = map[input.CursorShape]glfw.StandardCursor{
	input.CursorArrow:            glfw.ArrowCursor,
	input.CursorText:             glfw.IBeamCursor,
	input.CursorHand:             glfw.HandCursor,
	input.CursorResizeHorizontal: glfw.HResizeCursor,
	input.CursorResizeVertical:   glfw.VResizeCursor,
	input.CursorMove:             glfw.ResizeAllCursor,
}

var mouseButtonsFromGLFW = map[glfw.MouseButton]input.MouseButton{
	glfw.MouseButtonLeft:   input.MouseLeft,
	glfw.MouseButtonRight:  input.MouseRight,
	glfw.MouseButtonMiddle: input.MouseMiddle,
}

// ForwardInput sends GLFW keyboard, mouse, scroll and text events into an input.State, and
// gives it the system clipboard.
func ForwardInput(window *glfw.Window, state *input.State) {
	window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, scancode int, action glfw.Action, _ glfw.ModifierKey) {
		mapped, known := keyOf(key, glfw.GetKeyName(key, scancode))
		if !known {
			return
		}
		if action == glfw.Repeat {
			state.Feed().RepeatKey(mapped)
			return
		}
		state.Feed().SetKey(mapped, action == glfw.Press)
	})
	window.SetCharCallback(func(_ *glfw.Window, character rune) { state.Feed().AddCharacter(character) })
	window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		if mapped, known := mouseButtonsFromGLFW[button]; known {
			state.Feed().SetButton(mapped, action == glfw.Press)
		}
	})
	window.SetCursorPosCallback(func(moved *glfw.Window, x, y float64) {
		shrink := missingScaleOf(moved) * zoom
		state.Feed().MoveCursor(x/shrink, y/shrink)
	})
	window.SetScrollCallback(func(_ *glfw.Window, horizontalScroll, verticalScroll float64) {
		state.Feed().AddScroll(verticalScroll)
		state.Feed().AddHorizontalScroll(horizontalScroll)
	})
	window.SetContentScaleCallback(func(scaled *glfw.Window, _, _ float32) { UpdateSizes(scaled, state) })
	state.Clipboard = windowClipboard{window}
	UpdateSizes(window, state)
}

// On macOS and Wayland a window on a dense screen is measured in points, and its framebuffer has
// more pixels than the window has points: the framebuffer carries the screen's scale. On X11 (and
// Windows) a window is measured in pixels, framebuffer and window match, and the scale is only in
// the window's content scale (Xft.dpi on X11). An interface laid out by the window's size would then
// be drawn at half size on a screen of twice the density. So the window's size and the cursor are
// given in layout pixels: divided by the scale the framebuffer does not carry.

// zoom is how much larger than the screen's scale the program draws, 1 unless SetZoom changed it;
// appliedZoom is the zoom the input state's sizes were last given with. Only the main thread uses
// them.
var zoom, appliedZoom = 1.0, 1.0

// SetZoom makes the program draw that many times larger than the screen's scale (1 for its own
// size), as a browser's zoom does: the window's size and the cursor reach the input state divided
// by it, so an interface laid out by them grows. The next frame comes at once with the new size.
// Call it on the main thread, from an update step.
func SetZoom(factor float64) {
	if factor <= 0 || factor == zoom {
		return
	}
	zoom = factor
	Wake()
}

// Zoom is the zoom SetZoom gave, 1 by default.
func Zoom() float64 { return zoom }

// missingScales are, for each window with forwarded input, the scale its framebuffer does not
// carry (see missingScale). Only the main thread uses them, where GLFW's callbacks run.
var missingScales = map[*glfw.Window]float64{}

// missingScaleOf is the scale a window's framebuffer does not carry, as UpdateSizes last found it.
func missingScaleOf(window *glfw.Window) float64 {
	if missing, found := missingScales[window]; found {
		return missing
	}
	return 1
}

// missingScale is how much of the screen's content scale a window's framebuffer does not carry:
// 2 for a window on X11 on a screen of twice the density; 1 where the framebuffer has the scale's
// pixels already (macOS, Wayland), or where the screen is not scaled.
func missingScale(framebufferWidth, windowWidth int, contentScale float32) float64 {
	if framebufferWidth <= 0 || windowWidth <= 0 {
		return 1
	}
	carried := float64(framebufferWidth) / float64(windowWidth)
	return max(float64(contentScale)/carried, 1)
}

// UpdateSizes copies the framebuffer and window sizes into the input state, the window's in layout
// pixels: divided by the scale the framebuffer does not carry, and by the zoom.
func UpdateSizes(window *glfw.Window, state *input.State) {
	state.FramebufferWidth, state.FramebufferHeight = window.GetFramebufferSize()
	width, height := window.GetSize()
	contentScale, _ := window.GetContentScale()
	missing := missingScale(state.FramebufferWidth, width, contentScale)
	missingScales[window], appliedZoom = missing, zoom
	shrink := missing * zoom
	state.WindowWidth = int(math.Round(float64(width) / shrink))
	state.WindowHeight = int(math.Round(float64(height) / shrink))
}

// ApplyCursorLock captures or releases the cursor when the application changes its request.
func ApplyCursorLock(window *glfw.Window, state *input.State, currentlyLocked *bool) {
	if state.CursorLocked() == *currentlyLocked {
		return
	}
	*currentlyLocked = state.CursorLocked()
	if *currentlyLocked {
		window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	} else {
		window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	}
}

// windowClipboard is the system clipboard through GLFW; it must be used on the main thread, where
// update steps run.
type windowClipboard struct{ window *glfw.Window }

func (clipboard windowClipboard) ClipboardText() string { return clipboard.window.GetClipboardString() }

func (clipboard windowClipboard) SetClipboardText(text string) {
	clipboard.window.SetClipboardString(text)
}

// CursorShapes shows the pointer shape the application asks for, creating each system cursor once.
type CursorShapes struct {
	current input.CursorShape
	cursors map[input.CursorShape]*glfw.Cursor
}

// Apply shows the pointer shape the state asks for, if it is not already shown.
func (shapes *CursorShapes) Apply(window *glfw.Window, state *input.State) {
	wanted := state.CursorShape()
	if wanted == shapes.current {
		return
	}
	shapes.current = wanted
	if shapes.cursors == nil {
		shapes.cursors = map[input.CursorShape]*glfw.Cursor{}
	}
	cursor, created := shapes.cursors[wanted]
	if !created {
		standard, known := cursorShapesToGLFW[wanted]
		if !known {
			standard = glfw.ArrowCursor
		}
		cursor = glfw.CreateStandardCursor(standard)
		shapes.cursors[wanted] = cursor
	}
	window.SetCursor(cursor)
}
