package scene

import "math"

// The pieces the built-in textures are made of: hashes, noise and cells on lattices that wrap at
// the texture's edges.

// hashOf is a repeatable pseudo-random number from 0 to 1 for a cell of a lattice and a seed.
func hashOf(column, row, seed int) float64 {
	value := uint32(column)*0x8da6b343 ^ uint32(row)*0xd8163841 ^ uint32(seed)*0xcb1ab31f
	value ^= value >> 13
	value *= 0x5bd1e995
	value ^= value >> 15
	return float64(value&0xffffff) / 0x1000000
}

// wrap brings a cell's index back into 0 to count−1, so the lattice repeats.
func wrap(index, count int) int { return ((index % count) + count) % count }

// periodicNoise is smooth value noise over a lattice of columns × rows cells across the square,
// from 0 to 1.
func periodicNoise(u, v float64, columns, rows, seed int) float64 {
	x, y := u*float64(columns), v*float64(rows)
	column, row := int(math.Floor(x)), int(math.Floor(y))
	easedX, easedY := easedStep(0, 1, x-math.Floor(x)), easedStep(0, 1, y-math.Floor(y))
	corner := func(offsetColumn, offsetRow int) float64 {
		return hashOf(wrap(column+offsetColumn, columns), wrap(row+offsetRow, rows), seed)
	}
	return mix(mix(corner(0, 0), corner(1, 0), easedX), mix(corner(0, 1), corner(1, 1), easedX), easedY)
}

// cells is where a point lies among a lattice's cells: its distance to the nearest cell's centre
// and to the second nearest (in cells), and the nearest cell's tone (from 0 to 1).
type cells struct{ nearest, second, tone float64 }

// nearestCells finds a point's nearest cells on a lattice of columns × rows, each centre moved at
// random by up to jitter, odd rows shifted half a cell when offset (hexagons; rows must be even).
func nearestCells(u, v float64, columns, rows int, jitter float64, offset bool, seed int) cells {
	x, y := u*float64(columns), v*float64(rows)
	found := cells{nearest: 8, second: 8}
	for row := int(math.Floor(y)) - 1; row <= int(math.Floor(y))+1; row++ {
		shift := 0.0
		if offset && wrap(row, 2) == 1 {
			shift = 0.5
		}
		for column := int(math.Floor(x-shift)) - 1; column <= int(math.Floor(x-shift))+1; column++ {
			wrappedColumn, wrappedRow := wrap(column, columns), wrap(row, rows)
			centreX := float64(column) + shift + 0.5 + jitter*(hashOf(wrappedColumn, wrappedRow, seed)-0.5)
			centreY := float64(row) + 0.5 + jitter*(hashOf(wrappedColumn, wrappedRow, seed+1)-0.5)
			distance := math.Hypot(x-centreX, y-centreY)
			found = found.with(distance, hashOf(wrappedColumn, wrappedRow, seed+2))
		}
	}
	return found
}

// with counts one more cell at a distance with a tone.
func (found cells) with(distance, tone float64) cells {
	switch {
	case distance < found.nearest:
		return cells{nearest: distance, second: found.nearest, tone: tone}
	case distance < found.second:
		found.second = distance
	}
	return found
}

// lineDistance is how far a coordinate is from the nearest of count evenly spaced lines, the first
// at 0.
func lineDistance(coordinate float64, count int) float64 {
	spacing := 1 / float64(count)
	inside := math.Mod(coordinate, spacing)
	return min(inside, spacing-inside)
}

// edgeFade is 0 at both ends of a cell (0 and 1) and 1 once width away from them: a joint.
func edgeFade(inside, width float64) float64 {
	return easedStep(0, width, inside) * easedStep(1, 1-width, inside)
}

// easedStep goes smoothly from 0 at low to 1 at high.
func easedStep(low, high, value float64) float64 {
	share := clampToUnit((value - low) / (high - low))
	return share * share * (3 - 2*share)
}

func mix(from, to, share float64) float64 { return from + (to-from)*share }

func clampToUnit(value float64) float64 { return math.Max(0, math.Min(1, value)) }
