package overlay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/opengl/overlay/lucide"
	"golang.org/x/image/font/gofont/goregular"
)

// withFontFolder makes a folder of system fonts for a test, with files of these names and fonts,
// and returns the default fonts with it as their only folder.
func withFontFolder(test *testing.T, files map[string][]byte) Fonts {
	test.Helper()
	folder := test.TempDir()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(folder, name), data, 0o644); err != nil {
			test.Fatal(err)
		}
	}
	return Fonts{Folders: []string{folder}}
}

func TestACharacterIsLookedForInTheFontNamedForItsScript(test *testing.T) {
	for character, words := range map[rune][]string{
		'ب': {"arabic"}, '中': {"cjk", "han", "droidsansfallback"}, 'か': {"cjk", "hiragana", "droidsansfallback"},
		'क': {"devanagari"}, '★': {"symbols", "math"}, 0xe000: nil,
	} {
		if got := scriptNames(character); !equalWords(got, words) {
			test.Errorf("%q: %v, want %v", character, got, words)
		}
	}
}

func equalWords(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func TestThePlainestFontComesFirst(test *testing.T) {
	files := []fontFile{{name: plainName("NotoSerifArabic-Bold.ttf")}, {name: plainName("NotoNaskhArabic-Regular.ttf")},
		{name: plainName("NotoSansArabic-Italic.ttf")}, {name: plainName("NotoSansArabic-Regular.ttf")}}
	sorted := byPreference(files)
	if sorted[0].name != "notonaskharabicregular" || sorted[1].name != "notosansarabicregular" || sorted[3].name != "notoserifarabicbold" {
		test.Fatalf("regular upright first, a bold serif last: %v", sorted)
	}
	if isFontFile("NotoColorEmoji.ttf") || !isFontFile("NotoSansCJK-Regular.ttc") || isFontFile("fonts.xml") {
		test.Fatal("colour emoji are left out of the fonts drawn in one colour")
	}
}

func TestAFontNamedForTheScriptIsFound(test *testing.T) {
	icons := lucide.Font()
	fonts := withFontFolder(test, map[string][]byte{"NotoSansSymbols-Regular.ttf": goregular.TTF, "NotoSansLucide-Regular.ttf": icons})
	systemFonts, _ := systemFontsIn(fonts.Folders)
	star, _ := lucide.Codepoint("star")
	if systemFonts.fontFor('λ') == nil {
		test.Fatal("a Greek letter is in the folder's Go font")
	}
	if systemFonts.fontFor(star) == nil {
		test.Fatal("a character of no script is looked for in every font, the last resort")
	}
	if systemFonts.fontFor('中') != nil {
		test.Fatal("no font in the folder has Chinese")
	}
}

func TestTextTheFontsLackIsMeasuredInTheSystemsFont(test *testing.T) {
	fonts := withFontFolder(test, map[string][]byte{"Icons.ttf": lucide.Font()})
	measurer, err := NewMeasurer(fonts)
	if err != nil {
		test.Fatal(err)
	}
	star, _ := lucide.Codepoint("star")
	drawn := measurer.MeasureText(string(star), 16, draw2d.Regular, 1)
	question := measurer.MeasureText("?", 16, draw2d.Regular, 1)
	if drawn.Width == question.Width || drawn.Width <= 0 {
		test.Fatalf("the system's glyph, not a question mark: %v and %v", drawn.Width, question.Width)
	}
	nowhere, err := NewMeasurer(Fonts{Folders: []string{}})
	if err != nil {
		test.Fatal(err)
	}
	if measured := nowhere.MeasureText(string(rune(0xe001)), 16, draw2d.Regular, 1); measured.Width != question.Width {
		test.Fatalf("with no system fonts it is a question mark: %v", measured.Width)
	}
}

func TestAFontFilesPlainNameAllocatesOnlyItsText(test *testing.T) {
	plainName("Warm-up.ttf") // the shared replacer is built on its first use
	allocations := testing.AllocsPerRun(100, func() { plainName("Noto Sans_Arabic-Bold.ttf") })
	const atMost = 3 // the lower-case name, the plain one and their joining
	if allocations > atMost {
		test.Errorf("a plain name costs %.0f allocations, want at most %d", allocations, atMost)
	}
}
