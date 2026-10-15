package scene

import (
	"slices"
	"strconv"
	"strings"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
)

// Overlay is what is drawn over the 3D view, in screen space: labels that float above things in
// the world, a header pinned to the top-left corner and short notices under it, and a free
// drawing on top of everything (Drawing). The overlay itself handles no input: an interface library
// fills Drawing with its panels and buttons and reads the mouse and keyboard on its own.
//
// A program changes it from the viewer's update function, which runs on the same goroutine as the
// drawing, so no locking is needed. The zero value is an empty overlay ready to use.
type Overlay struct {
	Labels  []*Label
	Header  []HeaderLine
	Notices []Notice
	// Drawing is drawn last, over the labels, header and notices: flat shapes, text and icons in
	// screen space, filled every frame by a program or an interface library. nil = nothing.
	Drawing *draw2d.List
	// Clock counts the seconds the overlay has been shown; the viewer advances it at every step.
	// Notices remember the clock reading when they were added, so they fade on the viewer's time.
	Clock float64
	// NoticeDuration is how long a notice stays, in seconds, fading at the end; 0 = DefaultNoticeDuration.
	NoticeDuration float64
	// MaximumNotices is how many notices are shown at once, the newest; 0 = DefaultMaximumNotices.
	MaximumNotices int
}

// DefaultNoticeDuration is how long a notice stays when the overlay does not say.
const DefaultNoticeDuration = 6.0

// DefaultMaximumNotices is how many notices are shown at once when the overlay does not say.
const DefaultMaximumNotices = 5

// noticeFadeSeconds is the last stretch of a notice's life, over which it fades out.
const noticeFadeSeconds = 1.0

// Label is a small tag drawn above something in the world: an icon, a text and an optional badge
// under them. It keeps the same size on screen however far the camera is.
type Label struct {
	// Node anchors the label to a node: it follows the node's world transform and hides with it.
	// When nil, the label stands at Point.
	Node  *Node
	Point graphics.Vec4 // world position of the anchor when there is no Node
	// Height lifts the label above its anchor, in world units (the top of a box, for example).
	Height float32
	Text   string
	Icon   string // name of a Lucide icon (https://lucide.dev), drawn before the text or alone
	Badge  string // a short text in a small pill under the label
	// Tint colours the icon and the badge, as a screen colour (see ColorFromHex); alpha 0 = the
	// default text colour. The text itself stays dark on a light pill, so it always reads well.
	Tint   graphics.Vec4
	Hidden bool
}

// IsEmpty tells whether the label has nothing to draw.
func (label *Label) IsEmpty() bool {
	return label.Text == "" && label.Icon == "" && label.Badge == ""
}

// HeaderLine is one line of the header in the top-left corner of the screen.
type HeaderLine struct {
	Icon string
	Text string
	Tint graphics.Vec4 // colour of the icon (alpha 0 = the default text colour)
}

// Notice is a short message shown under the header for a few seconds, newest first.
type Notice struct {
	Icon  string
	Text  string
	Tint  graphics.Vec4 // colour of the icon (alpha 0 = the default text colour)
	Added float64       // the overlay's Clock when the notice was added
}

// AddLabel shows a label; the caller keeps the pointer to change or remove it later.
func (overlay *Overlay) AddLabel(label *Label) {
	overlay.Labels = append(overlay.Labels, label)
}

// RemoveLabel stops showing a label.
func (overlay *Overlay) RemoveLabel(label *Label) {
	overlay.Labels = slices.DeleteFunc(overlay.Labels, func(candidate *Label) bool { return candidate == label })
}

// AddNotice shows a message now; it fades after the notice duration.
func (overlay *Overlay) AddNotice(icon, text string, tint graphics.Vec4) {
	overlay.Notices = append(overlay.Notices, Notice{Icon: icon, Text: text, Tint: tint, Added: overlay.Clock})
}

