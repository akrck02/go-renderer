package svg

import (
	"strconv"
	"strings"
)

// point is a place in a document's units.
type point struct{ x, y float64 }

// stepKind is what a step of an outline does.
type stepKind int

const (
	moveStep  stepKind = iota // starts a new run of the outline, without drawing
	lineStep                  // a straight piece
	curveStep                 // a cubic Bézier curve: two control points and its end
	closeStep                 // a straight piece back to where the run started
)

// step is one piece of an outline. A move, line or close keeps its end in points[0]; a curve its
// two control points, then its end.
type step struct {
	kind   stepKind
	points [3]point
}

// end is where a step leaves the pen.
func (piece step) end() point {
	if piece.kind == curveStep {
		return piece.points[2]
	}
	return piece.points[0]
}

// pathReader reads a path's data: the commands of SVG's path grammar, turned into moves, lines
// and cubic curves only, so what draws them knows one kind of curve.
type pathReader struct {
	data           string
	at             int
	steps          []step
	pen, runStart  point
	lastCommand    byte
	cubicControl   point // the second control of the last C or S, mirrored by S
	quadraticPoint point // the control of the last Q or T, mirrored by T
}

// readPathData reads the d of a path into its steps. An empty one is no steps and no error; one
// with an error gives the steps read before it, with BadPath.
func readPathData(data string) ([]step, error) {
	reader := &pathReader{data: data}
	command := byte(0)
	for reader.skipSeparators(); reader.at < len(data); reader.skipSeparators() {
		if letter := data[reader.at]; isLetter(letter) {
			command = letter
			reader.at++
		} else if command == 0 {
			return nil, BadPath
		}
		if reader.steps == nil && command != 'M' && command != 'm' {
			return nil, BadPath // a path starts with a move
		}
		if err := reader.command(command); err != nil {
			return reader.steps, err // what was read before the error is drawn, as a browser does
		}
		reader.lastCommand = command
		switch command {
		case 'M', 'm':
			command-- // a move's further pairs are lines: M is L + 1, m is l + 1
		case 'Z', 'z':
			command = 0 // a close takes no numbers: one after it is an error, not a loop
		}
	}
	return reader.steps, nil
}

// command reads the numbers of one command and adds its steps.
func (reader *pathReader) command(command byte) error {
	relative := command >= 'a'
	switch command | 0x20 { // the lower case of a letter
	case 'm', 'l':
		return reader.moveOrLine(command|0x20 == 'm', relative)
	case 'h', 'v':
		return reader.straight(command|0x20 == 'h', relative)
	case 'c', 's':
		return reader.cubic(command|0x20 == 's', relative)
	case 'q', 't':
		return reader.quadratic(command|0x20 == 't', relative)
	case 'a':
		return reader.arc(relative)
	case 'z':
		reader.add(step{kind: closeStep, points: [3]point{reader.runStart}})
		return nil
	}
	return BadPath
}

// moveOrLine reads M, m, L or l.
func (reader *pathReader) moveOrLine(isMove, relative bool) error {
	to, err := reader.place(relative)
	if err != nil {
		return err
	}
	if isMove {
		reader.runStart = to
		reader.add(step{kind: moveStep, points: [3]point{to}})
		return nil
	}
	reader.add(step{kind: lineStep, points: [3]point{to}})
	return nil
}

// straight reads H, h, V or v: a line along one axis.
func (reader *pathReader) straight(across, relative bool) error {
	value, err := reader.number()
	if err != nil {
		return err
	}
	to := reader.pen
	switch {
	case across && relative:
		to.x += value
	case across:
		to.x = value
	case relative:
		to.y += value
	default:
		to.y = value
	}
	reader.add(step{kind: lineStep, points: [3]point{to}})
	return nil
}

// cubic reads C, c, S or s. A smooth one (S) takes its first control as the mirror of the last
// curve's second, or the pen when the command before was not a cubic curve.
func (reader *pathReader) cubic(smooth, relative bool) error {
	first := reader.pen
	if smooth && strings.IndexByte("CcSs", reader.lastCommand) >= 0 {
		first = mirror(reader.cubicControl, reader.pen)
	}
	if !smooth {
		var err error
		if first, err = reader.place(relative); err != nil {
			return err
		}
	}
	second, err := reader.place(relative)
	if err != nil {
		return err
	}
	to, err := reader.place(relative)
	if err != nil {
		return err
	}
	reader.cubicControl = second
	reader.add(step{kind: curveStep, points: [3]point{first, second, to}})
	return nil
}

