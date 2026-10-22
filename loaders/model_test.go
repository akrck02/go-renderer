package loaders

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

func TestLoadModelReadsPositionsAndShortIndices(test *testing.T) {
	positions := littleEndianFloats(0, 0, 0, 1, 0, 0, 0, 1, 0)
	indices := binary.LittleEndian.AppendUint16(nil, 0)
	indices = binary.LittleEndian.AppendUint16(indices, 1)
	indices = binary.LittleEndian.AppendUint16(indices, 2)
	data := append(append([]byte{}, positions...), indices...)
	document := map[string]any{
		"asset":  map[string]any{"version": "2.0"},
		"meshes": []any{map[string]any{"primitives": []any{map[string]any{"attributes": map[string]int{"POSITION": 0}, "indices": 1}}}},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3"},
			map[string]any{"bufferView": 1, "componentType": 5123, "count": 3, "type": "SCALAR"},
		},
		"bufferViews": []any{
			map[string]any{"buffer": 0, "byteOffset": 0, "byteLength": len(positions)},
			map[string]any{"buffer": 0, "byteOffset": len(positions), "byteLength": len(indices)},
		},
		"buffers": []any{map[string]any{"byteLength": len(data)}},
	}
	model, err := LoadModel(writeGLB(test, document, data))
	if err != nil {
		test.Fatal(err)
	}
	if len(model.Meshes) != 1 {
		test.Fatalf("got %d meshes, want 1", len(model.Meshes))
	}
	mesh := model.Meshes[0]
	if len(mesh.Vertices) != 3 || mesh.Vertices[1] != (graphics.Vector{1, 0, 0, 1}) {
		test.Fatalf("vertices = %v", mesh.Vertices)
	}
	if len(mesh.Indices) != 3 || mesh.Indices[2] != 2 {
		test.Fatalf("indices = %v", mesh.Indices)
	}
}

func TestLoadModelOfAMissingFileIsFileNotRead(test *testing.T) {
	if _, err := LoadModel("no-such-model.glb"); !errors.Is(err, FileNotRead) {
		test.Fatalf("got %v, want FileNotRead", err)
	}
}
