// Package stats turns recorded keystrokes into analytics. Everything here is
// a pure function of typing.Result values, so the same code powers the live
// readout while typing, the results screen, and all-time history.
package stats

import (
	"math"
	"time"

	"github.com/nihaar/typist/internal/typing"
)

// charsPerWord is the standard "word" used by typing tests: five keystrokes.
const charsPerWord = 5

// Summary holds the headline numbers for one test.
type Summary struct {
	Duration    time.Duration
	WPM         float64 // net speed: correct characters in the final input
	RawWPM      float64 // every character keystroke, right or wrong
	Accuracy    float64 // share of character keystrokes that were right, 0–1
	Consistency float64 // how steady the pace was second to second, 0–1
	PeakWPM     float64 // best rolling-window speed during the test
	Correct     int     // correct characters in the final input
	Keystrokes  int     // character keystrokes, excluding backspaces
	Errors      int     // character keystrokes that were wrong
	Corrections int     // backspaces
}

// Summarize computes the headline numbers for r.
func Summarize(r typing.Result) Summary {
	s := Summary{Duration: r.Duration}
	for _, k := range r.Keystrokes {
		switch {
		case k.Kind == typing.KindBackspace:
			s.Corrections++
		case k.Correct():
			s.Keystrokes++
		default:
			s.Keystrokes++
			s.Errors++
		}
	}
	target, typed := []rune(r.Target), []rune(r.Typed)
	for i := range min(len(target), len(typed)) {
		if target[i] == typed[i] {
			s.Correct++
		}
	}

	s.WPM = wpm(s.Correct, r.Duration)
	s.RawWPM = wpm(s.Keystrokes, r.Duration)
	if s.Keystrokes > 0 {
		s.Accuracy = float64(s.Keystrokes-s.Errors) / float64(s.Keystrokes)
	}

	timeline := Timeline(r)
	s.Consistency = consistency(timeline)
	for _, p := range timeline {
		s.PeakWPM = max(s.PeakWPM, p.WPM)
	}
	return s
}

// wpm converts a character count over a duration to words per minute.
func wpm(chars int, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(chars) / charsPerWord / d.Minutes()
}

// consistency scores how evenly keystrokes were spread across seconds. It
// maps the coefficient of variation into 0–1 so a perfectly even pace is 1.
func consistency(timeline []Point) float64 {
	if len(timeline) < 2 {
		return 0
	}
	var sum, sumSq float64
	for _, p := range timeline {
		n := float64(p.Keystrokes)
		sum += n
		sumSq += n * n
	}
	n := float64(len(timeline))
	mean := sum / n
	if mean == 0 {
		return 0
	}
	std := math.Sqrt(max(0, sumSq/n-mean*mean))
	return 1 - math.Tanh(std/mean)
}
