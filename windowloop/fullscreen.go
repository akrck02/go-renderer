package windowloop

import (
	"github.com/akrck02/go-renderer/input"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// Fullscreen puts a window on the whole screen when the application asks (input.State's
// SetFullscreen), and back in its place and at its size when it asks no more.
type Fullscreen struct {
	on bool
	// where the window was before it went on the whole screen, in screen coordinates
	x, y, width, height int
}

// Apply follows the application's request, when it changed.
func (fullscreen *Fullscreen) Apply(window *glfw.Window, state *input.State) {
	wanted := state.Fullscreen()
	if wanted == fullscreen.on {
		return
	}
	fullscreen.on = wanted
	if !wanted {
		window.SetMonitor(nil, fullscreen.x, fullscreen.y, fullscreen.width, fullscreen.height, glfw.DontCare)
		return
	}
	fullscreen.x, fullscreen.y = window.GetPos()
	fullscreen.width, fullscreen.height = window.GetSize()
	monitor := monitorUnder(window)
	if monitor == nil {
		fullscreen.on = false // no screen to fill (a headless display): it stays a window
		state.SetFullscreen(false)
		return
	}
	mode := monitor.GetVideoMode()
	window.SetMonitor(monitor, 0, 0, mode.Width, mode.Height, mode.RefreshRate)
}

// monitorUnder is the screen that holds most of a window, or the main one; nil without screens.
func monitorUnder(window *glfw.Window) *glfw.Monitor {
	x, y := window.GetPos()
	width, height := window.GetSize()
	var best *glfw.Monitor
	bestShared := 0
	for _, monitor := range glfw.GetMonitors() {
		left, top := monitor.GetPos()
		mode := monitor.GetVideoMode()
		if mode == nil {
			continue
		}
		shared := overlap(x, x+width, left, left+mode.Width) * overlap(y, y+height, top, top+mode.Height)
		if best == nil || shared > bestShared {
			best, bestShared = monitor, shared
		}
	}
	return best
}

// overlap is how long two spans on a line share.
func overlap(start, end, otherStart, otherEnd int) int {
	return max(0, min(end, otherEnd)-max(start, otherStart))
}
