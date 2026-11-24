package svg

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors svg returns.
var (
	// NotAnSVG is a document that is not SVG: not XML, or not an <svg> at its root.
	NotAnSVG = failure.Define("svg.not-an-svg", "the document is not an SVG picture")
	// BadPath is a path's data (its d) that does not follow SVG's path grammar.
	BadPath = failure.Define("svg.bad-path", "the path's data cannot be read")
)
