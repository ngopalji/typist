package stats

import (
	"math"
	"time"

	"github.com/nihaar/typist/internal/typing"
)

// smoothing is the trailing window speed and accuracy are measured over, so
// the curves show trends instead of per-second noise.
const smoothing = 5 * time.Second

// Point describes one second of a test.
type Point struct {
	Second     int     // 1-based: the second that just ended
	WPM        float64 // net speed over the trailing window
	RawWPM     float64 // raw speed over the trailing window
	Accuracy   float64 // over the trailing window; NaN if nothing was typed
	Keystrokes int     // character keystrokes during this second
	Errors     int     // wrong keystrokes during this second
}

// Timeline breaks r into one Point per second.
func Timeline(r typing.Result) []Point {
	seconds := int(math.Ceil(r.Duration.Seconds()))
	if seconds == 0 {
		return nil
	}
	correct := make([]int, seconds)
	total := make([]int, seconds)
	errs := make([]int, seconds)
	for _, k := range r.Keystrokes {
		if k.Kind != typing.KindChar {
			continue
		}
		s := min(int(k.At/time.Second), seconds-1)
		total[s]++
		if k.Correct() {
			correct[s]++
		} else {
			errs[s]++
		}
	}

	window := int(smoothing / time.Second)
	points := make([]Point, seconds)
	var sumCorrect, sumTotal int
	for s := range seconds {
		sumCorrect += correct[s]
		sumTotal += total[s]
		if s >= window {
			sumCorrect -= correct[s-window]
			sumTotal -= total[s-window]
		}
		span := time.Duration(min(s+1, window)) * time.Second
		p := Point{
			Second:     s + 1,
			WPM:        wpm(sumCorrect, span),
			RawWPM:     wpm(sumTotal, span),
			Accuracy:   math.NaN(),
			Keystrokes: total[s],
			Errors:     errs[s],
		}
		if sumTotal > 0 {
			p.Accuracy = float64(sumCorrect) / float64(sumTotal)
		}
		points[s] = p
	}
	return points
}
