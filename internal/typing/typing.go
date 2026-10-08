// Package typing is the typing-test engine. It tracks what the user has typed
// against a target text and records every keystroke with its timing.
//
// It has no UI or storage dependencies: callers feed it runes and timestamps,
// and read back state and a Result. That keeps the rules of a test in one
// place and makes them trivially testable.
package typing

import (
	"time"

	"github.com/ngopalji/typist/internal/content"
)

// Kind distinguishes the keystrokes the engine records.
type Kind uint8

const (
	KindChar      Kind = iota // a printable character was entered
	KindBackspace             // one character was deleted
)

// Keystroke is a single recorded key press. Keystrokes are the raw material
// for every statistic, so they carry everything needed to reconstruct a test.
type Keystroke struct {
	At       time.Duration // since the test's first keystroke
	Pos      int           // index into the target the key applied to
	Kind     Kind
	Expected rune // target rune at Pos; zero for backspaces
	Typed    rune // rune entered; zero for backspaces
}

// Correct reports whether k entered the expected character.
func (k Keystroke) Correct() bool { return k.Kind == KindChar && k.Typed == k.Expected }

// Wrong reports whether k entered something other than the expected character.
func (k Keystroke) Wrong() bool { return k.Kind == KindChar && k.Typed != k.Expected }

// Result is a finished (or in-progress) test: everything needed to analyze or
// persist it.
type Result struct {
	Mode       content.Mode
	Limit      time.Duration // the test's configured length
	StartedAt  time.Time     // wall-clock time of the first keystroke
	Duration   time.Duration // time from the first keystroke to the end
	Target     string        // the portion of the text that was reached
	Typed      string        // the final input, after corrections
	Keystrokes []Keystroke
}

// CharState is how one position of the target should be displayed.
type CharState uint8

const (
	Pending CharState = iota
	Correct
	Incorrect
)

// Test is one timed typing test. The clock starts on the first keystroke and
// the test finishes once Limit has elapsed.
type Test struct {
	mode    content.Mode
	limit   time.Duration
	target  []rune
	typed   []rune
	strokes []Keystroke
	start   time.Time
	done    bool
}

// New returns an empty test; add text with Append before typing.
func New(mode content.Mode, limit time.Duration) *Test {
	return &Test{mode: mode, limit: limit}
}

// Append adds a token to the end of the target, separated by a space.
func (t *Test) Append(token string) {
	if len(t.target) > 0 {
		t.target = append(t.target, ' ')
	}
	t.target = append(t.target, []rune(token)...)
}

// Type enters r at the cursor. Keys pressed after the time limit are dropped
// and finish the test.
func (t *Test) Type(r rune, now time.Time) {
	if !t.accept(now) || len(t.typed) >= len(t.target) {
		return
	}
	pos := len(t.typed)
	t.typed = append(t.typed, r)
	t.record(now, Keystroke{Pos: pos, Kind: KindChar, Expected: t.target[pos], Typed: r})
}

// Backspace deletes the character before the cursor.
func (t *Test) Backspace(now time.Time) {
	if len(t.typed) == 0 || !t.accept(now) {
		return
	}
	t.typed = t.typed[:len(t.typed)-1]
	t.record(now, Keystroke{Pos: len(t.typed), Kind: KindBackspace})
}

// DeleteWord deletes back to the start of the current word, like ctrl+w in a
// shell. Each deleted character is recorded as its own backspace.
func (t *Test) DeleteWord(now time.Time) {
	n := len(t.typed)
	for n > 0 && t.typed[n-1] == ' ' {
		n--
	}
	for n > 0 && t.typed[n-1] != ' ' {
		n--
	}
	for len(t.typed) > n && !t.done {
		t.Backspace(now)
	}
}

// Tick finishes the test if its time is up. Call it periodically while the
// test is running.
func (t *Test) Tick(now time.Time) {
	if t.Started() && now.Sub(t.start) >= t.limit {
		t.done = true
	}
}

// accept starts the clock on the first keystroke and reports whether a key at
// now still counts.
func (t *Test) accept(now time.Time) bool {
	if t.done {
		return false
	}
	if !t.Started() {
		t.start = now
		return true
	}
	t.Tick(now)
	return !t.done
}

func (t *Test) record(now time.Time, k Keystroke) {
	k.At = now.Sub(t.start)
	t.strokes = append(t.strokes, k)
}

// Started reports whether the first key has been pressed.
func (t *Test) Started() bool { return !t.start.IsZero() }

// Done reports whether the time limit has been reached.
func (t *Test) Done() bool { return t.done }

// Mode returns the kind of text being typed.
func (t *Test) Mode() content.Mode { return t.mode }

// Limit returns the configured length of the test.
func (t *Test) Limit() time.Duration { return t.limit }

// Elapsed returns how long the test has been running, capped at its limit.
func (t *Test) Elapsed(now time.Time) time.Duration {
	if !t.Started() {
		return 0
	}
	return min(now.Sub(t.start), t.limit)
}

// Remaining returns how much time is left.
func (t *Test) Remaining(now time.Time) time.Duration {
	return t.limit - t.Elapsed(now)
}

// Cursor returns the index of the next character to type.
func (t *Test) Cursor() int { return len(t.typed) }

// Len returns the number of runes of target text buffered.
func (t *Test) Len() int { return len(t.target) }

// Buffered returns how many runes of target text remain ahead of the cursor.
func (t *Test) Buffered() int { return len(t.target) - len(t.typed) }

// Rune returns the target rune at i.
func (t *Test) Rune(i int) rune { return t.target[i] }

// State returns how position i of the target should be displayed.
func (t *Test) State(i int) CharState {
	switch {
	case i >= len(t.typed):
		return Pending
	case t.typed[i] == t.target[i]:
		return Correct
	default:
		return Incorrect
	}
}

// Keystrokes returns every keystroke recorded so far. Callers must not
// modify the returned slice.
func (t *Test) Keystrokes() []Keystroke { return t.strokes }

// Snapshot returns the test's state as a Result, measured at now.
func (t *Test) Snapshot(now time.Time) Result {
	return Result{
		Mode:       t.mode,
		Limit:      t.limit,
		StartedAt:  t.start,
		Duration:   t.Elapsed(now),
		Target:     string(t.target[:len(t.typed)]),
		Typed:      string(t.typed),
		Keystrokes: append([]Keystroke(nil), t.strokes...),
	}
}
