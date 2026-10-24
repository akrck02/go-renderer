package frameloop

import "time"

// How long a Headless application waits for its picture to stop moving: it goes on drawing while
// the program asks for a frame within motionFrame (something eases into place), up to settleLimit.
// A frame asked for later is not motion (a caret's next blink, a note's end) and is not waited for.
const (
	settleLimit = 2 * time.Second
	motionFrame = 100 * time.Millisecond
)

// Settle draws frames while the program asks for the next one soon, so a capture shows the
// interface at rest rather than half way through a motion.
func Settle(nextRequest func() (time.Duration, bool), drawFrame func() error, sleep func(time.Duration)) error {
	waited := time.Duration(0)
	for waited < settleLimit {
		delay, requested := nextRequest()
		if !requested || delay > motionFrame {
			return nil
		}
		sleep(delay)
		waited += max(delay, time.Millisecond)
		if err := drawFrame(); err != nil {
			return err
		}
	}
	return nil
}
