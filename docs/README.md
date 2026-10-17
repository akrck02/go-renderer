# go-renderer documentation

A small graphics renderer in Go, to learn how 3D graphics work from the inside and to show scenes:
terrain, forests, towns, the sea and the sky, by day and by night.

## 🚀 Start here

| | |
|---|---|
| **[Get started](get-started.md)** | Install the tools, open the sample island, and show your first scene. Fifteen minutes. |
| **[Tutorial: an island with a lighthouse](tutorial/README.md)** | Six lessons, one program: meshes, motion, sky and water, labels, editing with the pointer, glTF files. |

## 📖 Guides

How each part works and how to use it, one subject a page.

| Guide | What it covers |
|---|---|
| [Architecture](architecture.md) | The immediate-mode contract and its backends, the retained scene, the glTF loader, the scene renderers and their passes, input, cameras and the viewer |
| [The viewer](viewer.md) | `cmd/visor`: every control, key and option; `viewer.Edit` for editors |
| [The overlay](overlay.md) | Labels, icons and badges over the 3D view, the header and the notices, and the free 2D drawing (`draw2d`) interfaces paint with: how to use them and how they are drawn |
| [glTF extras](gltf-extras.md) | What the renderer reads from a scene's extras: environment, sky, water, materials, levels of detail, walkable ground, overlay labels |
| [In a web browser](web.md) | The island and interfaces through WebGL2: the shared OpenGL ES renderers (`gles`), the canvas and its loop (`browser`), why this route and what is missing |
| [On Android](android.md) | go-renderer's part in running interfaces on phones, and why OpenGL ES 2 |
| [Metal](metal.md) | The Metal backend for flat drawing on macOS: why, how it fits next to OpenGL, what was measured against it |

## 📚 Reference

| | |
|---|---|
| [Packages](reference/README.md) | Every package, type and function, generated from the doc comments, and the conventions they share (axes, units, colours, goroutines) |

## 🧭 The project

| | |
|---|---|
| [Optimization plan](optimization-plan.md) | Measurements and the plan that guided the performance work |
| [Roadmap](roadmap.md) | Goals for the next versions |

## The family

go-renderer draws. Other projects build on it:

| Project | Adds |
|---|---|
| [go-ui](https://github.com/akrck02/go-ui) | Panels, buttons, fields and lists over a scene, or a whole application with no scene |
| [go-ui-android](https://github.com/akrck02/go-ui-android) | Material Design 3 pieces, and a tool that packages a go-ui program as an Android app |
| [go-scene-builder](https://github.com/akrck02/go-scene-builder) | Scenes (`.glb`) from a 2D map and a relief |
| [go-simulator](https://github.com/akrck02/go-simulator) | Simulations that change a scene while the viewer shows it |
