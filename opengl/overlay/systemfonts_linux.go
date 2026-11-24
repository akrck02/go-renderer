//go:build linux && !android

package overlay

import (
	"os/exec"
	"strings"
)

// desktopFonts are the fonts the desktop is set to: the families GNOME's settings name (which
// most desktops keep), or else fontconfig's sans-serif and monospace, found with fc-match. Without
// fc-match, or a file it names, it returns SystemFontNotFound and the fonts are looked for by name.
func desktopFonts() (Fonts, error) {
	family := desktopFamily("font-name", "sans-serif")
	regularPath, regular := matchedFont(family)
	if regular == nil {
		return Fonts{}, SystemFontNotFound
	}
	bold := regular
	// a variable font holds every weight in one file: fc-match then names the same one
	if boldPath, found := matchedFont(family + ":bold"); found != nil && boldPath != regularPath {
		bold = found
	}
	_, mono := matchedFont(desktopFamily("monospace-font-name", "monospace"))
	return Fonts{Regular: regular, Medium: regular, Bold: bold, Mono: mono}, nil
}

// desktopFamily is the family a GNOME setting names, or fallback where there is no such setting.
func desktopFamily(setting, fallback string) string {
	output, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", setting).Output()
	if err != nil {
		return fallback // no gsettings, or not a GNOME setting here: fontconfig's own choice
	}
	if family := familyOf(string(output)); family != "" {
		return family
	}
	return fallback
}

// familyOf reads the family out of a GNOME font setting as gsettings prints it, the size after
// it: "'Adwaita Sans 11'" is "Adwaita Sans".
func familyOf(setting string) string {
	name := strings.Trim(strings.TrimSpace(setting), "'")
	words := strings.Fields(name)
	if len(words) > 1 && strings.Trim(words[len(words)-1], "0123456789.") == "" {
		words = words[:len(words)-1]
	}
	return strings.Join(words, " ")
}

// matchedFont is the file fontconfig draws a pattern ("Adwaita Sans", "monospace:bold") with,
// and its bytes; nil when fc-match is missing or its file cannot be read.
func matchedFont(pattern string) (string, []byte) {
	output, err := exec.Command("fc-match", "--format=%{file}", pattern).Output()
	if err != nil {
		return "", nil // no fontconfig: the fonts are looked for by name instead
	}
	path := strings.TrimSpace(string(output))
	// kept for as long as the program draws text: never released
	data, _, err := mapFile(path)
	if err != nil {
		return path, nil // a file fontconfig names but cannot be read is as no file
	}
	return path, data
}
