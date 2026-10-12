// Package lucide embeds the Lucide icon font (https://lucide.dev, ISC licence, see LICENSE) and
// the table from icon names ("anchor", "sailboat", "clock"…) to its characters, so the overlay
// can draw icons by name. The font and the table are copied from the lucide-static npm package;
// generate.go explains how to update them.
package lucide

import _ "embed"

//go:embed lucide.ttf
var font []byte

// Font returns the TrueType data of the icon font.
func Font() []byte { return font }

// Codepoint returns the character of an icon in the font, and false for an unknown name.
func Codepoint(name string) (rune, bool) {
	character, found := codepoints[name]
	return character, found
}
