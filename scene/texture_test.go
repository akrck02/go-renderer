package scene

import (
	"slices"
	"testing"
)

func TestATextureRepeatsSmoothlyInFullColourUnlessToldOtherwise(test *testing.T) {
	texture := Texture{}
	if texture.Wrap != Repeat || texture.Format != RGBA8 || texture.Sharp {
		test.Errorf("a new texture is %+v; wanted it repeated, smooth and in RGBA8", texture)
	}
}

func TestATextureScaleOfZeroShowsTheTextureOnce(test *testing.T) {
	cases := map[[2]float32][2]float32{{0, 0}: {1, 1}, {4, 0}: {4, 1}, {2, 3}: {2, 3}}
	for scale, wanted := range cases {
		material := Material{TextureScale: scale}
		if got := material.EffectiveTextureScale(); got != wanted {
			test.Errorf("scale %v: %v, wanted %v", scale, got, wanted)
		}
	}
}

func TestAMeshHasUVsOnlyWithTwoForEachVertex(test *testing.T) {
	triangle := &Mesh{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}}
	if triangle.HasUVs() {
		test.Error("a mesh without UVs has them")
	}
	triangle.UVs = []float32{0, 0, 1, 0, 0, 1}
	if !triangle.HasUVs() {
		test.Error("a mesh with two UVs a vertex has none")
	}
	triangle.UVs = triangle.UVs[:4]
	if triangle.HasUVs() {
		test.Error("a mesh with too few UVs has them")
	}
}

func TestAMeshWithoutUVsTakesThemFromWhereItsVerticesAre(test *testing.T) {
	floor := &Mesh{Positions: []float32{1, 0, 2, 3, 0, 2, 1, 0, 5}, Normals: []float32{0, 1, 0, 0, 1, 0, 0, 1, 0}}
	if got, wanted := floor.SurfaceUVs(), []float32{1, 2, 3, 2, 1, 5}; !slices.Equal(got, wanted) {
		test.Errorf("a floor's UVs are %v, wanted its x and z %v", got, wanted)
	}
	wall := &Mesh{Positions: []float32{1, 0, 7, 3, 0, 7, 1, 4, 7}, Normals: []float32{0, 0, 1, 0, 0, 1, 0, 0, 1}}
	if got, wanted := wall.SurfaceUVs(), []float32{1, 0, 3, 0, 1, 4}; !slices.Equal(got, wanted) {
		test.Errorf("a wall facing along z has UVs %v, wanted its x and y %v", got, wanted)
	}
	side := &Mesh{Positions: []float32{7, 0, 1, 7, 0, 3, 7, 4, 1}, Normals: []float32{1, 0, 0, 1, 0, 0, 1, 0, 0}}
	if got, wanted := side.SurfaceUVs(), []float32{1, 0, 3, 0, 1, 4}; !slices.Equal(got, wanted) {
		test.Errorf("a wall facing along x has UVs %v, wanted its z and y %v", got, wanted)
	}
	floor.UVs = []float32{0, 0, 1, 0, 0, 1}
	if got := floor.SurfaceUVs(); !slices.Equal(got, floor.UVs) {
		test.Errorf("a mesh's own UVs were replaced by %v", got)
	}
}
