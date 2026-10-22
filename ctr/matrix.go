// Package ctr draws through the Nintendo 3DS's graphics chip, the PICA200, with citro3d. The
// drawing part is built only for the nintendo3ds target (TinyGo); the matrix conversions here are
// plain Go, so they are tested on any computer.
package ctr

import "github.com/akrck02/go-renderer/graphics"

// PICAMatrix lays out a matrix the way citro3d and the PICA200 read it: row by row, and inside
// each row the components in reverse order (w, z, y, x). graphics.Matrix is column by column.
func PICAMatrix(matrix graphics.Matrix) [16]float32 {
	var laidOut [16]float32
	for row := range 4 {
		for column := range 4 {
			laidOut[row*4+3-column] = matrix[column*4+row]
		}
	}
	return laidOut
}

// OrthographicTilt is citro3d's Mtx_OrthoTilt for a right-handed view: an orthographic projection
// with the depth range the PICA200 wants, [-1, 0], turned a quarter turn around the view axis,
// because the 3DS's screens are portrait panels mounted on their side.
func OrthographicTilt(left, right, bottom, top, near, far float32) graphics.Matrix {
	var matrix graphics.Matrix
	// Row 0 gives x from y, and row 1 gives y from x: the quarter turn.
	set(&matrix, 0, 1, 2/(top-bottom))
	set(&matrix, 0, 3, (bottom+top)/(bottom-top))
	set(&matrix, 1, 0, 2/(left-right))
	set(&matrix, 1, 3, (left+right)/(right-left))
	set(&matrix, 2, 2, 1/(near-far))
	set(&matrix, 2, 3, 0.5*(near+far)/(near-far)-0.5)
	set(&matrix, 3, 3, 1)
	return matrix
}

// set writes the element at a row and a column of a column-major matrix.
func set(matrix *graphics.Matrix, row, column int, value float32) {
	matrix[column*4+row] = value
}
