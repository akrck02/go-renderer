# 1. The audit before 1.0

> Part of [proposal 1](1-towards-version-1.md), task T3 · Written 2026-10-22 · **Awaits the
> author:** each package's plan is approved (or changed) by him before T4 rewrites it.

## How it was done

Every package was read against [llm-guidelines](https://github.com/akrck02/llm-guidelines) (and
its Go and docs chapters) and against the proposal's two imperatives: **a simple, efficient API**
(is the public API the smallest that does the job? what would a newcomer stumble on?) and
**performance** (what does a frame cost, where does it allocate?). Three tools helped the reading:

- a small syntax-tree scan of every `.go` file, whatever its build tags: functions over 40 lines,
  names of one or two letters, abbreviations, `errors.New`/`fmt.Errorf`, `panic`, printing,
  exported names without a comment;
- the [baseline benchmarks](../6-project/1-optimization-plan.md#baseline-2026-10-22) and memory
  profiles of the frames that still allocate;
- `go doc` of each package, and the imports of go-ui and go-engine, to know who uses what.

Nothing was changed while auditing. Line numbers are those of commit `2adf4ce`.

**Hand-written code.** The `Renderer` contract and `models`, the `graphics` maths, the first
OpenGL work (`opengl/opengl.go`, `opengl/renderer.go`, `opengl/window.go`, `loaders/glb.go`), the
root `main.go` and the `software`, `vulkan`, `webgpu` and `metal` stubs are the author's. Their
findings are listed like the others but marked **✋ author's decision**: this audit proposes, it
does not plan to rewrite them.

Severity: **high** breaks a rule that costs users or frames today, or blocks 1.0's promise;
**medium** is a rule broken where a newcomer would stumble; **low** is polish. Effort: **S** under
a day, **M** a few days, **L** a week or more.

## Summary

| Package | State | Main problems | Effort |
|---|---|---|---|
| `graphics` ✋ | needs work | `Vec4`/`Mat4` shorthand everywhere (529 one- or two-letter names); shaders of every backend live beside the maths; `Move` allocates | L (rename touches every package) |
| `models` ✋ | needs work | a package named for a kind of file; `Application` holds 25 fields of three concerns; `Camera` lives here, not in `camera`; no comments on public names | M |
| root `main.go` ✋ | needs work | an example that panics, a package-level `first`, a fixed backend constant | S |
| `software`, `vulkan`, `webgpu` ✋ | listed only | stubs; return their codes since T1 | — |
| `metal` (AI parts) | good, one risk | `ErrUnavailable` is text, not a code; `MTLStorageModeManaged` (T6); the hand-written `Metal` stub still returns a plain text error ✋ | S |
| `opengl` | needs work | `panic`, `log` and `fmt.Print` in a library (hand-written parts ✋); `OpenGLRenderer` stutters; a frame allocates 3–4 times; package-level `firstFrame` | M |
| `opengl/overlay` | good, two fixes | one allocation per frame (`sort.SliceStable`); global `UseFonts` and `FontFolders` read at import; `Audit…` benchmark and test names | S |
| `sceneplan` | good, one fix | two allocations per frame (water nodes); 32 exported functions only backends use: belongs in `internal/` | M |
| `scene` | good | `ColorFromHex` duplicates `draw2d.ParseHex`; `DefaultNoticeDuration` has no unit; `Kind…` prefix vs `…Kind` suffix elsewhere | S |
| `scene/sample` | good | none worth a rewrite; magic numbers are its point (a made-up island) | — |
| `draw2d` | good | `Hex` panics; text errors; `Rect` abbreviation; `Command.Text` doubles as an icon's name | S |
| `input` | good, large | `State` mixes what programs read and what backends write (56 methods); `MultiClickDistance` without unit | M |
| `camera` | good | `Plan` collides with `sceneplan` in a reader's head; `float64`/`float32` mixed in constructors | S |
| `loaders` | needs work | `LoadGLB` ✋ is 123 lines and prints to stdout; 14 text errors; three names for two things | M |
| `viewer` | needs work | `Run` takes a benchmark-only argument; `Options` has 20 fields of three kinds; a 300-character `Usage` constant; prints | M |
| `windowloop` | fair | re-exports `frameloop` (`Clock` alias, `RunUpdateStep`); `Run` 51 lines; 5.5 % tested | S |
| `frameloop` | good | none | — |
| `gles` | good | `Functions` mirrors GL's names (`Uniform1i`, `TexImage2D`): a choice to confirm | S |
| `android` | fair | 50 exported GL functions without comments; platform services (notifications, palettes, haptics, video) in a renderer; `Channel`/`Tones` mutable globals | M |
| `browser` | fair | 50 exported WebGL functions without comments; text errors | S |
| `ctr` | fair | device names in a generic API (`OpenBottomScreen`, `Slider3D`, `PICAMatrix`, `CStick`); two ways to read buttons and begin a frame | M |
| `cmd/*` | fair | `visor` uses `os.Exit` (fine in `main`); 3DS examples with `main` of 52–59 lines | S |
| `failure` | new | — | — |
| repository | fair | `.gitignore` lacks `.env` and `out/`; compiled `.shbin` shaders in git | S |

### The ten findings that matter most

1. **`graphics.Vec4` and `graphics.Mat4`** are the shorthand the guidelines name by example
   (`Vector`, not `Vec3`). They are the most used names of the module (go-ui 35 uses, go-engine
   42). ✋ Hand-written: renaming is the author's call, and the largest breaking change of 1.0.
2. **A frame still allocates.** Planning the sample island: 2 allocations (288 B), both a
   `scene.Node` made for the water every frame (`sceneplan/commands.go:161`, `:176`). The overlay:
   1 (`sort.SliceStable` in `opengl/overlay/layout.go:243`). An OpenGL 2D frame: 3–4 (`gl.Ptr` and
   `GetIntegerv` taking addresses in `opengl/overlay.go:263`, `:110`). All four can reach zero.
3. **No error has a code** outside the stubs: 64 `errors.New`/`fmt.Errorf` in 27 files, plus
   `metal.ErrUnavailable` and `browser.ErrNoCanvas`/`ErrNoWebGL2` that are plain sentinels.
4. **Library code panics, logs and prints.** `opengl/opengl.go:129–149` and
   `opengl/window.go:14,43` panic (✋); `opengl/renderer.go:282` calls `log.Fatalf` (✋);
   `loaders/glb.go:103–180` prints every mesh (✋); `opengl/opengl.go:162` and
   `viewer/viewer.go:434` print. A program cannot silence or handle them.
5. **`models` is a package named for a kind of file**, and `models.Application` carries
   window settings, frame callbacks, input and the immediate-mode matrices in one struct
   (`models/application.go:59`). `models.Camera` belongs in `camera`. ✋ Hand-written core.
6. **`sceneplan` exports 32 functions and 21 types that only the module's backends use**
   (`InterleaveVertices`, `ScaleOrOne`, `ColorOrWhite`…). As `internal/sceneplan`, the public API
   shrinks by a third and nothing outside breaks (go-ui and go-engine import none of it).
7. **`viewer.Run(world, options, title, update, loadingTime)`**: `loadingTime` only feeds the
   benchmark printout, yet every caller must pass it. `Options` mixes start-up view, rendering
   switches and benchmark settings in 20 flat fields.
8. **`input.State` is one struct for two audiences**: programs read it (`Pressed`, `Clicks`), and
   backends write it (`SetKey`, `MoveCursor`, `AddCharacter`, `PressBack`…). A newcomer reading
   `go doc input.State` sees 56 methods, half of which they must not call.
9. **Global mutable state in libraries**: `overlay.UseFonts` sets a package variable
   (`opengl/overlay/fonts.go:41`), `overlay.FontFolders` reads the home folder at import
   (`fallback.go:26`), `android.Channel` and `android.Tones` are exported variables, and
   `opengl.firstFrame` (`opengl.go:156`) is shared by every window.
10. **Generic names**: `ctr` speaks the device (`OpenBottomScreen`, `BottomWidth`, `Slider3D`,
    `CStick`, `PICAMatrix`), where the guidelines ask for `OpenSecondScreen`, `RightStick`. The
    package name `ctr` itself is the console's code name, an abbreviation to a newcomer.

### Breaking changes for go-ui and go-engine

If every plan below is approved as written, these are what the two projects would have to change.
Each comes with a release note and, where cheap, a deprecated alias for one minor version.

| Change | go-ui | go-engine |
|---|---|---|
| `graphics.Vec4` → `graphics.Vector`, `Mat4` → `Matrix` ✋ | 35 uses, + 2 `Identity`/`Translate` | 42 uses, + `Mat4` 3, `Identity` 6, `Scale`, `RotateY`, `FromTranslationRotationScale` |
| `models.Camera` → `camera.Camera`; `models.Application` → split (see `models`) ✋ | `Camera` 8, `Application` 13, `WindowApplication`/`HeadlessApplication` | `Camera` 10 |
| `opengl.OpenGLRenderer` → `opengl.Renderer`, `opengl.OpenGL` → `opengl.Window` ✋ | 1 + 1 | — |
| `ctr` renamed and its device names made generic | `ScreenWidth`, `BottomWidth`/`Height`, `OpenBottomScreen`, `Slider3D`, `Buttons`… (≈20 lines) | `Reading` 33, `Button…` ≈40, `ReadControls`, `StartMotionControls`, `Gyroscope` |
| `input.State`'s writing methods move to a backend side (`input.Feed` or similar) | its own backends that call `SetKey`, `MoveCursor`… (the window adapters) | none (it only reads) |
| `viewer.Run` loses `loadingTime`; `Options` grouped | 3 calls of `Run`, 4 of `DefaultOptions` | — |
| `overlay.UseFonts` → an option of `NewPainter`/`NewMeasurer` | 1 | — |
| `scene.ColorFromHex` removed (use `draw2d.ParseHex`) | check | check |
| errors checked with `errors.Is(err, <package>.<Code>)` instead of text | `metal.ErrUnavailable` 1 | — |
| `loaders.LoadGLB` → `loaders.LoadModel` ✋, `ParseSceneGLB` → `ReadScene` | — | `ParseSceneGLB` 1 |

`sceneplan` going internal, the allocations, the shaders' move and the comments break nothing
outside.

---

## graphics ✋

Hand-written maths (`transformations.go`, `space.go`, `shaders.go`), extended by AI with shader
sources for every backend (`scene_shaders.go`, `overlay_shaders.go`, `light_shaders.go`,
`backdrop_shaders.go`). 46 % tested.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| high ✋ | `transformations.go:9,13` | `Vec4`, `Mat4`: mathematical shorthand the guidelines forbid by name | `Vector`, `Matrix` (a `Vec4` alias kept one version, deprecated) |
| medium ✋ | `transformations.go` (529 names) | `a`, `b`, `m`, `v`, `res`, `col`, `bColIdx`, `fovyRad` | whole words: `left`, `right`, `matrix`, `vector`, `result`, `column`, `fieldOfViewRadians` |
| medium | `scene_shaders.go` and the other `*_shaders.go` | shader sources of OpenGL's scene, overlay, lights and backdrop sit in the maths package: it grows with every backend, and `gles`/`metal` carry their own copies anyway | move each source beside the backend that compiles it (`opengl/shaders/…` or `opengl/scene_shaders.go`); `graphics` keeps maths only |
| medium ✋ | `transformations.go:149` | `Move` returns a new slice: allocates per call | move in place, or document it as load-time only |
| medium ✋ | `transformations.go:188–224` | `PerspectiveOpenGL`, `PerspectiveVulkan`, `PerspectiveMetal`: one function per backend in a generic package | one `Perspective(fieldOfView, aspect, near, far, depthRange)` with a `DepthRange` (`MinusOneToOne`, `ZeroToOne`), or keep and document |
| low ✋ | `transformations.go:240` | `Inverse` returns `(Mat4, bool)`: fine; the comment does not say when it fails | say "false for a matrix that cannot be inverted (its determinant is zero)" |
| low | `light_shaders.go:97` | `AmbientLight` is not maths: it is the lights' shader input | goes with the shaders |
| low ✋ | tests | 2 test files, `t *testing.T`, `TestInverseRoundTrip` | sentence names; a test per transform (the proposal's R7) |

**Plan.** (1) Move the shader sources to their backends: no API outside the module uses them
(neither go-ui nor go-engine). (2) The author decides on `Vector`/`Matrix` and the names inside;
if yes, it is one mechanical rename with `gofmt -r`, a deprecated alias for one version, and both
dependants updated in the same week. (3) Tests of each transform.

```go
// before                                   // after (if approved)
graphics.Vec4{0, 1, 0, 0}                   graphics.Vector{0, 1, 0, 0}
graphics.Mat4                               graphics.Matrix
graphics.PerspectiveOpenGL(fov, a, n, f)    graphics.Perspective(fieldOfView, aspect, near, far, graphics.MinusOneToOne)
graphics.SceneVertexShader                  (inside opengl)
```

## models ✋

The immediate-mode contract (`Renderer`, `GraphicsApi`, `Mesh`, `Model`) and `Application` and
`Camera`. Hand-written, extended by AI (`DrawOnDemand`, `Unchanged`, `FlatOnly`, `Input`…). 0 tests.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| high ✋ | package name | `models` names a kind of file, which the guidelines ask to avoid ("folders say what the program is about") | split by concept: the contract to `immediate` (or the root package), `Camera` to `camera`, `Application` to `window`/`app` |
| high ✋ | `application.go:59` | `Application` mixes the window (size, title, vsync, flat-only), the loop (`Update`, `Draw`, `FixedStep`, `DrawOnDemand`, `Unchanged`, `Stop`), input and the immediate-mode matrices: 25 fields. A newcomer cannot tell which matter | group: `Window` settings, `Loop` callbacks, the matrices stay with the immediate-mode contract |
| medium ✋ | `application.go:15` | `Camera` is in `models`, its controllers in `camera`: a reader looks in the wrong place | `camera.Camera` |
| medium ✋ | `application.go:19–21` | `FieldOfView float64` beside `Near, Far float32` | one type (float32, as the GPU takes) |
| medium ✋ | `application.go:48` | `0.7853981633974483`: a magic number | `math.Pi / 4`, named `defaultFieldOfView` |
| medium ✋ | `graphics_api.go` | `GraphicsApi`, `GraphicsApiBackend`: `Api` stutter and an acronym | `Backend` |
| medium ✋ | `renderer.go` | the contract takes `shader *string` and returns `error` for every shape: a newcomer cannot tell what the string is; errors per triangle cost a check per call | the author's contract: noted, not planned |
| low ✋ | all | 8 exported names without a comment (`Application`, `Camera`, `Renderer`, `Mesh`…) | comments that start with the name |

**Plan.** None until the author decides. The smallest step that breaks nothing: comments, named
constants, tests of `Camera.Projection`. The larger (split `Application`, move `Camera`) is the
second-largest breaking change and pays most for a newcomer.

```go
// before                              // after (one possible shape)
app := &models.Application{            app := &window.Application{
    Type: models.WindowApplication,        Window: window.Settings{Title: "…", Width: 1000, Height: 800},
    Title: "…", Width: 1000, …             Update: update, Draw: draw,
    Camera: models.Camera{…},          }
}                                       view := camera.Camera{…}
```

## Root `main.go` ✋

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium ✋ | `main.go:47,71` | `panic` where a `main` should print and exit | `log.Fatal` (allowed in `main`) |
| medium ✋ | `main.go:15` | `graphicsApi = models.OpenGL`: the example cannot try another backend without editing | a `-backend` flag |
| low ✋ | `main.go:77` | `var first = true` is unused | delete |
| low ✋ | `main.go:79` | `Draw` ignores the errors of `RenderTriangle`/`RenderRectangle` | return them |
| low | layout | a library's root holding `package main` means `import "github.com/akrck02/go-renderer"` imports nothing useful | move to `cmd/immediate` (or `examples/triangle`) |

## software, vulkan, webgpu ✋ (listed only)

Since T1 each function returns `<backend>.not-implemented`, tested. Not judged for rewrite. Note
for the author: `SetMatrices` returns nothing, so it cannot say it is not implemented; and the
empty `window.go` files could go.

## metal

AI-written renderers (`FlatWindow`, `FlatRenderer`, `SceneWindow`, `SceneRenderer`, the Objective-C
and the shaders) beside the hand-written stubs `metal.go`, `renderer.go`, `window.go` ✋. 35 %
tested on Linux (the darwin files cannot run here).

| Severity | Where | What and why | What to do |
|---|---|---|---|
| high | `flat.go:7` | `ErrUnavailable = errors.New("metal unavailable")`: a sentinel without a code; `flat_darwin.go:52`, `flat_other.go:17,25` wrap it with `fmt.Errorf` | `Unavailable = failure.Define("metal.unavailable", …)` and `failure.Wrap(Unavailable, reason, nil)`; keep `ErrUnavailable` as a deprecated alias one version (go-ui uses it once) |
| high | `scene_darwin.m:273`, `flat_darwin.m:651` | `MTLStorageModeManaged`, deprecated on newer SDKs | T6: `MTLStorageModeShared` |
| medium ✋ | `metal.go`, `renderer.go` | the stubs return `"Graphics API not implemented yet!"` as text | as the other stubs: `metal.NotImplemented` (the author asked it for the three others; Metal's stub is his call) |
| medium | `MetalRenderer` ✋, `FlatRenderer`… | `metal.MetalRenderer` stutters | the stub's name is the author's |
| low | `shader_check_darwin.go` | two text errors | codes |

**Plan.** Codes (S), T6 (S, on the author's Mac). The stubs: the author's decision.

## opengl

The OpenGL 4.1 backend: the immediate-mode window (✋ `opengl.go`, `renderer.go`, `window.go`),
the scene renderer, the flat renderer and the overlay's GPU side. 53 % tested (with a display).

| Severity | Where | What and why | What to do |
|---|---|---|---|
| high ✋ | `opengl.go:129,140,144,149`; `window.go:14,43` | `panic` when GL or a shader fails: a program cannot fall back (to Metal, to a message) | return coded errors (`opengl.ShaderNotCompiled`, `opengl.WindowNotOpened`) |
| high ✋ | `renderer.go:282–291` | `log.Fatalf` and `panic` when a texture fails | return the error |
| high | `overlay.go:263`, `overlay.go:110` | an OpenGL 2D frame allocates 3–4 times (182–466 B): `gl.Ptr(vertices)` escapes the slice header, `gl.GetIntegerv(…, &framebuffer)` the integer | keep `framebuffer` in the renderer; pass `unsafe.Pointer(&vertices[0])` as `gl.Ptr` does internally, or keep a pointer made once per buffer growth; a `testing.AllocsPerRun` test |
| medium ✋ | `opengl.go:156` | `var firstFrame = true`: shared by every window, and only prints | delete with the print |
| medium ✋ | `opengl.go:58,76,133,162` | `println`, `fmt.Printf`, `log.Println` | none in a library; the version can be a field (`OpenGL.Version()`) |
| medium ✋ | `opengl.go:82` | `setInputs` (42 lines) hard-codes arrow keys and WASD into the window when no `Input` is given: hidden behaviour | delete, or move to the example |
| medium ✋ | `opengl.go:221` | `DrawOnDisk` 61 lines; its comment starts `SaveFrameToDisk` | split into read pixels / flip / encode; `SaveFrame` |
| medium ✋ | names | `opengl.OpenGL`, `opengl.OpenGLRenderer` stutter; `InitWindow` is an abbreviation | `opengl.Window`, `opengl.Renderer`, `OpenWindow` |
| medium | `scene.go:33–45` | `ShadowSettings`, `FrameStatistics`, `PassTimes` are aliases of `sceneplan`'s: fine, but `DefaultShadowSettings` is duplicated | keep the aliases when `sceneplan` goes internal; they are the public face |
| medium | `opengl/opengl.go:32` | `StartLoop` 48 lines; `Draw` 47 | split once the hand-written parts are decided |
| low | `renderer.go` | `vao`, `vbo`, `ebo`, `img`, `rgba`, `r` (22×) | whole words (`vertexArray`, `vertexBuffer`, `elementBuffer`) ✋ |
| low | `shadows.go`, `opengl.go:34` | text errors | codes |
| low | tests | `t *testing.T` in 2 files | `test` |

**Plan.** AI parts first (S): the allocations to zero with a test, codes for its errors, `test`
names. Hand-written parts (M) wait for the author: they are where every panic and print is.

## opengl/overlay

Lays out labels, notices and draw lists into quads of one atlas; fonts, emoji, icons, the measurer.
84 % tested; the best-measured package (14 benchmarks).

| Severity | Where | What and why | What to do |
|---|---|---|---|
| high | `layout.go:243` | `sort.SliceStable` with a closure: the one allocation of a frame (24 B) | `slices.SortStableFunc(placed, byDepth)` with a function declared once; `AllocsPerRun` test |
| medium | `fonts.go:41` | `UseFonts` sets a package variable read by every painter: two windows cannot use two fonts, and tests interfere | an option: `NewPainter(Fonts)`, `NewMeasurer(Fonts)`; zero value = the default fonts |
| medium | `fallback.go:26`, `fonts.go:52` | `FontFolders` reads the home folder and `PrebuiltCharacters` builds a list at import: "no package reads anything when imported" | functions (`DefaultFontFolders()`), or computed on first use |
| medium | `audit_*_test.go` | `TestAuditFrameShape`, `BenchmarkAudit…`: named for the review that wrote them, not what they check; `t *testing.T`, `b *testing.B` | rename (`BenchmarkPaintingTwoThousandCommands`…) |
| medium | `colour.go` | 7 text errors; `pictureIn` 44 lines | codes; split |
| low | API | 12 exported types (`Entry`, `Snapshot`, `LightQuad`, `Lighting`, `Pictures`, `Projection`…) of which go-ui uses `Frame`, `Painter`, `Measurer`, `Fonts`, `Quad` | the rest could be unexported or internal; measure with go-ui's backends first |
| low | `style.go:33–44` | pill styles as literals of many numbers | named fields already; fine |

**Plan (S).** The allocation, fonts as an option, nothing read at import, test names, codes.

## sceneplan

What a frame of a scene draws, for every backend. 63 % tested.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| high | `commands.go:161`, `:176` | `&scene.Node{…}` for the water and its floor, every frame: the 2 allocations of `Plan` (288 B) | keep both nodes in the `Planner` with the meshes, update their transform |
| high | API | 32 exported functions and 21 types used only by `opengl`, `metal`, `gles`, `ctr` (`InterleaveVertices`, `ScaleOrOne`, `ColorOrWhite`, `ElementCount`, `LineColor`, `MergeInstances`…). They read as public API to a newcomer and would be frozen by 1.0 | `internal/sceneplan`; `opengl` keeps the aliases it exports |
| medium | `commands.go:157`, `:173` | `{0.05, 0.22, 0.28, 0.85}`, `0.6`: magic colour and darkening | named constants (`defaultDeepWater`, `waterFloorDarkening`) |
| medium | names | `ComputeCameraMatrices`, `BaseColorOf`, `ShallowColorOf`, `WaveLengthFor`: three styles of naming the same kind of function | one style: what they give back (`CameraMatrices`, `BaseColor`, `ShallowColor`, `WaveLength`) |
| low | tests | `t *testing.T` in 5 files; `TestBoxDistanceAndTransform` | sentence names |

**Plan (M).** Water nodes kept (S, with an `AllocsPerRun` test on `Plan`); move to `internal/`
(S, mechanical); names (S).

## scene

The retained scene. 51 % tested.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium | `overlay.go:191` | `ColorFromHex` (returns `bool`) duplicates `draw2d.ParseHex` (returns an error) | one: `draw2d.ParseHex` returns a colour both use, or `scene` takes `graphics.Vector` from it |
| medium | `overlay.go:36` | `DefaultNoticeDuration = 6.0`: no unit in the name | `DefaultNoticeSeconds` |
| medium | `scene.go:78`, `:105` | changes are told two ways: `Dirty bool` on meshes and instances, and `Instances.MarkChanged` | keep both (they do different jobs) but say so in `Mesh`'s comment |
| low | `MaterialKind` | `KindLit`, `KindUnlit` (prefix) where `draw2d` writes `RectangleKind` (suffix) | pick one convention for the module |
| low | tests | `TestMeshTop`, `TestColorFromHex`, `t *testing.T` | sentences; R7's bounds and instances tests |

**Plan (S).** Names with units, one hex parser, tests of bounds and instances (R7).

## scene/sample

The made-up island. Its numbers are its content. No plan.

## draw2d

Flat draw lists. 90 % tested; building 300 controls costs 31 µs and allocates nothing.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium | `draw2d.go:66` | `Hex` panics in a library | rename `MustHex` (Go's convention for "panics on bad constants", as `regexp.MustCompile`) so the name warns |
| medium | `draw2d.go:76,83`, `image.go` | `fmt.Errorf` | `draw2d.InvalidColour` code, wrapped |
| low | `draw2d.go:21` | `Rect` is an abbreviation | `Rectangle` (the `List.Rectangle` method is in another namespace) — breaking: go-ui and go-engine use `Rect` widely, so weigh it |
| low | `list.go:39` | `Command.Text` is the icon's name for icons | document, or an `Icon string` field (costs 16 B per command) |
| low | `image.go:62` | `ID()` | `Identity()` reads odd; `ID` is universal: keep |
| low | tests | `TestRectIntersect`, `t` in 2 files | sentences |

**Plan (S).** `MustHex`, codes, test names. `Rect` → `Rectangle` only if the author wants the
churn.

## input

Keyboard, mouse, touch, pads, clipboard, cursor shape, back button. 96 % tested; reading a pad
costs 19 ns and nothing.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium | `input.go:130` | `State` has 56 methods for two audiences: programs read, backends write (`SetKey`, `RepeatKey`, `AddCharacter`, `SetButton`, `MoveCursor`, `PressBack`, `PointerLeaves`, `AddScroll`, `SetPadButton`, touches…) | keep one `State`, but move the writers onto a `Feed` view (`state.Feed().SetKey(…)`) or a separate `input/backend`-style type; `go doc input.State` then shows only what programs use |
| medium | `input.go:126` | `MultiClickDistance = 4.0` has no unit | `MultiClickDistancePixels` |
| low | `input.go:152` | `StickX, StickY`, `RightStickX` exported fields beside methods for everything else | methods `Stick()`, `RightStick()` returning `(x, y)`, as `ctr.Reading` does |
| low | tests | `TestPointerButtons`, `TestHasInput`, `t` in 4 files | sentences |

**Plan (M).** The backend side (breaking for go-ui's adapters only), units, test names.

## camera

Orbit, walk and map controllers; rays through pixels. 66 % tested.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium | `plan.go:19` | `camera.Plan` (a map view) beside `sceneplan.Plan` (a frame): the same word for two things | `camera.Map` |
| medium | `orbit.go:29`, `plan.go:34`, `walk.go:27` | `NewOrbit(…, sceneSize float64)`, `NewWalk(x, z, eyeHeight, speed float32)`: mixed float types | one (float32, as `graphics`) |
| medium | `plan.go:36` | `0.0005`, `3`, `0.12`, `4` | named constants |
| low | — | `Camera` itself lives in `models` (see there) | `camera.Camera` if the author agrees |

**Plan (S).** Rename `Plan`, one float type, constants.

## loaders

glTF 2.0 with the renderer's extras (AI), and the first `LoadGLB` ✋. 64 % tested.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| high ✋ | `glb.go:60` | `LoadGLB` is 123 lines and prints every mesh and the bounds to stdout (`:103,119,159,180`); only reads float positions (`// Assuming float32 VEC3`) | the author's: split, no prints; or rebuild it on the AI glTF reader that already handles every type |
| medium ✋ | `glb.go` | `mIdx`, `pIdx`, `posIdx`, `idx`, `i` | whole words |
| medium | `gltf.go` (7), `sky.go` (3), `level_variation.go` (3) | text errors: a program cannot tell "file missing" from "bad glTF" | `loaders.FileNotRead`, `loaders.InvalidGLTF`, `loaders.UnsupportedAccessor` |
| medium | API | `LoadGLB` (→ `models.Model`), `LoadScene` (→ `scene.Scene`), `ParseSceneGLB` (bytes → scene): a newcomer asks which | `LoadModel`, `LoadScene`, `ReadScene(io.Reader)` |
| low | tests | `t` in 6 files | `test` |

**Plan (M).** Codes, names, and the author's decision on `LoadGLB`.

## viewer

The explorer. 40 % tested.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| high | `viewer.go:74` | `Run(world, options, title, update, loadingTime)`: `loadingTime` is only printed by benchmarks; go-ui's three callers pass `0` | put it in `Options.Benchmark` (or measure it inside) |
| medium | `options.go` | 20 flat fields of three kinds (where to start looking, rendering switches, benchmark/capture) | group: `Start` (`LookAt`, `Heading`, `Pitch`, `Walking`, `Map`), `Rendering`, `Measure` |
| medium | `options.go:74` | `Usage`, a 300-character constant that repeats `RegisterFlags` | built from the flag set (`flags.PrintDefaults`) |
| medium | `viewer.go:127`, `:434` | `log.Printf`, `fmt.Printf` | return the numbers (`viewer.Measurements`), print in `cmd/visor` |
| low | `options.go` | 3 text errors | codes |

**Plan (M).** `Run` without `loadingTime`, grouped options, no prints. Breaking for go-ui (3 calls).

## windowloop

The GLFW loop OpenGL and Metal share. 5.5 % tested.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium | `loop.go:75`, `:81` | `Clock = frameloop.Clock` and `RunUpdateStep` re-export `frameloop`: two names for one thing | callers use `frameloop` |
| medium | `loop.go:20` | `Run` 51 lines: events, pads, steps, drawing, cursor, fullscreen, title | split (`pollOrWait`, `applyRequests`, `updateTitle`) |
| medium | `loop.go:66` | `fmt.Sprintf` once a second for the title: fine, but the FPS title is a choice a program cannot turn off | an option |
| medium | tests | 5.5 %: fullscreen and pads untested (R7) | tests with a fake window interface |

**Plan (S).** Drop the re-exports, split `Run`, tests.

## frameloop

The fixed-step clock. Small, clear, tested. No plan.

## gles

Renderers written once for OpenGL ES (WebGL2, Android). 94 % tested.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium | `functions.go` | `Functions` mirrors GL (`Uniform1i`, `TexImage2D`, `RGBA`, `Enum`): against whole words, but a reader who knows GL finds them at once | **author's call**: keep GL's vocabulary at this boundary (recommended) and say why in the package comment |
| low | `program.go` | 2 text errors | codes |

## android

The NativeActivity: window, touches, keyboard, screen reader, and platform services.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium | `gl.go:46–244` | 49 exported GL functions without comments (`CreateShader`, `Uniform4f`…) | they implement `gles.Functions`: unexport them behind a `GL` value, or one comment for the block |
| medium | `notify.go`, `palettes.go`, `feeling.go`, `services.go`, `video.go`, `yuv.go` | notifications, wallpaper palettes, haptics, sharing, links and video in a renderer package | a second package, `android/system` (or into go-ui's platform layer): `android` keeps the window, input and accessibility |
| medium | `notify.go:16`, `palettes.go:20` | `Channel` and `Tones` exported variables: anyone can change them | constants or functions |
| low | `video.go` | text error | code |

**Plan (M).** Split services out; comments; no mutable globals.

## browser

The canvas and loop for WebAssembly.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium | `webgl.go:87–…` | ~45 exported WebGL functions without comments | as in `android` |
| medium | `canvas.go:31` | `ErrNoCanvas`, `ErrNoWebGL2` are text sentinels | `browser.CanvasNotFound`, `browser.WebGL2Missing` |
| low | `canvas.go:307` | `ClipboardText` without comment | comment |

## ctr

The 3DS: screens, controls, flat and scene renderers. 93 % tested on the host.

| Severity | Where | What and why | What to do |
|---|---|---|---|
| high | package name | `ctr` is the console's code name: a newcomer cannot guess it | `nintendo3ds` (matches the build tag) |
| medium | API | `OpenBottomScreen`, `BottomWidth`, `Slider3D`, `CStick`, `Circle`, `PICAMatrix`, `OrthographicTiltFromCitro3D`: device words in names; the guidelines ask for `OpenSecondScreen`, `RightStick`, `Stick`, `DepthSlider` | generic names; device words stay in comments |
| medium | `ctr.go:176`, `:214`; `controls_read.go:90` | `ButtonsPressed()` and `Controls.Pressed`, `BeginFrame()` and `Screen.BeginFrame`: two ways to do one thing | one each |
| medium | `ctr.go` (5), `flat.go` | text errors | codes |
| low | `ctr/shaders/*.shbin` | compiled shaders in git | a script that builds them (picasso), or document why they are kept |

**Plan (M).** Rename the package and the device words (breaking for go-engine: ≈80 lines, mostly
`Reading` and `Button…`; and go-ui: ≈20).

## cmd/*

| Severity | Where | What and why | What to do |
|---|---|---|---|
| low | `cmd/triangle3ds/main.go:36`, `island3ds/main.go:37` | `main` of 59 and 52 lines | split into `openScreens`, `loop` |
| low | `cmd/visor` | 0 % of statements tested (one test of its backends) | fine for a `main` |
| low | `cmd/web-island/main.go:27` | `log.Fatal` | fine in `main` |

## failure

New in T1; copied from go-engine. Nothing to audit.

## The repository

| Severity | Where | What and why | What to do |
|---|---|---|---|
| medium | `.gitignore` | lacks `.env` and `out/`, which every repository ignores | add them |
| low | `README.md` | the warning says APIs will break: true until 1.0 | T7 |

## Order proposed

1. **No breaking change, frame cost first** (S each): the four allocations to zero
   (`sceneplan`, `opengl/overlay`, `opengl`), each with an `AllocsPerRun` test; error codes in
   every AI package; no prints in AI code; test names.
2. **Shrink the API without breaking outside** (S–M): `internal/sceneplan`, shaders to their
   backends, `windowloop`'s re-exports, `overlay` fonts as an option.
3. **Breaking, AI-written** (M each): `viewer.Run`, `input`'s backend side, `ctr` → `nintendo3ds`,
   `camera.Plan` → `Map`, `loaders` names, `android` services.
4. **The author's code** (✋): `Vec4`/`Mat4`, `models`, the panics and prints of the first OpenGL
   work, `LoadGLB`, the Metal stub. Decided by the author, ideally together, because they are
   1.0's largest break and go-ui and go-engine should move once.

---

[← Proposal 1](1-towards-version-1.md)
