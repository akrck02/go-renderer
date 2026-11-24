// Package svg reads SVG pictures, the format icon sets are drawn in, and draws them into
// pictures of any size. It reads the part of SVG that icons use: paths and basic shapes, groups,
// transforms, uses of a shape, fills, and strokes with their caps, joins and dashes, in plain
// colours. What it does not draw (text, masks, clip paths, filters) is left out without
// failing, and a gradient is drawn in one colour.
package svg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"

	"github.com/akrck02/go-renderer/failure"
)

// Drawing is an SVG document read: its shapes, in the order they are painted, in the units of
// its view box.
type Drawing struct {
	viewBox rectangle
	shapes  []shape
}

// rectangle is an area in a document's units.
type rectangle struct{ x, y, width, height float64 }

// shape is an outline and how it is painted.
type shape struct {
	steps []step
	look
}

// attributes are an element's attributes by name, its style's properties among them.
type attributes map[string]string

// number is an attribute read as a number, 0 when it is missing or not one.
func (all attributes) number(name string) float64 { return length(all[name], 0) }

// hiddenElements hold what is not drawn where it is: definitions, masks and clips, text, pictures.
// Gradients and shapes with an id inside them are still read, for what refers to them; a use is
// drawn as the shape it refers to.
var hiddenElements = map[string]bool{
	"defs": true, "mask": true, "clipPath": true, "symbol": true, "pattern": true, "marker": true,
	"text": true, "use": true, "image": true, "foreignObject": true, "style": true, "title": true,
}

// frame is what an element passes to the ones inside it.
type frame struct {
	look      look
	placement transform
	hidden    bool
}

// Read reads an SVG document. A document that is not XML, or not an <svg>, is NotAnSVG. A path
// that cannot be read is BadPath, and the drawing is still given, with what could be read of it,
// as a browser draws it.
func Read(source []byte) (*Drawing, error) {
	decoder := xml.NewDecoder(bytes.NewReader(source))
	reader := &documentReader{drawing: &Drawing{}, gradients: map[string]*gradient{}, definitions: map[string]definition{}}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, failure.Wrap(NotAnSVG, "", err)
		}
		if err := reader.take(token); err != nil {
			return nil, err
		}
	}
	if !reader.rootSeen {
		return nil, NotAnSVG
	}
	reader.finish()
	return reader.drawing, reader.pathError
}
