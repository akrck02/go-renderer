//go:build js && wasm

// Package browser runs go-renderer programs in a web browser, built with GOOS=js GOARCH=wasm: a
// canvas stands in for the window, its DOM events become an input.State, WebGL2 draws through the
// gles renderers, and requestAnimationFrame drives the same fixed-step loop as the desktop, drawing
// only when something happens.
package browser

import (
	"errors"
	"math"
	"syscall/js"

	"github.com/akrck02/go-renderer/input"
)

// Canvas is the page's drawing surface and the source of its input. It fills its place in the
// page (its CSS size) and draws at the screen's density, so text stays sharp on high-density
// screens.
type Canvas struct {
	element   js.Value
	state     *input.State
	listeners []js.Func
	cursor    input.CursorShape
	// changed is called after every event, so the loop can schedule a frame.
	changed func()
}

// ErrNoCanvas is returned when the page has no canvas with the given id.
var ErrNoCanvas = errors.New("browser: no canvas with that id in the page")

// OpenCanvas takes the page's canvas with this id (the first canvas when id is empty).
func OpenCanvas(id string) (*Canvas, error) {
	document := js.Global().Get("document")
	var element js.Value
	if id == "" {
		element = document.Call("querySelector", "canvas")
	} else {
		element = document.Call("getElementById", id)
	}
	if element.IsNull() || element.IsUndefined() {
		return nil, ErrNoCanvas
	}
	// the canvas takes keyboard focus, and touches on it are the program's, not the page's scroll
	element.Set("tabIndex", 0)
	element.Get("style").Set("touchAction", "none")
	element.Get("style").Set("outline", "none")
	return &Canvas{element: element, changed: func() {}}, nil
}

// WebGL returns the canvas's WebGL2 context as gles.Functions. antialias smooths the edges of 3D
// geometry; a flat interface does not need it.
func (canvas *Canvas) WebGL(antialias bool) (*WebGL, error) {
	return newWebGL(canvas.element, antialias)
}

// Attach makes the canvas fill an input.State from now on: pointer, wheel, keyboard and size.
func (canvas *Canvas) Attach(state *input.State) {
	canvas.state = state
	state.Clipboard = &pageClipboard{}
	canvas.updateSize()
	canvas.listen(canvas.element, "pointermove", canvas.pointerMoved)
	canvas.listen(canvas.element, "pointerdown", canvas.pointerPressed)
	canvas.listen(canvas.element, "pointerup", canvas.pointerReleased)
	canvas.listen(canvas.element, "pointercancel", canvas.pointerReleased)
	canvas.listen(canvas.element, "wheel", canvas.wheelTurned)
	canvas.listen(canvas.element, "contextmenu", func(event js.Value) { event.Call("preventDefault") })
	canvas.listen(canvas.element, "keydown", canvas.keyPressed)
	canvas.listen(canvas.element, "keyup", canvas.keyReleased)
	canvas.listen(canvas.element, "blur", func(js.Value) { canvas.releaseEverything() })
	canvas.listen(js.Global().Get("document"), "paste", canvas.pasted)
	canvas.observeSize()
}

// listen adds an event listener that cannot be passive, so it may stop the page's own reaction
// (scrolling on a wheel turn, moving focus on Tab).
func (canvas *Canvas) listen(target js.Value, event string, handle func(event js.Value)) {
	function := js.FuncOf(func(_ js.Value, arguments []js.Value) any {
		handle(arguments[0])
		canvas.changed()
		return nil
	})
	canvas.listeners = append(canvas.listeners, function)
	target.Call("addEventListener", event, function, map[string]any{"passive": false})
}

func (canvas *Canvas) pointerMoved(event js.Value) {
	canvas.state.MoveCursor(event.Get("offsetX").Float(), event.Get("offsetY").Float())
}

func (canvas *Canvas) pointerPressed(event js.Value) {
	canvas.element.Call("focus")
	// keep receiving the pointer's events while it is held, even outside the canvas (a drag)
	canvas.element.Call("setPointerCapture", event.Get("pointerId"))
	canvas.pointerMoved(event)
	if button, known := mouseButtonFrom(event.Get("button").Int()); known {
		canvas.state.SetButton(button, true)
	}
	event.Call("preventDefault")
}

func (canvas *Canvas) pointerReleased(event js.Value) {
	canvas.pointerMoved(event)
	if button, known := mouseButtonFrom(event.Get("button").Int()); known {
		canvas.state.SetButton(button, false)
	}
}

