# 2. Textures on meshes

> Status: proposed · Author's approval: proposal ☐ plan ☐

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

1. **Where a texture lives:** open. Proposed: a small `scene.Texture` that holds a `draw2d.Image`
   for its pixels (one way to keep pixels, for 2D and 3D alike) and the settings only 3D needs:
   wrapping, filtering, the 3DS's format.
2. **Filtering:** chosen per texture, smooth or sharp.
3. **The 3DS's formats:** save memory where it can: RGBA8, RGB565 and ETC1, chosen per texture.
4. **The procedural patterns:** they become built-in textures, so every target, the 3DS included,
   draws them; the pattern code in the desktop's shaders goes.

## Result

Not built yet: waiting for the author's approval of this proposal.