// Advance moves the overlay's clock on and forgets the notices that have faded out.
func (overlay *Overlay) Advance(seconds float64) {
	overlay.Clock += seconds
	duration := overlay.noticeDuration()
	overlay.Notices = slices.DeleteFunc(overlay.Notices, func(notice Notice) bool {
		return overlay.Clock-notice.Added >= duration
	})
}

// VisibleNotice is a notice as it should be drawn now.
type VisibleNotice struct {
	Notice
	Opacity float32 // 1 while fresh, falling to 0 as the notice ends
}

// VisibleNotices returns the notices to draw now, newest first, at most MaximumNotices.
func (overlay *Overlay) VisibleNotices() []VisibleNotice {
	duration := overlay.noticeDuration()
	var visible []VisibleNotice
	for index := len(overlay.Notices) - 1; index >= 0 && len(visible) < overlay.maximumNotices(); index-- {
		notice := overlay.Notices[index]
		opacity := noticeOpacity(overlay.Clock-notice.Added, duration)
		if opacity <= 0 {
			continue
		}
		visible = append(visible, VisibleNotice{Notice: notice, Opacity: opacity})
	}
	return visible
}

// noticeOpacity is 1 for most of a notice's life and falls linearly to 0 over its last second.
func noticeOpacity(age, duration float64) float32 {
	if age < 0 {
		return 1
	}
	remaining := duration - age
	if remaining <= 0 {
		return 0
	}
	fade := min(noticeFadeSeconds, duration)
	if remaining >= fade {
		return 1
	}
	return float32(remaining / fade)
}

func (overlay *Overlay) noticeDuration() float64 {
	if overlay.NoticeDuration > 0 {
		return overlay.NoticeDuration
	}
	return DefaultNoticeDuration
}

func (overlay *Overlay) maximumNotices() int {
	if overlay.MaximumNotices > 0 {
		return overlay.MaximumNotices
	}
	return DefaultMaximumNotices
}

// IsEmpty tells whether there is nothing to draw on the overlay.
func (overlay *Overlay) IsEmpty() bool {
	return len(overlay.Labels) == 0 && len(overlay.Header) == 0 && len(overlay.Notices) == 0 && overlay.Drawing.IsEmpty()
}

// WorldTransforms returns the world transform of every visible node, for anchoring labels. Nodes
// that are hidden (or under a hidden node) or no longer in the scene are missing.
func (scene *Scene) WorldTransforms(wanted map[*Node]bool) map[*Node]graphics.Mat4 {
	transforms := make(map[*Node]graphics.Mat4, len(wanted))
	if len(wanted) == 0 {
		return transforms
	}
	scene.Walk(func(node *Node, world graphics.Mat4) {
		if wanted[node] {
			transforms[node] = world
		}
	})
	return transforms
}

// Top returns the highest y of the mesh's vertices (0 for an empty mesh), to lift a label above it.
func (mesh *Mesh) Top() float32 {
	if mesh == nil || mesh.VertexCount() == 0 {
		return 0
	}
	top := mesh.Positions[1]
	for index := 4; index < len(mesh.Positions); index += 3 {
		top = max(top, mesh.Positions[index])
	}
	return top
}

// ColorFromHex reads a colour written as on a web page, "#rrggbb" (or "#rgb"), into the 0–1 range
// with alpha 1. Overlay colours are screen colours, used as they are.
func ColorFromHex(text string) (graphics.Vec4, bool) {
	digits := strings.TrimPrefix(strings.TrimSpace(text), "#")
	if len(digits) == 3 {
		digits = string([]byte{digits[0], digits[0], digits[1], digits[1], digits[2], digits[2]})
	}
	if len(digits) != 6 {
		return graphics.Vec4{}, false
	}
	value, err := strconv.ParseUint(digits, 16, 32)
	if err != nil {
		return graphics.Vec4{}, false
	}
	channel := func(shift uint) float32 { return float32((value>>shift)&0xff) / 255 }
	return graphics.Vec4{channel(16), channel(8), channel(0), 1}, true
}
