//go:build !linux || android

package overlay

// desktopFonts is where a desktop names its fonts; elsewhere they are looked for by name.
func desktopFonts() (Fonts, error) { return Fonts{}, SystemFontNotFound }
