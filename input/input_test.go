package input

import (
	"testing"
	"time"
)

// fakeClock is a clock tests move by hand.
type fakeClock struct{ now time.Time }

func (clock *fakeClock) read() time.Time { return clock.now }

func newStateWithClock() (*State, *fakeClock) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	state := NewState()
	state.Now = clock.read
	return state, clock
}

func click(state *State, button MouseButton) {
	state.Feed().SetButton(button, true)
	state.Feed().SetButton(button, false)
}

func TestButtonEdgesLastOneStep(test *testing.T) {
	state := NewState()
	state.Feed().SetButton(MouseLeft, true)
	if !state.ButtonPressed(MouseLeft) || state.ButtonReleased(MouseLeft) || !state.Button(MouseLeft) {
		test.Fatal("a press is seen as pressed and held")
	}
	state.Feed().EndFrame()
	if state.ButtonPressed(MouseLeft) || !state.Button(MouseLeft) {
		test.Fatal("the press lasts one step; the button stays held")
	}
	state.Feed().SetButton(MouseLeft, false)
	if !state.ButtonReleased(MouseLeft) || state.Button(MouseLeft) {
		test.Fatal("letting go is seen as released")
	}
}

func TestAQuickClickWithinOneStepIsStillAClick(test *testing.T) {
	state := NewState()
	click(state, MouseLeft)
	if !state.ButtonPressed(MouseLeft) || !state.ButtonReleased(MouseLeft) || state.Button(MouseLeft) {
		test.Fatal("a press and a release in the same step are both seen")
	}
}

func TestQuickClicksInOnePlaceCountAsDoubleAndTriple(test *testing.T) {
	state, clock := newStateWithClock()
	state.Feed().MoveCursor(100, 100)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 1 {
		test.Fatalf("first click, got %d", state.Clicks(MouseLeft))
	}
	state.Feed().EndFrame()
	clock.now = clock.now.Add(200 * time.Millisecond)
	state.Feed().MoveCursor(102, 101)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 2 {
		test.Fatalf("a second click soon after, close by, makes a double click; got %d", state.Clicks(MouseLeft))
	}
	state.Feed().EndFrame()
	clock.now = clock.now.Add(200 * time.Millisecond)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 3 {
		test.Fatalf("triple click, got %d", state.Clicks(MouseLeft))
	}
	state.Feed().EndFrame()
	if state.Clicks(MouseLeft) != 0 {
		test.Fatal("no press this step, no clicks")
	}
}

func TestSlowOrDistantClicksStartAgain(test *testing.T) {
	state, clock := newStateWithClock()
	click(state, MouseLeft)
	state.Feed().EndFrame()
	clock.now = clock.now.Add(time.Second)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 1 {
		test.Fatal("too slow for a double click")
	}
	state.Feed().EndFrame()
	clock.now = clock.now.Add(100 * time.Millisecond)
	state.Feed().MoveCursor(50, 0)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 1 {
		test.Fatal("too far for a double click")
	}
	state.Feed().EndFrame()
	clock.now = clock.now.Add(100 * time.Millisecond)
	click(state, MouseRight)
	if state.Clicks(MouseRight) != 1 {
		test.Fatal("another button starts a new count")
	}
}

func TestARepeatedKeyIsTypedAgainAndCharactersAreKept(test *testing.T) {
	state := NewState()
	state.Feed().SetKey(KeyBackspace, true)
	if !state.Typed(KeyBackspace) {
		test.Fatal("a press types the key")
	}
	state.Feed().EndFrame()
	if state.Typed(KeyBackspace) {
		test.Fatal("holding without a repeat does not type again")
	}
	state.Feed().RepeatKey(KeyBackspace)
	if !state.Typed(KeyBackspace) || state.Pressed(KeyBackspace) {
		test.Fatal("a system repeat types the key but is not a new press")
	}
	state.Feed().AddCharacter('ñ')
	state.Feed().AddCharacter('o')
	if state.TypedText() != "ño" {
		test.Fatalf("got %q", state.TypedText())
	}
	state.Feed().EndFrame()
	if state.TypedText() != "" {
		test.Fatal("characters last one step")
	}
}

