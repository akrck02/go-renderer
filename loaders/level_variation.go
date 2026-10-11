package loaders

import (
	"fmt"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// applyLevelVariation reads a varying water surface (tides, for example) from the water extras:
//
//	"variation": {"minimum": [x, z], "maximum": [x, z], "columns": c, "rows": r,
//	              "inPhase": accessor, "inQuadrature": accessor, "angle": radians}
//
// The accessors hold rows×columns heights (row by row from the minimum z); the offset at a point
// is inPhase·cos(angle) + inQuadrature·sin(angle).
func (file *gltfFile) applyLevelVariation(environment *scene.Environment, extras map[string]any) error {
	water, _ := extras["water"].(map[string]any)
	definition, found := water["variation"].(map[string]any)
	if !found || environment.Water == nil {
		return nil
	}
	minimum := extraVector(definition, "minimum", graphics.Vec4{})
	maximum := extraVector(definition, "maximum", graphics.Vec4{})
	variation := &scene.LevelVariation{
		MinimumX: minimum[0], MinimumZ: minimum[1], MaximumX: maximum[0], MaximumZ: maximum[1],
		Columns: int(extraNumber(definition, "columns", 0)), Rows: int(extraNumber(definition, "rows", 0)),
		Angle: extraNumber(definition, "angle", 0),
	}
	var err error
	if variation.InPhase, err = file.readGrid(definition, "inPhase", variation); err != nil {
		return err
	}
	if variation.InQuadrature, err = file.readGrid(definition, "inQuadrature", variation); err != nil {
		return err
	}
	environment.Water.Variation = variation
	return nil
}

// readGrid reads the accessor named by a key and checks it has one value per cell.
func (file *gltfFile) readGrid(definition map[string]any, key string, variation *scene.LevelVariation) ([]float32, error) {
	accessor, isNumber := definition[key].(float64)
	if !isNumber {
		return nil, fmt.Errorf("water variation: %s needs an accessor", key)
	}
	values, _, err := file.readAccessor(int(accessor))
	if err != nil {
		return nil, fmt.Errorf("water variation %s: %w", key, err)
	}
	if len(values) != variation.Columns*variation.Rows {
		return nil, fmt.Errorf("water variation %s: %d values for %d×%d cells", key, len(values), variation.Columns, variation.Rows)
	}
	return values, nil
}
