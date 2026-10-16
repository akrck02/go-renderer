# 📚 go-renderer documentation

| | Page | What is in it |
|---|---|---|
| 🧭 | [Architecture](architecture.md) | The two ways to draw (the immediate-mode contract and its backends, the retained scene), the packages, the glTF loader, the scene renderers and the passes of a frame, input, cameras and the viewer. |
| 🔭 | [The viewer](viewer.md) | `cmd/visor`: the orbit, walking and map views, the sun and sky, every key and option, captures and benchmarks, editing a scene. |
| 🏷️ | [The overlay](overlay.md) | Labels, icons and badges over the 3D view, the header and the notices, and the free 2D drawing (`draw2d`) interfaces paint with: how to use them and how they are drawn. |
| 🧾 | [glTF extras](gltf-extras.md) | What the renderer reads from a scene's extras: environment, sky, water, materials, levels of detail, walkable ground, overlay labels. |
| 🌐 | [In a web browser](web.md) | The island and interfaces through WebGL2: the shared OpenGL ES renderers (`gles`), the canvas and its loop (`browser`), why this route. |
| 📱 | [On Android](android.md) | go-renderer's part in running interfaces on phones: its own activity, why OpenGL ES 2, typing, notifications and screen readers. |
| 🎮 | [On the Nintendo 3DS](3ds.md) | A first drawing through the 3DS's PICA200 with citro3d (`ctr`): what is different about it, building with TinyGo, what is missing. |
| 🍎 | [Metal](metal.md) | The Metal backend on macOS, flat and 3D: why, how it fits next to OpenGL, what was measured against it. |
| ⚡ | [Optimization plan](optimization-plan.md) | Measurements and the plan that guided the performance work, phase by phase. |
| 🗺️ | [Roadmap](roadmap.md) | Goals for the next versions. |

New here? Start with the [project README](../README.md), then the [architecture](architecture.md).

The pictures are of the built-in sample island (`go run ./cmd/visor -sample island`), a made-up
place that uses every feature the renderers draw.

## 🚧 Not done yet

- **In a browser:** the full scene renderer in WebGL2 (shadows, the sky dome with stars and clouds,
  waves and tides, patterns, wind and snow), loading `.glb` files in the page, locking the pointer
  for walking, the on-screen keyboard and the system clipboard at any time, WebGPU, and smaller
  files with TinyGo. See [in a web browser](web.md#not-yet).
- **On Android:** keyboards for Chinese and Japanese (not tried yet), long presses, and the full
  scene renderer (shadows, sky, waves) on phones. See [on Android](android.md#not-yet).
- **Browsers and phones both:** true instanced drawing (WebGL2, OpenGL ES 3), so very large
  forests do not cost a copy of each mesh.
- **On the Nintendo 3DS:** the scene renderer, textures, the overlay, input and loading scenes from
  the SD card. See [on the Nintendo 3DS](3ds.md#not-yet).
- **Metal:** a look on screen at live resizing and at a second screen with another scale; sky
  images and simulations that move instances every frame are not tested yet. See [Metal](metal.md).
- **The immediate-mode contract:** OpenGL is its only complete backend; the others are the goals of
  the [roadmap](roadmap.md).
