package overlay

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAMappedFileReadsAsItsContents(test *testing.T) {
	path := filepath.Join(test.TempDir(), "Font.ttf")
	if err := os.WriteFile(path, []byte("every glyph"), 0o644); err != nil {
		test.Fatal(err)
	}
	data, release, err := mapFile(path)
	if err != nil {
		test.Fatal(err)
	}
	defer release()
	if string(data) != "every glyph" {
		test.Errorf("mapped %q", data)
	}
}

func TestAnEmptyFileIsReadRatherThanMapped(test *testing.T) {
	path := filepath.Join(test.TempDir(), "Empty.ttf")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	data, release, err := mapFile(path)
	if err != nil || len(data) != 0 {
		test.Errorf("an empty file gave %q, %v", data, err)
	}
	release()
}

func TestAMappedFontCostsTheProgramsMemoryNothing(test *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "js" {
		test.Skip("this system reads fonts whole")
	}
	path := filepath.Join(test.TempDir(), "Large.ttf")
	const size = 8 << 20
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		test.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	data, release, err := mapFile(path)
	if err != nil {
		test.Fatal(err)
	}
	defer release()
	runtime.ReadMemStats(&after)
	if grown := after.TotalAlloc - before.TotalAlloc; grown > size/2 {
		test.Errorf("mapping %d bytes allocated %d in the program's memory", len(data), grown)
	}
}
