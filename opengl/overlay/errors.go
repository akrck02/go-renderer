package overlay

import "github.com/akrck02/go-renderer/failure"

// FontNotRead is returned by NewPainter and NewMeasurer when one of the fonts (the default ones or
// those given in Fonts) is not a TrueType or OpenType font.
var FontNotRead = failure.Define("overlay.font-not-read", "a font could not be read")

// colourFontInvalid is a system's colour emoji font that cannot be read; the painter then draws
// emoji in black and white, so it never reaches a program.
var colourFontInvalid = failure.Define("overlay.colour-font-invalid", "the colour emoji font cannot be read")

// SystemFontNotFound is returned by SystemFonts when none of the system's interface fonts it knows
// is on this device; the Go fonts are still there to draw with.
var SystemFontNotFound = failure.Define("overlay.system-font-not-found", "the system's interface font is not on this device")
