# 6. SVG pictures

> Status: built, pending review · Author's approval: proposal ☑ ("dale", 2026-11-24) plan ☑ ("dale", 2026-11-24)

## Why

Go Renderer draws icons from a font (Lucide) and pictures from PNG or JPEG, but not from SVG, the
format nearly every icon set is drawn in. Coddy is the first to need it: it reads Zed's icon themes
(Coddy's proposal 10), thousands of small SVG files in their own colours. Any app that shows
icons it did not draw itself needs the same.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | `svg.Read` reads an SVG document into a drawing: `path` (every command, relative and absolute, arcs included), `rect` (rounded too), `circle`, `ellipse`, `line`, `polyline`, `polygon`, `g`, `viewBox`, `width` and `height`, `transform` (all six kinds), and fill, stroke and opacity given as attributes, in `style`, or inherited from a `g` | a test for each, on small made-up documents |
| R2 | A drawing becomes a picture (`draw2d.Image`) of any size in pixels, in its own colours, its edges smoothed, filled by the non-zero or the even-odd rule | tests on shapes whose coverage is known: a square half over a pixel, a ring drawn with each rule |
| R3 | Strokes have their width, caps (butt, round, square) and joins (miter with its limit, round, bevel) | tests on a line and on a corner of each kind |
| R4 | What it does not draw (text, masks, clip paths, filters, CSS classes) is left out without failing; a `use` of a shape paints that shape; a gradient is drawn in the plain colour of its stops' average | test: `a document with what is not drawn still gives its shapes`; a test per case |
| R5 | A document that is not SVG, or a path that cannot be read, gives an error with its code (`svg.not-an-svg`, `svg.bad-path`) | the errors test, one per code |
| R6 | Icons look as a browser draws them: on Catppuccin's (5,248 icons) and Material's (1,252) sets, every file reads without error and its picture differs little from ImageMagick's | a probe, not run in CI (it needs both sets and `magick`): it lists the 20 most different icons, which a person looks at |

## Out of scope

- The whole of SVG: text, masks, clip paths, filters, patterns, markers, `use` of a group, CSS stylesheets,
  animation, and drawing gradients as gradients.
- Drawing an SVG every frame at any size: a picture is made once per size and kept by whoever
  shows it, as a glyph is.

## Design

A new package, `svg`, beside `draw2d`, that knows SVG and nothing of windows:

- `read.go` reads the XML (`encoding/xml`) into a list of shapes: each a path in the document's
  units, the transform that places it, its fill and its stroke (a colour with its opacity, a
  width, caps, joins, a miter limit) and its fill rule. A `g` passes what it sets to its children.
- `pathdata.go` reads a path's `d` into moves, lines and cubic curves: quadratic curves and arcs
  are turned into cubic ones, so the rest knows only one kind of curve.
- `stroke.go` turns a stroke into a shape to fill: a quadrilateral for each straight piece, with
  its caps and joins as small polygons, all filled together by the non-zero rule.
- `rasterise.go` fills shapes into a mask: curves cut into straight pieces, then, for each of 16
  rows inside a pixel, where the edges cross it, filled between them by the rule; the rows'
  coverage, added up, is the pixel's opacity. Each shape is then laid over the picture in its
  colour, in the document's order.
- `errors.go` holds the codes. `(*Drawing).Picture(width, height)` is the one call that draws.

**Alternatives not taken.** `golang.org/x/image/vector`, already a dependency of the module, fills
paths well but only by the non-zero rule, and icon sets use even-odd too; a small scanline
rasteriser of our own is about 150 lines and teaches how a path becomes pixels, which is what this
repository is for. `oksvg` and `rasterx` would read and draw it all, but they are two new
dependencies for something we can build and explain. The existing `Path.Stroke`, which measures
each pixel's distance to the line, draws one colour only and would be slow for filled shapes.

## Tasks

- [x] T1 — read a path's `d`: every command, relative and absolute, arcs to curves · `svg/pathdata.go`, `svg/errors.go` · R1, R5 · tests: `every path command reads into moves lines and curves`, `an arc becomes curves through its ends`, `a path that cannot be read has its code`
- [x] T2 — read a document: elements, transforms, colours, style, inheritance · `svg/read.go` · R1, R4, R5 · tests: `each shape reads into a path`, `a group passes its colours down`, `a document with what is not drawn still gives its shapes`
- [x] T3 — fill into a picture by either rule, smoothed, colours laid in order · `svg/rasterise.go`, `svg/picture.go` · R2 · tests: `a square half over a pixel covers half of it`, `a ring is filled as each rule says`
- [x] T4 — strokes: width, caps, joins · `svg/stroke.go` · R3 · tests: `a line's caps reach as far as their kind says`, `a corner is joined as its kind says`
- [x] T5 — gradients as the average of their stops; the probe on both sets against ImageMagick · `svg/read.go`, `svg/probe_test.go` (build tag `probe`) · R4, R6
- [x] T6 — docs: a reference page for `svg`, and a guide "Drawing an SVG icon"; the README's AI disclosure · `docs/5-reference/`, `docs/4-guides/`

## Result

Built on 24 November 2026, pending the author's review.

- `svg.Read` and `Drawing.Picture`: about 1,600 lines of code and 600 of tests, no new dependency.
- Two things were added to the plan once the probe showed them in real icons: a `use` of a single
  shape (Material draws its folders' letters with it) and dashed strokes (`stroke-dasharray`,
  Catppuccin's Go template icon). Both have their tests.
- The probe (R6), run against librsvg 2.60 through ImageMagick: all 6,500 icons read; 1,300 of
  them (every fifth) compared at 32 pixels differ by 0.17 % on average, the most different by 3 %.
  A person looked at the 20 most different side by side: they cannot be told apart, but for masks
  and gradients drawn plain. The first run compared against a picture drawn at 8 times the size
  and shrunk, and showed half-pixel shifts that were the shrinking's, not ours.
- Seen on screen in Coddy (its proposal 10): Catppuccin's and Material's icons in the sidebar and
  the settings, in light and dark.

**Review.** A reviewer with a clean context found files that could stop or hang the drawing,
which matters because icon themes come from anyone. All are fixed, each with its test:

- Arcs of radii like `1e200`, and `NaN` or `Inf` written as numbers, crashed `Picture`: such arcs
  are now lines, such numbers are not read, and a span that is not a number is left out.
- Dashes a millionth of a pixel long, or a dashed line far longer than the picture, never
  finished: dashes shorter than half a pixel are drawn whole, and a run is cut into 4,096 at most.
- A move alone, or a closed outline's start, got a dot from round and square caps; a closed run
  of two points got caps instead of joins.
- `visibility="visible"` inside a hidden group, `inherit` for caps and joins, transform numbers
  that run together (`translate(4-0)`), and colours without their `#` were read wrong.
- Strokes were slow: edges are now sorted from the top so each row looks only at those it may
  cross, and a round join is drawn as a bevel where the two differ by less than a pixel's
  hundredth. A stroked circle at 32 pixels went from 3.7 ms to 1.1 ms with round joins.
- The guide now says what is drawn more simply than a browser: other colour formats, opacity per
  shape, and dashes over a closed outline's start. The probe gives the same figures as before.
