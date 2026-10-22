# 1. Go Renderer towards 1.0

> Status: proposed · Author's approval: proposal ☐ plan ☐

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

## Open questions for the author

1. **The stub backends.** Implement, remove, or keep them marked unavailable? Each one separately?
2. **What 1.0 promises.** Which packages are stable at 1.0, and which stay experimental?
3. **Frames in CI.** Is Mesa's software driver under Xvfb good enough as a reference, or do you
   prefer a software painter that needs no driver at all?
4. **The review.** Which AI-written parts must you review before 1.0, and which can stay marked
   as pending?
5. **The Mac.** Which Mac and macOS versions count for the checklist?

## Result

Not built yet, except R1: waiting for the author's approval of this proposal.
