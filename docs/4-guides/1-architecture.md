# 🧭 Architecture

Go Renderer is a small renderer written in Go to learn how 3D graphics work from the inside. It
offers two ways to draw:

- **Immediate mode**: your program says "draw this triangle, this image, this model" every frame.
  Simple, and good for learning the pipeline.
- **A retained scene**: a whole world (terrain, towns, forests, sea, sky) is loaded once from a glTF
  file and the renderer draws it every frame, as fast as it can. This is what the viewer and the
  simulators use.

The renderer only draws. It does not know what a town or a season is: other programs change the
scene, and it shows the result.

<p align="center">
  <img alt="Immediate mode: your program's Update and Draw call the hand-written immediate.Renderer contract, drawn by the software, opengl, vulkan, metal or webgpu backend. A retained scene: glTF 2.0 through loaders into scene and sceneplan, drawn by the scene renderers: opengl and metal on the desktop, gles in browsers and on Android" src="../images/two-ways.svg" width="880">
</p>

## Packages

| Package | Role |
|---|---|
| `graphics` | Vectors, matrices, coordinate spaces and transformations |
| `window` | A program in a window: what the window is (`Options`), what runs every frame (`Callbacks`: a fixed-step `Update`, `Draw`) and what both share (`Application`) |
| `immediate` | The hand-written immediate-mode contract: `Renderer`, `Mesh`, `Model` |
| `opengl`, `metal` | Backends behind the contract (`opengl.Window` and `opengl.Renderer`); OpenGL is the complete one. `metal` also has `FlatWindow` and `FlatRenderer`, which draw a flat interface through Metal on macOS ([design](7-metal.md)) |
| `software`, `vulkan`, `webgpu` | Backends behind the contract that will be built later. Until then every function returns its package's `NotImplemented` code (`software.not-implemented`, `vulkan.not-implemented`, `webgpu.not-implemented`), checked with `errors.Is` |
| `failure` | Error codes: `failure.Define` declares one, `failure.Wrap` adds details; check them with `errors.Is` |
| `internal/shading` | What the backends' shaders agree on (a sharp picture's offset, ambient light, the backdrop blur's weights); each backend keeps its own shader sources (the OpenGL ones in `opengl/*_shaders.go`) |
| `internal/sceneplan` | Decides what a frame of a scene draws, for any backend: the draw commands in order, culling and levels of detail, instance batches, where the shadow and seabed maps look |
| `scene/sample` | A made-up island that uses every feature the renderers draw, for tests and comparisons |
| `gles` | Renderers written once for OpenGL ES (WebGL2 in browsers, Android): a flat `FlatRenderer` and a simple `SceneRenderer`, over a small interface each platform implements ([in a browser](5-web.md)) |
| `browser` | Built only for WebAssembly: a canvas as the window, its events as input, WebGL2, and the loop over `requestAnimationFrame` |
| `android` | Built only for Android: the program as a NativeActivity's shared library, its window, touches and drawing thread, direct OpenGL ES calls ([on Android](6-android.md)) |
| `frameloop` | The fixed-step clock and update step that the desktop and browser loops share |
| `windowloop` | The window loop both OpenGL and Metal use: GLFW events and input, drawing on demand, the fixed-step clock, redrawing while resizing, the pointer's shape and lock |
| `scene` | The retained, backend-independent scene, with its overlay (labels, header, notices) |
| `loaders` | glTF 2.0 loading, including the renderer's extras |
| `opengl/overlay` | Lays out the overlay as rectangles of one atlas of glyphs, icons and rounded corners (no graphics context needed) |
| `draw2d` | A list of flat shapes (rounded rectangles, shadows, text, icons, pictures, clips, layers) to paint over the view, filled by a program or an interface library |
| `input` | A backend-independent snapshot of keyboard and mouse state: held keys and buttons, presses and releases, double clicks, typed characters, clipboard, pointer shape, and what an interface claimed |
| `camera` | The `Camera` and its controllers: orbit, walk and map; the ray through a point of the window (`RayThrough`) and where it meets a plane or the ground |
| `viewer` | The interactive explorer behind `cmd/visor` |
| `cmd/visor` | The viewer as a program |

## The immediate-mode contract

An application implements an update and a draw, and calls the renderer every frame:

```go
type Renderer interface {
	RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error
	RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error
	RenderRectangle(space graphics.CoordinateSpace, centre graphics.Vector, width, height float32, shader *string, fill bool) error
	RenderImage(space graphics.CoordinateSpace, bytes []byte, centre graphics.Vector, width, height float32, shader *string) error
	RenderObject(space graphics.CoordinateSpace, vertices []graphics.Vector, shader *string) error
	RenderModel(space graphics.CoordinateSpace, model *Model, shader *string) error
	SetMatrices(projection, view, model graphics.Matrix)
}
```

Coordinates can be given in local, world or viewport space; the backend does the transformations.
`window.Application.Update` runs at a fixed time step before every frame, so motion does not
depend on the frame rate.

