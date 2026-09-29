# Go Renderer

This is a tiny, educational graphics renderer built in Go. The core goal of this project is to explore and implement complex graphics pipelines from scratch, while exposing a clean, simple contract (API) for building applications on top.

> ⚠️ **Warning:** This software is an educational experiment and is absolutely **not suitable for production use**. APIs will break, performance is unoptimized, and errors are highly probable.

##  Project Goals
* **Understand the Pipeline:** Deep-dive into vertex processing, clipping, rasterization, and fragment shading.
* **Clean Abstraction:** Create an intuitive, idiomatic Go API that hides backend rendering complexity from the application developer.
* **Minimal depencies:** Build core math and rendering concepts with minimal dependencies to fully understand the underlying mechanics.

## Scene rendering (OpenGL)
Besides the immediate-mode `Renderer` contract, the project can load and draw complete scenes:

* `scene`: retained, backend-independent scene — nodes with transforms, meshes (positions, normals, per-vertex colors, indices), materials (`lit`, `unlit`, `water`, `waterfall`), GPU instancing and an environment (sun, sky, fog, water plane, vertical exaggeration). Meshes and instances can be marked `Dirty` so simulations can change them and the backend uploads them again. `GroundHeight(x, z)` answers the height of walkable meshes through a grid index.
* `loaders.LoadScene`: glTF 2.0 loader (`.glb` or `.gltf`) with node hierarchy, `COLOR_0`, `EXT_mesh_gpu_instancing` (plus a `_COLOR` per instance) and renderer extras (`environment`, material `kind`, `ground`).
* `opengl.SceneRenderer`: uploads each mesh once and draws instanced geometry in a single call; sky, sunlight with sky ambient, fog, animated water and waterfalls.
* `input`: backend-independent keyboard and mouse state, fed by the window backend.
* `camera`: `Orbit` (drag, pan, zoom) and `Walk` (first person on the ground) controllers.
* `models.Application.Update`: fixed time step before every frame, the place for simulations.

### Visor
```bash
go run ./cmd/visor scene.glb
go run ./cmd/visor -walk scene.glb
go run ./cmd/visor -capture frame.png scene.glb   # one frame to PNG, hidden window
```
Left drag rotates, right drag pans, scroll zooms. `Tab` switches to walking (WASD or arrows, mouse to look, shift to run). `J`/`L` turn the sun, `I`/`K` raise or lower it, `Z`/`X` change the vertical exaggeration, `F` toggles fog.

### glTF extras read by the renderer
* scene `extras.environment`: `sunDirection`, `sunColor`, `skyZenith`, `skyHorizon`, `groundAmbient`, `ambient`, `fog {color, near, far}`, `water {level, deep, shallow, size}`, `verticalScale`
* material `extras.kind`: `lit` | `unlit` | `water` | `waterfall`
* mesh, primitive or node `extras.ground: true`: walkable surface

## Roadmap
This are the goals for the next versions of the library. The feature implementation order can change.

### v1.0.0 Prototype
* Local space, world space and viewport space coordinate transformations.
* Render of 2D objects on world space
* Render of 2D objects on viewport space
* Render of 3D objects on world space
* Render of 3D objects on viewport space
* Camera movement
* Input bindings

### v2.0.0 Support expansion
* Metal support (Apple silicon)
* Vulkan support
* WebGPU support
