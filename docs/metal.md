# Metal: a backend for flat drawing

**Status:** built (October 2026): the flat path, which go-ui's window draws through on macOS by
default, and the 3D scene renderer, which the viewer draws through on macOS by default (see *The 3D
scene renderer* below). This page started as the design; *What was built* and *Measured* say how it turned
out. Pending a look on screen: live resizing by dragging an edge, and a second screen with another
scale.

## Why

OpenGL has been **deprecated on macOS since 2018** and is frozen at version 4.1. On Apple silicon it
runs as a translation layer over Metal (the driver reports "4.1 Metal"). It still ships in macOS 26.
Apple has announced no removal, but there are warning signs: a report from a macOS 27 beta of an
OpenGL application that would not start, and Chrome, which dropped its OpenGL backend on macOS and
draws only through Metal.

It is **not** needed for performance or battery today. An interface that draws on demand sleeps
while idle, and each frame it does draw is one draw call; the translation layer costs about
0.1–0.2 ms of such a frame. Metal is insurance, plus a few things OpenGL cannot do on macOS:
smooth live resizing (`presentsWithTransaction`), choosing the low-power GPU, and pacing frames
with the display.

## What was built

| Piece | Where |
|---|---|
| The window and its `CAMetalLayer`, the pipeline, the atlas texture, instanced quads, the frame, captures | `metal/flat_darwin.m` (Objective-C), `metal/flat_darwin.go` (cgo) |
| Packing the painter's quads into 36-byte instances | `metal/instances.go`, with tests |
| The window's loop through Metal, and one-frame captures | `metal/flat_window_darwin.go` (`FlatWindow`) |
| The loop both backends share | `windowloop` (moved out of `opengl`, which now calls it) |
| `ErrUnavailable`, and a `FlatWindow` that returns it outside macOS | `metal/flat.go`, `metal/flat_other.go` |
| Choosing the backend: `Options.Backend`, `-backend automatic|metal|opengl`, falling back to OpenGL | go-ui `window` |

Where it differs from the design below:

- **An instance is 36 bytes, not 32:** the screen and atlas rectangles stay as floats (the atlas
  places glyphs at half pixels), and the colour is four bytes.
- **The layer keeps two drawables** (`maximumDrawableCount = 2`) instead of three: a window that
  draws on demand never queues more than one frame, and each drawable is a whole window of pixels.
- **A new atlas texture each time the atlas changes**, instead of `replaceRegion` on the changed
  rectangle: a frame still being drawn keeps reading the old texture, and the atlas changes only
  when glyphs or corners are added. Uploading only the changed rectangle is left for later, as on
  the OpenGL path.
- **The OpenGL path keeps its six vertices per quad** for now; moving it to the same instances is
  step 5 of go-ui's performance audit.

## Measured

On a laptop with an Apple M4 and a 60 Hz screen, macOS 26, a 1280 × 800 window on a high-density
screen.

**Captures:** the register application captured through both backends differs in fewer than 1 %
of its pixels, by at most 1/255 in a channel (the colour rounded to a byte), light and dark. go-ui's
`cmd/app` test `TestMetalDrawsLikeOpenGL` checks it.

**The register application, idle (drawing on demand):**

| | OpenGL | Metal |
|---|---|---|
| Memory (system footprint) | 93–102 MB | **66–70 MB** |
| Processor | 0 % | 0 % |

**A window whose content changes every frame for 20 seconds (60 frames a second), then stays
still** (a probe with 30 lines of text, `-continuous`):

| | OpenGL | Metal |
|---|---|---|
| Processor while changing | 13.6 % of a core | **9 %** |
| GPU while changing | 19–20 % | **12–16 %** |
| Memory while changing | 163 MB | **107 MB** |
| Memory once still | 109 MB | **60 MB** |

Most of the memory saved is the window's buffers and the driver's caches: OpenGL on macOS runs on
a translation layer over Metal, which keeps more buffers and caches of its own.

## The 3D scene renderer

Built after the flat path, in the same way:

- **One plan, two backends.** What a frame draws (commands in order, culling, levels of detail,
  instance batches, the shadow and seabed regions) moved out of `opengl` into `sceneplan`, which
  makes no graphics calls. OpenGL's captures came out identical, pixel for pixel, before and after
  the move. `metal.SceneRenderer` draws the same plan.
- **Shaders translated one to one** (`metal/scene_shaders.go`), compiled at start with precise
  maths: the noise behind the patterns, stars and clouds takes sines of large numbers, and fast
  maths would round them differently. The uniform blocks are mirrored in Go byte for byte, checked
  by tests on both sides.
