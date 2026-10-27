//go:build js && wasm

// Command web-emoji draws a few labels with emoji in a web browser, through WebGL2, to see that a
// page draws them in colour as Android and the desktop do. Build and serve it with:
//
//	GOOS=js GOARCH=wasm go build -o cmd/web-emoji/emoji.wasm ./cmd/web-emoji
//	cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" cmd/web-emoji/
//	python3 -m http.server -d cmd/web-emoji 8080
package main

import (
	"log"

	"github.com/akrck02/go-renderer/browser"
	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/window"
)

// Where the labels go and how big they are, in layout pixels.
const (
	margin     = 24
	lineHeight = 56
	textSize   = 28
	smallSize  = 16
	firstLine  = 48
)

var (
	background = draw2d.MustHex("#f4f2ef")
	ink        = draw2d.MustHex("#1c1f24")
)

// lines are drawn one under another: emoji by default, one asked for with U+FE0F, and a small size.
var lines = []struct {
	text string
	size float32
}{
	{"North light 🌊 😀 🚀", textSize},
	{"A heart asked as emoji: ❤️", textSize},
	{"Small and inline: 👍 done ✅", smallSize},
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	canvas, err := browser.OpenCanvas("view")
	if err != nil {
		return err
	}
	functions, err := canvas.WebGL(true)
	if err != nil {
		return err
	}
	renderer, err := gles.NewFlatRenderer(functions, overlay.Fonts{})
	if err != nil {
		return err
	}
	app := &window.Application{Input: input.NewState(), Options: window.Options{DrawOnDemand: true}}
	app.Update = func(*window.Application, float64) error { return nil }
	var list draw2d.List
	draw := func() error {
		state := app.Input
		scale := float32(state.FramebufferWidth) / float32(max(state.WindowWidth, 1))
		list.Reset()
		for index, line := range lines {
			baseline := float32(firstLine + index*lineHeight)
			list.Text(line.text, margin, baseline, line.size, draw2d.Regular, ink)
		}
		renderer.Draw(&list, background, state.FramebufferWidth, state.FramebufferHeight, scale)
		return nil
	}
	return browser.Run(canvas, app, draw)
}
