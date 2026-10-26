//go:build android

// The controllers: one action set, its actions bound to the Touch controllers and, for any other
// controller, to the simple controller every runtime knows, and a space for each hand's grip and aim.

#include <string.h>

#include "bridge.h"

static XrResult path(xr_program *program, const char *text, XrPath *result) {
	return program->call.xrStringToPath(program->instance, text, result);
}

// create_action makes an action; one per hand reads each hand on its own (the left trigger and the
// right one are one action, asked of either hand).
static XrResult create_action(xr_program *program, XrAction *action, const char *name, const char *shown,
	XrActionType type, int per_hand) {
	XrActionCreateInfo info = {XR_TYPE_ACTION_CREATE_INFO};
	strncpy(info.actionName, name, XR_MAX_ACTION_NAME_SIZE - 1);
	strncpy(info.localizedActionName, shown, XR_MAX_LOCALIZED_ACTION_NAME_SIZE - 1);
	info.actionType = type;
	if (per_hand) {
		info.countSubactionPaths = XR_HANDS;
		info.subactionPaths = program->controls.hands;
	}
	return program->call.xrCreateAction(program->controls.set, &info, action);
}

static XrResult create_actions(xr_program *program) {
	xr_controls *controls = &program->controls;
	struct {
		XrAction *action;
		const char *name, *shown;
		XrActionType type;
		int per_hand;
	} actions[] = {
		{&controls->grip, "grip", "Grip pose", XR_ACTION_TYPE_POSE_INPUT, 1},
		{&controls->aim, "aim", "Aim pose", XR_ACTION_TYPE_POSE_INPUT, 1},
		{&controls->trigger, "trigger", "Trigger", XR_ACTION_TYPE_FLOAT_INPUT, 1},
		{&controls->squeeze, "squeeze", "Squeeze", XR_ACTION_TYPE_FLOAT_INPUT, 1},
		{&controls->stick, "stick", "Thumbstick", XR_ACTION_TYPE_VECTOR2F_INPUT, 1},
		{&controls->stick_click, "stick_click", "Thumbstick click", XR_ACTION_TYPE_BOOLEAN_INPUT, 1},
		{&controls->upper, "upper_button", "B and Y", XR_ACTION_TYPE_BOOLEAN_INPUT, 1},
		{&controls->lower, "lower_button", "A and X", XR_ACTION_TYPE_BOOLEAN_INPUT, 1},
		{&controls->menu, "menu", "Menu", XR_ACTION_TYPE_BOOLEAN_INPUT, 0},
		{&controls->vibration, "vibration", "Vibration", XR_ACTION_TYPE_VIBRATION_OUTPUT, 1},
	};
	for (size_t index = 0; index < sizeof actions / sizeof actions[0]; index++) {
		XrResult result = create_action(program, actions[index].action, actions[index].name, actions[index].shown,
			actions[index].type, actions[index].per_hand);
		if (XR_FAILED(result)) return result;
	}
	return XR_SUCCESS;
}

typedef struct {
	XrAction action;
	const char *path;
} binding;

#define XR_MOST_BINDINGS 32

// suggest tells the runtime which of a controller's inputs each action reads.
static XrResult suggest(xr_program *program, const char *profile, const binding *bindings, int count) {
	XrActionSuggestedBinding suggested[XR_MOST_BINDINGS];
	for (int index = 0; index < count; index++) {
		suggested[index].action = bindings[index].action;
		XrResult result = path(program, bindings[index].path, &suggested[index].binding);
		if (XR_FAILED(result)) return result;
	}
	XrInteractionProfileSuggestedBinding info = {XR_TYPE_INTERACTION_PROFILE_SUGGESTED_BINDING};
	XrResult result = path(program, profile, &info.interactionProfile);
	if (XR_FAILED(result)) return result;
	info.countSuggestedBindings = count;
	info.suggestedBindings = suggested;
	return program->call.xrSuggestInteractionProfileBindings(program->instance, &info);
}

#define XR_COUNT(list) ((int)(sizeof list / sizeof list[0]))

static XrResult suggest_touch(xr_program *program) {
	xr_controls *controls = &program->controls;
	const binding bindings[] = {
		{controls->grip, "/user/hand/left/input/grip/pose"},
		{controls->grip, "/user/hand/right/input/grip/pose"},
		{controls->aim, "/user/hand/left/input/aim/pose"},
		{controls->aim, "/user/hand/right/input/aim/pose"},
		{controls->trigger, "/user/hand/left/input/trigger/value"},
		{controls->trigger, "/user/hand/right/input/trigger/value"},
		{controls->squeeze, "/user/hand/left/input/squeeze/value"},
		{controls->squeeze, "/user/hand/right/input/squeeze/value"},
		{controls->stick, "/user/hand/left/input/thumbstick"},
		{controls->stick, "/user/hand/right/input/thumbstick"},
		{controls->stick_click, "/user/hand/left/input/thumbstick/click"},
		{controls->stick_click, "/user/hand/right/input/thumbstick/click"},
		{controls->upper, "/user/hand/left/input/y/click"},
		{controls->upper, "/user/hand/right/input/b/click"},
		{controls->lower, "/user/hand/left/input/x/click"},
		{controls->lower, "/user/hand/right/input/a/click"},
		{controls->menu, "/user/hand/left/input/menu/click"},
		{controls->vibration, "/user/hand/left/output/haptic"},
		{controls->vibration, "/user/hand/right/output/haptic"},
	};
	return suggest(program, "/interaction_profiles/oculus/touch_controller", bindings, XR_COUNT(bindings));
}

