# In a web browser

**Status:** a first step (October 2026). The sample island and its labels draw in a browser
through WebGL2, and go-ui interfaces run there too (see go-ui's `docs/web.md`). The desktop is
unchanged. What is still missing is listed at the end.

![The sample island in a browser: forests, the river, the harbour house and its label](images/web-island.jpg)

```bash
GOOS=js GOARCH=wasm go build -o cmd/web-island/island.wasm ./cmd/web-island
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" cmd/web-island/
python3 -m http.server -d cmd/web-island 8080      # then open http://localhost:8080
```

Drag to turn around the island, drag with the right button to move, turn the wheel to come
closer.

## What was already portable

Most of go-renderer is plain Go and builds for WebAssembly as it is: the scene (`scene`), the
frame planning (`sceneplan`), cameras (`camera`), glTF loading (`loaders`), input (`input`), the
2D draw lists (`draw2d`) and the overlay painter with its atlas and fonts (`opengl/overlay`, which
rasterises on the processor and embeds its fonts with `go:embed`). Only three packages are tied to
the desktop, through GLFW and cgo: `opengl`, `windowloop` and `viewer`.

So the browser needed three things: **a window** (a canvas and its events), **a loop** that does
not use GLFW, and **a renderer** that speaks WebGL2.

## How it fits together

```text
your program              the same scene, camera and update function as on the desktop
browser   (js only)       the canvas, its events → input.State, requestAnimationFrame, WebGL2
gles                      FlatRenderer and SceneRenderer, in GLSL ES 3.00, over gles.Functions
frameloop                 the fixed-step clock, shared with windowloop
overlay, scene, camera…   unchanged
```

### `gles`: renderers written once for OpenGL ES 3

WebGL2 *is* OpenGL ES 3.0, and Android phones draw with OpenGL ES 3 too. So the renderers are
written once, in package `gles`, against a small interface, `gles.Functions`: about forty calls
to make shaders, buffers and textures and to draw. Each platform implements it:

| Platform | Implementation |
|---|---|
| Browser | `browser.WebGL`, over the canvas's `WebGL2RenderingContext` through `syscall/js` |
| Android | over `golang.org/x/mobile/gl`, in go-ui's window (so go-renderer gains no dependency) |
| Tests | a fake that records the calls, so the renderers are tested on any computer, without a display |

`gles.FlatRenderer` paints a `draw2d.List` on a plain background, like `opengl.FlatRenderer`: the
same atlas, one draw call. `gles.SceneRenderer` draws a scene in its **simplest form**: lit meshes
with vertex colours, instances, the light of the time of day (`Environment.Daylight`), fog,
translucent water and, over everything, the overlay. What a frame draws is still decided by
`sceneplan`, so culling and levels of detail work as on the desktop.

Each rectangle of the overlay is six vertices and each instance its own draw call, because
OpenGL ES through `golang.org/x/mobile/gl` cannot draw instances. That is fine for scenes like the
sample island (about a thousand calls a frame); large forests need instancing, which WebGL2 has,
as a later step.

### `browser`: the canvas as a window

- **Size and density.** The canvas fills its place in the page; its drawing buffer is that size
  times `devicePixelRatio`, so text is rasterised at its real pixel size. As on the desktop, the
  cursor is in layout pixels (`WindowWidth`) and drawing in device pixels (`FramebufferWidth`).
- **Input.** Pointer events (mouse, pen and touch alike), the wheel and the keyboard fill an
  `input.State`. Keys follow their place on the keyboard (`KeyboardEvent.code`), as GLFW's do;
  typed characters come from `KeyboardEvent.key`. The canvas takes the focus when pressed, and lets
  go of held keys when it loses it.
- **Clipboard.** Copied text is kept by the page and offered to the system clipboard; pasting takes
  what the browser's paste event brings. Reading the system clipboard at any other time needs a
  permission, so it is not done.
- **The loop.** `browser.Run` is `windowloop.Run`'s twin: the same fixed-step clock (now in
  `frameloop`, shared by both), the same drawing on demand. It asks for an animation frame only
  when something happens (an event, a size change, `RequestFrame`, `browser.Wake`), so a page left
  alone does nothing at all: measured, no frames in two seconds of idling, and two after moving
  the pointer once (the frame and its follow-up).

## Why this route

| Option | For | Against | Choice |
|---|---|---|---|
| Go, `GOOS=js GOARCH=wasm`, WebGL2 through `syscall/js` | The same compiler and standard library; `go:embed` and the fonts work; WebGL2 is in every browser; the same shaders as Android | A large file (below); each call into JavaScript has a small cost | **Chosen** |
| Go with WebGPU | A modern interface, close to Metal | Not in every browser yet; new shaders in WGSL; the hand-written `webgpu` package is the author's | Later |
| TinyGo | Much smaller files | Partial support for reflection and `go:embed`, which the fonts and the glTF loader use; a second compiler to keep working | Not now |
| Software rendering to a 2D canvas | No GPU needed | Far too slow for the scene | No |

**Size.** The island is a 5.4 MB `.wasm` file, about 1.8 MB compressed with gzip, which web
servers usually do on the fly. The fonts and icons are inside it, so the page loads nothing else.

## Testing

- `go test ./gles ./frameloop` on any computer: the renderers against the fake (shaders are
  OpenGL ES 3.00, one draw call for an interface, the atlas and meshes uploaded only when they
  change).
- `GOOS=js GOARCH=wasm go test ./browser ./gles` runs the browser package's own tests under
  Node.js (key, wheel and button mapping); it needs `$(go env GOROOT)/lib/wasm` in the `PATH`.
- In a browser: the island, turning and zooming, and the idle check above, were looked at in
  Chromium on macOS. Other browsers, phones and tablets have not been tried yet.

## Not yet

- The full scene renderer in WebGL2: shadows, the sky dome with stars and clouds, waves and tides,
  patterns, wind and snow. It is a translation of the desktop shaders to GLSL ES, a larger piece of
  work.
- Instanced drawing for big forests.
- Loading `.glb` files in the page (`fetch`, then `loaders` on the bytes).
- Locking the pointer for the walking camera, the on-screen keyboard, and the system clipboard
  at any time.
- WebGPU, and smaller files with TinyGo.
