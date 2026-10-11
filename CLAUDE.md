# go-renderer

A tiny, educational graphics renderer in Go: an immediate-mode `Renderer` contract with several
backends, plus a retained glTF scene, an OpenGL scene renderer and a viewer (`cmd/visor`). Start
with `docs/architecture.md`; the viewer controls are in `docs/viewer.md`.

## Commands

```bash
go vet ./...
go test ./camera ./opengl ./graphics ./loaders ./scene
go run ./cmd/visor scene.glb                     # needs a display and OpenGL 4.1
go run ./cmd/visor -capture frame.png scene.glb  # one frame to PNG
```

The OpenGL packages use cgo (GLFW): they need the OpenGL, X11/Wayland and GLFW build headers.

## Rules

- **Generic only.** This project knows nothing about any particular world, game or story. Never put
  names, places, dates, numbers or settings from a user's own project in code, comments, tests,
  examples, docs or commit messages. Examples use made-up places (like the example island in the docs)
  and round, neutral numbers.
- **Readable code.** Full-word names (no acronyms, no one-letter variables), small functions that do
  one thing, early returns instead of nested branches, comments that say *why*. Match the code
  around you.
- **Tests with every change.** `go vet ./...` and `go test ./...` must pass before committing.
- **Docs live in `docs/`**, written for people: explanatory, friendly, with 2D images where they
  help. Keep the README short and point to `docs/`. Update the docs in the same commit as the
  behaviour they describe.
- **AI disclosure.** Commits written with an AI assistant end with a `Co-Authored-By: Claude …`
  line, and the *AI disclosure* section of the README stays true. The
  code is pending human review: say so, don't overclaim.
- **Git.** Commit only when asked; never push, force-push, rewrite history or delete branches or
  the repository without the author's explicit permission. Do not change the global git config.

## Human and AI code

- `main` holds hand-written code only. AI work goes on feature branches (`feature/…`) and is merged
  by the author after review.
- The immediate-mode `Renderer` contract and the `software`, `vulkan`, `metal` and `webgpu`
  backends are hand-written: don't change them without asking. The AI-written parts are listed in
  the README's *AI disclosure*.
- Rendering changes can't be checked without a display: say what was tested (build, unit tests)
  and what still needs a look on screen.
