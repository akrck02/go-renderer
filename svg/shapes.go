package svg

import (
	"fmt"
	"math"
	"strings"
)

// shapeData is a shape's outline written as a path's data, so every shape is read by the one
// path reader. A shape that draws nothing (no size, an element that is not a shape) is "".
func shapeData(element string, all attributes) string {
	switch element {
	case "path":
		return all["d"]
	case "rect":
		return rectangleData(all)
	case "circle":
		return ellipseData(all.number("cx"), all.number("cy"), all.number("r"), all.number("r"))
	case "ellipse":
		return ellipseData(all.number("cx"), all.number("cy"), all.number("rx"), all.number("ry"))
	case "line":
		return fmt.Sprintf("M%g %g L%g %g", all.number("x1"), all.number("y1"), all.number("x2"), all.number("y2"))
	case "polyline", "polygon":
		points := strings.TrimSpace(all["points"])
		if points == "" {
			return ""
		}
		if element == "polygon" {
			return "M" + points + "Z"
		}
		return "M" + points
	}
	return ""
}

// rectangleData is a rectangle's outline, its corners rounded by rx and ry: one of them given
// stands for both, and neither is more than half the side it rounds.
func rectangleData(all attributes) string {
	x, y, width, height := all.number("x"), all.number("y"), all.number("width"), all.number("height")
	if width <= 0 || height <= 0 {
		return ""
	}
	radiusX, radiusY := all.number("rx"), all.number("ry")
	if _, given := all["rx"]; !given {
		radiusX = radiusY
	}
	if _, given := all["ry"]; !given {
		radiusY = radiusX
	}
	radiusX, radiusY = math.Min(math.Max(radiusX, 0), width/2), math.Min(math.Max(radiusY, 0), height/2)
	if radiusX == 0 || radiusY == 0 {
		return fmt.Sprintf("M%g %g H%g V%g H%g Z", x, y, x+width, y+height, x)
	}
	corner := fmt.Sprintf("A%g %g 0 0 1 ", radiusX, radiusY)
	return fmt.Sprintf("M%g %g H%g ", x+radiusX, y, x+width-radiusX) +
		corner + fmt.Sprintf("%g %g V%g ", x+width, y+radiusY, y+height-radiusY) +
		corner + fmt.Sprintf("%g %g H%g ", x+width-radiusX, y+height, x+radiusX) +
		corner + fmt.Sprintf("%g %g V%g ", x, y+height-radiusY, y+radiusY) +
		corner + fmt.Sprintf("%g %gZ", x+radiusX, y)
}

// ellipseData is an ellipse's outline: two half turns from its left end to its right and back.
func ellipseData(centreX, centreY, radiusX, radiusY float64) string {
	if radiusX <= 0 || radiusY <= 0 {
		return ""
	}
	half := fmt.Sprintf("A%g %g 0 1 0 ", radiusX, radiusY)
	return fmt.Sprintf("M%g %g ", centreX-radiusX, centreY) + half + fmt.Sprintf("%g %g ", centreX+radiusX, centreY) +
		half + fmt.Sprintf("%g %gZ", centreX-radiusX, centreY)
}
