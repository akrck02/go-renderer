package loaders

import (
	"testing"

	"github.com/akrck02/go-renderer/scene"
)

func TestMaterialExtrasGiveBuiltInTexturesAndGlow(test *testing.T) {
	file := &gltfFile{document: gltfDocument{Materials: []gltfMaterialDefinition{
		{Name: "street", Extras: map[string]any{"kind": "lit", "texture": "cobbles", "textureScale": 0.5}},
		{Name: "flowers", Extras: map[string]any{"kind": "glow"}},
		{Name: "odd", Extras: map[string]any{"texture": "marble"}},
	}}}
	materials := file.buildMaterials()
	cobbles, _ := scene.BuiltInTexture("cobbles")
	if materials[0].Texture != cobbles || materials[0].TextureScale != [2]float32{0.5, 0.5} {
		test.Errorf("street: %+v", materials[0])
	}
	if materials[1].Kind != scene.GlowKind {
		test.Errorf("flowers should glow: %+v", materials[1])
	}
	if materials[2].Texture != nil {
		test.Errorf("an unknown texture is none: %+v", materials[2])
	}
}