An application that is mostly still (a window of controls) sets `DrawOnDemand`: the loop then
sleeps in `glfw.WaitEvents` and draws only when input arrives, when the program asked for a frame
with `input.State.RequestFrame` (a caret's next blink), or when another goroutine calls
`opengl.Wake` (or `windowloop.Wake`). Each such frame runs one update step, and a frame with input is followed by one
more so hover looks catch up with a layout that just changed. Idle, it uses no processor. `Application.Unchanged` lets an application say a frame would repeat
the last one shown (Go UI compares `draw2d.List.Fingerprint`), and the loop then neither draws nor
presents it. In the default, continuous mode a frame is drawn only after an update step, so a
display faster than the updates does not draw the same picture twice. While a
window is being resized, it is redrawn from GLFW's refresh callback, since macOS holds the loop
until the mouse is let go.

This loop lives in `windowloop` and knows nothing about the graphics interface: `windowloop.Run`
takes the window, the application and a function that draws and shows one frame. The OpenGL
backend and the Metal flat window both use it, so they behave the same way.

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
[extras](4-gltf-extras.md): environment, sky, water, material kinds, levels of detail and walkable
ground.

`loaders.ReadScene` reads the same from a `.glb`'s content (any `io.Reader`), for programs with no file system
(a web page) or that carry their scene inside them.

## The scene renderers

There are two, which draw the same frame: `opengl.SceneRenderer` and, on macOS, `metal.SceneRenderer`
(the viewer chooses with `-backend`; Metal by default on macOS). What a frame draws is decided once,
in `internal/sceneplan`, which makes no graphics calls; each backend only uploads and draws that plan, with
shaders that are faithful translations of each other. Their captures of the sample island match
except for rounding and where two surfaces lie in the same plane (`cmd/visor`'s
`TestMetalDrawsTheSceneLikeOpenGL`; [measurements](7-metal.md)).

<table>
<tr><th width="50%">OpenGL (<code>-backend opengl</code>)</th><th width="50%">Metal (<code>-backend metal</code>)</th></tr>
<tr>
<td><img alt="The sample island drawn through OpenGL" src="../images/backend-opengl.png"></td>
<td><img alt="The same frame of the sample island drawn through Metal, alike to the eye" src="../images/backend-metal.png"></td>
</tr>
<tr><td colspan="2" align="center">The same plan, drawn by each backend: <code>go run ./cmd/visor -sample island -time 10 -backend … -capture frame.png</code></td></tr>
</table>

Each uploads each mesh once and draws instanced geometry in a single call, so a
forest of a hundred thousand trees costs a few draw calls. Every frame goes through the same passes:

<p align="center">
  <img alt="The six passes of a frame in order: 1 culling, what is in view at what detail; 2 shadows, the scene seen from the sun; 3 sky, day, night, stars and clouds; 4 scene, light, fog, patterns, snow and wind; 5 water, depth, waves, tides and foam; 6 overlay, labels, header, notices and drawing" src="../images/frame-passes.svg" width="880">
</p>

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
6. **Overlay.** Labels over things in the world, a header and fading notices, and on top of them
   a free 2D drawing (`Overlay.Drawing`, a `draw2d.List`), in screen space with one texture atlas
   and one draw call (see [the overlay](3-overlay.md)).

## Input, cameras and the viewer

The window backend feeds `input` with a snapshot of the keyboard and the mouse: what is held, and
what happened since the last update step (keys and buttons that went down or up, keys repeated by
the system, characters typed, which click of a double click a press was). An interface drawn over
the view can **claim** the mouse or the keyboard for a step (`ClaimMouse`, `ClaimKeyboard`); the
viewer gives its cameras and shortcuts `State.Unclaimed()`, a copy without what was claimed, so a
slider being dragged does not turn the camera. `camera.Orbit`
(drag, pan, zoom), `camera.Walk` (first person on the ground) and `camera.Map` (a flat map, from
above or isometric) turn it into a camera placement.

A game pad is read by position, whatever its maker: `PadA` is always the bottom face button. What
is printed on that button is not: A on an Xbox pad, ✕ on a PlayStation pad, B on a Nintendo pad.
`State.PadLayout()` says which (`XboxLayout`, `PlayStationLayout`, `NintendoLayout`), so a game can
write a hint with the name the player sees. The window loop sets it when a pad connects, from the
USB vendor in the pad's GUID (`PadLayoutFromGUID`); the browser from the vendor in the Gamepad
API's id (`PadLayoutFromName`). A vendor it does not know, Android and the 3DS read as an Xbox pad:
the layout most pads copy (and the 3DS's own buttons already carry Nintendo's names).

On a touch screen the first finger is the pointer (`input.TouchPointer`): it moves the cursor and
holds the left button, so a tap is a click and a drag turns the orbit. A second finger lets the
button go, and every finger on the screen is in `State.Touches`. `input.TwoFingers` turns two
fingers, from one step to the next, into a `Gesture`: how much they spread apart, how much they
turned and how far their middle moved. `Orbit.Gesture` follows it as a map does: the ground between
the fingers stays between them while they pinch, twist and move; `Walk.Steer` walks as an on-screen stick says and `Walk.Turn` looks
around by a finger's movement. The interface's copy (`Unclaimed`) keeps the touches, so a program
can tell which finger is on its own control. A camera with an `OrthographicHeight` is
drawn without perspective: it shows that many scene units from the bottom to the top of the
window, and shadows and levels of detail follow that size instead of the distance to the camera.
The `viewer` package puts it all together: cameras, sun, fog, shadows, vertical exaggeration,
headless captures and benchmarks. `viewer.Run` takes an optional update function called at every
fixed step, so another program can change the scene while it is shown. Without one, the viewer
draws on demand: only while input arrives, a key is held or the view is still moving.
`viewer.Edit` is for programs that edit the scene: they run at every step with a `viewer.View`
that turns the pointer into a point on the ground, and the viewer still draws on demand.

## Simulations

The renderer draws the scene as it is and knows nothing about simulations. Moving things, wind
gusts, drifting clouds, the day cycle or growing cities live in the sibling project
[Go Simulator](https://github.com/akrck02/go-simulator), which changes the scene through `viewer.Run`. The renderer only
reads the state they leave: node transforms, `Environment.Wind`, `Sky.Clouds.Offset`, the sun
direction, the snow line and the foliage tint.

---

[← All the documentation](../1-overview.md)
