package loaders

import (
	"math"
	"sort"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// readOverlay turns the overlay extras of the nodes into the scene's overlay: a node with
// "label", "icon" or "badge" gets a label above it (tinted by "label_color"), and one with
// "pinned" becomes a line of the header instead, in the order of the numbers.
func readOverlay(loaded *scene.Scene) {
	var pinned []pinnedLine
	loaded.Walk(func(node *scene.Node, world graphics.Matrix) {
		label, found := labelFromTags(node.Tags)
		if !found {
			return
		}
		if line, isPinned := pinnedFromTags(node.Tags, label); isPinned {
			pinned = append(pinned, line)
			return
		}
		label.Node = node
		label.Height = node.Mesh.Top() * verticalScaleOf(world)
		loaded.Overlay.AddLabel(label)
	})
	sort.SliceStable(pinned, func(first, second int) bool { return pinned[first].order < pinned[second].order })
	for _, line := range pinned {
		loaded.Overlay.Header = append(loaded.Overlay.Header, line.line)
	}
}

// labelFromTags reads a label from extras; false when the node has none.
func labelFromTags(tags map[string]any) (*scene.Label, bool) {
	label := &scene.Label{}
	label.Text, _ = tags["label"].(string)
	label.Icon, _ = tags["icon"].(string)
	label.Badge, _ = tags["badge"].(string)
	if color, isText := tags["label_color"].(string); isText {
		if tint, err := draw2d.ParseHex(color); err == nil { // a colour that is not one leaves the label untinted
			label.Tint = graphics.Vector(tint)
		}
	}
	return label, !label.IsEmpty()
}

type pinnedLine struct {
	order float64
	line  scene.HeaderLine
}

func pinnedFromTags(tags map[string]any, label *scene.Label) (pinnedLine, bool) {
	order, isNumber := tags["pinned"].(float64)
	if !isNumber {
		return pinnedLine{}, false
	}
	return pinnedLine{order: order, line: scene.HeaderLine{Icon: label.Icon, Text: label.Text, Tint: label.Tint}}, true
}

// verticalScaleOf returns how much a transform stretches heights, to turn a mesh's top into world
// units.
func verticalScaleOf(transform graphics.Matrix) float32 {
	return float32(math.Sqrt(float64(transform[4]*transform[4] + transform[5]*transform[5] + transform[6]*transform[6])))
}
