package loaders

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSceneLinksDetailLevels(t *testing.T) {
	triangle := make([]byte, 36)
	for index, value := range []float32{0, 0, 0, 1, 0, 0, 0, 1, 0} {
		binary.LittleEndian.PutUint32(triangle[index*4:], math.Float32bits(value))
	}
	document := `{"asset": {"version": "2.0"}, "scene": 0,
		"buffers": [{"byteLength": 36, "uri": "triangle.bin"}],
		"bufferViews": [{"buffer": 0, "byteLength": 36}],
		"accessors": [{"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3"}],
		"meshes": [
			{"name": "full", "primitives": [{"attributes": {"POSITION": 0}}], "extras": {"detail": [{"mesh": 1, "screenSize": 0.05}]}},
			{"name": "simple", "primitives": [{"attributes": {"POSITION": 0}}]}],
		"nodes": [{"mesh": 0}], "scenes": [{"nodes": [0]}]}`
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "triangle.bin"), triangle, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "detail.gltf")
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	world, err := LoadScene(path)
	if err != nil {
		t.Fatal(err)
	}
	details := world.Nodes[0].Mesh.Details
	if len(details) != 1 || details[0].Mesh.Name != "simple" || details[0].ScreenSize != 0.05 {
		t.Fatalf("detail level not linked: %+v", details)
	}
}