- **A thin Objective-C layer** (`scene_darwin.m`): buffers, textures, pipelines, passes and draw
  calls; the logic stays in Go. A frame waits for the previous one, so buffers can be rewritten in
  place (a simulation moving instances).
- **Memoryless depth buffer** where the GPU allows it (Apple silicon): nothing reads it after the
  frame, so it takes no memory.
- **The viewer** chooses with `-backend` (Metal on macOS, falling back to OpenGL), and the visor
  opens a built-in sample island (`-sample island`) and draws at a fixed time (`-time`) for captures
  that can be compared.

Measured with the sample island and a factory floor scene, 1280 × 800 on a high-density screen,
drawing every frame:

| | OpenGL | Metal |
|---|---|---|
| Captures | | differ only by rounding (1/255) and where two surfaces share a plane |
| Draw calls and triangles | 15 and 160 756 | the same |
| Processor per frame (benchmark) | 0.76 ms | **0.32 ms** |
| Processor, the viewer open | 17.5 % of a core | **11.4 %** |
| GPU busy | 52–73 % | 60–92 % (precise maths) |
| Memory | 305–322 MB | **214–230 MB** |

Not tested yet: sky images (panoramas and cube faces), a second screen, simulations that move
instances every frame.

## Scope

**First:** the flat path, `FlatRenderer` — everything an interface-only window needs (go-ui's
`window` package). It is small: about 25 OpenGL calls, two short shaders, one atlas texture and one
draw call a frame.

**Later (now done, see above):** the 3D scene renderer (shadows, sky, water, levels of
detail, instancing; about 3400 lines and 660 lines of GLSL). That is three to six weeks and a
decision of its own (see *Options*).

## Options

| Option | Fit |
|---|---|
| **Native Metal through cgo and Objective-C** | cgo is already required by GLFW, so nothing new. Smallest and most direct; how Gio draws on Apple systems. **Chosen for the flat path.** |
| Metal through `purego` (no cgo) | How Ebitengine does it. No gain while GLFW needs cgo anyway, and the bindings are ours to write. |
| WebGPU through wgpu-native | One renderer for Metal, Vulkan, OpenGL ES and Direct3D, with WGSL shaders: attractive for the 3D renderer and Android. A large native library and young Go bindings. **Reconsider when porting the 3D renderer.** |

Android keeps its own small OpenGL ES flat renderer, as planned in go-ui's `docs/android.md`.

## How it fits together

```text
go-ui window          chooses the backend: Metal on macOS, OpenGL elsewhere (-backend to force one)
draw2d.List           unchanged: what an interface draws
overlay painter       unchanged: turns the list into quads of one atlas (plain Go, no OpenGL)
metal.FlatRenderer    NEW: uploads the atlas and the quads, one draw call, presents the drawable
window loop           shared by both backends: GLFW events, drawing on demand, the resize follower
GLFW                  creates the window; with Metal it creates no OpenGL context (NoAPI)
```

The painter lives in `opengl/overlay` but imports nothing from OpenGL, so the Metal renderer uses
it as it is. Moving it to a neutral package (for example `draw2d/paint`) is a tidy-up for later.

## The pieces

### 1. The window and the layer (`metal/`, Objective-C through cgo)

- GLFW creates the window with `glfw.ClientAPI` set to `glfw.NoAPI`: no OpenGL context.
- `window.GetCocoaWindow()` gives the `NSWindow`; its content view gets a `CAMetalLayer`
  (`wantsLayer = YES`, `layer = metalLayer`).
- The layer:
  - `device`: the system's default device, or the low-power one when there is a choice.
  - `pixelFormat`: `BGRA8Unorm`.
  - `framebufferOnly = YES`, `opaque = YES`, `contentsScale` = the screen's backing scale.
  - `displaySyncEnabled = YES`, unless vsync is off.
  - `drawableSize` follows the framebuffer size callback. The scale follows the window when it moves
    to another screen.
- **Live resize:** while the window is being resized, `presentsWithTransaction = YES` and the frame is
  drawn from the refresh callback (as the OpenGL loop already does). This presents the new size in
  the same transaction as the window's frame, so nothing stretches.

### 2. The flat renderer (`metal.FlatRenderer`)

It has the same method as the OpenGL one:

```go
func NewFlatRenderer(window *glfw.Window, vsync bool) (*FlatRenderer, error)
func (renderer *FlatRenderer) Draw(list *draw2d.List, background draw2d.Color, width, height int, scale float32)
```

