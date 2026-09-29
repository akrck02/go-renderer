# Go Renderer

This is a tiny, educational graphics renderer built in Go. The core goal of this project is to explore and implement complex graphics pipelines from scratch, while exposing a clean, simple contract (API) for building applications on top.

> ⚠️ **Warning:** This software is an educational experiment and is absolutely **not suitable for production use**. APIs will break, performance is unoptimized, and errors are highly probable.

##  Project Goals
* **Understand the Pipeline:** Deep-dive into vertex processing, clipping, rasterization, and fragment shading.
* **Clean Abstraction:** Create an intuitive, idiomatic Go API that hides backend rendering complexity from the application developer.
* **Minimal depencies:** Build core math and rendering concepts with minimal dependencies to fully understand the underlying mechanics.

## Scene rendering (OpenGL)
Besides the immediate-mode `Renderer` contract, the project can load and draw complete scenes:

* `scene`: retained, backend-independent scene — nodes with transforms, meshes (positions, normals, per-vertex colors, indices), materials (`lit`, `unlit`, `water`, `waterfall`), GPU instancing and an environment (sun, sky, fog, water plane, vertical exaggeration). `scene.Sky` describes the sky over a whole day and `Environment.Daylight()` derives, from the sun height, the sky colors and the light of the moment (sun by day, moon by night). Meshes and instances can be marked `Dirty` so simulations can change them and the backend uploads them again. `GroundHeight(x, z)` answers the height of walkable meshes through a grid index.
* `loaders.LoadScene`: glTF 2.0 loader (`.glb` or `.gltf`) with node hierarchy, `COLOR_0`, `EXT_mesh_gpu_instancing` (plus a `_COLOR` per instance) and renderer extras (`environment`, material `kind`, `ground`).
* `opengl.SceneRenderer`: uploads each mesh once and draws instanced geometry in a single call; a day and night sky (procedural day gradient, twilight glow, stars that turn around the celestial pole, a moon with phases that lights the night, clouds, constellations, or skybox images: equirectangular panoramas or six cube faces, one for the day and one for the night), sun or moon light with sky ambient, sun shadows (a shadow map fitted around the camera target, PCF filtered, `ShadowSettings`), fog, animated water colored by its depth (a top-down depth map of the ground, rendered once per scene and again when a ground mesh is Dirty) and waterfalls.
* `input`: backend-independent keyboard and mouse state, fed by the window backend.
* `camera`: `Orbit` (drag, pan, zoom) and `Walk` (first person on the ground) controllers.
* `models.Application.Update`: fixed time step before every frame, the place for simulations.

### Visor
```bash
go run ./cmd/visor scene.glb
go run ./cmd/visor -walk scene.glb
go run ./cmd/visor -capture frame.png scene.glb   # one frame to PNG, hidden window
```

#### Orbit camera (default)
| Control | Action |
|---|---|
| Left drag | Rotate around the target |
| Right drag (or shift + left drag) | Pan |
| Scroll | Zoom |
| `←` `→` | Rotate |
| `↑` `↓` | Tilt |
| `+` `-` | Zoom in or out |

#### Walking (`Tab` switches)
| Control | Action |
|---|---|
| `W` `A` `S` `D` | Move forward, left, back, right |
| `↑` `↓` | Move forward or back |
| `←` `→` | Turn |
| Drag with any mouse button | Look around |
| `Shift` | Run |
| `[` `]` | Walk slower or faster (×1.5 per press) |

#### Sun, sky and scene (both modes)
| Key | Action |
|---|---|
| `J` `L` | Turn the sun |
| `I` `K` | Raise or lower the sun (below the horizon it is night) |
| `N` | Jump between day and night |
| `T` | Let time pass, a day lasts two minutes (toggle) |
| `Z` `X` | Less or more vertical exaggeration (orbit only) |
| `F` | Fog (toggle) |
| `O` | Shadows (toggle) |

#### Flags
| Flag | Effect |
|---|---|
| `-walk` | Start walking |
| `-night` | Start at night |
| `-sun-elevation D` / `-sun-azimuth D` | Place the sun (degrees; azimuth clockwise from north) |
| `-zoom F` | Start closer (0.1 = ten times closer) |
| `-no-shadows` | Start without shadows |
| `-width` / `-height` | Window size |
| `-capture frame.png` | Render one frame to a PNG, hidden window |

### glTF extras read by the renderer
* scene `extras.environment`: `sunDirection`, `sunColor`, `skyZenith`, `skyHorizon`, `groundAmbient`, `ambient`, `fog {color, near, far}`, `water {level, deep, shallow, size, waveLength, floorDepth, colorDepth}`, `verticalScale`, `sky`:
  * `nightZenith`, `nightHorizon`, `nightAmbient`, `twilightColor`, `celestialPole`: vectors
  * `stars {density, brightness, twinkle, daytimeVisibility}`
  * `moon {direction, size (degrees), color, phase (0 new, 0.5 full), light}` or `false`
  * `clouds {coverage, color, speed, scale}`
  * `constellations [{name, stars: [[x, y, z], ...] as seen at midnight, lines: [[first, second], ...], color}]`
  * `dayImage`, `nightImage`: `"panorama.png"` or `{faces: [+X, -X, +Y, -Y, +Z, -Z]}`, paths relative to the glTF file (PNG or JPEG); the night image turns with the stars
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
