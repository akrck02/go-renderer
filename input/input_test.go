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
	state.SetButton(button, true)
	state.SetButton(button, false)
}

func TestButtonEdgesLastOneStep(test *testing.T) {
	state := NewState()
	state.SetButton(MouseLeft, true)
	if !state.ButtonPressed(MouseLeft) || state.ButtonReleased(MouseLeft) || !state.Button(MouseLeft) {
		test.Fatal("a press is seen as pressed and held")
	}
	state.EndFrame()
	if state.ButtonPressed(MouseLeft) || !state.Button(MouseLeft) {
		test.Fatal("the press lasts one step; the button stays held")
	}
	state.SetButton(MouseLeft, false)
	if !state.ButtonReleased(MouseLeft) || state.Button(MouseLeft) {
		test.Fatal("letting go is seen as released")
	}
}

func TestQuickClickWithinOneStep(test *testing.T) {
	state := NewState()
	click(state, MouseLeft)
	if !state.ButtonPressed(MouseLeft) || !state.ButtonReleased(MouseLeft) || state.Button(MouseLeft) {
		test.Fatal("a press and a release in the same step are both seen")
	}
}

func TestDoubleAndTripleClicks(test *testing.T) {
	state, clock := newStateWithClock()
	state.MoveCursor(100, 100)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 1 {
		test.Fatalf("first click, got %d", state.Clicks(MouseLeft))
	}
	state.EndFrame()
	clock.now = clock.now.Add(200 * time.Millisecond)
	state.MoveCursor(102, 101)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 2 {
		test.Fatalf("a second click soon after, close by, makes a double click; got %d", state.Clicks(MouseLeft))
	}
	state.EndFrame()
	clock.now = clock.now.Add(200 * time.Millisecond)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 3 {
		test.Fatalf("triple click, got %d", state.Clicks(MouseLeft))
	}
	state.EndFrame()
	if state.Clicks(MouseLeft) != 0 {
		test.Fatal("no press this step, no clicks")
	}
}

func TestSlowOrDistantClicksStartAgain(test *testing.T) {
	state, clock := newStateWithClock()
	click(state, MouseLeft)
	state.EndFrame()
	clock.now = clock.now.Add(time.Second)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 1 {
		test.Fatal("too slow for a double click")
	}
	state.EndFrame()
	clock.now = clock.now.Add(100 * time.Millisecond)
	state.MoveCursor(50, 0)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 1 {
		test.Fatal("too far for a double click")
	}
	state.EndFrame()
	clock.now = clock.now.Add(100 * time.Millisecond)
	click(state, MouseRight)
	if state.Clicks(MouseRight) != 1 {
		test.Fatal("another button starts a new count")
	}
}

func TestRepeatsAndCharacters(test *testing.T) {
	state := NewState()
	state.SetKey(KeyBackspace, true)
	if !state.Typed(KeyBackspace) {
		test.Fatal("a press types the key")
	}
	state.EndFrame()
	if state.Typed(KeyBackspace) {
		test.Fatal("holding without a repeat does not type again")
	}
	state.RepeatKey(KeyBackspace)
	if !state.Typed(KeyBackspace) || state.Pressed(KeyBackspace) {
		test.Fatal("a system repeat types the key but is not a new press")
	}
	state.AddCharacter('ñ')
	state.AddCharacter('o')
	if state.TypedText() != "ño" {
		test.Fatalf("got %q", state.TypedText())
	}
	state.EndFrame()
	if state.TypedText() != "" {
		test.Fatal("characters last one step")
	}
}

func TestUnclaimedHidesWhatTheInterfaceUses(test *testing.T) {
	state := NewState()
	state.SetButton(MouseLeft, true)
	state.MoveCursor(0, 0)
	state.MoveCursor(10, 0)
	state.AddScroll(1)
	state.SetKey(KeyW, true)
	state.AddCharacter('w')

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

	state.EndFrame()
	if state.MouseClaimed() || state.KeyboardClaimed() {
		test.Fatal("claims last one step")
	}
}

type memoryClipboard struct{ text string }

func (clipboard *memoryClipboard) ClipboardText() string        { return clipboard.text }
func (clipboard *memoryClipboard) SetClipboardText(text string) { clipboard.text = text }

func TestClipboardAndCursorShape(test *testing.T) {
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
	state.EndFrame()
	if state.CursorShape() != CursorText {
		test.Fatal("the cursor shape stays until changed")
	}
}

func TestFrameRequestsKeepTheEarliest(test *testing.T) {
	state := NewState()
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	state.Now = func() time.Time { return now }
	if _, wanted := state.TakeFrameRequest(); wanted {
		test.Fatal("no request made")
	}
	state.RequestFrame(500 * time.Millisecond)
	state.RequestFrame(200 * time.Millisecond)
	state.RequestFrame(time.Second)
	now = now.Add(50 * time.Millisecond)
	if delay, wanted := state.TakeFrameRequest(); !wanted || delay != 150*time.Millisecond {
		test.Fatalf("got %v, %v", delay, wanted)
	}
	if _, wanted := state.TakeFrameRequest(); wanted {
		test.Fatal("a request is taken once")
	}
}

func TestHasInput(test *testing.T) {
	state := NewState()
	if state.HasInput() {
		test.Fatal("nothing happened yet")
	}
	state.SetKey(KeyA, true)
	if !state.HasInput() {
		test.Fatal("a key went down")
	}
	state.EndFrame()
	if state.HasInput() {
		test.Fatal("a key held from before is not new input")
	}
	state.MoveCursor(1, 1)
	state.MoveCursor(2, 1)
	if !state.HasInput() {
		test.Fatal("the pointer moved")
	}
}

func TestAnyHeldFollowsKeysAndButtons(test *testing.T) {
	state := NewState()
	if state.AnyHeld() {
		test.Fatal("nothing is held at first")
	}
	state.SetKey(KeyW, true)
	if !state.AnyHeld() {
		test.Error("a key held down counts")
	}
	state.SetKey(KeyW, false)
	state.SetButton(MouseLeft, true)
	if !state.AnyHeld() {
		test.Error("a button held down counts")
	}
	state.SetButton(MouseLeft, false)
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
	state.EndFrame()
	copied := state.Unclaimed()
	if !state.Fullscreen() || !copied.Fullscreen() {
		test.Fatal("the request does not last the frame, or does not travel with a copy")
	}
	copied.SetFullscreen(false)
	if !state.Fullscreen() {
		test.Fatal("a copy changed the state it was made from")
	}
}
