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

func TestButtonEdgesLastOneStep(t *testing.T) {
	state := NewState()
	state.SetButton(MouseLeft, true)
	if !state.ButtonPressed(MouseLeft) || state.ButtonReleased(MouseLeft) || !state.Button(MouseLeft) {
		t.Fatal("a press is seen as pressed and held")
	}
	state.EndFrame()
	if state.ButtonPressed(MouseLeft) || !state.Button(MouseLeft) {
		t.Fatal("the press lasts one step; the button stays held")
	}
	state.SetButton(MouseLeft, false)
	if !state.ButtonReleased(MouseLeft) || state.Button(MouseLeft) {
		t.Fatal("letting go is seen as released")
	}
}

func TestQuickClickWithinOneStep(t *testing.T) {
	state := NewState()
	click(state, MouseLeft)
	if !state.ButtonPressed(MouseLeft) || !state.ButtonReleased(MouseLeft) || state.Button(MouseLeft) {
		t.Fatal("a press and a release in the same step are both seen")
	}
}

func TestDoubleAndTripleClicks(t *testing.T) {
	state, clock := newStateWithClock()
	state.MoveCursor(100, 100)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 1 {
		t.Fatalf("first click, got %d", state.Clicks(MouseLeft))
	}
	state.EndFrame()
	clock.now = clock.now.Add(200 * time.Millisecond)
	state.MoveCursor(102, 101)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 2 {
		t.Fatalf("a second click soon after, close by, makes a double click; got %d", state.Clicks(MouseLeft))
	}
	state.EndFrame()
	clock.now = clock.now.Add(200 * time.Millisecond)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 3 {
		t.Fatalf("triple click, got %d", state.Clicks(MouseLeft))
	}
	state.EndFrame()
	if state.Clicks(MouseLeft) != 0 {
		t.Fatal("no press this step, no clicks")
	}
}

func TestSlowOrDistantClicksStartAgain(t *testing.T) {
	state, clock := newStateWithClock()
	click(state, MouseLeft)
	state.EndFrame()
	clock.now = clock.now.Add(time.Second)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 1 {
		t.Fatal("too slow for a double click")
	}
	state.EndFrame()
	clock.now = clock.now.Add(100 * time.Millisecond)
	state.MoveCursor(50, 0)
	click(state, MouseLeft)
	if state.Clicks(MouseLeft) != 1 {
		t.Fatal("too far for a double click")
	}
	state.EndFrame()
	clock.now = clock.now.Add(100 * time.Millisecond)
	click(state, MouseRight)
	if state.Clicks(MouseRight) != 1 {
		t.Fatal("another button starts a new count")
	}
}

func TestRepeatsAndCharacters(t *testing.T) {
	state := NewState()
	state.SetKey(KeyBackspace, true)
	if !state.Typed(KeyBackspace) {
		t.Fatal("a press types the key")
	}
	state.EndFrame()
	if state.Typed(KeyBackspace) {
		t.Fatal("holding without a repeat does not type again")
	}
	state.RepeatKey(KeyBackspace)
	if !state.Typed(KeyBackspace) || state.Pressed(KeyBackspace) {
		t.Fatal("a system repeat types the key but is not a new press")
	}
	state.AddCharacter('ñ')
	state.AddCharacter('o')
	if state.TypedText() != "ño" {
		t.Fatalf("got %q", state.TypedText())
	}
	state.EndFrame()
	if state.TypedText() != "" {
		t.Fatal("characters last one step")
	}
}

func TestUnclaimedHidesWhatTheInterfaceUses(t *testing.T) {
	state := NewState()
	state.SetButton(MouseLeft, true)
	state.MoveCursor(0, 0)
	state.MoveCursor(10, 0)
	state.AddScroll(1)
	state.SetKey(KeyW, true)
	state.AddCharacter('w')

	free := state.Unclaimed()
	if !free.Button(MouseLeft) || free.CursorDeltaX != 10 || !free.Down(KeyW) || free.TypedText() != "w" {
		t.Fatal("nothing claimed: the copy is the same")
	}

	state.ClaimMouse()
	free = state.Unclaimed()
	if free.Button(MouseLeft) || free.CursorDeltaX != 0 || free.ScrollDelta != 0 || !free.Down(KeyW) {
		t.Fatal("a claimed mouse is hidden, the keyboard is not")
	}
	if free.CursorX != 10 {
		t.Fatal("the pointer position is still known")
	}

	state.ClaimKeyboard()
	free = state.Unclaimed()
	if free.Down(KeyW) || free.TypedText() != "" {
		t.Fatal("a claimed keyboard is hidden")
	}

	state.EndFrame()
	if state.MouseClaimed() || state.KeyboardClaimed() {
		t.Fatal("claims last one step")
	}
}

type memoryClipboard struct{ text string }

func (clipboard *memoryClipboard) ClipboardText() string        { return clipboard.text }
func (clipboard *memoryClipboard) SetClipboardText(text string) { clipboard.text = text }

func TestClipboardAndCursorShape(t *testing.T) {
	state := NewState()
	state.SetClipboardText("lost")
	if state.ClipboardText() != "" {
		t.Fatal("without a clipboard, nothing is kept")
	}
	state.Clipboard = &memoryClipboard{}
	state.SetClipboardText("Faro")
	if state.ClipboardText() != "Faro" {
		t.Fatal("the clipboard keeps the text")
	}
	state.SetCursorShape(CursorText)
	state.EndFrame()
	if state.CursorShape() != CursorText {
		t.Fatal("the cursor shape stays until changed")
	}
}

func TestFrameRequestsKeepTheEarliest(t *testing.T) {
	state := NewState()
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	state.Now = func() time.Time { return now }
	if _, wanted := state.TakeFrameRequest(); wanted {
		t.Fatal("no request made")
	}
	state.RequestFrame(500 * time.Millisecond)
	state.RequestFrame(200 * time.Millisecond)
	state.RequestFrame(time.Second)
	now = now.Add(50 * time.Millisecond)
	if delay, wanted := state.TakeFrameRequest(); !wanted || delay != 150*time.Millisecond {
		t.Fatalf("got %v, %v", delay, wanted)
	}
	if _, wanted := state.TakeFrameRequest(); wanted {
		t.Fatal("a request is taken once")
	}
}

func TestHasInput(t *testing.T) {
	state := NewState()
	if state.HasInput() {
		t.Fatal("nothing happened yet")
	}
	state.SetKey(KeyA, true)
	if !state.HasInput() {
		t.Fatal("a key went down")
	}
	state.EndFrame()
	if state.HasInput() {
		t.Fatal("a key held from before is not new input")
	}
	state.MoveCursor(1, 1)
	state.MoveCursor(2, 1)
	if !state.HasInput() {
		t.Fatal("the pointer moved")
	}
}
