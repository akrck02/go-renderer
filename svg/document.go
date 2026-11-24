package svg

import (
	"encoding/xml"
	"strings"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/failure"
)

// documentReader follows a document's elements, each placed and painted by the groups around it.
type documentReader struct {
	drawing   *Drawing
	frames    []frame
	rootSeen  bool
	gradients map[string]*gradient
	gradient  *gradient // the gradient whose stops are being read
	pathError error     // the first path that could not be read
	// definitions are the shapes with an id, which a use paints again
	definitions map[string]definition
}

// definition is a shape with an id: its element's name and attributes.
type definition struct {
	name string
	all  attributes
}

// shapeElements are the elements that are a shape, which a use may paint.
var shapeElements = map[string]bool{"path": true, "rect": true, "circle": true, "ellipse": true, "line": true, "polyline": true, "polygon": true}

// gradient is a gradient's stops, drawn as their average, or another gradient's it refers to.
type gradient struct {
	stops     []draw2d.Color
	reference string
}

// take reads one token of the document.
func (reader *documentReader) take(token xml.Token) error {
	switch element := token.(type) {
	case xml.StartElement:
		return reader.start(element)
	case xml.EndElement:
		if element.Name.Local == "linearGradient" || element.Name.Local == "radialGradient" {
			reader.gradient = nil
		}
		if len(reader.frames) > 0 {
			reader.frames = reader.frames[:len(reader.frames)-1]
		}
	}
	return nil
}

// start reads an element's opening: the root's view box, a gradient's stops, a shape.
func (reader *documentReader) start(element xml.StartElement) error {
	name, all := element.Name.Local, attributesOf(element)
	if !reader.rootSeen {
		if name != "svg" {
			return NotAnSVG
		}
		reader.rootSeen = true
		reader.drawing.viewBox = viewBoxOf(all)
	}
	around := frame{look: startingLook, placement: identity}
	if len(reader.frames) > 0 {
		around = reader.frames[len(reader.frames)-1]
	}
	inside := frame{look: around.look.with(all), placement: around.placement.then(readTransform(all["transform"])), hidden: around.hidden}
	inside.hidden = inside.hidden || hiddenElements[name] || all["display"] == "none"
	reader.frames = append(reader.frames, inside)
	reader.readGradient(name, all)
	if shapeElements[name] && all["id"] != "" {
		reader.definitions[all["id"]] = definition{name, all}
	}
	if name == "use" && !around.hidden {
		reader.addUse(all, frame{look: around.look.with(all), placement: around.placement.then(readTransform(all["transform"]))})
	}
	if !inside.hidden && !inside.look.invisible {
		reader.addShape(name, all, inside)
	}
	return nil
}

// addShape keeps a shape, its outline placed where its groups put it.
func (reader *documentReader) addShape(name string, all attributes, inside frame) {
	data := shapeData(name, all)
	if data == "" {
		return
	}
	steps, err := readPathData(data)
	if err != nil && reader.pathError == nil {
		reader.pathError = failure.Wrap(BadPath, shortened(data), nil)
	}
	if len(steps) == 0 {
		return
	}
	shown := inside.look
	scale := inside.placement.scale()
	shown.strokeWidth, shown.dashOffset = shown.strokeWidth*scale, shown.dashOffset*scale
	shown.dashes = scaledLengths(shown.dashes, scale)
	reader.drawing.shapes = append(reader.drawing.shapes, shape{steps: inside.placement.placed(steps), look: shown})
}

// addUse paints again the shape a use refers to, placed by the use (its transform, then its x and
// y), its look the use's with its own over it. A use of what is not a shape, or of what is not
// defined yet, paints nothing.
func (reader *documentReader) addUse(all attributes, placed frame) {
	defined, found := reader.definitions[strings.TrimPrefix(all["href"], "#")]
	if !found {
		return
	}
	shift := transform{1, 0, 0, 1, all.number("x"), all.number("y")}
	placed.placement = placed.placement.then(shift).then(readTransform(defined.all["transform"]))
	placed.look = placed.look.with(defined.all)
	reader.addShape(defined.name, defined.all, placed)
}

