package loaders

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/akrck02/go-renderer/scene"
)

// twoPixelPNG is a PNG of two pixels, red then blue.
func twoPixelPNG(test *testing.T) []byte {
	test.Helper()
	picture := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	picture.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	picture.SetNRGBA(1, 0, color.NRGBA{0, 0, 255, 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		test.Fatal(err)
	}
	return encoded.Bytes()
}

// texturedTriangle is a document of a triangle with UVs and two materials that share one texture,
// clamped and sharp, whose picture is given by image (embedded or by URI).
func texturedTriangle(uvs []byte, positions []byte, image map[string]any) map[string]any {
	textured := map[string]any{"pbrMetallicRoughness": map[string]any{"baseColorTexture": map[string]any{"index": 0}}}
	return map[string]any{
		"asset":  map[string]any{"version": "2.0"},
		"scene":  0,
		"scenes": []any{map[string]any{"nodes": []int{0, 1}}},
		"nodes":  []any{map[string]any{"name": "first", "mesh": 0}, map[string]any{"name": "second", "mesh": 1}},
		"meshes": []any{
			map[string]any{"primitives": []any{map[string]any{"attributes": map[string]int{"POSITION": 0, "TEXCOORD_0": 1}, "material": 0}}},
			map[string]any{"primitives": []any{map[string]any{"attributes": map[string]int{"POSITION": 0, "TEXCOORD_0": 1}, "material": 1}}},
		},
		"materials": []any{textured, textured},
		"textures":  []any{map[string]any{"source": 0, "sampler": 0}},
		"samplers":  []any{map[string]any{"magFilter": 9728, "wrapS": 33071, "wrapT": 33071}},
		"images":    []any{image},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3"},
			map[string]any{"bufferView": 1, "componentType": 5126, "count": 3, "type": "VEC2"},
		},
		"bufferViews": []any{
			map[string]any{"buffer": 0, "byteOffset": 0, "byteLength": len(positions)},
			map[string]any{"buffer": 0, "byteOffset": len(positions), "byteLength": len(uvs)},
			map[string]any{"buffer": 0, "byteOffset": len(positions) + len(uvs), "byteLength": 0},
		},
		"buffers": []any{map[string]any{"byteLength": 0}},
	}
}

func checkTexturedTriangle(test *testing.T, loaded *scene.Scene) {
	test.Helper()
	first, second := loaded.Nodes[0].Mesh, loaded.Nodes[1].Mesh
	if !slices.Equal(first.UVs, []float32{0, 0, 1, 0, 0, 1}) {
		test.Errorf("UVs %v, wanted the file's TEXCOORD_0", first.UVs)
	}
	texture := first.Material.Texture
	if texture == nil || texture.Picture == nil {
		test.Fatalf("no texture on %+v", first.Material)
	}
	if width, height := texture.Picture.Size(); width != 2 || height != 1 || texture.Picture.Pixels.NRGBAAt(1, 0).B != 255 {
		test.Errorf("the picture is %d × %d, its second pixel %v", width, height, texture.Picture.Pixels.NRGBAAt(1, 0))
	}
	if texture.Wrap != scene.Clamp || !texture.Sharp {
		test.Errorf("wrap %v and sharp %v, wanted the sampler's clamp and nearest", texture.Wrap, texture.Sharp)
	}
	if second.Material.Texture != texture {
		test.Error("two materials of one texture got two textures")
	}
}

func TestATexturedGLBGivesUVsAndOneSharedTexture(test *testing.T) {
	positions := littleEndianFloats(0, 0, 0, 1, 0, 0, 0, 0, 1)
	uvs := littleEndianFloats(0, 0, 1, 0, 0, 1)
	picture := twoPixelPNG(test)
	document := texturedTriangle(uvs, positions, map[string]any{"bufferView": 2, "mimeType": "image/png"})
	document["bufferViews"].([]any)[2].(map[string]any)["byteLength"] = len(picture)
	binaryData := append(append(append([]byte{}, positions...), uvs...), picture...)
	document["buffers"] = []any{map[string]any{"byteLength": len(binaryData)}}
	content, err := os.ReadFile(writeGLB(test, document, binaryData))
	if err != nil {
		test.Fatal(err)
	}
	loaded, err := ReadScene(bytes.NewReader(content))
	if err != nil {
		test.Fatal(err)
	}
	checkTexturedTriangle(test, loaded)
}

func TestATexturesPictureIsReadBesideTheFile(test *testing.T) {
	positions := littleEndianFloats(0, 0, 0, 1, 0, 0, 0, 0, 1)
	uvs := littleEndianFloats(0, 0, 1, 0, 0, 1)
	document := texturedTriangle(uvs, positions, map[string]any{"uri": "stones.png"})
	binaryData := append(append([]byte{}, positions...), uvs...)
	document["buffers"] = []any{map[string]any{"byteLength": len(binaryData)}}
	path := writeGLB(test, document, binaryData)
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "stones.png"), twoPixelPNG(test), 0o644); err != nil {
		test.Fatal(err)
	}
	loaded, err := LoadScene(path)
	if err != nil {
		test.Fatal(err)
	}
	checkTexturedTriangle(test, loaded)
	content, _ := os.ReadFile(path)
	if _, err := ReadScene(bytes.NewReader(content)); err == nil {
		test.Error("ReadScene read a picture beside a file it has no folder for")
	}
}
