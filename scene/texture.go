package scene

import "github.com/akrck02/go-renderer/draw2d"

// Texture is a picture a material lays over its surfaces: a tiling detail (panels, stones, a
// weave) or a whole picture. A program makes one, keeps it, and can give it to any number of
// materials: backends upload its pixels once, and again only after Picture.Changed().
type Texture struct {
	Picture *draw2d.Image // its pixels, the same kind of picture 2D lists draw
	Wrap    Wrap          // what the surface shows past the picture's edges
	// Sharp shows the nearest pixel, as pixel art wants, instead of blending between pixels (and
	// between smaller copies of the picture far away).
	Sharp  bool
	Format TextureFormat // how the 3DS keeps it in its memory; other targets keep RGBA8
	// Gain multiplies the texture's colour (0 = 1): a picture darker on average than white, as a
	// pattern with dark joints is, would darken every surface that wears it. The built-in
	// textures set it so that, on average, they keep the material's brightness.
	Gain float32
}

// EffectiveGain is the texture's Gain, 1 when unset.
func (texture *Texture) EffectiveGain() float32 {
	if texture.Gain <= 0 {
		return 1
	}
	return texture.Gain
}

// Wrap is what a surface shows where its UVs leave the picture (below 0 or above 1).
type Wrap int

// The ways a picture wraps.
const (
	Repeat Wrap = iota // the picture again, as tiles
	Clamp              // the colour at the picture's edge, stretched
	Mirror             // the picture again, flipped every other time, so its edges always meet
)

// TextureFormat is how the 3DS keeps a texture's pixels: in full, in half the memory, or in an
// eighth. Other targets keep every texture in RGBA8.
type TextureFormat int

// The texture formats.
const (
	RGBA8  TextureFormat = iota // 32 bits a pixel: every colour and opacity
	RGB565                      // 16 bits a pixel: no opacity, colours in steps of 1/32 (green 1/64)
	ETC1                        // 4 bits a pixel, compressed in blocks of 4×4: no opacity, small loss in detail
)

// HasUVs tells whether the mesh carries its own UVs, two for each vertex.
func (mesh *Mesh) HasUVs() bool {
	return len(mesh.UVs) > 0 && len(mesh.UVs) == 2*mesh.VertexCount()
}

// SurfaceUVs returns the mesh's UVs, or, when it has none, UVs projected from where its vertices
// are, one unit of the model a repeat: x and z on what faces up or down, along the wall and up on
// what faces sideways (its facing taken from its normals; up when it has none). A mesh made in
// code, or a model without UVs, can wear a tiling texture this way.
func (mesh *Mesh) SurfaceUVs() []float32 {
	if mesh.HasUVs() {
		return mesh.UVs
	}
	uvs := make([]float32, 2*mesh.VertexCount())
	for vertex := range mesh.VertexCount() {
		uvs[2*vertex], uvs[2*vertex+1] = mesh.projectedUV(vertex)
	}
	return uvs
}

// projectedUV is a vertex's UV seen along the axis its normal points most towards.
func (mesh *Mesh) projectedUV(vertex int) (float32, float32) {
	x, y, z := mesh.Positions[3*vertex], mesh.Positions[3*vertex+1], mesh.Positions[3*vertex+2]
	if len(mesh.Normals) != len(mesh.Positions) {
		return x, z
	}
	normalX, normalY, normalZ := abs(mesh.Normals[3*vertex]), abs(mesh.Normals[3*vertex+1]), abs(mesh.Normals[3*vertex+2])
	switch {
	case normalY >= normalX && normalY >= normalZ:
		return x, z
	case normalX >= normalZ:
		return z, y
	}
	return x, y
}

func abs(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

// EffectiveTextureScale is how many times the texture repeats for each unit of UV, along each
// side, with 0 meaning 1.
func (material *Material) EffectiveTextureScale() [2]float32 {
	scale := material.TextureScale
	for side := range scale {
		if scale[side] == 0 {
			scale[side] = 1
		}
	}
	return scale
}
