package draw2d

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors draw2d returns.
var (
	// InvalidColour is a colour that is not written as "#rrggbb" or "#rrggbbaa".
	InvalidColour = failure.Define("draw2d.invalid-colour", "a colour must be written as #rrggbb or #rrggbbaa")
	// ImageNotDecoded is a picture that is not a PNG, JPEG or GIF, or is damaged.
	ImageNotDecoded = failure.Define("draw2d.image-not-decoded", "the picture could not be decoded")
)
