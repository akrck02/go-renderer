package svg

import (
	"math"
	"regexp"
)

// transform places a shape: a point (x, y) goes to (a·x + c·y + e, b·x + d·y + f), with
// a b c d e f in this order, as SVG's matrix() writes them.
type transform [6]float64

// identity leaves every point where it is.
var identity = transform{1, 0, 0, 1, 0, 0}

// then is this transform applied after inner: inner first, as a list in SVG reads ("A B" places
// a point by B, then by A).
func (outer transform) then(inner transform) transform {
	return transform{
		outer[0]*inner[0] + outer[2]*inner[1],
		outer[1]*inner[0] + outer[3]*inner[1],
		outer[0]*inner[2] + outer[2]*inner[3],
		outer[1]*inner[2] + outer[3]*inner[3],
		outer[0]*inner[4] + outer[2]*inner[5] + outer[4],
		outer[1]*inner[4] + outer[3]*inner[5] + outer[5],
	}
}

// place is where the transform puts a point.
func (matrix transform) place(at point) point {
	return point{matrix[0]*at.x + matrix[2]*at.y + matrix[4], matrix[1]*at.x + matrix[3]*at.y + matrix[5]}
}

// scale is how much the transform grows a length, on average over its two axes: what a stroke's
// width is multiplied by.
func (matrix transform) scale() float64 {
	return math.Sqrt(math.Abs(matrix[0]*matrix[3] - matrix[1]*matrix[2]))
}

// transformPattern is one function of a transform list: its name and its numbers.
var transformPattern = regexp.MustCompile(`([a-zA-Z]+)\s*\(([^)]*)\)`)

// readTransform reads a transform attribute: a list of matrix, translate, scale, rotate, skewX
// and skewY. What it cannot read leaves the shape where it is.
func readTransform(text string) transform {
	result := identity
	for _, function := range transformPattern.FindAllStringSubmatch(text, -1) {
		result = result.then(transformFunction(function[1], numbers(function[2])))
	}
	return result
}

// transformFunction is one function of a transform list, from its name and numbers.
func transformFunction(name string, values []float64) transform {
	value := func(index int, otherwise float64) float64 {
		if index < len(values) {
			return values[index]
		}
		return otherwise
	}
	switch name {
	case "matrix":
		if len(values) == 6 {
			return transform(values)
		}
	case "translate":
		return transform{1, 0, 0, 1, value(0, 0), value(1, 0)}
	case "scale":
		return transform{value(0, 1), 0, 0, value(1, value(0, 1)), 0, 0}
	case "rotate":
		angle := value(0, 0) * math.Pi / 180
		cosine, sine := math.Cos(angle), math.Sin(angle)
		around := point{value(1, 0), value(2, 0)}
		turn := transform{cosine, sine, -sine, cosine, 0, 0}
		return transform{1, 0, 0, 1, around.x, around.y}.then(turn).then(transform{1, 0, 0, 1, -around.x, -around.y})
	case "skewX":
		return transform{1, 0, math.Tan(value(0, 0) * math.Pi / 180), 1, 0, 0}
	case "skewY":
		return transform{1, math.Tan(value(0, 0) * math.Pi / 180), 0, 1, 0, 0}
	}
	return identity
}

// numbers reads a list of numbers as a path's are read: apart by spaces or commas, or running
// together ("4-2" is 4 and -2). The list ends where something that is not a number starts.
func numbers(text string) []float64 {
	var values []float64
	reader := pathReader{data: text}
	for reader.skipSeparators(); reader.at < len(reader.data); reader.skipSeparators() {
		value, err := reader.number()
		if err != nil || !isFinite(value) {
			return values
		}
		values = append(values, value)
	}
	return values
}

// placed is an outline moved by a transform.
func (matrix transform) placed(steps []step) []step {
	if matrix == identity {
		return steps
	}
	for index := range steps {
		for which := range steps[index].points {
			steps[index].points[which] = matrix.place(steps[index].points[which])
		}
	}
	return steps
}