// suggest_simple binds what the simple controller has: poses, select as the trigger, and menu.
static XrResult suggest_simple(xr_program *program) {
	xr_controls *controls = &program->controls;
	const binding bindings[] = {
		{controls->grip, "/user/hand/left/input/grip/pose"},
		{controls->grip, "/user/hand/right/input/grip/pose"},
		{controls->aim, "/user/hand/left/input/aim/pose"},
		{controls->aim, "/user/hand/right/input/aim/pose"},
		{controls->trigger, "/user/hand/left/input/select/click"},
		{controls->trigger, "/user/hand/right/input/select/click"},
		{controls->menu, "/user/hand/left/input/menu/click"},
		{controls->menu, "/user/hand/right/input/menu/click"},
		{controls->vibration, "/user/hand/left/output/haptic"},
		{controls->vibration, "/user/hand/right/output/haptic"},
	};
	return suggest(program, "/interaction_profiles/khr/simple_controller", bindings, XR_COUNT(bindings));
}

static XrResult create_hand_space(xr_program *program, XrAction action, XrPath hand, XrSpace *space) {
	XrActionSpaceCreateInfo info = {XR_TYPE_ACTION_SPACE_CREATE_INFO};
	info.action = action;
	info.subactionPath = hand;
	info.poseInActionSpace.orientation.w = 1;
	return program->call.xrCreateActionSpace(program->session, &info, space);
}

static XrResult create_hand_spaces(xr_program *program) {
	xr_controls *controls = &program->controls;
	for (int hand = 0; hand < XR_HANDS; hand++) {
		XrResult result = create_hand_space(program, controls->grip, controls->hands[hand], &controls->grip_spaces[hand]);
		if (XR_FAILED(result)) return result;
		result = create_hand_space(program, controls->aim, controls->hands[hand], &controls->aim_spaces[hand]);
		if (XR_FAILED(result)) return result;
	}
	return XR_SUCCESS;
}

static XrResult attach(xr_program *program) {
	XrSessionActionSetsAttachInfo info = {XR_TYPE_SESSION_ACTION_SETS_ATTACH_INFO};
	info.countActionSets = 1;
	info.actionSets = &program->controls.set;
	return program->call.xrAttachSessionActionSets(program->session, &info);
}

// xr_create_controls makes the action set and its actions, binds them, and attaches them to the
// session. The simple controller's bindings are a fallback: a runtime without that profile is not
// an error.
XrResult xr_create_controls(xr_program *program) {
	xr_controls *controls = &program->controls;
	XrResult result = path(program, "/user/hand/left", &controls->hands[0]);
	if (XR_SUCCEEDED(result)) result = path(program, "/user/hand/right", &controls->hands[1]);
	if (XR_FAILED(result)) return result;
	XrActionSetCreateInfo info = {XR_TYPE_ACTION_SET_CREATE_INFO};
	strncpy(info.actionSetName, "controls", XR_MAX_ACTION_SET_NAME_SIZE - 1);
	strncpy(info.localizedActionSetName, "Controls", XR_MAX_LOCALIZED_ACTION_SET_NAME_SIZE - 1);
	result = program->call.xrCreateActionSet(program->instance, &info, &controls->set);
	if (XR_FAILED(result)) return result;
	result = create_actions(program);
	if (XR_FAILED(result)) return result;
	result = suggest_touch(program);
	if (XR_FAILED(result)) return result;
	suggest_simple(program);
	result = create_hand_spaces(program);
	if (XR_FAILED(result)) return result;
	result = attach(program);
	controls->attached = XR_SUCCEEDED(result);
	return result;
}

static float read_float(xr_program *program, XrAction action, XrPath hand) {
	XrActionStateGetInfo info = {XR_TYPE_ACTION_STATE_GET_INFO, NULL, action, hand};
	XrActionStateFloat state = {XR_TYPE_ACTION_STATE_FLOAT};
	if (XR_FAILED(program->call.xrGetActionStateFloat(program->session, &info, &state)) || !state.isActive) return 0;
	return state.currentState;
}

