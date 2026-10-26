# 2. Textures on meshes

> Status: building · Author's approval: proposal ☑ (2026-10-26, with the decisions below) plan ☑ (2026-10-26, "Dale, implementa")

## Why

A 3D surface in Go Renderer has one colour: its material's `BaseColor` times its vertex colours,
plus six procedural patterns (cobbles, masonry, plaster, shingles, planks, grass) that only the
desktop's shaders draw. A game that shows more needs pictures on its surfaces:

- **Tiling detail:** panels, a grid, stripes, carbon weave, hexagons, wear, camouflage, windows —
  small pictures repeated across a part and tinted by its colour.
- **Pictures made at run time:** a game generates a pattern, or downloads one, and gives it to many
  meshes at once.
- **Models that carry their own:** a `.glb` file's textures are read and thrown away today.
- **The same on every target**, the 3DS included, within its limits.

Today `scene.Mesh` has no texture coordinates and `scene.Material` no texture, on any backend.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | A mesh built from code carries texture coordinates (`UVs`, two per vertex) | test: a mesh with UVs uploads and draws; a mesh without them draws as today |
| R2 | A material can carry a texture: the surface's colour is `BaseColor` × vertex colour × the texture, lit as before | frames compared under Mesa: a tinted texture, the same texture in two colours |
| R3 | A texture repeats (or clamps, or mirrors) across the surface, with a scale and an offset on the material, so one small picture tiles a large part | frames compared: a 64 px picture repeated eight times across a box |
| R4 | A texture is made from Go pixels at run time (an `image.Image`, or pixels written and marked changed), kept and shared by many meshes; changing it changes every mesh that shows it, uploaded once | test: one texture on a hundred meshes is uploaded once; a change uploads it again once |
| R5 | The 3DS draws textured meshes too: the texture multiplies the baked colour, its sizes and formats are written down (powers of two from 8 to 1024 px a side; which formats), and a texture that does not fit is scaled or refused with an error code | Azahar capture of a tiled, tinted part; tests of the size rules |
| R6 | `loaders.ReadScene` and `LoadScene` read a glTF's base colour textures and texture coordinates (`TEXCOORD_0`, `baseColorTexture`, its sampler's wrapping) | test with a small `.glb` that carries a texture |
| R7 | A frame with textures allocates nothing, and a scene without textures costs what it costs today | the benchmarks of the optimization plan, before and after |

## Out of scope

- Normal, roughness, metallic and emissive maps: one colour texture first.
- Texture atlases built by the renderer, and texture compression on the desktop.
- Fetching pictures from the network: a program downloads them and hands over the pixels.

## Decisions

The author's answers (2026-10-26):

1. **Where a texture lives:** a small `scene.Texture` that holds a `draw2d.Image` for its pixels
   (one way to keep pixels, for 2D and 3D alike) and the settings only 3D needs: wrapping,
   filtering, the 3DS's format ("Me parece correcto").
2. **Filtering:** chosen per texture, smooth or sharp.
3. **The 3DS's formats:** save memory where it can: RGBA8, RGB565 and ETC1, chosen per texture.
4. **The procedural patterns:** they become built-in textures, so every target, the 3DS included,
   draws them; the pattern code in the desktop's shaders goes.

## Design

**In `scene`.** `Texture` is small and kept by the program:

```go
type Texture struct {
	Picture   *draw2d.Image // its pixels; Picture.Changed() uploads them again
	Wrap      Wrap          // Repeat (the zero value), Clamp, Mirror
	Sharp     bool          // nearest pixel instead of smooth (with mipmaps on the desktop)
	Format    TextureFormat // on the 3DS: RGBA8 (default), RGB565 or ETC1; the desktop keeps RGBA8
}
```

`Mesh` gains `UVs []float32` (two per vertex, as `Positions` has three). `Material` gains
`Texture *Texture` and `TextureScale`, `TextureOffset` (`[2]float32`; a zero scale is 1): the
surface's colour is `BaseColor` × vertex colour × the texture at (UV × scale + offset). A material
without a texture draws exactly as today. A textured mesh without UVs takes them from its
positions (`Mesh.SurfaceUVs`: x and z on what faces up, along and up on walls, one model unit a
repeat), as the procedural patterns did from the world position: the island's terrain, a box made
in code and a model without `TEXCOORD_0` can still wear a tiling texture.

