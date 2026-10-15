package sceneplan

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// LineRange is the part of the constellation buffer that holds one constellation's lines, with
// its colour.
type LineRange struct {
	First, Count int32
	Color        graphics.Vec4
}

// ConstellationVertices lays out every star first (drawn as points) and then the line segments of
// each constellation, three floats (a direction) per vertex.
func ConstellationVertices(constellations []scene.Constellation) (vertices []float32, pointCount int32, lineRanges []LineRange) {
	for _, constellation := range constellations {
		for _, star := range constellation.Stars {
			vertices = append(vertices, star[0], star[1], star[2])
			pointCount++
		}
	}
	next := pointCount
	for _, constellation := range constellations {
		first := next
		for _, line := range constellation.Lines {
			if !validLine(line, len(constellation.Stars)) {
				continue
			}
			for _, end := range line {
				star := constellation.Stars[end]
				vertices = append(vertices, star[0], star[1], star[2])
				next++
			}
		}
		lineRanges = append(lineRanges, LineRange{First: first, Count: next - first, Color: ColorOrWhite(constellation.Color)})
	}
	return vertices, pointCount, lineRanges
}

func validLine(line [2]int, starCount int) bool {
	return line[0] >= 0 && line[1] >= 0 && line[0] < starCount && line[1] < starCount
}

// ConstellationVisibility is how much the constellations show: fully at night, faintly by day as
// much as the stars do.
func ConstellationVisibility(sky *scene.Sky, light scene.Daylight) float32 {
	darkness := float32(math.Pow(float64(1-light.DayAmount), 4))
	return float32(math.Min(1, float64(darkness+sky.Stars.DaytimeVisibility*(1-darkness))))
}

// LineColor is the colour a constellation's lines are drawn with: its own, more transparent.
func LineColor(color graphics.Vec4) graphics.Vec4 {
	return graphics.Vec4{color[0], color[1], color[2], color[3] * 0.55}
}