func TestUnclaimedHidesWhatTheInterfaceUses(test *testing.T) {
	state := NewState()
	state.Feed().SetButton(MouseLeft, true)
	state.Feed().MoveCursor(0, 0)
	state.Feed().MoveCursor(10, 0)
	state.Feed().AddScroll(1)
	state.Feed().SetKey(KeyW, true)
	state.Feed().AddCharacter('w')

	free := state.Unclaimed()
	if !free.Button(MouseLeft) || free.CursorDeltaX != 10 || !free.Down(KeyW) || free.TypedText() != "w" {
		test.Fatal("nothing claimed: the copy is the same")
	}

	state.ClaimMouse()
	free = state.Unclaimed()
	if free.Button(MouseLeft) || free.CursorDeltaX != 0 || free.ScrollDelta != 0 || !free.Down(KeyW) {
		test.Fatal("a claimed mouse is hidden, the keyboard is not")
	}
	if free.CursorX != 10 {
		test.Fatal("the pointer position is still known")
	}

	state.ClaimKeyboard()
	free = state.Unclaimed()
	if free.Down(KeyW) || free.TypedText() != "" {
		test.Fatal("a claimed keyboard is hidden")
	}

	state.Feed().EndFrame()
	if state.MouseClaimed() || state.KeyboardClaimed() {
		test.Fatal("claims last one step")
	}
}

type memoryClipboard struct{ text string }

func (clipboard *memoryClipboard) ClipboardText() string        { return clipboard.text }
func (clipboard *memoryClipboard) SetClipboardText(text string) { clipboard.text = text }

func TestTheClipboardAndTheCursorShapeGoThroughTheState(test *testing.T) {
	state := NewState()
	state.SetClipboardText("lost")
	if state.ClipboardText() != "" {
		test.Fatal("without a clipboard, nothing is kept")
	}
	state.Clipboard = &memoryClipboard{}
	state.SetClipboardText("Faro")
	if state.ClipboardText() != "Faro" {
		test.Fatal("the clipboard keeps the text")
	}
	state.SetCursorShape(CursorText)
	state.Feed().EndFrame()
	if state.CursorShape() != CursorText {
		test.Fatal("the cursor shape stays until changed")
	}
}

func TestFrameRequestsKeepTheEarliest(test *testing.T) {
	state := NewState()
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	state.Now = func() time.Time { return now }
	if _, wanted := state.Feed().TakeFrameRequest(); wanted {
		test.Fatal("no request made")
	}
	state.RequestFrame(500 * time.Millisecond)
	state.RequestFrame(200 * time.Millisecond)
	state.RequestFrame(time.Second)
	now = now.Add(50 * time.Millisecond)
	if delay, wanted := state.Feed().TakeFrameRequest(); !wanted || delay != 150*time.Millisecond {
		test.Fatalf("got %v, %v", delay, wanted)
	}
	if _, wanted := state.Feed().TakeFrameRequest(); wanted {
		test.Fatal("a request is taken once")
	}
}

func TestAnyEventCountsAsInput(test *testing.T) {
	state := NewState()
	if state.Feed().HasInput() {
		test.Fatal("nothing happened yet")
	}
	state.Feed().SetKey(KeyA, true)
	if !state.Feed().HasInput() {
		test.Fatal("a key went down")
	}
	state.Feed().EndFrame()
	if state.Feed().HasInput() {
		test.Fatal("a key held from before is not new input")
	}
	state.Feed().MoveCursor(1, 1)
	state.Feed().MoveCursor(2, 1)
	if !state.Feed().HasInput() {
		test.Fatal("the pointer moved")
	}
}

func TestAnyHeldFollowsKeysAndButtons(test *testing.T) {
	state := NewState()
	if state.AnyHeld() {
		test.Fatal("nothing is held at first")
	}
	state.Feed().SetKey(KeyW, true)
	if !state.AnyHeld() {
		test.Error("a key held down counts")
	}
	state.Feed().SetKey(KeyW, false)
	state.Feed().SetButton(MouseLeft, true)
	if !state.AnyHeld() {
		test.Error("a button held down counts")
	}
	state.Feed().SetButton(MouseLeft, false)
	if state.AnyHeld() {
		test.Error("nothing is held once both are let go")
	}
}

func TestAFullscreenRequestStaysUntilChangedAndTravelsWithACopy(test *testing.T) {
	state := NewState()
	if state.Fullscreen() {
		test.Fatal("a new state asks for fullscreen")
	}
	state.SetFullscreen(true)
	state.Feed().EndFrame()
	copied := state.Unclaimed()
	if !state.Fullscreen() || !copied.Fullscreen() {
		test.Fatal("the request does not last the frame, or does not travel with a copy")
	}
	copied.SetFullscreen(false)
	if !state.Fullscreen() {
		test.Fatal("a copy changed the state it was made from")
	}
}

func TestFeedingTheStateAllocatesNothing(test *testing.T) {
	state := NewState()
	allocations := testing.AllocsPerRun(100, func() {
		state.Feed().SetKey(KeyA, true)
		state.Feed().MoveCursor(10, 20)
		state.Feed().SetStick(0.5, -0.5)
		state.Feed().EndFrame()
	})
	if allocations != 0 {
		test.Fatalf("feeding a frame allocated %v times", allocations)
	}
}
