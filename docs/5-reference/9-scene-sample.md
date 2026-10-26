# scene/sample

```go
import "github.com/akrck02/go-renderer/scene/sample"
```

Package sample builds a small made-up island that uses every feature the
renderers draw: relief with snow, the sea with a tide, a river, instanced trees
with levels of detail that sway, a house with stone, shingle and plank textures,
a paved square, a glowing lamp, a waterfall, an unlit flag, a glass box,
the sky with clouds, moon and a constellation, and a label. It needs no files,
so tests and backend comparisons can draw the same scene anywhere.

### Functions

| Function | |
|---|---|
| `func Island() *scene.Scene` | Island returns the scene, with the sun in the afternoon. |
| `func Night() *scene.Scene` | Night returns the island an hour before midnight, under the moon and the stars. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
