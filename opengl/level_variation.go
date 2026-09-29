package opengl

import (
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// levelVariationUnit is the texture unit of the water level variation.
const levelVariationUnit = 7

// levelVariationTexture holds the harmonic grids of a varying water surface on the GPU, and a
// placeholder for scenes without one (some drivers check every sampler on every draw).
type levelVariationTexture struct {
	placeholder uint32
}

func newLevelVariationTexture() *levelVariationTexture {
	return &levelVariationTexture{placeholder: uploadHarmonicTexture(1, 1, []float32{0, 0})}
}

// bind gives the scene program the variation of the water, or turns it off.
func (textures *levelVariationTexture) bind(program *shaderProgram, water *scene.Water) {
	program.setInteger("levelVariation", levelVariationUnit)
	gl.ActiveTexture(gl.TEXTURE0 + levelVariationUnit)
	defer gl.ActiveTexture(gl.TEXTURE0)
	if water == nil || water.Variation == nil {
		gl.BindTexture(gl.TEXTURE_2D, textures.placeholder)
		program.setInteger("levelVariationEnabled", 0)
		return
	}
	variation := water.Variation
	gl.BindTexture(gl.TEXTURE_2D, uploadVariation(variation))
	program.setInteger("levelVariationEnabled", 1)
	program.setVector4("levelVariationArea", [4]float32{variation.MinimumX, variation.MinimumZ, variation.MaximumX, variation.MaximumZ})
	program.setFloat("levelVariationAngle", variation.Angle)
}

// uploadVariation returns the texture of a variation, uploading it when new or changed.
func uploadVariation(variation *scene.LevelVariation) uint32 {
	texture, uploaded := variation.GPU.(uint32)
	if uploaded && !variation.Dirty {
		return texture
	}
	if uploaded {
		gl.DeleteTextures(1, &texture)
	}
	texture = uploadHarmonicTexture(variation.Columns, variation.Rows, interleaveHarmonics(variation))
	variation.GPU, variation.Dirty = texture, false
	return texture
}

// interleaveHarmonics packs the in-phase and in-quadrature grids as two channels.
func interleaveHarmonics(variation *scene.LevelVariation) []float32 {
	cells := variation.Columns * variation.Rows
	data := make([]float32, 2*cells)
	for cell := 0; cell < cells && cell < len(variation.InPhase) && cell < len(variation.InQuadrature); cell++ {
		data[2*cell], data[2*cell+1] = variation.InPhase[cell], variation.InQuadrature[cell]
	}
	return data
}

// uploadHarmonicTexture creates a two-channel float texture, linearly filtered and clamped.
func uploadHarmonicTexture(columns, rows int, data []float32) uint32 {
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RG32F, int32(columns), int32(rows), 0, gl.RG, gl.FLOAT, gl.Ptr(data))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	return texture
}
