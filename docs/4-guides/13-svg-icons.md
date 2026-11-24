# 🖼️ Drawing an SVG icon

This guide is for a Go programmer who wants to show icons drawn by someone else: a file icon, a
logo, a set made for an editor. At the end you will know how to read an SVG file, draw it at the
size you need, and what happens between the two.

## The shortest program

`svg.Read` reads the document once; `Picture` draws it at a size in pixels, as a `draw2d.Image`
that a `draw2d.List` (or Go UI's `Image`) shows like any other picture:

```go
source, err := os.ReadFile("icons/lighthouse.svg")
if err != nil {
	return err
}
drawing, err := svg.Read(source)
if err != nil {
	return err
}
picture := drawing.Picture(32, 32) // keep it: draw it again only when the size changes
list.Image(picture, draw2d.Rectangle{X: 10, Y: 10, Width: 16, Height: 16}, 0, draw2d.Color{1, 1, 1, 1})
```

Here a 16-pixel icon is drawn into a 32-pixel picture, so it stays sharp on a screen of twice the
density. `ExampleDrawing_Picture`, in `svg/example_test.go`, runs this and checks what it paints.

Drawing is the slow part, about a millisecond for a stroked icon at 32 pixels: do it once per
size, never every frame, as the overlay does with the letters of a font.

## What it reads, and what it leaves out

Icon sets use a small part of SVG, and that part is what `svg` reads: paths with every command,
rectangles, circles, ellipses, lines and polygons; groups that pass their colours and their
transforms down; fills by either rule; strokes with their caps, joins and dashes; and `use` of a
shape defined elsewhere.

What it does not draw it leaves out quietly: text, masks, clip paths, filters. A few things it
draws more simply than a browser:

- A gradient is drawn in the plain colour of its stops' average, which at 16 pixels is hard to
  tell from the real thing.
- Colours are read as `#rgb`, `#rrggbb`, `#rrggbbaa`, `black`, `white` and `none`; `rgb(…)` and
  CSS's other names are drawn black. The icon sets we checked use none of them.
- Opacity fades each shape on its own, not its group as one picture: overlapping shapes of a
  half-faded group darken where they meet, and a shape's fill shows through the inner half of its
  stroke.
- A dash that runs over a closed outline's start is cut there, into two dashes with caps.

Numbers no icon needs (a radius of `1e200`, a width of `Inf`, dashes of a millionth of a pixel)
are drawn as near as they can be, never stopping or slowing the program: icon themes come from
anyone.
Two errors have codes: `svg.not-an-svg` for a document that is not SVG at all, and `svg.bad-path`
for a path whose data breaks SVG's grammar. With the second, `Read` still gives the drawing, with
everything it could read, as a browser would.

We checked it against librsvg, the library GNOME draws SVG with, on Catppuccin's and Material's
icon sets for Zed, 6,500 icons: every one reads, and at 32 pixels they differ from librsvg's by
0.2 % on average. `svg/probe_test.go` runs that comparison again (it needs both sets and
ImageMagick, so CI does not run it).

## How a path becomes pixels

The interesting part is the drawing, and it fits in a few ideas.

**Everything becomes one kind of curve.** SVG has straight lines, quadratic and cubic curves, and
arcs of ellipses. The reader turns them all into moves, lines and cubic curves (`pathdata.go`,
`arc.go`); even a circle is four cubic curves. What draws them then only knows those three.

**Curves become straight pieces.** A cubic curve is cut into pieces about a pixel long
(`rasterise.go`). At that length nobody can tell the pieces from the curve.

**A shape is filled row by row.** Each row of pixels is sampled 16 times, at 16 heights. At each
height, every edge of the shape either crosses it or not; where it crosses is a single number, the
x of the crossing. Sorted along the row, the crossings cut it into spans, and the fill rule says
which spans are inside: *non-zero* counts edges going down as +1 and going up as −1, and is inside
where the count is not zero; *even-odd* is inside after an odd number of crossings. A span adds to
each pixel the part of it that it covers, a sixteenth at most. Add the 16 heights, and a pixel half
covered by the shape ends up half opaque: the edges come out smooth.

**A stroke is a shape too.** A line two pixels wide is a long thin rectangle around it
(`stroke.go`): one rectangle for each straight piece, a circle at each end for round caps, a small
triangle or a circle at each corner for the joins. Filled together by the non-zero rule, they are
the stroke. They must all go round the same way, or where two overlap they would cancel out and
leave a hole.

**Colours are laid one over another.** Each shape is painted in its colour, as opaque as its
coverage says, over the ones before it, in the document's order.

## Alternatives we did not take

`golang.org/x/image/vector` fills paths beautifully, but only by the non-zero rule, and icon sets
use even-odd as well. `oksvg` and `rasterx` read and draw nearly all of SVG, but they are two
libraries for something that fits in a small package we can read and explain, and that is what
this repository is for.