// quadratic reads Q, q, T or t, turned into the cubic curve that draws the same line: each cubic
// control two thirds of the way from an end to the quadratic one.
func (reader *pathReader) quadratic(smooth, relative bool) error {
	control := reader.pen
	if smooth && strings.IndexByte("QqTt", reader.lastCommand) >= 0 {
		control = mirror(reader.quadraticPoint, reader.pen)
	}
	if !smooth {
		var err error
		if control, err = reader.place(relative); err != nil {
			return err
		}
	}
	to, err := reader.place(relative)
	if err != nil {
		return err
	}
	from := reader.pen
	reader.quadraticPoint = control
	first := point{from.x + (control.x-from.x)*2/3, from.y + (control.y-from.y)*2/3}
	second := point{to.x + (control.x-to.x)*2/3, to.y + (control.y-to.y)*2/3}
	reader.add(step{kind: curveStep, points: [3]point{first, second, to}})
	return nil
}

// arc reads A or a: two radii, the ellipse's turn in degrees, two flags and the end.
func (reader *pathReader) arc(relative bool) error {
	var numbers [3]float64
	for index := range numbers {
		value, err := reader.number()
		if err != nil {
			return err
		}
		numbers[index] = value
	}
	large, err := reader.flag()
	if err != nil {
		return err
	}
	sweep, err := reader.flag()
	if err != nil {
		return err
	}
	to, err := reader.place(relative)
	if err != nil {
		return err
	}
	shape := ellipse{radiusX: numbers[0], radiusY: numbers[1], turnDegrees: numbers[2]}
	for _, piece := range arcSteps(reader.pen, to, shape, large, sweep) {
		reader.add(piece)
	}
	return nil
}

// add keeps a step and moves the pen to its end.
func (reader *pathReader) add(piece step) {
	reader.steps = append(reader.steps, piece)
	reader.pen = piece.end()
}

// place reads a pair of numbers, a point; relative to the pen for a lower case command.
func (reader *pathReader) place(relative bool) (point, error) {
	x, err := reader.number()
	if err != nil {
		return point{}, err
	}
	y, err := reader.number()
	if err != nil {
		return point{}, err
	}
	if relative {
		return point{reader.pen.x + x, reader.pen.y + y}, nil
	}
	return point{x, y}, nil
}

// number reads a number as SVG writes it: a sign, digits, a point, digits, an exponent. Numbers
// may run together without a separator ("1.5.5" is 1.5 and .5, "1-2" is 1 and -2).
func (reader *pathReader) number() (float64, error) {
	reader.skipSeparators()
	start, data := reader.at, reader.data
	if reader.at < len(data) && (data[reader.at] == '-' || data[reader.at] == '+') {
		reader.at++
	}
	reader.skipDigits()
	if reader.at < len(data) && data[reader.at] == '.' {
		reader.at++
		reader.skipDigits()
	}
	if reader.at < len(data) && (data[reader.at] == 'e' || data[reader.at] == 'E') {
		reader.at++
		if reader.at < len(data) && (data[reader.at] == '-' || data[reader.at] == '+') {
			reader.at++
		}
		reader.skipDigits()
	}
	value, err := strconv.ParseFloat(data[start:reader.at], 64)
	if err != nil {
		return 0, BadPath
	}
	return value, nil
}

// flag reads an arc's flag: one character, 0 or 1, that needs no separator after it.
func (reader *pathReader) flag() (bool, error) {
	reader.skipSeparators()
	if reader.at >= len(reader.data) || (reader.data[reader.at] != '0' && reader.data[reader.at] != '1') {
		return false, BadPath
	}
	reader.at++
	return reader.data[reader.at-1] == '1', nil
}

func (reader *pathReader) skipDigits() {
	for reader.at < len(reader.data) && reader.data[reader.at] >= '0' && reader.data[reader.at] <= '9' {
		reader.at++
	}
}

// skipSeparators steps over spaces and commas.
func (reader *pathReader) skipSeparators() {
	for reader.at < len(reader.data) && strings.IndexByte(" \t\n\r\f,", reader.data[reader.at]) >= 0 {
		reader.at++
	}
}

func isLetter(character byte) bool {
	lower := character | 0x20
	return lower >= 'a' && lower <= 'z' && lower != 'e' // e is a number's exponent
}

// mirror is a control point reflected through the pen, for a smooth curve.
func mirror(control, pen point) point {
	return point{2*pen.x - control.x, 2*pen.y - control.y}
}

// outlineBounds is the smallest rectangle that holds every point of an outline, control points
// included: a curve never leaves it.
func outlineBounds(steps []step) rectangle {
	if len(steps) == 0 {
		return rectangle{}
	}
	low, high := steps[0].points[0], steps[0].points[0]
	for _, piece := range steps {
		count := 1
		if piece.kind == curveStep {
			count = 3
		}
		for _, at := range piece.points[:count] {
			low = point{min(low.x, at.x), min(low.y, at.y)}
			high = point{max(high.x, at.x), max(high.y, at.y)}
		}
	}
	return rectangle{low.x, low.y, high.x - low.x, high.y - low.y}
}
