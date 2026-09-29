package scene

import "math"

// LevelVariation raises and lowers a water surface with one harmonic that differs from place to
// place: at each point the offset is InPhase·cos(Angle) + InQuadrature·sin(Angle), which is
// Amplitude·cos(Angle − Phase) with InPhase = Amplitude·cos(Phase) and InQuadrature =
// Amplitude·sin(Phase). A simulation moves Angle (tides advance it a full turn per period); the
// grids stay the same. Outside the area the values of the nearest edge apply.
type LevelVariation struct {
	MinimumX, MinimumZ float32 // corner of the area (world units)
	MaximumX, MaximumZ float32 // opposite corner; grid values sit at cell centres
	Columns, Rows      int
	InPhase            []float32 // Rows×Columns, row by row from MinimumZ; heights in world units
	InQuadrature       []float32
	Angle              float32 // radians
	Dirty              bool    // the grids changed: backends upload them again
	GPU                any
}

// OffsetAt returns the height of the surface above the water level at a point.
func (variation *LevelVariation) OffsetAt(x, z float32) float32 {
	inPhase, inQuadrature := variation.sample(variation.InPhase, x, z), variation.sample(variation.InQuadrature, x, z)
	angle := float64(variation.Angle)
	return inPhase*float32(math.Cos(angle)) + inQuadrature*float32(math.Sin(angle))
}

// sample interpolates a grid bilinearly at a point, clamped to the area.
func (variation *LevelVariation) sample(grid []float32, x, z float32) float32 {
	if variation.Columns == 0 || variation.Rows == 0 || len(grid) < variation.Columns*variation.Rows {
		return 0
	}
	column := gridCoordinate(x, variation.MinimumX, variation.MaximumX, variation.Columns)
	row := gridCoordinate(z, variation.MinimumZ, variation.MaximumZ, variation.Rows)
	firstColumn, firstRow := int(column), int(row)
	nextColumn, nextRow := minimumInteger(firstColumn+1, variation.Columns-1), minimumInteger(firstRow+1, variation.Rows-1)
	alongColumns, alongRows := column-float32(firstColumn), row-float32(firstRow)
	at := func(row, column int) float32 { return grid[row*variation.Columns+column] }
	top := at(firstRow, firstColumn) + (at(firstRow, nextColumn)-at(firstRow, firstColumn))*alongColumns
	bottom := at(nextRow, firstColumn) + (at(nextRow, nextColumn)-at(nextRow, firstColumn))*alongColumns
	return top + (bottom-top)*alongRows
}

// gridCoordinate converts a position to a fractional cell index (cell centres at whole numbers).
func gridCoordinate(value, minimum, maximum float32, cells int) float32 {
	if maximum <= minimum {
		return 0
	}
	coordinate := (value-minimum)/(maximum-minimum)*float32(cells) - 0.5
	return float32(math.Max(0, math.Min(float64(cells-1), float64(coordinate))))
}

func minimumInteger(first, second int) int {
	if first < second {
		return first
	}
	return second
}
