# loaders

```go
import "github.com/akrck02/go-renderer/loaders"
```

### Functions

| Function | |
|---|---|
| `func LoadGLB(path string) (*models.Model, error)` |  |
| `func LoadScene(path string) (*scene.Scene, error)` | LoadScene reads a glTF 2.0 file (.glb, or .gltf with external buffers) into a scene.Scene. |
| `func ParseSceneGLB(content []byte) (*scene.Scene, error)` | ParseSceneGLB reads a scene from the bytes of a .glb file, as LoadScene does from its path: for programs with no file system (a web page) or that carry their scene inside them. |

---

[← Reference](README.md) · generated from the doc comments by `generate.py`