- **One pipeline:** the vertex and fragment functions, with blending set in the pipeline the same way
  as the OpenGL path's `BlendFuncSeparate`.
- **Instanced quads:** about 32 bytes each (screen rectangle, atlas rectangle, colour), drawn as a
  four-vertex triangle strip per instance. Today's OpenGL path uses 192 bytes per quad, so the
  OpenGL path should take the same layout at the same time and both stay alike.
- **Vertex buffers:** three, used in turn and guarded by a semaphore, so the CPU never writes one
  the GPU is still reading. A buffer grows by doubling and is kept between frames.
- **The atlas:** an `R8Unorm` texture. `replaceRegion` uploads only the rectangle that changed; a
  larger texture is made only when the atlas grows.
- **The shaders:** about 30 lines of Metal Shading Language: position to clip space, atlas
  coverage times colour. They are compiled from source when the renderer starts
  (`newLibraryWithSource`), so building needs no Xcode, only the command-line tools.
- **The frame:**
  1. take the next drawable;
  2. a render pass that clears to the background;
  3. one draw call;
  4. present;
  5. commit.

  No depth buffer.

### 3. A loop shared by both backends

Today the window loop lives in `opengl.StartLoop`. Its parts are not about OpenGL:
- GLFW events,
- drawing on demand,
- frame requests and `Wake`,
- the resize follower,
- cursor shape and lock,
- the fixed-step clock.

They move to a package both backends use. Each backend gives the loop a small interface to make
its window and to draw and show a frame. The OpenGL backend keeps its behaviour, and its tests must
pass unchanged.

The existing hand-written `metal.Metal` and `metal.MetalRenderer` stubs (the immediate-mode
`Renderer` contract) are left as they are. Filling them in is a separate decision for the author.

### 4. Captures without a screen

`-capture` draws into an off-screen texture with shared storage and reads it back into a PNG, as
the OpenGL path does with `ReadPixels`. The same frame captured through both backends must match
within a small tolerance. That comparison is the main test of the port.

### 5. Choosing the backend (go-ui)

- `window.Options.Backend`: `Automatic` (Metal on macOS when a device exists, else OpenGL), `Metal` or
  `OpenGL`.
- The flag `-backend` forces one.
- If Metal fails to start, the window falls back to OpenGL and says so once.

## Iterations

| # | What | Done when | State |
|---|---|---|---|
| 0 | GLFW window with no OpenGL context, a `CAMetalLayer`, clear to a colour, drawing on demand, a capture | A window shows the background, sleeps while idle, and `-capture` writes the colour | Done |
| 1 | `metal.FlatRenderer`: pipeline, atlas, instanced quads, shaders | The register application's capture matches the OpenGL one within 1/255 per channel, light and dark | Done |
| 2 | The shared loop; go-ui backend choice; live resize in a transaction; moving between screens | The register runs on Metal, resizes without stretching or flicker, and its CPU, GPU and memory are measured against OpenGL | Done; resizing and a second screen still need a look on screen |
| 3 | Docs, tests, README disclosure | `go vet` and `go test` pass in both repositories; this page says what was measured | Done |

It was estimated at three to four days. Each step kept the OpenGL path working.

## Testing

- **Without a window:** unit tests for packing quads into instances, for the dirty rectangle of the
  atlas, and for choosing the backend.
- **With a window:** captures through both backends compared pixel by pixel; the register idle
  for a minute (no frames, no processor); resizing by dragging an edge; a second screen with
  another scale.
- **Measured against OpenGL:** processor time while moving the pointer, GPU use, memory footprint.

## Risks

- **Pacing:** `nextDrawable` blocks when every drawable is in use. Drawing on demand never holds
  one while idle. Continuous mode relies on display sync, and on skipping frames with no update.
- **Retina and scale changes:** the layer's `contentsScale` and `drawableSize` must follow the window
  as it moves between screens, or text turns blurry. This is tested on a second screen.
- **Two backends to keep alike:** the captures comparison catches drift, and both share the painter
  and the quad layout.
- **OpenGL removal may never come:** the work is still small, and it brings the resize and power
  gains with it.

## Decisions

1. **Native Metal for the flat path and WebGPU only for a later 3D port.** Taken.
2. **Shaders compiled from source at start-up rather than shipped as a compiled library.** Taken:
   it needs no Xcode, and costs a few milliseconds at start.
3. **Metal as the default on macOS once captures match.** Taken, with `-backend opengl` to go back.
4. **The OpenGL flat path takes the same instanced quads.** Still to do (step 5 of the audit).
