package sceneplan

import (
	"slices"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/scene"
)

func TestOpaqueDrawsAreGroupedByTextureAndKeepTheirOrderOtherwise(test *testing.T) {
	stones := &scene.Texture{Picture: draw2d.NewBlankImage(8, 8)}
	planks := &scene.Texture{Picture: draw2d.NewBlankImage(8, 8)}
	plain, stone, wood := &scene.Material{Name: "plain"}, &scene.Material{Name: "stone", Texture: stones}, &scene.Material{Name: "wood", Texture: planks}
	commands := []Command{{Material: wood}, {Material: plain}, {Material: stone}, {Material: wood},
		{Material: &scene.Material{Name: "glass", Transparent: true}, Transparent: true}, {Material: plain}}
	for index := range commands {
		commands[index].textureOrder = textureOrder(commands[index].Material)
	}
	sortOpaqueBeforeTransparent(commands)
	var names []string
	for _, command := range commands {
		names = append(names, command.Material.Name)
	}
	if wanted := []string{"plain", "plain", "stone", "wood", "wood", "glass"}; !slices.Equal(names, wanted) {
		test.Errorf("drawn in the order %v, wanted %v", names, wanted)
	}
}

func TestATexturesPlacementIsItsScaleAndOffset(test *testing.T) {
	material := &scene.Material{TextureScale: [2]float32{4, 0}, TextureOffset: [2]float32{0.5, 0.25}}
	if got, wanted := TexturePlacement(material), [4]float32{4, 1, 0.5, 0.25}; got != wanted {
		test.Errorf("placement %v, wanted %v", got, wanted)
	}
}

func TestOnlyAMaterialWithAPictureIsTextured(test *testing.T) {
	cases := map[*scene.Material]bool{
		{}:                          false,
		{Texture: &scene.Texture{}}: false,
		{Texture: &scene.Texture{Picture: draw2d.NewBlankImage(8, 8)}}: true,
	}
	for material, wanted := range cases {
		if IsTextured(material) != wanted {
			test.Errorf("%+v textured: %v, wanted %v", material, !wanted, wanted)
		}
	}
}

func TestMergedInstancesKeepTheirMeshsUVs(test *testing.T) {
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 0, 1}, UVs: []float32{0, 0, 1, 0, 0, 1}}
	packed := make([]float32, 2*FloatsPerInstance)
	for instance := range 2 {
		for diagonal := range 4 {
			packed[instance*FloatsPerInstance+5*diagonal] = 1
		}
	}
	merged := &scene.Mesh{}
	MergeInstances(merged, mesh, packed, 2)
	if wanted := append(slices.Clone(mesh.UVs), mesh.UVs...); !slices.Equal(merged.UVs, wanted) {
		test.Errorf("merged UVs %v, wanted %v", merged.UVs, wanted)
	}
}
