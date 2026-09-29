package loaders

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akrck02/go-renderer/scene"
)

// writeGLB packs a glTF JSON document and a binary buffer into a .glb file.
func writeGLB(t *testing.T, document map[string]any, binaryData []byte) string {
	t.Helper()
	jsonData, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	for len(jsonData)%4 != 0 {
		jsonData = append(jsonData, ' ')
	}
	for len(binaryData)%4 != 0 {
		binaryData = append(binaryData, 0)
	}
	var container bytes.Buffer
	total := 12 + 8 + len(jsonData) + 8 + len(binaryData)
	binary.Write(&container, binary.LittleEndian, []uint32{glbMagic, 2, uint32(total)})
	binary.Write(&container, binary.LittleEndian, []uint32{uint32(len(jsonData)), glbJSONChunk})
	container.Write(jsonData)
	binary.Write(&container, binary.LittleEndian, []uint32{uint32(len(binaryData)), glbBinaryChunk})
	container.Write(binaryData)
	path := filepath.Join(t.TempDir(), "test.glb")
	if err := os.WriteFile(path, container.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func littleEndianFloats(values ...float32) []byte {
	var encoded bytes.Buffer
	binary.Write(&encoded, binary.LittleEndian, values)
	return encoded.Bytes()
}

func totalLength(parts ...[]byte) int {
	length := 0
	for _, part := range parts {
		length += len(part)
	}
	return length
}

func TestLoadSceneGroundInstancesAndExtras(t *testing.T) {
	// triangle on the XZ plane rising to y=10 at the far corner
	positions := littleEndianFloats(0, 0, 0, 10, 0, 0, 0, 10, 10)
	colors := []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255}
	indices := []byte{0, 0, 1, 0, 2, 0, 0, 0} // uint16 indices + padding
	translations := littleEndianFloats(1, 0, 0, 2, 0, 0)
	binaryData := append(append(append(append([]byte{}, positions...), colors...), indices...), translations...)
	document := map[string]any{
		"asset":  map[string]any{"version": "2.0"},
		"scene":  0,
		"scenes": []any{map[string]any{"nodes": []int{0, 1}, "extras": map[string]any{"environment": map[string]any{"verticalScale": 2, "water": map[string]any{"level": 0.5}}}}},
		"nodes": []any{
			map[string]any{"name": "terrain", "mesh": 0, "extras": map[string]any{"ground": true}},
			map[string]any{"name": "trees", "mesh": 0, "extensions": map[string]any{"EXT_mesh_gpu_instancing": map[string]any{"attributes": map[string]int{"TRANSLATION": 3}}}},
		},
		"meshes":    []any{map[string]any{"primitives": []any{map[string]any{"attributes": map[string]int{"POSITION": 0, "COLOR_0": 1}, "indices": 2, "material": 0}}}},
		"materials": []any{map[string]any{"pbrMetallicRoughness": map[string]any{"baseColorFactor": []float32{1, 1, 1, 0.5}}, "alphaMode": "BLEND", "extras": map[string]any{"kind": "water"}}},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3"},
			map[string]any{"bufferView": 1, "componentType": 5121, "normalized": true, "count": 3, "type": "VEC4"},
			map[string]any{"bufferView": 2, "componentType": 5123, "count": 3, "type": "SCALAR"},
			map[string]any{"bufferView": 3, "componentType": 5126, "count": 2, "type": "VEC3"},
		},
		"bufferViews": []any{
			map[string]any{"buffer": 0, "byteOffset": 0, "byteLength": len(positions)},
			map[string]any{"buffer": 0, "byteOffset": totalLength(positions), "byteLength": len(colors)},
			map[string]any{"buffer": 0, "byteOffset": totalLength(positions, colors), "byteLength": 6},
			map[string]any{"buffer": 0, "byteOffset": totalLength(positions, colors, indices), "byteLength": len(translations)},
		},
		"buffers": []any{map[string]any{"byteLength": len(binaryData)}},
	}
	loaded, err := LoadScene(writeGLB(t, document, binaryData))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Nodes) != 2 || loaded.Nodes[0].Mesh == nil {
		t.Fatalf("unexpected nodes: %+v", loaded.Nodes)
	}
	terrain := loaded.Nodes[0].Mesh
	if len(terrain.Indices) != 3 || len(terrain.Colors) != 12 || math.Abs(float64(terrain.Colors[0]-1)) > 1e-6 {
		t.Fatalf("mesh not read correctly: %+v", terrain)
	}
	if terrain.Material == nil || terrain.Material.Kind != scene.KindWater || !terrain.Material.Transparent {
		t.Fatalf("material not read: %+v", terrain.Material)
	}
	trees := loaded.Nodes[1].Instances
	if trees == nil || len(trees.Transforms) != 2 || trees.Transforms[1][12] != 2 {
		t.Fatalf("instances not read: %+v", trees)
	}
	if loaded.Environment.VerticalScale != 2 || loaded.Environment.Water == nil || loaded.Environment.Water.Level != 0.5 {
		t.Fatalf("environment not read: %+v", loaded.Environment)
	}
	// ground height inside the triangle (the instanced node is not ground)
	height, found := loaded.GroundHeight(1, 5)
	if !found || math.Abs(float64(height-5)) > 1e-4 {
		t.Fatalf("ground height = %v %v, want 5", height, found)
	}
	if _, found := loaded.GroundHeight(9, 9); found {
		t.Fatal("point outside the triangle should have no ground")
	}
}