// readGradient keeps a gradient's stops, wherever it is: in definitions or not.
func (reader *documentReader) readGradient(name string, all attributes) {
	switch name {
	case "linearGradient", "radialGradient":
		reader.gradient = &gradient{reference: strings.TrimPrefix(all["href"], "#")}
		reader.gradients[all["id"]] = reader.gradient
	case "stop":
		if reader.gradient == nil {
			return
		}
		colour, _, found := readPaint(strings.TrimSpace(all["stop-color"]))
		if !found {
			colour = draw2d.Color{0, 0, 0, 1}
		}
		colour[3] *= float32(length(all["stop-opacity"], 1))
		reader.gradient.stops = append(reader.gradient.stops, colour)
	}
}

// finish paints the shapes with their gradients, found by now, and lays their opacities on.
func (reader *documentReader) finish() {
	for index := range reader.drawing.shapes {
		shown := &reader.drawing.shapes[index].look
		shown.fill = reader.resolved(shown.fill, shown.fillOpacity*shown.opacity)
		shown.stroke = reader.resolved(shown.stroke, shown.strokeOpacity*shown.opacity)
	}
}

// resolved is a paint's plain colour: its own, or its gradient's average, faded by an opacity.
func (reader *documentReader) resolved(fill paint, opacity float64) paint {
	if fill.reference != "" {
		fill = paint{colour: reader.average(fill.reference, 0)}
	}
	fill.colour[3] *= float32(opacity)
	return fill
}

// average is a gradient's colour: the average of its stops, or of the gradient it takes its
// stops from. A gradient not found draws nothing.
func (reader *documentReader) average(id string, depth int) draw2d.Color {
	found := reader.gradients[id]
	if found == nil || depth > 8 { // a chain of references that long is a loop
		return draw2d.Color{}
	}
	if len(found.stops) == 0 {
		return reader.average(found.reference, depth+1)
	}
	var sum draw2d.Color
	for _, stop := range found.stops {
		for channel := range sum {
			sum[channel] += stop[channel] / float32(len(found.stops))
		}
	}
	return sum
}

// attributesOf are an element's attributes by name, the properties of its style among them,
// winning over attributes of the same name as CSS says.
func attributesOf(element xml.StartElement) attributes {
	all := attributes{}
	for _, attribute := range element.Attr {
		all[attribute.Name.Local] = attribute.Value
	}
	for _, declaration := range strings.Split(all["style"], ";") {
		if name, value, found := strings.Cut(declaration, ":"); found {
			all[strings.TrimSpace(name)] = strings.TrimSpace(value)
		}
	}
	return all
}

// viewBoxOf is the area of the document its shapes are drawn in: its viewBox, or else its width
// and height, or else SVG's 300 by 150.
func viewBoxOf(all attributes) rectangle {
	if values := numbers(all["viewBox"]); len(values) == 4 && values[2] > 0 && values[3] > 0 {
		return rectangle{values[0], values[1], values[2], values[3]}
	}
	width, height := all.number("width"), all.number("height")
	if width <= 0 || height <= 0 {
		return rectangle{0, 0, 300, 150}
	}
	return rectangle{0, 0, width, height}
}

// scaledLengths are lengths multiplied by a scale, in a list of their own.
func scaledLengths(lengths []float64, scale float64) []float64 {
	if lengths == nil {
		return nil
	}
	scaled := make([]float64, len(lengths))
	for index, each := range lengths {
		scaled[index] = each * scale
	}
	return scaled
}

// shownPathLength is how much of a path an error shows: enough to find it, not kilobytes of it
const shownPathLength = 40

// shortened is the start of a path, for an error.
func shortened(data string) string {
	if len(data) <= shownPathLength {
		return data
	}
	return data[:shownPathLength] + "…"
}
