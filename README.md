# Go Renderer

A tiny, educational graphics renderer built in Go. The goal is to explore and implement graphics
pipelines from scratch while exposing a clean, simple API for building applications on top: draw
shapes every frame, or load a whole glTF scene (terrain, forests, towns, sea and sky) and explore
it in a viewer.

> ⚠️ **Warning:** This software is an educational experiment and is **not suitable for production
> use**. APIs will break and errors are likely.

```bash
go run ./cmd/visor scene.glb          # explore a scene: drag to orbit, Tab to walk
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
[go-simulator](https://github.com/akrck02/go-3d-simulator), which changes the scene while the viewer shows it.
