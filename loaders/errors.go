package loaders

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors LoadScene and ParseSceneGLB return. Their details say which file, buffer or
// accessor; the error underneath, when there is one, is kept.
var (
	// FileNotRead is a scene, buffer or image file that could not be opened or read.
	FileNotRead = failure.Define("loaders.file-not-read", "a file of the scene could not be read")
	// InvalidGLTF is a file that is not glTF 2.0, or breaks its rules (a chunk cut short, an
	// accessor past its buffer, a sky cube without six faces).
	InvalidGLTF = failure.Define("loaders.invalid-gltf", "the file is not valid glTF")
	// UnsupportedAccessor is an accessor of a type or component type the loader does not read.
	UnsupportedAccessor = failure.Define("loaders.unsupported-accessor", "an accessor's type is not supported")
	// ImageNotDecoded is a sky image that is not a PNG or JPEG, or is damaged.
	ImageNotDecoded = failure.Define("loaders.image-not-decoded", "an image of the scene could not be decoded")
)
