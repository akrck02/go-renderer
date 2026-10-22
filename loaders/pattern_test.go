package loaders

import (
	"testing"

	"github.com/akrck02/go-renderer/scene"
)

func TestMaterialExtrasGivePatternsAndGlow(test *testing.T) {
	file := &gltfFile{document: gltfDocument{Materials: []gltfMaterialDefinition{
		{Name: "street", Extras: map[string]any{"kind": "lit", "pattern": "cobbles", "patternScale": 0.0004}},
		{Name: "flowers", Extras: map[string]any{"kind": "glow"}},
		{Name: "odd", Extras: map[string]any{"pattern": "marble"}},
	}}}
	materials := file.buildMaterials()
	if materials[0].Pattern != scene.PatternCobbles || materials[0].PatternScale != 0.0004 {
		test.Errorf("street: %+v", materials[0])
	}
	if materials[1].Kind != scene.GlowKind {
		test.Errorf("flowers should glow: %+v", materials[1])
	}
	if materials[2].Pattern != scene.PatternNone {
		test.Errorf("an unknown pattern is none: %+v", materials[2])
	}
}
