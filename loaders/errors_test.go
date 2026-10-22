package loaders

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/akrck02/go-renderer/failure"
)

func TestAMissingFileIsFileNotRead(test *testing.T) {
	_, err := LoadScene(filepath.Join(test.TempDir(), "missing.glb"))
	if !errors.Is(err, FileNotRead) || !errors.Is(err, os.ErrNotExist) {
		test.Fatalf("%v, not %s over the missing file", err, FileNotRead.Name())
	}
}

func TestBrokenFilesAreInvalidGLTF(test *testing.T) {
	cutShort := make([]byte, 20)
	binary.LittleEndian.PutUint32(cutShort, glbMagic)
	binary.LittleEndian.PutUint32(cutShort[12:], 100) // a chunk longer than the file
	broken := map[string][]byte{
		"not a .glb":      []byte("hello"),
		"a chunk cut off": cutShort,
	}
	for name, content := range broken {
		if _, err := ParseSceneGLB(content); !errors.Is(err, InvalidGLTF) {
			test.Errorf("%s: %v, not %s", name, err, InvalidGLTF.Name())
		}
	}
	path := filepath.Join(test.TempDir(), "broken.gltf")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		test.Fatal(err)
	}
	if _, err := LoadScene(path); !errors.Is(err, InvalidGLTF) {
		test.Errorf("bad JSON: %v, not %s", err, InvalidGLTF.Name())
	}
}

func TestAnAccessorOfAnUnknownTypeIsUnsupportedAccessor(test *testing.T) {
	file := &gltfFile{}
	file.document.Accessors = append(file.document.Accessors, gltfAccessorDefinition{Type: "MAT9", ComponentType: 5126, Count: 1})
	if _, _, err := file.readAccessor(0); !errors.Is(err, UnsupportedAccessor) {
		test.Fatalf("%v, not %s", err, UnsupportedAccessor.Name())
	}
	if _, _, err := file.readAccessor(1); !errors.Is(err, InvalidGLTF) {
		test.Fatalf("an accessor that does not exist: %v, not %s", err, InvalidGLTF.Name())
	}
}

func TestASkyImageThatIsNotAPictureIsImageNotDecoded(test *testing.T) {
	directory := test.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "sky.png"), []byte("not a picture"), 0o600); err != nil {
		test.Fatal(err)
	}
	file := &gltfFile{directory: directory}
	if _, err := file.readImage("sky.png"); !errors.Is(err, ImageNotDecoded) {
		test.Fatalf("%v, not %s", err, ImageNotDecoded.Name())
	}
	if _, err := file.readImage("missing.png"); failure.CodeOf(err) != FileNotRead {
		test.Fatalf("a missing image: %v, not %s", err, FileNotRead.Name())
	}
}
