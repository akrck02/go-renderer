# The viewer

`cmd/visor` opens a glTF 2.0 scene (`.glb` or `.gltf`) and lets you explore it: fly around it like
a bird, or walk on the ground at eye height; move the sun to see it at dawn or at night; take a
picture or measure how fast it draws.

```bash
go run ./cmd/visor scene.glb
go run ./cmd/visor -walk scene.glb
go run ./cmd/visor -capture frame.png scene.glb   # one frame to PNG, hidden window
go run ./cmd/visor -benchmark 120 scene.glb       # CPU and GPU times per pass, draw calls, triangles
```

## Orbit camera (default)

| Control | Action |
|---|---|
| Left drag | Rotate around the target |
| Right drag (or shift + left drag) | Pan |
| Scroll | Zoom |
| `←` `→` | Rotate |
| `↑` `↓` | Tilt |
| `+` `-` | Zoom in or out |

## Walking (`Tab` switches)

| Control | Action |
|---|---|
| `W` `A` `S` `D` | Move forward, left, back, right |
| `↑` `↓` | Move forward or back |
| `←` `→` | Turn |
| Drag with any mouse button | Look around |
| `Shift` | Run |
| `[` `]` | Walk slower or faster (×1.5 per press) |

## Sun, sky and scene (both modes)

| Key | Action |
|---|---|
| `J` `L` | Turn the sun |
| `I` `K` | Raise or lower the sun (below the horizon it is night) |
| `N` | Jump between day and night |
| `Z` `X` | Less or more vertical exaggeration (orbit only) |
| `F` | Fog (toggle) |
| `O` | Shadows (toggle) |

## Options

| Option | Effect |
|---|---|
| `-walk` | Start walking on the ground |
| `-night` | Start at night |
| `-sun-elevation D` / `-sun-azimuth D` | Place the sun (degrees; negative elevation is below the horizon; azimuth clockwise from north) |
| `-zoom F` | Initial orbit distance as a fraction of the default (0.1 = ten times closer) |
| `-look-at x,z` | Start centred on this point, in scene units; with `-walk`, start walking there |
| `-heading D` | With `-walk`, start facing this way: degrees clockwise from north (−z) |
| `-pitch D` | With `-walk`, start looking up this many degrees (negative = down) |
| `-eye-height H` | With `-walk`, eye height in scene units (0 = a share of the scene size) |
| `-no-shadows` | Start without sun shadows |
| `-width` / `-height` | Window size |
| `-capture frame.png` | Render one frame to a PNG with a hidden window, and exit |
| `-benchmark N` | Measure N frames, print CPU and GPU times per pass, draw calls and triangles, and exit |
| `-no-vsync` | Do not wait for the display; the window title shows the real frames per second |
| `-cpu-profile file` | Write a CPU profile of the session for `go tool pprof` |

Other programs reuse these options: `viewer.Options.RegisterFlags` adds them to any flag set, as
go-simulator's live simulator does.
