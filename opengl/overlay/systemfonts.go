package overlay

import (
	"os"
	"path/filepath"
	"runtime"
)

// systemFace is one interface font a system ships: its regular and bold files, as names to find in
// the font folders.
type systemFace struct {
	regular, bold string
}

// systemFaces are the interface fonts of each system, the most usual first: what a web page's
// system-ui is drawn with there.
var systemFaces = map[string][]systemFace{
	"android": {{"Roboto-Regular.ttf", "Roboto-Bold.ttf"}},
	"darwin":  {{"SFNS.ttf", "SFNS.ttf"}, {"HelveticaNeue.ttc", "HelveticaNeue.ttc"}},
	"ios":     {{"SFUI.ttf", "SFUI.ttf"}},
	"windows": {{"segoeui.ttf", "segoeuib.ttf"}},
	"linux": {
		{"AdwaitaSans-Regular.ttf", "AdwaitaSans-Regular.ttf"},
		{"DejaVuSans.ttf", "DejaVuSans-Bold.ttf"},
		{"NotoSans-Regular.ttf", "NotoSans-Bold.ttf"},
		{"Cantarell-Regular.otf", "Cantarell-Bold.otf"},
	},
}

// systemMonoFaces are the monospaced fonts each system reads code in (its terminal's, its code
// editor's), the most usual first.
var systemMonoFaces = map[string][]string{
	"android": {"DroidSansMono.ttf"},
	"darwin":  {"SFNSMono.ttf"},
	"windows": {"consola.ttf"},
	"linux":   {"AdwaitaMono-Regular.ttf", "DejaVuSansMono.ttf", "NotoSansMono-Regular.ttf"},
}

// SystemFonts are the fonts the system draws its own interface with, so a program reads like the
// rest of the device and like a web page using system-ui on it, and the one it reads code in
// (Mono). The medium weight is the regular one, as a browser draws it when the font has no medium.
// A desktop that names its fonts (GNOME's settings, or fontconfig's sans-serif and monospace) is
// asked first; otherwise the fonts are looked for in DefaultFontFolders. When no known interface
// font is there it returns SystemFontNotFound, with the Go fonts for the interface (and the
// system's monospaced font, if one was found).
func SystemFonts() (Fonts, error) {
	if fonts, err := desktopFonts(); err == nil {
		return fonts, nil
	}
	folders := DefaultFontFolders()
	fonts, err := interfaceFontsIn(folders, systemFaces[runtime.GOOS])
	fonts.Mono = monoFontIn(folders, systemMonoFaces[runtime.GOOS])
	return fonts, err
}

// monoFontIn reads the first of these monospaced fonts found in folders; nil when there is none.
func monoFontIn(folders []string, names []string) []byte {
	for _, name := range names {
		if found := findFontFile(folders, name); found != nil {
			return found
		}
	}
	return nil
}

func interfaceFontsIn(folders []string, faces []systemFace) (Fonts, error) {
	for _, face := range faces {
		regular := findFontFile(folders, face.regular)
		bold := regular
		// A variable font holds every weight in one file, often a large one: read it once.
		if face.bold != face.regular {
			bold = findFontFile(folders, face.bold)
		}
		if regular == nil || bold == nil {
			continue
		}
		return Fonts{Regular: regular, Medium: regular, Bold: bold}, nil
	}
	return Fonts{}, SystemFontNotFound
}

// findFontFile reads the first file called name under any of folders; nil when there is none.
func findFontFile(folders []string, name string) []byte {
	for _, folder := range folders {
		var found []byte
		// A folder that cannot be walked is skipped: the next one may hold the font.
		_ = filepath.WalkDir(folder, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || entry.Name() != name {
				return nil
			}
			// kept for as long as the program draws text: never released
			found, _, err = mapFile(path)
			if err != nil {
				return nil // unreadable: keep looking
			}
			return filepath.SkipAll
		})
		if found != nil {
			return found
		}
	}
	return nil
}
