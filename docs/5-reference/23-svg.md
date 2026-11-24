# svg

```go
import "github.com/akrck02/go-renderer/svg"
```

Package svg reads SVG pictures, the format icon sets are drawn in, and draws
them into pictures of any size. It reads the part of SVG that icons use: paths
and basic shapes, groups, transforms, uses of a shape, fills, and strokes with
their caps, joins and dashes, in plain colours. What it does not draw (text,
masks, clip paths, filters) is left out without failing, and a gradient is drawn
in one colour.

### Constants and variables

```go
var (
	// NotAnSVG is a document that is not SVG: not XML, or not an <svg> at its root.
	NotAnSVG = failure.Define("svg.not-an-svg", "the document is not an SVG picture")
	// BadPath is a path's data (its d) that does not follow SVG's path grammar.
	BadPath = failure.Define("svg.bad-path", "the path's data cannot be read")
)
```

Codes of the errors svg returns.

### Drawing

Drawing is an SVG document read: its shapes, in the order they are painted, in the units of its view box.

```go
type Drawing struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func Read(source []byte) (*Drawing, error)` | Read reads an SVG document. |
| `func (drawing *Drawing) Picture(width, height int) *draw2d.Image` | Picture draws the drawing into a picture of a size in pixels, in its own colours. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
