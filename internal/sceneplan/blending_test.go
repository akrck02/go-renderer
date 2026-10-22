package sceneplan

import (
	"testing"

	"github.com/akrck02/go-renderer/scene"
)

func TestAnAdditiveSurfaceIsDrawnWithTheTransparentOnes(test *testing.T) {
	additive := &scene.Material{Additive: true}
	command := Command{Material: additive, Transparent: IsTransparent(additive)}
	if !command.Transparent || BlendingOf(command) != BlendAdditive {
		test.Fatalf("an additive surface is transparent %v and blends as %v", command.Transparent, BlendingOf(command))
	}
	translucent := &scene.Material{Transparent: true}
	if blending := BlendingOf(Command{Material: translucent, Transparent: true}); blending != BlendAlpha {
		test.Fatalf("a translucent surface blends as %v", blending)
	}
	if blending := BlendingOf(Command{Material: &scene.Material{}}); blending != BlendNone {
		test.Fatalf("an opaque surface blends as %v", blending)
	}
}
