package ctr

import "unsafe"

// A mesh is baked once for each way it is drawn (bake.go): each node's turn, material and light,
// and, under lamps, each place it stands. These are the bakes kept for a mesh, and how many.

// meshOnGPU is a mesh in linear memory, in pieces of 16-bit indices.
type meshOnGPU struct {
	pieces []pieceOnGPU
}

type pieceOnGPU struct {
	vertices, indices unsafe.Pointer
	count             int
}

// bakedVariant is one bake of a mesh, and the frame it was last drawn in.
type bakedVariant struct {
	key      bakeKey
	onGPU    *meshOnGPU
	lastUsed int
}

// maximumBakedVariants is how many ways of turning one mesh are kept baked. A mesh drawn by a
// few nodes turned differently (the same tree, rotated) keeps one for each; a node that turns all
// the time without being marked Moving would otherwise leave a new one behind every frame, and
// fill linear memory. Past this, the least recently drawn one is baked again for the new turn.
const maximumBakedVariants = 4

// maximumLampPlacements is how many places one mesh is kept baked in under lamps: there, every
// node that shares the mesh is lit differently, so a street of twenty crates keeps twenty bakes,
// baked once while nothing moves, instead of baking again and again in turns of four.
const maximumLampPlacements = 64

// placeFor is where a new bake of a mesh goes among its bakes: -1 to add it, else the index of
// the one it replaces, the least recently drawn of its own sort (under lamps or not) once that
// sort has as many as it keeps.
func placeFor(variants []bakedVariant, underLamps bool) int {
	count, oldest := 0, -1
	for index := range variants {
		variant := &variants[index]
		if (variant.key.lamps.count > 0) != underLamps {
			continue
		}
		count++
		if oldest < 0 || variant.lastUsed < variants[oldest].lastUsed {
			oldest = index
		}
	}
	limit := maximumBakedVariants
	if underLamps {
		limit = maximumLampPlacements
	}
	if count < limit {
		return -1
	}
	return oldest
}
