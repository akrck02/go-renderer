# Reference

Every public package, type and function, with the first sentence of its documentation. The pages
are generated from the doc comments (`python3 docs/5-reference/generate.py` from the repository's
root), so they say exactly what `go doc` says; for the whole comment of anything, run
`go doc github.com/akrck02/go-renderer/scene.Mesh` or open the source.

## Which package do I need?

| I want to… | Package |
|---|---|
| Show a scene in a window, with cameras, a sun and captures | [`viewer`](2-viewer.md) |
| Describe a scene: nodes, meshes, materials, instances, sky, water, labels | [`scene`](1-scene.md) |
| Open a `.glb` or `.gltf` | [`loaders`](6-loaders.md) |
| Read the keyboard, the mouse and fingers | [`input`](3-input.md) |
| Move a camera myself, or find what is under the pointer | [`camera`](4-camera.md) |
| Paint flat shapes, text, icons and pictures over the view | [`draw2d`](5-draw2d.md) |
| Vectors, matrices, transforms | [`graphics`](7-graphics.md) |
| A made-up island to test with | [`scene/sample`](9-scene-sample.md) |
| Run my own program in a window: its options, update and draw | [`window`](8-window.md) |
| Draw triangles myself every frame (immediate mode) | [`immediate`](22-immediate.md) |

### Core

| Package | What it is |
|---|---|
| [`scene`](1-scene.md) | The retained scene: nodes, meshes, materials, instances, levels of detail, the environment (sun, sky, fog, water, wind, snow) and the overlay (labels, header, notices, a free drawing) |
| [`viewer`](2-viewer.md) | The interactive explorer: `Run` for programs that animate, `Edit` for programs that change the scene in answer to the person; options and flags |
| [`input`](3-input.md) | A snapshot of the keyboard, mouse and touch screen at each step; claims, the clipboard, the pointer's shape, frames asked for |
| [`camera`](4-camera.md) | The `Camera`; orbit, walking and map controllers; the ray through a pixel |
| [`draw2d`](5-draw2d.md) | A list of flat shapes in layout pixels, which the backends paint over everything |
| [`loaders`](6-loaders.md) | glTF 2.0 with the renderer's extras |
| [`graphics`](7-graphics.md) | `Vector`, `Matrix`, transformations and projections |
| [`window`](8-window.md) | A program in a window: its `Options`, its `Callbacks` (update, draw) and the `Application` they make |
| [`immediate`](22-immediate.md) | The immediate-mode `Renderer` contract: shapes drawn at once, every frame |
| [`scene/sample`](9-scene-sample.md) | A made-up island that uses every feature, needing no files |
| [`failure`](18-failure.md) | Error codes: `Define`, `Wrap`, checked with `errors.Is` |

### Platforms and internals

Programs rarely use these directly: `viewer` and go-ui's `window` choose and drive them.

| Package | What it is |
|---|---|
| [`opengl`](16-opengl.md) | The OpenGL 4.1 backend: the window, the scene renderer, the flat renderer |
| [`opengl/overlay`](15-opengl-overlay.md) | Lays the overlay out as quads of one atlas: fonts, icons, the text measurer |
| [`metal`](14-metal.md) | Metal on macOS: flat and scene windows and renderers |
| [`gles`](13-gles.md) | Renderers written once for OpenGL ES: WebGL2 in browsers, Android |
| [`browser`](11-browser.md) | A canvas as the window, built with `GOOS=js GOARCH=wasm` |
| [`android`](10-android.md) | The program as a NativeActivity: window, touches, keyboard, screen reader, built with `GOOS=android` |
| [`windowloop`](17-windowloop.md) | The GLFW window loop OpenGL and Metal share |
| [`frameloop`](12-frameloop.md) | The fixed-step clock every loop shares |
| [`software`](19-software.md), [`vulkan`](20-vulkan.md), [`webgpu`](21-webgpu.md) | Backends of the immediate-mode contract, to be built later: every function returns `<backend>.not-implemented` |

The backends also share two packages of the module's own, which programs cannot import:
`internal/sceneplan` (what a frame of a scene draws, worked out once for every backend) and
`internal/shading` (what every backend's shaders agree on).

## Conventions

These hold across every package.

| Topic | Rule |
|---|---|
| **Axes** | y is up; the ground is the x–z plane; north is −z |
| **Units** | The scene's units are yours (metres, kilometres); some defaults scale with the scene's size |
| **Points and directions** | A `Vector` with 1 last is a point, with 0 a direction; colours use it as opacity |
| **Matrices** | Column-major, as OpenGL; `a.Multiply(b)` applies `b` first; translation in elements 12, 13, 14 |
| **Scene colours** | Linear RGB(A), 0 to 1, lit and gamma-corrected by the renderer |
| **Screen colours** | `draw2d.Color` and the overlay's tints are as seen on screen, not gamma-corrected |
| **Screen positions** | `draw2d` and touches: layout pixels from the top-left corner, multiplied by the screen's density when drawn. `input.State.CursorX`/`CursorY` and `viewer.View`: window pixels |
| **Time** | Update functions run at a fixed step (1/60 s by default), whatever the frame rate |
| **Goroutines** | Update and edit functions run on the drawing goroutine: change the scene there, with no locks. From another goroutine, wake the window (`opengl.Wake`) and hand the change over |
| **Uploads** | Meshes and instances are uploaded once; set `Dirty` (or `Instances.MarkChanged`) after changing them |
| **Zero values** | A zero field usually means "the default" (`FogFar` 0 = no fog, `VerticalScale` 0 = 1); the field's comment says which |
| **Icons** | By [Lucide](https://lucide.dev/icons) name, everywhere; the brand logos Lucide withdrew are not offered |

---

[← Documentation](../1-overview.md)
