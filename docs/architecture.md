# Architecture

go-renderer is a small renderer written in Go to learn how 3D graphics work from the inside. It
offers two ways to draw:

- **Immediate mode**: your program says "draw this triangle, this image, this model" every frame.
  Simple, and good for learning the pipeline.
- **A retained scene**: a whole world (terrain, towns, forests, sea, sky) is loaded once from a glTF
  file and the renderer draws it every frame, as fast as it can. This is what the viewer and the
  simulators use.

The renderer only draws. It does not know what a town or a season is: other programs change the
scene, and it shows the result.

## Packages

| Package | Role |
|---|---|
| `graphics` | Vectors, matrices, coordinate spaces and transformations; the shader sources |
| `models` | The application loop (`Application`, a fixed time step before every frame), the `Renderer` contract and the backend enumeration |
| `software`, `opengl`, `vulkan`, `metal`, `webgpu` | Backends behind the contract; OpenGL is the complete one |
| `scene` | The retained, backend-independent scene |
| `loaders` | glTF 2.0 loading, including the renderer's extras |
| `input` | A backend-independent snapshot of keyboard and mouse state |
| `camera` | Camera controllers: orbit and walk |
| `viewer` | The interactive explorer behind `cmd/visor` |
| `cmd/visor` | The viewer as a program |

## The immediate-mode contract

An application implements an update and a draw, and calls the renderer every frame:

```go
type Renderer interface {
	RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error
	RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error
	RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vec4, width, height float32, shader *string, fill bool) error
	RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vec4, width, height float32, shader *string) error
	Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error
	Render3dModel(space graphics.CoordinateSpace, model *Model, shader *string) error
	SetMatrices(projection, view, model graphics.Mat4)
}
```

Coordinates can be given in local, world or viewport space; the backend does the transformations.
`models.Application.Update` runs at a fixed time step before every frame, so motion does not
depend on the frame rate.

## The retained scene

`scene.Scene` is a description of a 3D scene that does not depend on any backend:

- **Nodes** with transforms, children, names and tags (the glTF extras), drawing meshes.
- **Meshes** with positions, per-vertex normals and colours, indices and a **material**: `lit`,
  `unlit`, `water` or `waterfall`, with an optional procedural pattern (cobbles, masonry, plaster,
  shingles, planks, grass), a glow, a sway (how much the wind bends it) and a depth bias.
- **Instances**: one mesh drawn many times (GPU instancing), each copy with its transform and tint.
- **Levels of detail**: simpler versions of a mesh for when it looks small on screen.
- **The environment**: sun, sky, fog, water plane, wind, snow and foliage tint, and the vertical
  exaggeration. `scene.Sky` describes the sky over a whole day, and `Environment.Daylight()`
  derives, from the sun's height, the sky colours and the light of the moment (sun by day, moon by
  night).

Meshes and instances can be marked `Dirty` so a simulation changes them and the backend uploads
them again; `Instances.MarkChanged(indices...)` uploads only the instances that changed.
`GroundHeight(x, z)` answers the height of the walkable meshes through a grid index.

## Loading

`loaders.LoadScene` reads glTF 2.0 (`.glb` or `.gltf`): the node hierarchy, `COLOR_0`, quantized
attributes (`KHR_mesh_quantization`: integer positions, normals, colours and indices),
`EXT_mesh_gpu_instancing` (with a `_COLOR` per instance), and the renderer's
[extras](gltf-extras.md): environment, sky, water, material kinds, levels of detail and walkable
ground.

## The OpenGL scene renderer

`opengl.SceneRenderer` uploads each mesh once and draws instanced geometry in a single call, so a
forest of a hundred thousand trees costs a few draw calls. Every frame goes through the same passes:

```mermaid
flowchart LR
    A[Culling<br/>what is in view,<br/>at what detail] --> B[Shadows<br/>the scene seen<br/>from the sun] --> C[Sky<br/>day, night,<br/>stars, clouds] --> D[Scene<br/>light, fog, patterns,<br/>snow, wind] --> E[Water<br/>depth, waves,<br/>tides, foam]
```


1. **Culling.** Instances are kept in grid cells; cells outside the view frustum are skipped, and
   each cell picks a level of detail by its size on screen.
2. **Shadows.** A shadow map fitted around the camera target, rendered from the sun, filtered with
   PCF (`ShadowSettings`).
3. **Sky.** A procedural day gradient, twilight glow, stars that turn around the celestial pole, a
   moon with phases that lights the night, clouds that drift with the wind, constellations, or
   skybox images (equirectangular panoramas or six cube faces, one for the day and one for the
   night).
4. **Scene.** Sun or moon light with sky ambient, fog, procedural surface patterns, glowing
   materials, snow on surfaces facing up, the season's tint on foliage, wind sway on plants.
5. **Water.** Coloured by its depth, read from a top-down depth map of the ground that is rendered
   once per scene (and again when a ground mesh is `Dirty`); waves, foam on the shore, tides that
   rise and fall differently from place to place, and waterfalls.

## Input, cameras and the viewer

The window backend feeds `input` with a snapshot of the keyboard and the mouse. `camera.Orbit`
(drag, pan, zoom) and `camera.Walk` (first person on the ground) turn it into a camera placement.
The `viewer` package puts it all together: cameras, sun, fog, shadows, vertical exaggeration,
headless captures and benchmarks. `viewer.Run` takes an optional update function called at every
fixed step, so another program can change the scene while it is shown.

## Simulations

The renderer draws the scene as it is and knows nothing about simulations. Moving things, wind
gusts, drifting clouds, the day cycle or growing cities live in the sibling project
[go-simulator](https://github.com/akrck02/go-3d-simulator), which changes the scene through `viewer.Run`. The renderer only
reads the state they leave: node transforms, `Environment.Wind`, `Sky.Clouds.Offset`, the sun
direction, the snow line and the foliage tint.