static XrVector2f read_vector(xr_program *program, XrAction action, XrPath hand) {
	XrActionStateGetInfo info = {XR_TYPE_ACTION_STATE_GET_INFO, NULL, action, hand};
	XrActionStateVector2f state = {XR_TYPE_ACTION_STATE_VECTOR2F};
	if (XR_FAILED(program->call.xrGetActionStateVector2f(program->session, &info, &state)) || !state.isActive) {
		return (XrVector2f){0, 0};
	}
	return state.currentState;
}

// read_boolean is a button held; with went_down, a button that went down since the last sync.
static int read_boolean(xr_program *program, XrAction action, XrPath hand, int went_down) {
	XrActionStateGetInfo info = {XR_TYPE_ACTION_STATE_GET_INFO, NULL, action, hand};
	XrActionStateBoolean state = {XR_TYPE_ACTION_STATE_BOOLEAN};
	if (XR_FAILED(program->call.xrGetActionStateBoolean(program->session, &info, &state)) || !state.isActive) return 0;
	if (went_down) return state.currentState && state.changedSinceLastSync;
	return state.currentState;
}

// locate_hand finds a hand's pose in the world space; it returns whether the runtime knows both
// where it is and which way it points.
static int locate_hand(xr_program *program, XrSpace space, int64_t time, XrPosef *pose) {
	XrSpaceLocation location = {XR_TYPE_SPACE_LOCATION};
	*pose = (XrPosef){{0, 0, 0, 1}, {0, 0, 0}};
	if (XR_FAILED(program->call.xrLocateSpace(space, program->world, time, &location))) return 0;
	const XrSpaceLocationFlags valid = XR_SPACE_LOCATION_ORIENTATION_VALID_BIT | XR_SPACE_LOCATION_POSITION_VALID_BIT;
	if ((location.locationFlags & valid) != valid) return 0;
	*pose = location.pose;
	return 1;
}

static void read_hand(xr_program *program, int index, int64_t time, xr_hand *hand) {
	xr_controls *controls = &program->controls;
	XrPath path = controls->hands[index];
	hand->tracked = locate_hand(program, controls->grip_spaces[index], time, &hand->grip);
	locate_hand(program, controls->aim_spaces[index], time, &hand->aim);
	hand->trigger = read_float(program, controls->trigger, path);
	hand->squeeze = read_float(program, controls->squeeze, path);
	XrVector2f stick = read_vector(program, controls->stick, path);
	hand->stick_x = stick.x;
	hand->stick_y = stick.y;
	hand->stick_click = read_boolean(program, controls->stick_click, path, 0);
	hand->upper = read_boolean(program, controls->upper, path, 0);
	hand->lower = read_boolean(program, controls->lower, path, 0);
}

// xr_read_controls brings the actions up to date and reads both hands at the frame's display
// time; menu is the menu button going down since the last read.
XrResult xr_read_controls(xr_program *program, int64_t time, xr_hand *hands, int *menu) {
	XrActiveActionSet active = {program->controls.set, XR_NULL_PATH};
	XrActionsSyncInfo sync = {XR_TYPE_ACTIONS_SYNC_INFO};
	sync.countActiveActionSets = 1;
	sync.activeActionSets = &active;
	XrResult result = program->call.xrSyncActions(program->session, &sync);
	if (XR_FAILED(result)) return result;
	for (int index = 0; index < XR_HANDS; index++) read_hand(program, index, time, &hands[index]);
	*menu = read_boolean(program, program->controls.menu, XR_NULL_PATH, 1);
	return result;
}

// xr_vibrate shakes a hand's controller at an amplitude from 0 to 1 for a time, at the frequency
// the runtime finds best. A vibration replaces the one the hand still has.
XrResult xr_vibrate(xr_program *program, int hand, float amplitude, int64_t nanoseconds) {
	if (hand < 0 || hand >= XR_HANDS || !program->controls.attached) return XR_ERROR_HANDLE_INVALID;
	XrHapticActionInfo info = {XR_TYPE_HAPTIC_ACTION_INFO};
	info.action = program->controls.vibration;
	info.subactionPath = program->controls.hands[hand];
	XrHapticVibration vibration = {XR_TYPE_HAPTIC_VIBRATION};
	vibration.amplitude = amplitude;
	vibration.duration = nanoseconds;
	vibration.frequency = XR_FREQUENCY_UNSPECIFIED;
	return program->call.xrApplyHapticFeedback(program->session, &info, (const XrHapticBaseHeader *)&vibration);
}

void xr_destroy_controls(xr_program *program) {
	xr_controls *controls = &program->controls;
	for (int hand = 0; hand < XR_HANDS; hand++) {
		if (controls->grip_spaces[hand] != XR_NULL_HANDLE) program->call.xrDestroySpace(controls->grip_spaces[hand]);
		if (controls->aim_spaces[hand] != XR_NULL_HANDLE) program->call.xrDestroySpace(controls->aim_spaces[hand]);
	}
	if (controls->set != XR_NULL_HANDLE) program->call.xrDestroyActionSet(controls->set); // its actions too
	*controls = (xr_controls){0};
}
