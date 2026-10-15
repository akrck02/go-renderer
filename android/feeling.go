package android

// Feeling is a kind of buzz a touch gives, as Android's own views give them.
type Feeling int

const (
	// FeelingLongPress is the buzz of a touch that became a long press.
	FeelingLongPress Feeling = iota
	// FeelingTick is a light tick: something chosen, a switch flipped, a step on a dial.
	FeelingTick
	// FeelingConfirm says something went through. Before Android 11 it is a tick.
	FeelingConfirm
	// FeelingReject says something went wrong or needs a look. Before Android 11 it is the buzz of a
	// long press, the strongest the older phones have.
	FeelingReject
)

// Android's HapticFeedbackConstants, and the API levels that brought them.
const (
	longPressConstant    = 0
	clockTickConstant    = 4  // API 21
	contextClickConstant = 6  // API 23
	confirmConstant      = 16 // API 30
	rejectConstant       = 17 // API 30
)

// constantFor is the HapticFeedbackConstants value a feeling has on a phone of this API level.
func constantFor(feeling Feeling, level int) int {
	switch feeling {
	case FeelingTick:
		return clockTickConstant
	case FeelingConfirm:
		if level >= 30 {
			return confirmConstant
		}
		if level >= 23 {
			return contextClickConstant
		}
		return clockTickConstant
	case FeelingReject:
		if level >= 30 {
			return rejectConstant
		}
		return longPressConstant
	}
	return longPressConstant
}
