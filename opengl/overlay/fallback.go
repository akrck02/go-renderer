package overlay

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// The fonts a program draws with (the Go fonts, or its own) have Latin, Greek and Cyrillic. A
// character they lack (Chinese, Arabic, Devanagari, a symbol) is drawn with one of the system's
// fonts instead, as every system's own text does: a phone carries a Noto font for every script.
// The fonts are found by their file names, which name their script (NotoSansArabic,
// NotoSansCJK); a character no font is named for is looked for in every font, the last resort.
// Each file is read only when a character needs it, and once.

// DefaultFontFolders are where the system's fonts are looked for by default, for characters the
// program's fonts lack: Android's own, or the usual folders of the desktop's system.
func DefaultFontFolders() []string {
	home, _ := os.UserHomeDir() // without a home folder, only the system's folders are searched
	switch runtime.GOOS {
	case "android":
		return []string{"/system/fonts", "/product/fonts"}
	case "darwin":
		return []string{"/System/Library/Fonts", "/Library/Fonts", filepath.Join(home, "Library", "Fonts")}
	case "windows":
		return []string{filepath.Join(os.Getenv("WINDIR"), "Fonts")}
	}
	return []string{"/usr/share/fonts", "/usr/local/share/fonts", filepath.Join(home, ".local", "share", "fonts"), filepath.Join(home, ".fonts")}
}

// largestFallbackFile is the largest font file read in the last resort: the big ones (Chinese,
// Japanese and Korean) are found by their names, and reading them all for one symbol would stall.
const largestFallbackFile = 8 << 20

// systemFontSets keeps, for every list of folders, the system's fonts and colour emoji found in
// them, shared by every painter and measurer that looks there: a big font is read and parsed once.
var systemFontSets = struct {
	mutex sync.Mutex
	sets  map[string]systemFontSet
}{sets: map[string]systemFontSet{}}

type systemFontSet struct {
	fonts *fallbackFonts
	emoji *emojiFont
}

// systemFontsIn returns the system's fonts and colour emoji in these folders, read when first used.
func systemFontsIn(folders []string) (*fallbackFonts, *emojiFont) {
	key := strings.Join(folders, "\x00")
	systemFontSets.mutex.Lock()
	defer systemFontSets.mutex.Unlock()
	set, found := systemFontSets.sets[key]
	if !found {
		set = systemFontSet{fonts: &fallbackFonts{folders: slices.Clone(folders)}, emoji: &emojiFont{folders: slices.Clone(folders)}}
		systemFontSets.sets[key] = set
	}
	return set.fonts, set.emoji
}

// fallbackFonts finds the system's font for a character.
type fallbackFonts struct {
	mutex   sync.Mutex
	folders []string // the folders the files are listed from
	files   []fontFile
	read    map[string][]*opentype.Font // the fonts in each file read and kept, a collection's several
	found   map[rune]*opentype.Font     // the font found for a character; nil when none has it
}

// fontFile is a font file, with its name in lower case and without separators, to match a
// script's name in.
type fontFile struct {
	path, name string
	size       int64
}

// fontFor returns a system font that draws a character, or nil.
func (fallback *fallbackFonts) fontFor(character rune) *opentype.Font {
	fallback.mutex.Lock()
	defer fallback.mutex.Unlock()
	if font, known := fallback.found[character]; known {
		return font
	}
	fallback.list()
	found := fallback.search(character)
	if fallback.found == nil {
		fallback.found = map[rune]*opentype.Font{}
	}
	fallback.found[character] = found
	return found
}

// list lists the font files of the folders, the first time.
func (fallback *fallbackFonts) list() {
	if fallback.files != nil {
		return
	}
	fallback.files = []fontFile{}
	fallback.read = map[string][]*opentype.Font{}
	fallback.found = nil
	for _, folder := range fallback.folders {
		filepath.WalkDir(folder, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !isFontFile(path) {
				return nil
			}
			information, err := entry.Info()
			if err != nil {
				return nil
			}
			fallback.files = append(fallback.files, fontFile{path: path, name: plainName(path), size: information.Size()})
			return nil
		})
	}
}

func isFontFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".otf", ".ttc", ".otc":
		// a colour emoji font has pictures, not outlines: it is not drawn in one colour
		name := plainName(path)
		return !strings.Contains(name, "color") && !strings.Contains(name, "colour")
	}
	return false
}

