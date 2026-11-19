package scene

import "math"

// The brightness of each built-in texture at a point (u and v from 0 to 1). Each is built on
// lattices of whole cells across the square, wrapped at its edges, so it repeats without a seam.

func cobbles(u, v float64) float64 {
	cell := nearestCells(u, v, 5, 5, 0.7, false, 1)
	joint := easedStep(0.03, 0.12, cell.second-cell.nearest)
	return mix(0.55, 0.8+0.2*cell.tone, joint) * (1 - 0.1*cell.nearest)
}

func masonry(u, v float64) float64 {
	const courses, blocksPerCourse = 4, 2
	course := math.Floor(v * courses)
	along := u*blocksPerCourse + 0.5*math.Mod(course, 2)
	block := hashOf(wrap(int(math.Floor(along)), blocksPerCourse), wrap(int(course), courses), 2)
	joint := edgeFade(math.Mod(v*courses, 1), 0.06) * edgeFade(math.Mod(along, 1), 0.03)
	return mix(0.6, 0.82+0.15*block+0.05*periodicNoise(u, v, 16, 16, 3), joint)
}

func plaster(u, v float64) float64 {
	return 0.86 + 0.08*periodicNoise(u, v, 4, 4, 4) + 0.06*periodicNoise(u, v, 16, 16, 5)
}

func shingles(u, v float64) float64 {
	const rows, perRow = 6, 4
	row := math.Floor(v * rows)
	along := u*perRow + 0.5*math.Mod(row, 2)
	tone := 0.82 + 0.18*hashOf(wrap(int(math.Floor(along)), perRow), wrap(int(row), rows), 6)
	overlap := 0.72 + 0.28*math.Mod(v*rows, 1)
	return tone * overlap * mix(0.7, 1, edgeFade(math.Mod(along, 1), 0.05))
}

func planks(u, v float64) float64 {
	const boards = 4
	board := math.Floor(v * boards)
	gap := edgeFade(math.Mod(v*boards, 1), 0.05)
	grain := 0.92 + 0.08*periodicNoise(u, v, 2, 32, 7)
	return mix(0.6, (0.82+0.16*hashOf(wrap(int(board), boards), 0, 8))*grain, gap)
}

func grass(u, v float64) float64 {
	return 0.78 + 0.12*periodicNoise(u, v, 4, 4, 9) + 0.06*periodicNoise(u, v, 16, 16, 10) + 0.04*periodicNoise(u, v, 32, 32, 11)
}

func panels(u, v float64) float64 {
	const perSide = 2
	insideU, insideV := math.Mod(u*perSide, 1), math.Mod(v*perSide, 1)
	seam := edgeFade(insideU, 0.02) * edgeFade(insideV, 0.02)
	bevel := 1 - 0.08*(1-edgeFade(insideU, 0.08)*edgeFade(insideV, 0.08))
	rivet := 1.0
	for _, corner := range [][2]float64{{0.07, 0.07}, {0.93, 0.07}, {0.07, 0.93}, {0.93, 0.93}} {
		if math.Hypot(insideU-corner[0], insideV-corner[1]) < 0.025 {
			rivet = 0.7
		}
	}
	return mix(0.45, 0.9*bevel*rivet, seam)
}

func grid(u, v float64) float64 {
	const lines = 4
	near := min(lineDistance(u, lines), lineDistance(v, lines))
	return mix(0.55, 0.95, easedStep(0.012, 0.02, near))
}

func stripes(u, v float64) float64 {
	const stripesAcross = 4
	along := math.Mod((u+v)*stripesAcross, 1)
	return mix(0.55, 0.95, easedStep(0.45, 0.5, along)*edgeFade(along, 0.05))
}

func carbon(u, v float64) float64 {
	const bundles = 8
	column, row := int(u*bundles), int(v*bundles)
	insideU, insideV := math.Mod(u*bundles, 1), math.Mod(v*bundles, 1)
	across := insideV
	if (column+row)%4 >= 2 {
		across = insideU
	}
	return 0.5 + 0.4*math.Sin(math.Pi*across)
}

func hexagons(u, v float64) float64 {
	cell := nearestCells(u, v, 4, 4, 0, true, 12)
	edge := easedStep(0.02, 0.06, cell.second-cell.nearest)
	return mix(0.5, 0.85+0.1*cell.tone, edge)
}

func worn(u, v float64) float64 {
	blotch := easedStep(0.55, 0.7, periodicNoise(u, v, 4, 4, 13))
	scratch := easedStep(0.8, 0.9, periodicNoise(u, v, 2, 32, 14))
	return 0.92 - 0.2*blotch - 0.15*scratch + 0.04*periodicNoise(u, v, 32, 32, 15)
}

func camouflage(u, v float64) float64 {
	shape := 0.65*periodicNoise(u, v, 3, 3, 16) + 0.35*periodicNoise(u, v, 6, 6, 17)
	tones := []float64{0.45, 0.6, 0.78, 0.95}
	return tones[min(int(shape*float64(len(tones))), len(tones)-1)]
}

func windows(u, v float64) float64 {
	const perSide = 2
	insideU, insideV := math.Mod(u*perSide, 1), math.Mod(v*perSide, 1)
	inWindow := math.Abs(insideU-0.5) < 0.3 && math.Abs(insideV-0.5) < 0.35
	inFrame := math.Abs(insideU-0.5) < 0.34 && math.Abs(insideV-0.5) < 0.39
	switch {
	case inWindow:
		return 0.25 + 0.2*insideV // glass, catching more sky at its top
	case inFrame:
		return 0.6
	}
	return 0.9
}
