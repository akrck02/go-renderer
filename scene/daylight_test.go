package scene

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

func TestDaylightFollowsTheSun(test *testing.T) {
	environment := DefaultEnvironment()
	environment.SunDirection = graphics.Vector{0, 1, 0.2, 0}.Normalize()
	day := environment.Daylight()
	environment.SunDirection = graphics.Vector{0, -1, 0.2, 0}.Normalize()
	night := environment.Daylight()
	if day.DayAmount < 0.99 || night.DayAmount > 0.01 {
		test.Fatalf("day amount: day %v night %v", day.DayAmount, night.DayAmount)
	}
	if night.LightDirection[1] <= 0 {
		test.Fatalf("at night the moon (above the horizon) should light the scene, got %v", night.LightDirection)
	}
	if night.Zenith[2] >= day.Zenith[2] {
		test.Fatalf("the night sky should be darker than the day sky")
	}
}

func TestStarsTurnWithTheSun(test *testing.T) {
	pole := graphics.Vector{0, 1, 0, 0}
	morning := starRotation(graphics.Vector{1, 0, 0, 0}, pole)
	evening := starRotation(graphics.Vector{-1, 0, 0, 0}, pole)
	star := graphics.Vector{0, 0, -1, 0}
	if morning.Apply(star) == evening.Apply(star) {
		test.Fatalf("the celestial direction of a star should change with the sun")
	}
}
