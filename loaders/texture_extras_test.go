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
	materials, err := file.buildMaterials()
	if err != nil {
		test.Fatal(err)
	}
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

func TestTheOldPatternExtrasStillGiveTheirTextureWithCellsOfTheirSize(test *testing.T) {
	file := &gltfFile{document: gltfDocument{Materials: []gltfMaterialDefinition{
		{Name: "street", Extras: map[string]any{"pattern": "cobbles", "patternScale": 0.5}},
	}}}
	materials, err := file.buildMaterials()
	if err != nil {
		test.Fatal(err)
	}
	cobbles, _ := scene.BuiltInTexture("cobbles")
	// five cobbles a repeat, each half a unit: a repeat every 2.5 units
	if materials[0].Texture != cobbles || materials[0].TextureScale != [2]float32{0.4, 0.4} {
		test.Errorf("street: texture %p (cobbles %p), scale %v", materials[0].Texture, cobbles, materials[0].TextureScale)
	}
}
