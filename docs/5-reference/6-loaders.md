# loaders

```go
import "github.com/akrck02/go-renderer/loaders"
```

### Functions

| Function | |
|---|---|
| `func LoadModel(path string) (*immediate.Model, error)` | LoadModel reads the meshes of a glTF 2.0 file (.glb, or .gltf with external buffers) into an immediate.Model, for the immediate-mode Renderer: each triangle primitive's positions and indices, in any of glTF's component types. |
| `func LoadScene(path string) (*scene.Scene, error)` | LoadScene reads a glTF 2.0 file (.glb, or .gltf with external buffers) into a scene.Scene. |
| `func ReadScene(reader io.Reader) (*scene.Scene, error)` | ReadScene reads a scene from a .glb file's content, as LoadScene does from its path: for programs with no file system (a web page) or that carry their scene inside them. |

### Constants and variables

```go
var (
	// FileNotRead is a scene, buffer or image file that could not be opened or read.
	FileNotRead = failure.Define("loaders.file-not-read", "a file of the scene could not be read")
	// InvalidGLTF is a file that is not glTF 2.0, or breaks its rules (a chunk cut short, an
	// accessor past its buffer, a sky cube without six faces).
	InvalidGLTF = failure.Define("loaders.invalid-gltf", "the file is not valid glTF")
	// UnsupportedAccessor is an accessor of a type or component type the loader does not read.
	UnsupportedAccessor = failure.Define("loaders.unsupported-accessor", "an accessor's type is not supported")
	// ImageNotDecoded is a sky image or a texture's picture that is not a PNG or JPEG, or is damaged.
	ImageNotDecoded = failure.Define("loaders.image-not-decoded", "an image of the scene could not be decoded")
)
```

Codes of the errors LoadScene, ReadScene and LoadModel return.

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
