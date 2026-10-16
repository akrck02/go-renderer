# 🎮 On the Nintendo 3DS

**Status:** a first step (October 2026). Package `ctr` draws through the 3DS's graphics chip, the
PICA200, with citro3d: `cmd/triangle3ds` turns a triangle of three colours on the top screen at 60
frames a second, and `cmd/island3ds` flies round the sample island with `ctr.SceneRenderer`. It runs as homebrew, from the Homebrew Launcher, built with a TinyGo that has a
`nintendo3ds` target (a separate fork of TinyGo, not part of this repository). The triangle is
checked in the Azahar emulator; the loading and planning figures below come from a New 3DS XL.

<p align="center">
  <img alt="The top screen of the emulator: a triangle with red, green and blue corners, turned a little, on a sky blue background" src="images/3ds-triangle.png" width="420">
  <img alt="The top screen of the emulator: the sample island seen from above at a distance, its forest, shore and central rock, in a sea that fades into fog towards the horizon" src="images/3ds-island.png" width="420">
</p>

## How it fits together

| Layer | What it does |
|---|---|
| Your program | Plans frames with `sceneplan`, moves things with `graphics` |
| `ctr` | The top screen, vertex shaders, vertex buffers and matrices, over citro3d (this package) |
| citro3d, libctru | devkitPro's libraries for the GPU and the system, reached through cgo |
| TinyGo `nintendo3ds` | Builds the program as a `.3dsx` for the Homebrew Launcher |

The rest of go-renderer is plain Go and already runs there: on the console, `encoding/json`,
`image/png`, `loaders.ParseSceneGLB` and the sample island load, and planning a frame of the island
takes under a millisecond. The tests of `graphics`, `scene`, `sceneplan`, `camera`, `draw2d`,
`input` and `frameloop` pass in the emulator.

## What is different about the PICA200

- **No fragment shaders.** Vertex shaders are written in the PICA200's assembly (`.v.pica`) and
  assembled with devkitPro's `picasso`; colours come out of a few fixed stages. `ctr` sets the first
  one to pass the vertex colour through.
- **Matrices row by row, components backwards.** citro3d stores a matrix as four rows, and each row
  as w, z, y, x. `graphics.Mat4` is column by column; `ctr.PICAMatrix` lays it out (tested on any
  computer).
- **A screen on its side.** The 3DS's screens are portrait panels turned a quarter turn, and depth
  runs from -1 to 0. `ctr.OrthographicTilt` is citro3d's `Mtx_OrthoTilt` for a right-handed view,
  as everywhere in go-renderer: what is visible lies between z = 0 and z = -far.
  `cmd/triangle3ds` checks it against citro3d's own on the console.
- **Vertex buffers outside the Go heap.** `ctr.NewVertexBuffer` copies vertices to the linear
  memory the GPU reads, so the garbage collector never scans them.

## The scene renderer

`ctr.SceneRenderer` draws a `scene.Scene` the way `gles` does: lit meshes and instances, the light
of the time of day, fog and translucent water, planned by `sceneplan`. What changes on the PICA200:

- **Light worked out on the processor.** Lighting every vertex on the GPU held the sample island
  to 30 frames a second on a New 3DS XL: the GPU took 34 ms a frame, the processor 0.4 ms. Sun and
  sky change slowly, so `ctr/bake.go` lights each vertex once, when a mesh is uploaded (sun, sky
  and ground light, glow, flat by brightness, then the screen's gamma), and keeps its colour in
  four bytes: a vertex is 16 bytes instead of 40. A mesh is baked again only when the light, the
  mesh or its node's turn changes. `ctr/shaders/baked.v.pica` then only places each vertex and
  fogs it by its distance to the camera, linearly: 27 instructions instead of 95.
- **Water on the GPU, on a coarser grid.** Water's reflection of the sky depends on where the
  camera looks from, so water goes through the full shader, `ctr/shaders/scene.v.pica` (gles's
  scene shader done per vertex, with its smoothstep fog and the chip's `lg2` and `ex2` for gamma).
  The planner's water grid of 256 × 256 cells, made for the desktop's waves, was 131 072 of the
  island's 160 756 triangles; `ctr` asks for 32 × 32 (`Planner.WaterGridDivisions`), still fine
  near the camera for the fog.
- **Not the chip's own fog.** The PICA200 can fog in fixed function, from a table of 128 steps of
  the depth buffer; but depth is far from linear (with the near plane at 0.1 and the far one at
  20 000, the whole island's fog would fall in the first step), so ctr fogs per vertex.
- **OpenGL's matrices, turned.** `sceneplan` plans frames with OpenGL's clip space; one fixed
  matrix (`ctr.PICAViewProjection`) gives the PICA200's quarter turn and depth range. Turning
  `graphics.OrthographicOpenGL` with it gives `ctr.OrthographicTilt` (tested on any computer).
- **16-bit indices.** Meshes with more than 65 535 vertices are drawn in pieces
  (`ctr/split.go`).
- **Instances merged on the processor,** as in `gles`: the merging moved to `sceneplan`
  (`MergedBatch`), which both renderers share.
- **Meshes in linear memory,** uploaded once and again only when `Dirty`.

## Building

```bash
tinygo build -target=nintendo3ds -o triangle3ds.3dsx ./cmd/triangle3ds
```

Copy the `.3dsx` to `/3ds/` on the SD card and open it from the Homebrew Launcher.

## Not yet

- How fast the renderer is on the console with the coarser water (baking the light alone went from
  30 to 36 frames a second: the chip was held back by the number of triangles more than by the
  shader). Positions in shorts and fewer triangles far away
  are the next steps if it is not enough.
- A sun that moves all the time bakes the meshes again whenever a colour would change by a step.
- Planning a frame makes about 4 KB of garbage; the collector would have to run every few minutes.
- Textures, the overlay and `draw2d`.
- Input beyond the START button, the bottom screen's touch, and the 3D slider.
- Reading scenes from the SD card: TinyGo's `os` has no file system there yet, so scenes are
  embedded in the program.
- Collection pauses: about a quarter of a second with 10 MB alive on the console. Keeping meshes
  in linear memory and making no garbage every frame (as `cmd/triangle3ds` does) avoids them.
