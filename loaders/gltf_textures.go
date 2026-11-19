package loaders

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/scene"
)

// The parts of a glTF document that lay a picture over a material: the material names a texture,
// the texture a picture (an image) and how to read it (a sampler).

type gltfTextureUsage struct {
	Index int `json:"index"`
}

type gltfTextureDefinition struct {
	Sampler *int `json:"sampler"`
	Source  *int `json:"source"`
}

type gltfImageDefinition struct {
	URI        string `json:"uri"`
	BufferView *int   `json:"bufferView"`
}

type gltfSamplerDefinition struct {
	MagFilter int `json:"magFilter"`
	WrapS     int `json:"wrapS"`
}

// The specification's numbers for a sampler's settings.
const (
	nearestFilter  = 9728
	clampToEdge    = 33071
	mirroredRepeat = 33648
)

// dataURIBase64Start is where a data URI's base64 bytes begin, after its media type.
const dataURIBase64Start = ";base64,"

// applyBaseColorTexture gives a material the texture its base colour names, if any.
func (file *gltfFile) applyBaseColorTexture(material *scene.Material, usage *gltfTextureUsage) error {
	if usage == nil {
		return nil
	}
	texture, err := file.texture(usage.Index)
	material.Texture = texture
	return err
}

// texture makes the document's texture of a number, once: every material that names it shares it.
func (file *gltfFile) texture(number int) (*scene.Texture, error) {
	if texture, made := file.textures[number]; made {
		return texture, nil
	}
	if number < 0 || number >= len(file.document.Textures) {
		return nil, failure.Wrap(InvalidGLTF, "texture "+strconv.Itoa(number)+" does not exist", nil)
	}
	definition := file.document.Textures[number]
	if definition.Source == nil || *definition.Source < 0 || *definition.Source >= len(file.document.Images) {
		return nil, failure.Wrap(InvalidGLTF, "texture "+strconv.Itoa(number)+" has no picture", nil)
	}
	picture, err := file.picture(file.document.Images[*definition.Source])
	if err != nil {
		return nil, err
	}
	texture := &scene.Texture{Picture: picture}
	file.applySampler(texture, definition.Sampler)
	if file.textures == nil {
		file.textures = map[int]*scene.Texture{}
	}
	file.textures[number] = texture
	return texture, nil
}

// applySampler reads how a texture wraps (along u: scene.Texture has one wrap) and whether it is
// sharp (magnified to the nearest pixel). Without a sampler a texture repeats, smooth.
func (file *gltfFile) applySampler(texture *scene.Texture, number *int) {
	if number == nil || *number < 0 || *number >= len(file.document.Samplers) {
		return
	}
	sampler := file.document.Samplers[*number]
	texture.Sharp = sampler.MagFilter == nearestFilter
	switch sampler.WrapS {
	case clampToEdge:
		texture.Wrap = scene.Clamp
	case mirroredRepeat:
		texture.Wrap = scene.Mirror
	}
}

// picture reads an image's PNG or JPEG: from a buffer view of the file, a data URI, or a file
// beside the scene's.
func (file *gltfFile) picture(definition gltfImageDefinition) (*draw2d.Image, error) {
	content, name, err := file.imageContent(definition)
	if err != nil {
		return nil, err
	}
	picture, err := draw2d.DecodeImage(bytes.NewReader(content))
	if err != nil {
		return nil, failure.Wrap(ImageNotDecoded, "texture picture "+name, err)
	}
	return picture, nil
}

// imageContent is an image's encoded bytes, and a name for it in errors.
func (file *gltfFile) imageContent(definition gltfImageDefinition) ([]byte, string, error) {
	if definition.BufferView != nil {
		content, err := file.bufferViewBytes(*definition.BufferView)
		return content, "in buffer view " + strconv.Itoa(*definition.BufferView), err
	}
	if start := strings.Index(definition.URI, dataURIBase64Start); strings.HasPrefix(definition.URI, "data:") && start >= 0 {
		content, err := base64.StdEncoding.DecodeString(definition.URI[start+len(dataURIBase64Start):])
		if err != nil {
			return nil, "in a data URI", failure.Wrap(InvalidGLTF, "a texture's data URI", err)
		}
		return content, "in a data URI", nil
	}
	if file.directory == "" {
		return nil, definition.URI, failure.Wrap(InvalidGLTF, "the .glb names a picture beside it ("+definition.URI+"), which needs its folder: use LoadScene", nil)
	}
	content, err := os.ReadFile(filepath.Join(file.directory, definition.URI))
	if err != nil {
		return nil, definition.URI, failure.Wrap(FileNotRead, "texture picture "+definition.URI, err)
	}
	return content, definition.URI, nil
}

// bufferViewBytes are the bytes of a buffer view.
func (file *gltfFile) bufferViewBytes(number int) ([]byte, error) {
	if number < 0 || number >= len(file.document.BufferViews) {
		return nil, failure.Wrap(InvalidGLTF, "buffer view "+strconv.Itoa(number)+" does not exist", nil)
	}
	view := file.document.BufferViews[number]
	if view.Buffer < 0 || view.Buffer >= len(file.buffers) || view.ByteOffset+view.ByteLength > len(file.buffers[view.Buffer]) {
		return nil, failure.Wrap(InvalidGLTF, "buffer view "+strconv.Itoa(number)+" lies past its buffer", nil)
	}
	return file.buffers[view.Buffer][view.ByteOffset : view.ByteOffset+view.ByteLength], nil
}
