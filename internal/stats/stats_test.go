package stats

import (
	"math"
	"testing"
	"time"

	"github.com/ngopalji/typist/internal/content"
	"github.com/ngopalji/typist/internal/typing"
)

// play types input against target, one key every step, and returns the
// result. '<' in input means backspace.
func play(target, input string, step time.Duration) typing.Result {
	t := typing.New(content.Numbers, time.Minute)
	t.Append(target)
	start := time.Date(2026, 9, 28, 12, 0, 0, 0, time.Local)
	now := start
	for _, r := range input {
		if r == '<' {
			t.Backspace(now)
		} else {
			t.Type(r, now)
		}
		now = now.Add(step)
	}
	return t.Snapshot(now.Add(-step))
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestSummarize(t *testing.T) {
	// 10 keys at 100ms: 9 intervals = 0.9s. One wrong key, then fixed.
	r := play("1234567890", "123x<45678", 100*time.Millisecond)
	s := Summarize(r)

	if s.Keystrokes != 9 || s.Errors != 1 || s.Corrections != 1 {
		t.Fatalf("counts = %+v", s)
	}
	if s.Correct != 8 {
		t.Fatalf("correct = %d, want 8", s.Correct)
	}
	if !near(s.Accuracy, 8.0/9.0) {
		t.Fatalf("accuracy = %v", s.Accuracy)
	}
	wantWPM := 8.0 / 5 / (900 * time.Millisecond).Minutes()
	if !near(s.WPM, wantWPM) {
		t.Fatalf("wpm = %v, want %v", s.WPM, wantWPM)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := Summarize(typing.Result{})
	if s.WPM != 0 || s.Accuracy != 0 || s.Consistency != 0 {
		t.Fatalf("empty summary = %+v", s)
	}
}

func TestTimeline(t *testing.T) {
	r := play("11111 11111 11111", "11111 11111 11111", 200*time.Millisecond)
	points := Timeline(r)
	if len(points) != 4 {
		t.Fatalf("got %d points, want 4 (3.2s of typing)", len(points))
	}
	if points[0].Keystrokes != 5 {
		t.Fatalf("first second had %d keystrokes, want 5", points[0].Keystrokes)
	}
	if !near(points[0].Accuracy, 1) {
		t.Fatalf("accuracy = %v", points[0].Accuracy)
	}
}

func TestBreakdown(t *testing.T) {
	b := NewBreakdown()
	b.Add(play("7878", "7x<878", 100*time.Millisecond).Keystrokes)

	seven, eight := b.Key('7'), b.Key('8')
	if eight.Misses != 1 || eight.Hits != 2 {
		t.Fatalf("8 = %+v", eight)
	}
	if seven.Hits != 2 || seven.Misses != 0 {
		t.Fatalf("7 = %+v", seven)
	}
	// The 8 typed right after a backspace is not clean flow, so only the
	// second 8 (after a correct 7) gives a latency sample.
	if eight.AvgLatency() != 100*time.Millisecond {
		t.Fatalf("8 latency = %v", eight.AvgLatency())
	}

	conf := b.Confusions(5)
	if len(conf) != 1 || conf[0] != (Confusion{Expected: '8', Typed: 'x', Count: 1}) {
		t.Fatalf("confusions = %+v", conf)
	}
	weak := b.Weakest(5, 1)
	if len(weak) != 1 || weak[0].Key != '8' {
		t.Fatalf("weakest = %+v", weak)
	}
	trans := b.SlowestTransitions(5, 1)
	if len(trans) == 0 || trans[0].Avg != 100*time.Millisecond {
		t.Fatalf("transitions = %+v", trans)
	}
}

func TestBreakdownIgnoresPauses(t *testing.T) {
	b := NewBreakdown()
	b.Add(play("12", "12", 5*time.Second).Keystrokes)
	if b.Key('2').AvgLatency() != 0 {
		t.Fatal("a 5s pause should not count as key latency")
	}
}

func TestOverview(t *testing.T) {
	now := time.Date(2026, 9, 28, 20, 0, 0, 0, time.Local)
	mk := func(daysAgo int, wpm float64) Session {
		return Session{
			Result:  typing.Result{StartedAt: now.AddDate(0, 0, -daysAgo), Duration: time.Minute},
			Summary: Summary{WPM: wpm, Accuracy: 0.9},
		}
	}
	sessions := []Session{mk(5, 40), mk(2, 50), mk(1, 60), mk(0, 70)}
	o := NewOverview(sessions, 2, now)

	if o.Tests != 4 || o.BestWPM != 70 || o.TimeTyping != 4*time.Minute {
		t.Fatalf("overview = %+v", o)
	}
	if !near(o.AvgWPM, 65) || !near(o.Trend, 20) {
		t.Fatalf("avg = %v trend = %v", o.AvgWPM, o.Trend)
	}
	if o.StreakDays != 3 {
		t.Fatalf("streak = %d, want 3", o.StreakDays)
	}
}