func (canvas *Canvas) wheelTurned(event js.Value) {
	mode := event.Get("deltaMode").Int()
	canvas.state.AddScroll(wheelLines(event.Get("deltaY").Float(), mode))
	canvas.state.AddHorizontalScroll(wheelLines(event.Get("deltaX").Float(), mode))
	event.Call("preventDefault")
}

func (canvas *Canvas) keyPressed(event js.Value) {
	shortcut := event.Get("ctrlKey").Bool() || event.Get("metaKey").Bool()
	if key, known := keyFromCode(event.Get("code").String()); known {
		if event.Get("repeat").Bool() {
			canvas.state.RepeatKey(key)
		} else {
			canvas.state.SetKey(key, true)
		}
	}
	if character, typed := typedCharacter(event.Get("key").String(), shortcut); typed {
		canvas.state.AddCharacter(character)
	}
	if !keepsDefault(event, shortcut) {
		event.Call("preventDefault")
	}
}

// keepsDefault tells whether the browser should still act on a key: copy, cut and paste, so the
// paste event brings the system clipboard's text, and reloading the page.
func keepsDefault(event js.Value, shortcut bool) bool {
	if !shortcut {
		return false
	}
	switch event.Get("code").String() {
	case "KeyC", "KeyX", "KeyV", "KeyR":
		return true
	}
	return false
}

func (canvas *Canvas) keyReleased(event js.Value) {
	if key, known := keyFromCode(event.Get("code").String()); known {
		canvas.state.SetKey(key, false)
	}
}

// releaseEverything lets go of held keys and buttons when the canvas loses focus: their release
// would otherwise never arrive, and a camera would keep turning.
func (canvas *Canvas) releaseEverything() {
	for key := input.KeyUnknown + 1; key <= input.KeySuper; key++ { // KeySuper is the last key
		if canvas.state.Down(key) {
			canvas.state.SetKey(key, false)
		}
	}
	for _, button := range []input.MouseButton{input.MouseLeft, input.MouseMiddle, input.MouseRight} {
		if canvas.state.Button(button) {
			canvas.state.SetButton(button, false)
		}
	}
}

func (canvas *Canvas) pasted(event js.Value) {
	text := event.Get("clipboardData").Call("getData", "text")
	if clipboard, ok := canvas.state.Clipboard.(*pageClipboard); ok {
		clipboard.text = text.String()
	}
}

// observeSize keeps the drawing buffer the canvas's size on screen times the screen's density.
func (canvas *Canvas) observeSize() {
	resized := js.FuncOf(func(js.Value, []js.Value) any {
		canvas.updateSize()
		canvas.changed()
		return nil
	})
	canvas.listeners = append(canvas.listeners, resized)
	js.Global().Get("ResizeObserver").New(resized).Call("observe", canvas.element)
}

// updateSize sets the canvas's buffer to its size on screen in device pixels and tells the input:
// the cursor is in CSS pixels (the window size), the frame in device pixels (the framebuffer).
func (canvas *Canvas) updateSize() {
	density := js.Global().Get("devicePixelRatio").Float()
	if density <= 0 {
		density = 1
	}
	width := canvas.element.Get("clientWidth").Int()
	height := canvas.element.Get("clientHeight").Int()
	framebufferWidth := int(math.Round(float64(width) * density))
	framebufferHeight := int(math.Round(float64(height) * density))
	if canvas.element.Get("width").Int() != framebufferWidth || canvas.element.Get("height").Int() != framebufferHeight {
		canvas.element.Set("width", framebufferWidth)
		canvas.element.Set("height", framebufferHeight)
	}
	canvas.state.WindowWidth, canvas.state.WindowHeight = width, height
	canvas.state.FramebufferWidth, canvas.state.FramebufferHeight = framebufferWidth, framebufferHeight
}

// applyCursor shows the pointer shape the program asked for.
func (canvas *Canvas) applyCursor() {
	shape := canvas.state.CursorShape()
	if shape == canvas.cursor {
		return
	}
	canvas.cursor = shape
	canvas.element.Get("style").Set("cursor", cursorStyles[shape])
}

// pageClipboard keeps copied text for the page. Copying also offers it to the system clipboard;
// reading the system's needs a permission, so pasting takes the text the browser's paste event
// brings (Control or Command and V).
type pageClipboard struct{ text string }

func (clipboard *pageClipboard) ClipboardText() string { return clipboard.text }

func (clipboard *pageClipboard) SetClipboardText(text string) {
	clipboard.text = text
	system := js.Global().Get("navigator").Get("clipboard")
	if system.IsUndefined() {
		return
	}
	system.Call("writeText", text) // a promise; failing (no permission) leaves the page's copy
}
