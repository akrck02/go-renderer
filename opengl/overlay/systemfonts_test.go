package overlay

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTheSystemFontIsTheFirstOneFoundWithBothWeights(test *testing.T) {
	folder := test.TempDir()
	nested := filepath.Join(folder, "truetype", "noto")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		test.Fatal(err)
	}
	for name, content := range map[string]string{"OnlyRegular.ttf": "a", "Noto.ttf": "regular", "NotoBold.ttf": "bold"} {
		if err := os.WriteFile(filepath.Join(nested, name), []byte(content), 0o644); err != nil {
			test.Fatal(err)
		}
	}
	faces := []systemFace{{"OnlyRegular.ttf", "Missing.ttf"}, {"Noto.ttf", "NotoBold.ttf"}}
	fonts, err := interfaceFontsIn([]string{folder}, faces)
	if err != nil || string(fonts.Regular) != "regular" || string(fonts.Medium) != "regular" || string(fonts.Bold) != "bold" {
		test.Errorf("fonts %q %q %q, %v", fonts.Regular, fonts.Medium, fonts.Bold, err)
	}
}

func TestWithoutASystemFontTheGoFontsStay(test *testing.T) {
	fonts, err := interfaceFontsIn([]string{test.TempDir()}, systemFaces["linux"])
	if !errors.Is(err, SystemFontNotFound) || fonts.Regular != nil {
		test.Errorf("fonts %v, %v", fonts, err)
	}
}

func TestAVariableFontIsReadOnceForBothWeights(test *testing.T) {
	folder := test.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "Variable.ttf"), []byte("every weight"), 0o644); err != nil {
		test.Fatal(err)
	}
	fonts, err := interfaceFontsIn([]string{folder}, []systemFace{{"Variable.ttf", "Variable.ttf"}})
	if err != nil {
		test.Fatal(err)
	}
	if &fonts.Regular[0] != &fonts.Bold[0] {
		test.Errorf("the regular and bold weights of one file are two copies in memory")
	}
}
