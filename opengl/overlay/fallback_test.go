package overlay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/opengl/overlay/lucide"
	"golang.org/x/image/font/gofont/goregular"
)

// withFontFolder makes a folder of system fonts for a test, with files of these names and fonts.
func withFontFolder(t *testing.T, files map[string][]byte) {
	t.Helper()
	folder := t.TempDir()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(folder, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before := FontFolders
	FontFolders = []string{folder}
	t.Cleanup(func() { FontFolders = before })
}

func TestACharacterIsLookedForInTheFontNamedForItsScript(t *testing.T) {
	for character, words := range map[rune][]string{
		'ب': {"arabic"}, '中': {"cjk", "han", "droidsansfallback"}, 'か': {"cjk", "hiragana", "droidsansfallback"},
		'क': {"devanagari"}, '★': {"symbols", "math"}, 0xe000: nil,
	} {
		if got := scriptNames(character); !equalWords(got, words) {
			t.Errorf("%q: %v, want %v", character, got, words)
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

func TestThePlainestFontComesFirst(t *testing.T) {
	files := []fontFile{{name: plainName("NotoSerifArabic-Bold.ttf")}, {name: plainName("NotoNaskhArabic-Regular.ttf")},
		{name: plainName("NotoSansArabic-Italic.ttf")}, {name: plainName("NotoSansArabic-Regular.ttf")}}
	sorted := byPreference(files)
	if sorted[0].name != "notonaskharabicregular" || sorted[1].name != "notosansarabicregular" || sorted[3].name != "notoserifarabicbold" {
		t.Fatalf("regular upright first, a bold serif last: %v", sorted)
	}
	if isFontFile("NotoColorEmoji.ttf") || !isFontFile("NotoSansCJK-Regular.ttc") || isFontFile("fonts.xml") {
		t.Fatal("colour emoji are left out of the fonts drawn in one colour")
	}
}

func TestAFontNamedForTheScriptIsFound(t *testing.T) {
	icons := lucide.Font()
	withFontFolder(t, map[string][]byte{"NotoSansSymbols-Regular.ttf": goregular.TTF, "NotoSansLucide-Regular.ttf": icons})
	star, _ := lucide.Codepoint("star")
	if systemFonts.fontFor('λ') == nil {
		t.Fatal("a Greek letter is in the folder's Go font")
	}
	if systemFonts.fontFor(star) == nil {
		t.Fatal("a character of no script is looked for in every font, the last resort")
	}
	if systemFonts.fontFor('中') != nil {
		t.Fatal("no font in the folder has Chinese")
	}
}

func TestTextTheFontsLackIsMeasuredInTheSystemsFont(t *testing.T) {
	withFontFolder(t, map[string][]byte{"Icons.ttf": lucide.Font()})
	measurer, err := NewMeasurer()
	if err != nil {
		t.Fatal(err)
	}
	star, _ := lucide.Codepoint("star")
	drawn := measurer.MeasureText(string(star), 16, draw2d.Regular, 1)
	question := measurer.MeasureText("?", 16, draw2d.Regular, 1)
	if drawn.Width == question.Width || drawn.Width <= 0 {
		t.Fatalf("the system's glyph, not a question mark: %v and %v", drawn.Width, question.Width)
	}
	FontFolders = nil
	if measured := measurer.MeasureText(string(rune(0xe001)), 16, draw2d.Regular, 1); measured.Width != question.Width {
		t.Fatalf("with no system fonts it is a question mark: %v", measured.Width)
	}
}