// plainName is a file's name in lower case, without its extension and separators.
func plainName(path string) string {
	name := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	return strings.NewReplacer("-", "", "_", "", " ", "").Replace(name)
}

// search looks for a font with the character: first among the files named for its script, then,
// the last resort, in every file not too big to read.
func (fallback *fallbackFonts) search(character rune) *opentype.Font {
	named := namedFor(fallback.files, scriptNames(character))
	for _, file := range named {
		if found := fallback.fontIn(file, character); found != nil {
			return found
		}
	}
	for _, file := range byPreference(fallback.files) {
		if file.size > largestFallbackFile || slices.Contains(named, file) {
			continue
		}
		if found := fallback.fontIn(file, character); found != nil {
			return found
		}
	}
	return nil
}

// fontIn reads a file, the first time, and returns its font that has the character. A file whose
// fonts lack it is not kept, as most are never needed; one that has it is.
func (fallback *fallbackFonts) fontIn(file fontFile, character rune) *opentype.Font {
	fonts, kept := fallback.read[file.path]
	if !kept {
		fonts = readFonts(file.path)
	}
	for _, font := range fonts {
		if hasCharacter(font, character) {
			fallback.read[file.path] = fonts
			return font
		}
	}
	return nil
}

// readFonts parses a font file, or each font of a collection. A file that cannot be read or is not
// a font has none.
func readFonts(path string) []*opentype.Font {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	collection, err := opentype.ParseCollection(data)
	if err != nil {
		return nil
	}
	var fonts []*opentype.Font
	for index := range collection.NumFonts() {
		if font, err := collection.Font(index); err == nil {
			fonts = append(fonts, font)
		}
	}
	return fonts
}

func hasCharacter(font *opentype.Font, character rune) bool {
	var buffer sfnt.Buffer
	index, err := font.GlyphIndex(&buffer, character)
	return err == nil && index != 0
}

// scriptNames are the words a font for a character's script has in its file's name.
func scriptNames(character rune) []string {
	script := scriptOf(character)
	switch script {
	case "":
		return nil
	case "Han", "Hiragana", "Katakana", "Hangul", "Bopomofo":
		// Chinese, Japanese and Korean share one font
		return []string{"cjk", strings.ToLower(script), "droidsansfallback"}
	case "Common", "Inherited":
		return []string{"symbols", "math"}
	}
	return []string{strings.ToLower(strings.ReplaceAll(script, "_", ""))}
}

// scriptOf is the name of a character's script, as Go's unicode package names it ("Arabic"), or
// empty for one outside every script (a private use character).
func scriptOf(character rune) string {
	for _, name := range scriptsInOrder {
		if unicode.Is(unicode.Scripts[name], character) {
			return name
		}
	}
	return ""
}

// scriptsInOrder are the scripts' names, Common and Inherited last: they hold the characters no
// script owns, and a script's own table is the better answer.
var scriptsInOrder = func() []string {
	var names []string
	for name := range unicode.Scripts {
		if name != "Common" && name != "Inherited" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return append(names, "Common", "Inherited")
}()

// namedFor are the files whose names have one of the words, the plainest first.
func namedFor(files []fontFile, words []string) []fontFile {
	var named []fontFile
	for _, file := range files {
		for _, word := range words {
			if strings.Contains(file.name, word) {
				named = append(named, file)
				break
			}
		}
	}
	return byPreference(named)
}

// byPreference orders files from the plainest: a sans serif in the regular weight upright before
// a serif, a bold or an italic.
func byPreference(files []fontFile) []fontFile {
	sorted := slices.Clone(files)
	slices.SortStableFunc(sorted, func(first, second fontFile) int {
		return fontPenalty(first.name) - fontPenalty(second.name)
	})
	return sorted
}

func fontPenalty(name string) int {
	penalty := 0
	for _, word := range []string{"bold", "italic", "oblique", "light", "thin", "black", "medium", "condensed", "mono"} {
		if strings.Contains(name, word) {
			penalty += 2
		}
	}
	if strings.Contains(name, "serif") && !strings.Contains(name, "sans") {
		penalty++
	}
	return penalty
}
