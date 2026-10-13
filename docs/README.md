# go-renderer documentation

| Document | What it covers |
|---|---|
| [Architecture](architecture.md) | The immediate-mode contract and its backends, the retained scene, the glTF loader, the OpenGL scene renderer, input, cameras and the viewer |
| [Viewer](viewer.md) | `cmd/visor`: every control, key and option |
| [Overlay](overlay.md) | Labels, icons and badges over the 3D view, the header and the notices, and the free 2D drawing (`draw2d`) interfaces paint with: how to use them and how they are drawn |
| [glTF extras](gltf-extras.md) | What the renderer reads from a scene's extras: environment, sky, water, materials, levels of detail, walkable ground, overlay labels |
| [In a web browser](web.md) | The island and interfaces through WebGL2: the shared OpenGL ES renderers (`gles`), the canvas and its loop (`browser`), why this route and what is missing |
| [Metal](metal.md) | The Metal backend for flat drawing on macOS: why, how it fits next to OpenGL, what was measured against it |
| [Optimization plan](optimization-plan.md) | Measurements and the plan that guided the performance work |
| [Roadmap](roadmap.md) | Goals for the next versions |
