package opengl

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
}

var mouseButtonsFromGLFW = map[glfw.MouseButton]input.MouseButton{
	glfw.MouseButtonLeft:   input.MouseLeft,
	glfw.MouseButtonRight:  input.MouseRight,
	glfw.MouseButtonMiddle: input.MouseMiddle,
}

// forwardInputEvents sends GLFW keyboard, mouse and scroll events into an input.State.
func forwardInputEvents(window *glfw.Window, state *input.State) {
	window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if mapped, known := keysFromGLFW[key]; known && action != glfw.Repeat {
			state.SetKey(mapped, action == glfw.Press)
		}
	})
	window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		if mapped, known := mouseButtonsFromGLFW[button]; known {
			state.SetButton(mapped, action == glfw.Press)
		}
	})
	window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) { state.MoveCursor(x, y) })
	window.SetScrollCallback(func(_ *glfw.Window, _, verticalScroll float64) { state.AddScroll(verticalScroll) })
	updateWindowSizes(window, state)
}

// updateWindowSizes copies the framebuffer and window sizes into the input state.
func updateWindowSizes(window *glfw.Window, state *input.State) {
	state.FramebufferWidth, state.FramebufferHeight = window.GetFramebufferSize()
	state.WindowWidth, state.WindowHeight = window.GetSize()
}

// applyCursorLock captures or releases the cursor when the application changes its request.
func applyCursorLock(window *glfw.Window, state *input.State, currentlyLocked *bool) {
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
