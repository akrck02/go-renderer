//go:build android

// The room behind the scene: XR_FB_passthrough's passthrough and the layer that shows it. Meta's
// runtime composes the cameras' view of the room under the program's layers; where the eyes' image
// is transparent, the room shows. (Its runtime has no ALPHA_BLEND environment blend mode: this
// layer is how a Quest mixes the room in.)

#include "bridge.h"

// xr_start_passthrough makes the passthrough and its layer, both running from the start: the room
// shows from the next frame. Call it after the session is made, and only when has_passthrough.
XrResult xr_start_passthrough(xr_program *program) {
	XrPassthroughCreateInfoFB passthrough = {XR_TYPE_PASSTHROUGH_CREATE_INFO_FB};
	passthrough.flags = XR_PASSTHROUGH_IS_RUNNING_AT_CREATION_BIT_FB;
	XrResult result = program->call.xrCreatePassthroughFB(program->session, &passthrough, &program->passthrough);
	if (XR_FAILED(result)) return result;
	XrPassthroughLayerCreateInfoFB layer = {XR_TYPE_PASSTHROUGH_LAYER_CREATE_INFO_FB};
	layer.passthrough = program->passthrough;
	layer.flags = XR_PASSTHROUGH_IS_RUNNING_AT_CREATION_BIT_FB;
	layer.purpose = XR_PASSTHROUGH_LAYER_PURPOSE_RECONSTRUCTION_FB;
	result = program->call.xrCreatePassthroughLayerFB(program->session, &layer, &program->passthrough_layer);
	if (XR_FAILED(result)) return result;
	program->passthrough_running = 1;
	return XR_SUCCESS;
}

// xr_stop_passthrough lets go of the layer and the passthrough, if they were made.
void xr_stop_passthrough(xr_program *program) {
	program->passthrough_running = 0;
	if (program->passthrough_layer != XR_NULL_HANDLE) {
		program->call.xrDestroyPassthroughLayerFB(program->passthrough_layer);
		program->passthrough_layer = XR_NULL_HANDLE;
	}
	if (program->passthrough != XR_NULL_HANDLE) {
		program->call.xrDestroyPassthroughFB(program->passthrough);
		program->passthrough = XR_NULL_HANDLE;
	}
}