*Changed while building (2026-10-26):* the plan said a mesh without UVs draws as today; the
patterns it replaces need no UVs, so without the projection every pattern would have been lost on
meshes that have none. The projection is worked out once, on the processor, when the mesh is
uploaded: every backend, the 3DS included, gets the same UVs and no shader changes for it.

**Built-in textures.** `scene.BuiltInTexture(name string) (*Texture, bool)` gives textures made in
Go the first time they are asked for, then kept: the six patterns of today (cobbles, masonry,
plaster, shingles, planks, grass) and common tiling ones (panels, grid, stripes, carbon, hexagons,
worn, camouflage, windows), each a small power of two that tiles without a seam. `Material.Pattern`
and `PatternScale` go, with the patterns' code in the desktop's shaders: a material asks for
`BuiltInTexture("cobbles")` and a `TextureScale` instead. Nobody outside the repository uses
`Pattern` (go-engine and Go UI checked).

**The backends.** Each keeps uploaded textures by the picture's id and version, as the 2D picture
atlas does: one upload for any number of meshes, again only after `Changed()`; opaque draws are
sorted by texture, so a texture is bound once a frame. (*Changed while building:* materials were
never sorted, and sorting by them would buy nothing, since a material is only uniforms; the
untextured draws keep their order, first.) A texture's pixels are sRGB, as a PNG's are: the
desktop and the browser turn them into linear light as they read them.
- **OpenGL (desktop), OpenGL ES (Android and the browser), Metal (its AI-written renderers):** a
  `texCoord` attribute, a sampler, wrapping and filtering as the texture says, mipmaps when smooth.
- **The 3DS:** the PICA200's texture combiner multiplies the texture by the baked vertex colour, so
  lighting stays baked and textures cost a texture read per pixel, no vertex work. Sizes are powers
  of two from 8 to 1024 a side (a larger or odd picture is scaled once, with a warning). RGB565 and
  ETC1 are encoded in Go when uploaded (ETC1: a small encoder, 4 bits a pixel, no alpha; a texture
  with transparency stays RGBA8 or is refused with an error code).

**glTF.** `ReadScene` and `LoadScene` read `TEXCOORD_0`, the base colour texture of the PBR
material (PNG or JPEG, embedded in a `.glb` or by URI beside the file) and its sampler's wrapping
and filtering.

**Alternatives not taken.** Textures in `draw2d.Image` itself: 3D-only settings in a 2D type. A
texture per mesh instead of per material: materials are already what meshes share. Lighting per
pixel on the 3DS for textured meshes: the measurements show its vertex work is the cost to avoid.

## Tasks

- [x] T1 — `scene.Texture`, `Wrap`, `TextureFormat`; `Mesh.UVs`; `Material.Texture`, `TextureScale`, `TextureOffset` · `scene` · R1, R2, R3 · tests of the defaults and of a mesh's UV count
- [x] T2 — built-in textures that tile without a seam; `Pattern` removed · `scene` · R4 · test: each tiles (its edges match), each is made once
- [x] T3 — OpenGL: upload once per version, wrapping, filtering, mipmaps, sorting by texture · `opengl`, `internal/sceneplan` · R2–R4, R7 · frames compared under Mesa (a tinted tiled box, sharp and smooth); benchmarks with and without textures
- [x] T4 — OpenGL ES, for Android and the browser · `gles` · R2–R4 · tests with the recorder; an APK on the emulator
- [x] T5 — Metal (its AI-written renderers) · `metal` · R2–R4 · the author builds and looks on his Mac
- [ ] T6 — the 3DS: tiles to powers of two, RGBA8, RGB565 and an ETC1 encoder, the combiner's multiply · `nintendo3ds` · R5 · tests of the encoders against reference blocks and of the size rules; Azahar capture; frame time measured with and without textures
- [ ] T7 — glTF textures and UVs · `loaders` · R6 · test with a small `.glb`
- [ ] T8 — docs: a guide *Textures on meshes*, the 3DS guide's limits, the reference; then go-engine's scenes can name a material's texture (a follow-up there) · `docs` · R1–R6

## Result

Being built.
