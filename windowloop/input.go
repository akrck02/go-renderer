package windowloop

import (
	"github.com/akrck02/go-renderer/input"
	"github.com/go-gl/glfw/v3.4/glfw"
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
	window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		mapped, known := keysFromGLFW[key]
		if !known {
			return
		}
		if action == glfw.Repeat {
			state.RepeatKey(mapped)
			return
		}
		state.SetKey(mapped, action == glfw.Press)
	})
	window.SetCharCallback(func(_ *glfw.Window, character rune) { state.AddCharacter(character) })
	window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		if mapped, known := mouseButtonsFromGLFW[button]; known {
			state.SetButton(mapped, action == glfw.Press)
		}
	})
	window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) { state.MoveCursor(x, y) })
	window.SetScrollCallback(func(_ *glfw.Window, horizontalScroll, verticalScroll float64) {
		state.AddScroll(verticalScroll)
		state.AddHorizontalScroll(horizontalScroll)
	})
	state.Clipboard = windowClipboard{window}
	UpdateSizes(window, state)
}

// UpdateSizes copies the framebuffer and window sizes into the input state.
func UpdateSizes(window *glfw.Window, state *input.State) {
	state.FramebufferWidth, state.FramebufferHeight = window.GetFramebufferSize()
	state.WindowWidth, state.WindowHeight = window.GetSize()
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
