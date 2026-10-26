package sceneplan

import "github.com/akrck02/go-renderer/scene"

// IsTextured tells whether a material lays a picture over its surfaces.
func IsTextured(material *scene.Material) bool {
	return material != nil && material.Texture != nil && material.Texture.Picture != nil
}

// TexturePlacement is how a material lays its texture: the scale along u and v (0 taken as 1),
// then the offset along u and v. A shader reads the texture at UV × scale + offset.
func TexturePlacement(material *scene.Material) [4]float32 {
	scale := material.EffectiveTextureScale()
	return [4]float32{scale[0], scale[1], material.TextureOffset[0], material.TextureOffset[1]}
}

// textureOrder is a number that groups the draws of one texture together: the picture's id, 0 for
// a material without one, so untextured draws keep coming first in their order.
func textureOrder(material *scene.Material) uint64 {
	if !IsTextured(material) {
		return 0
	}
	return material.Texture.Picture.ID()
}

// UVsToDraw are the UVs a mesh is uploaded with when drawn with a material: its own when it has
// them, ones projected from its positions when the material is textured (scene.Mesh.SurfaceUVs),
// and none otherwise, so an untextured mesh costs what it did before textures.
func UVsToDraw(mesh *scene.Mesh, material *scene.Material) []float32 {
	if mesh.HasUVs() {
		return mesh.UVs
	}
	if !IsTextured(material) {
		return nil
	}
	return mesh.SurfaceUVs()
}
