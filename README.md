# Go Renderer

> **Note:** parts of this project (the scene renderer, the viewer and the docs) were written with an AI assistant and are pending thorough review. See the [AI disclosure](#ai-disclosure).

A tiny, educational graphics renderer built in Go. The goal is to explore and implement graphics
pipelines from scratch while exposing a clean, simple API for building applications on top: draw
shapes every frame, or load a whole glTF scene (terrain, forests, towns, sea and sky) and explore
it in a viewer.

> ⚠️ **Warning:** This software is an educational experiment and is **not suitable for production
> use**. APIs will break and errors are likely.

```bash
go run ./cmd/visor scene.glb          # explore a scene: drag to orbit, Tab to walk, M for a 2D map
go run ./cmd/visor -walk -night scene.glb
```

## Goals

* **Understand the pipeline:** vertex processing, clipping, rasterization and fragment shading.
* **Clean abstraction:** an idiomatic Go API that hides backend complexity from the application.
* **Minimal dependencies:** core maths and rendering concepts built by hand, to understand them.

## Documentation

Everything is in [docs/](docs/README.md): [architecture](docs/architecture.md),
[the viewer](docs/viewer.md), [glTF extras](docs/gltf-extras.md),
[optimization plan](docs/optimization-plan.md) and [roadmap](docs/roadmap.md).

Simulations (moving things, weather, growing towns) live in the sibling project
[go-simulator](https://github.com/akrck02/go-simulator), which changes the scene while the viewer shows it.

## AI disclosure

I write most of my code by hand, so I want to be clear about where AI was used here.

- **Hand-written (May 2026):** the immediate-mode `Renderer` contract, the `software`, `vulkan`,
  `metal` and `webgpu` backends, the `graphics` maths and the first OpenGL work.
- **Written by an AI assistant (Claude, by Anthropic) under my direction, since September 2026:**
  the retained scene (`scene`), the glTF loader, the OpenGL scene renderer (sky, shadows, water,
  instancing, levels of detail), `camera`, `input`, `viewer`, `cmd/visor` and the documentation in
  `docs/`. It also extended `graphics`, `loaders` and `models`.

Commits made with AI carry a `Co-Authored-By: Claude` line (29 of the 49 commits on this branch).

**Status: pending thorough review.** The code compiles and its tests pass, but I have not yet
reviewed it line by line. Treat it as a working draft, not as code I vouch for.
