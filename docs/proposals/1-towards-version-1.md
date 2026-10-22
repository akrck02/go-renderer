# 1. Go Renderer towards 1.0

> Status: building · Author's approval: proposal ☑ (2026-10-22, with the decisions below) plan ☑ (2026-10-22, "Si")

## Why

Go Renderer is an educational experiment at v0.2.0. Games and libraries (Go UI, go-engine) already
build on it, so a version they can trust would help them. Today nothing promises that:

- **Little is checked automatically.** Until this branch, no workflow ran the tests. The OpenGL
  scene renderer is never checked by a program: its tests skip without a display, and a frame
  that came out wrong would pass.
- **Stub backends.** `software`, `vulkan` and `webgpu` are hand-written beginnings of the
  immediate-mode contract: every call returns "Graphics API not implemented yet!". A newcomer cannot tell that from the folder list.
- **Metal is tried by hand.** It runs on the author's Mac only, and newer SDKs (macOS 27)
  deprecate `MTLStorageModeManaged`, which its textures use.
- **No releases since v0.2.0.** Paths, strokes, fullscreen, pads and `DrawOver` landed on main
  since; nothing says what may break between versions.
- **AI-written parts are pending review.** The README says so; a 1.0 should not.
- **Central packages are thinly tested.** Coverage of `windowloop` is 6 %, `graphics` 46 % and
  `scene` 50 %.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | Format, vet and tests run on every push and pull request, and the browser packages are vetted for WebAssembly | `.github/workflows/test.yaml` runs green. **Done by `feature/docs-and-ci`** |
| R2 | The OpenGL scene renderer is tested automatically: frames of the sample island drawn under Mesa without a display (Xvfb or a surfaceless context), or by a software painter, compared with reference pictures within a tolerance | a CI job that draws and compares; a test fails when a pass is broken on purpose |
| R3 | Each stub backend (`software`, `vulkan`, `webgpu`) is implemented, removed, or marked unavailable with an error code and a line in the docs. They are the author's hand-written code: he decides which | the author's decision recorded here; the docs' backend table says the state of each |
| R4 | Metal is tried on a real Mac before each release, and builds without deprecation warnings on the newest SDK (`MTLStorageModeManaged` replaced where it is deprecated) | a checklist line in each release's notes; a clean build log on macOS 27 |
| R5 | Releases are tagged with notes, and 1.0 names what stays stable (the `Renderer` contract? `scene`, `draw2d`, `input`?) and what may still change. This ties to go-engine's [proposal 14](https://github.com/akrck02/go-engine/blob/main/docs/proposals/14-stable-releases.md), which needs Go Renderer released first | the release page; a page on what 1.0 promises |
| R6 | The author reviews the AI-written parts listed in the README, and the *AI disclosure* says what was reviewed | the README's disclosure, updated |
| R7 | The central packages are tested where a test proves behaviour (`windowloop`'s fullscreen and pads, `graphics`' transforms, `scene`'s bounds and instances); no tests written only to raise a percentage | the new tests' names say what must be true; coverage is reported, not targeted |

## Out of scope

- New features (fills of paths, areas in Metal and the browser): 1.0 is about trust, not reach.
- Building Android and the 3DS on every push: they need the NDK and a TinyGo fork; go-engine's
  proposal 14 covers them for the whole family.
- Changing the hand-written `Renderer` contract or backends without the author.

## Decisions

The author's answers (2026-10-22):

1. **The stub backends** (`software`, `vulkan`, `webgpu`) stay, to be built later. The docs say so,
   and every function returns an error that says it is not implemented, with a code.
2. **What 1.0 promises: every package.** So every package first passes an audit against the
   author's guidelines (llm-guidelines), and its API is made simple and efficient.
3. **Frames in CI:** Mesa's software driver under Xvfb.
4. **The review:** the author reviews every AI-written part once the audit is done and what it
   asks is simplified or rewritten.
5. **The Mac:** a MacBook Air M4, where Metal has been tried so far.

And above all: **performance and a simple API (developer experience) are imperative.** Every
change of this proposal is weighed by both: an API a newcomer reads in minutes, and frames that
cost no more than today, measured.

## Design

**One error package.** Go Renderer has no error codes today (33 files build errors from text). A
small package, `failure` as in go-engine (`Define`, `Wrap`, codes compared with `errors.Is`), is
shared by every package; each declares its codes in its own `errors.go`. The stub backends return
`<backend>.not-implemented`.

**The audit, before any rewrite.** A reviewer reads each package against llm-guidelines (names,
size of functions, early returns, error codes, generic names, nothing allocated per frame,
comments, tests that prove behaviour, docs) and against the two imperatives: is the API the
smallest that does the job, and what does a frame cost. Its findings go in one document,
`docs/proposals/1-audit.md`, package by package, each finding with a severity and what to do.
Nothing is changed while auditing: the author reads the findings and the plan of each package
before it is rewritten.

**Measured performance.** Before rewriting, the benchmarks that exist (34) are run and kept as a
baseline, with new ones where a frame's hot path has none (the scene renderer's frame, a 2D list
of three hundred controls, a pad's reading). A change that makes a frame slower or allocate more
does not merge.

**Frames compared in CI.** A test draws the sample island and a 2D list under Xvfb with Mesa
(llvmpipe) and compares them with reference pictures in the repository, within a tolerance; a
pass broken on purpose fails it. The CI installs Mesa and runs it under `xvfb-run`.

**Metal.** The AI-written Metal renderers use `MTLStorageModeShared` where `Managed` is deprecated
(it does nothing on Apple silicon); the author's hand-written stubs are left as they are. The
release checklist names the MacBook Air M4.

**1.0's promise.** A page, `docs/6-project/3-version-1.md`, says that every package is stable from
1.0: what that means (no breaking change without a major version), how a change is deprecated
first, and that releases carry notes (go-engine's proposal 14).

## Tasks

- [x] T1 — the `failure` package; the stub backends return `<backend>.not-implemented`; the docs' backend table and each stub's page say they come later · R3 · a test per stub that every function returns its code
- [x] T2 — baseline benchmarks, and new ones for a frame's hot paths · `*_bench_test.go` · imperatives · the numbers kept in `docs/6-project/1-optimization-plan.md`
- [x] T3 — the audit, every package, findings in [`docs/proposals/1-audit.md`](1-audit.md) (written 2026-10-22; awaits the author) with a plan per package · all · R6, R7 · **stop: the author reads it and approves each package's plan**
- [ ] T4 — rewrite package by package as the audit's approved plans say, each with its tests and docs, never slower nor allocating more than the baseline · per package · R6, R7 · tests, benchmarks against the baseline
  - Phases A and B of the [audit's order](1-audit.md#order-proposed) approved by the author for every package (2026-10-22, "Si"): A, no breaking change and a frame's cost first; B, a smaller API that breaks nothing outside. Phases C (breaking, AI-written) and D (his hand-written code) await him.
- [ ] T5 — frames compared in CI under Mesa · `opengl`, `.github/workflows` · R2 · the job; a broken pass fails it
- [ ] T6 — Metal: `MTLStorageModeShared` where deprecated · `metal` (AI-written parts) · R4 · builds clean on the newest SDK (on the author's Mac); checklist line
- [ ] T7 — the 1.0 page, error codes in the troubleshooting chapter, the README's status · docs · R5 · the pages
- [ ] T8 — the author's review of every AI-written part; the AI disclosure updated · R6 · the README

## Result

Building. R1 is done; T1 added the `failure` package and the stub backends' codes; T2 kept the benchmarks' baseline in the optimization plan; T3's [audit](1-audit.md) is written and awaits the author, package by package, before T4.
