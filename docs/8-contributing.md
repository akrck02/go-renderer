# 🤝 Contributing

This page is for someone who wants to change Go Renderer. At the end you will know where each
part lives, how to build and test it, and how a change is proposed.

## The layout of the code

Each folder is one idea, and dependencies point one way: the scene knows nothing of OpenGL.

| Folder | What lives there | Needs |
|---|---|---|
| `graphics`, `models` | Vectors, matrices, the application and the immediate-mode `Renderer` contract | plain Go |
| `scene`, `scene/sample`, `loaders` | The retained scene, glTF files, the sample island | plain Go |
| `sceneplan` | What a frame draws, decided once for every backend | plain Go |
| `input`, `camera`, `draw2d`, `frameloop` | Input, cameras, 2D draw lists, the clock | plain Go |
| `opengl`, `opengl/overlay`, `windowloop` | The desktop backend: GLFW, OpenGL 4.1, the overlay | cgo |
| `metal` | The Metal backend on macOS (Objective-C) | cgo, macOS |
| `gles`, `browser`, `android` | OpenGL ES renderers, the web page, the Android activity | WebAssembly or the NDK |
| `ctr` | The Nintendo 3DS, through citro3d | TinyGo |
| `software`, `vulkan`, `webgpu` | Backends of the immediate-mode contract, not finished | — |
| `viewer`, `cmd/…` | The viewer and the programs | cgo |
| `docs/` | These pages; `docs/5-reference` is generated | — |

The bigger picture is in [the architecture](4-guides/1-architecture.md).

## Building and testing

You need Go (the version in `go.mod`) and the headers of [troubleshooting](7-troubleshooting.md#building).

```sh
gofmt -l .        # prints nothing when everything is formatted
go vet ./...
go test ./...
```

Tests that need OpenGL open a hidden window, and skip themselves without a display. To run them
on a computer with no screen, use Xvfb (`xvfb-run go test ./opengl`); the CI has no display, so
it skips them. The examples in `*_test.go` (`ExampleFlatRenderer`,
`ExampleState_Pressed`…) are the guides' code: if a guide's code changes, change its example.

After changing a public name or its comment, regenerate the reference:

```sh
python3 docs/5-reference/generate.py
```

### What the CI checks

`.github/workflows/test.yaml` runs on every push and pull request:

| Job | Checks |
|---|---|
| `test` | `gofmt`, `go vet ./...` and `go test ./...` on Ubuntu, with the OpenGL, X11 and Wayland headers |
| `browser` | `go vet` with `GOOS=js GOARCH=wasm` of every package a web page uses |

It does not build Metal (macOS), Android or the 3DS yet, and it does not draw frames. A change to
those parts is tried by hand: say what you tried.

`.github/workflows/publish-docs.yaml` publishes `docs/` as a website on every push to `main`.

## Hand-written and AI-written code

Part of this project is written by hand by its author, and part by an AI assistant under his
direction. The README's *AI disclosure* lists which.

- **Hand-written, not to change without asking the author:** the immediate-mode `Renderer`
  contract (`models`), and the `software`, `vulkan`, `metal` (its hand-written stubs) and `webgpu`
  backends.
- **AI-written work** goes on a `feature/…` branch, never on `main`. Its commits end with a
  `Co-Authored-By:` line, and the author merges it after review. Until then it is pending review.

## Proposing a change

A small fix goes straight to a pull request. A change that spans several files or sessions starts
as a **proposal** in `docs/proposals/`, numbered in order:

| Part | What it says |
|---|---|
| Status | Proposed, approved or built, with the author's approval of the proposal and of the plan |
| Why | The problem, with facts |
| What | A table of requirements, each with an id (R1, R2…) and how it is checked |
| Out of scope | What it will not do |
| Open questions for the author | The decisions only he can take |
| Result | What was built, filled in at the end |

The author approves the proposal before a plan is written, and the plan before code. The first
one is [towards version 1.0](proposals/1-towards-version-1.md).

## Writing docs

Pages are for people: one task per guide, the shortest working code first, short sentences, and a
picture where something is spatial. Every code block runs as shown. Update the docs in the same
commit as the behaviour they describe.
