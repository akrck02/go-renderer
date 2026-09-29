package scene

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
)

// groundTriangle is a walkable triangle in world space.
type groundTriangle [3]graphics.Vec4

// groundIndex is a uniform grid over the XZ plane holding the triangles of walkable meshes,
// so that height queries only test the few triangles near the point.
type groundIndex struct {
	originX, originZ float32
	cellSize         float32
	columns, rows    int
	triangles        []groundTriangle
	cells            [][]int32
}

// InvalidateGround drops the ground index; call it after changing walkable meshes.
func (scene *Scene) InvalidateGround() { scene.ground = nil }

// GroundHeight returns the highest walkable surface at (x, z), before vertical scaling.
func (scene *Scene) GroundHeight(x, z float32) (float32, bool) {
	if scene.ground == nil {
		scene.ground = buildGroundIndex(collectGroundTriangles(scene))
	}
	return scene.ground.heightAt(x, z)
}

func collectGroundTriangles(scene *Scene) []groundTriangle {
	var triangles []groundTriangle
	scene.Walk(func(node *Node, world graphics.Mat4) {
		if node.Mesh == nil || !node.Mesh.Ground || node.Instances != nil {
			return
		}
		mesh := node.Mesh
		mesh.ForEachTriangle(func(first, second, third uint32) {
			triangles = append(triangles, groundTriangle{
				world.Apply(mesh.Vertex(first)), world.Apply(mesh.Vertex(second)), world.Apply(mesh.Vertex(third)),
			})
		})
	})
	return triangles
}

func buildGroundIndex(triangles []groundTriangle) *groundIndex {
	index := &groundIndex{triangles: triangles}
	if len(triangles) == 0 {
		return index
	}
	minimum, maximum := horizontalExtent(triangles)
	index.chooseGridLayout(minimum, maximum, len(triangles))
	index.assignTrianglesToCells()
	return index
}

// horizontalExtent returns the minimum and maximum x (first value) and z (second value).
func horizontalExtent(triangles []groundTriangle) (minimum, maximum [2]float32) {
	infinity := float32(math.Inf(1))
	minimum, maximum = [2]float32{infinity, infinity}, [2]float32{-infinity, -infinity}
	for _, triangle := range triangles {
		for _, corner := range triangle {
			minimum[0] = float32(math.Min(float64(minimum[0]), float64(corner[0])))
			minimum[1] = float32(math.Min(float64(minimum[1]), float64(corner[2])))
			maximum[0] = float32(math.Max(float64(maximum[0]), float64(corner[0])))
			maximum[1] = float32(math.Max(float64(maximum[1]), float64(corner[2])))
		}
	}
	return minimum, maximum
}

// chooseGridLayout sizes the cells so that each holds about four triangles on average.
func (index *groundIndex) chooseGridLayout(minimum, maximum [2]float32, triangleCount int) {
	area := float64((maximum[0] - minimum[0]) * (maximum[1] - minimum[1]))
	index.cellSize = float32(math.Max(math.Sqrt(area*4/float64(triangleCount)), 1e-3))
	index.originX, index.originZ = minimum[0], minimum[1]
	index.columns = int((maximum[0]-minimum[0])/index.cellSize) + 1
	index.rows = int((maximum[1]-minimum[1])/index.cellSize) + 1
	index.cells = make([][]int32, index.columns*index.rows)
}

func (index *groundIndex) assignTrianglesToCells() {
	for triangleNumber, triangle := range index.triangles {
		minimum, maximum := horizontalExtent([]groundTriangle{triangle})
		firstColumn, firstRow := index.cellOf(minimum[0], minimum[1])
		lastColumn, lastRow := index.cellOf(maximum[0], maximum[1])
		for row := firstRow; row <= lastRow && row < index.rows; row++ {
			for column := firstColumn; column <= lastColumn && column < index.columns; column++ {
				cell := row*index.columns + column
				index.cells[cell] = append(index.cells[cell], int32(triangleNumber))
			}
		}
	}
}

func (index *groundIndex) cellOf(x, z float32) (column, row int) {
	return int((x - index.originX) / index.cellSize), int((z - index.originZ) / index.cellSize)
}

func (index *groundIndex) heightAt(x, z float32) (float32, bool) {
	if index.columns == 0 {
		return 0, false
	}
	column, row := index.cellOf(x, z)
	if column < 0 || row < 0 || column >= index.columns || row >= index.rows {
		return 0, false
	}
	highest, found := float32(math.Inf(-1)), false
	for _, triangleNumber := range index.cells[row*index.columns+column] {
		if height, inside := heightOnTriangle(index.triangles[triangleNumber], x, z); inside && height > highest {
			highest, found = height, true
		}
	}
	return highest, found
}

// heightOnTriangle interpolates the triangle height at (x, z) with barycentric weights.
func heightOnTriangle(triangle groundTriangle, x, z float32) (float32, bool) {
	first, second, third := triangle[0], triangle[1], triangle[2]
	denominator := (second[2]-third[2])*(first[0]-third[0]) + (third[0]-second[0])*(first[2]-third[2])
	if denominator == 0 {
		return 0, false
	}
	weightFirst := ((second[2]-third[2])*(x-third[0]) + (third[0]-second[0])*(z-third[2])) / denominator
	weightSecond := ((third[2]-first[2])*(x-third[0]) + (first[0]-third[0])*(z-third[2])) / denominator
	weightThird := 1 - weightFirst - weightSecond
	const tolerance = -1e-4
	if weightFirst < tolerance || weightSecond < tolerance || weightThird < tolerance {
		return 0, false
	}
	return weightFirst*first[1] + weightSecond*second[1] + weightThird*third[1], true
}
