package loaders

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// A triangle written as KHR_mesh_quantization does: 16-bit positions (padded to 8 bytes) with the
// center and scale in the node, 8-bit normals (padded to 4 bytes), 8-bit colors and 16-bit indices.
func TestLoadSceneReadsQuantizedMeshes(t *testing.T) {
	var data []byte
	for _, position := range [][3]int16{{-32767, -32767, 0}, {32767, -32767, 0}, {0, 32767, 0}} {
		for _, component := range position {
			data = binary.LittleEndian.AppendUint16(data, uint16(component))
		}
		data = append(data, 0, 0)
	}
	for range 3 {
		data = append(data, 0, 127, 0, 0) // normal (0, 1, 0)
	}
	for range 3 {
		data = append(data, 255, 128, 0, 255) // color
	}
	for _, index := range []uint16{0, 1, 2} {
		data = binary.LittleEndian.AppendUint16(data, index)
	}
	document := `{"asset": {"version": "2.0"}, "scene": 0,
		"extensionsUsed": ["KHR_mesh_quantization"], "extensionsRequired": ["KHR_mesh_quantization"],
		"buffers": [{"byteLength": 54, "uri": "triangle.bin"}],
		"bufferViews": [
			{"buffer": 0, "byteOffset": 0, "byteLength": 24, "byteStride": 8},
			{"buffer": 0, "byteOffset": 24, "byteLength": 12, "byteStride": 4},
			{"buffer": 0, "byteOffset": 36, "byteLength": 12},
			{"buffer": 0, "byteOffset": 48, "byteLength": 6}],
		"accessors": [
			{"bufferView": 0, "componentType": 5122, "count": 3, "type": "VEC3"},
			{"bufferView": 1, "componentType": 5120, "normalized": true, "count": 3, "type": "VEC3"},
			{"bufferView": 2, "componentType": 5121, "normalized": true, "count": 3, "type": "VEC4"},
			{"bufferView": 3, "componentType": 5123, "count": 3, "type": "SCALAR"}],
		"meshes": [{"primitives": [{"attributes": {"POSITION": 0, "NORMAL": 1, "COLOR_0": 2}, "indices": 3}]}],
		"nodes": [{"mesh": 0, "translation": [10, 0, 0], "scale": [0.0001, 0.0001, 0.0001]}],
		"scenes": [{"nodes": [0]}]}`
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "triangle.bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "quantized.gltf")
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	world, err := LoadScene(path)
	if err != nil {
		t.Fatal(err)
	}
	node := world.Nodes[0]
	mesh := node.Mesh
	corner := node.Transform.Apply(mesh.Vertex(1))
	if math.Abs(float64(corner[0]-13.2767)) > 1e-3 || math.Abs(float64(corner[1]+3.2767)) > 1e-3 {
		t.Fatalf("dequantized corner %v", corner)
	}
	if mesh.Normals[1] != 1 || math.Abs(float64(mesh.Colors[1]-128.0/255)) > 1e-6 || len(mesh.Indices) != 3 || mesh.Indices[2] != 2 {
		t.Fatalf("normals %v colors %v indices %v", mesh.Normals[:3], mesh.Colors[:4], mesh.Indices)
	}
}
