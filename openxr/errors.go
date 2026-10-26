package openxr

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors a session returns. Their details name the OpenXR call that failed and what
// the runtime said (as XR_ERROR_FORM_FACTOR_UNAVAILABLE).
var (
	// LoaderNotFound is a program without libopenxr_loader.so, or a loader without its entry point.
	LoaderNotFound = failure.Define("openxr.loader-not-found", "the OpenXR loader could not be opened")
	// ExtensionMissing is a runtime without an extension the session cannot do without.
	ExtensionMissing = failure.Define("openxr.extension-missing", "the OpenXR runtime lacks a needed extension")
	// InstanceNotCreated is a runtime that would not start for the program.
	InstanceNotCreated = failure.Define("openxr.instance-not-created", "the OpenXR runtime could not be started")
	// HeadsetNotFound is a runtime with no headset to show the session on.
	HeadsetNotFound = failure.Define("openxr.headset-not-found", "no headset is available")
	// GraphicsUnsupported is an OpenGL ES context older than the runtime needs.
	GraphicsUnsupported = failure.Define("openxr.graphics-unsupported", "the OpenGL ES context is older than the runtime needs")
	// SessionNotCreated is a runtime that would not make a session, its spaces or its swapchains.
	SessionNotCreated = failure.Define("openxr.session-not-created", "the OpenXR session could not be made")
	// FramebufferIncomplete is a swapchain image OpenGL ES cannot draw into.
	FramebufferIncomplete = failure.Define("openxr.framebuffer-incomplete", "a swapchain image could not be made a framebuffer")
	// ControlsNotCreated is a runtime that would not take the controllers' actions.
	ControlsNotCreated = failure.Define("openxr.controls-not-created", "the controllers' actions could not be made")
	// FrameFailed is a frame the runtime would not wait for, begin, locate or take.
	FrameFailed = failure.Define("openxr.frame-failed", "the frame could not be shown")
	// PanelNotCreated is a panel the runtime would not make a swapchain for, or one past the most
	// a session has (MostPanels), or of no size.
	PanelNotCreated = failure.Define("openxr.panel-not-created", "the panel could not be made")
	// PanelNotDrawable is a panel's picture asked for outside a frame that is drawn.
	PanelNotDrawable = failure.Define("openxr.panel-not-drawable", "the panel can be drawn only in a frame that is shown")
	// PassthroughNotStarted is a runtime that has the passthrough extension but would not show the
	// room: the app may lack its permission or feature in its manifest.
	PassthroughNotStarted = failure.Define("openxr.passthrough-not-started", "the room could not be shown behind the scene")
	// SessionLost is a session the runtime took away: the program should end.
	SessionLost = failure.Define("openxr.session-lost", "the OpenXR session was lost")
)

// callError is an OpenXR call that failed, with what the runtime said: the call's name is what a
// person searches the specification for.
func callError(code *failure.Code, call, result string) error {
	return failure.Wrap(code, call+": "+result, nil)
}
