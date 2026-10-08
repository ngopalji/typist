package stats

import (
	"time"

	"github.com/ngopalji/typist/internal/typing"
)

// Session is a test paired with its summary, computed once up front.
type Session struct {
	typing.Result
	Summary Summary
}

// Analyze summarizes r.
func Analyze(r typing.Result) Session {
	return Session{Result: r, Summary: Summarize(r)}
}

// BreakdownOf folds every session's keystrokes into one Breakdown.
func BreakdownOf(sessions []Session) *Breakdown {
	b := NewBreakdown()
	for _, s := range sessions {
		b.Add(s.Keystrokes)
	}
	return b
}

// Overview aggregates many sessions into all-time numbers.
type Overview struct {
	Tests       int
	TimeTyping  time.Duration
	Keystrokes  int
	BestWPM     float64
	AvgWPM      float64 // mean over the most recent sessions
	AvgAccuracy float64 // mean over the most recent sessions
	Trend       float64 // AvgWPM minus the mean of the sessions before those; 0 if too few
	StreakDays  int     // consecutive days with a test, ending today or yesterday
}

// NewOverview summarizes sessions (oldest first). Averages cover the last
// `recent` sessions, and now anchors the day streak.
func NewOverview(sessions []Session, recent int, now time.Time) Overview {
	o := Overview{Tests: len(sessions)}
	for _, s := range sessions {
		o.TimeTyping += s.Duration
		o.Keystrokes += s.Summary.Keystrokes
		o.BestWPM = max(o.BestWPM, s.Summary.WPM)
	}

	last := sessions[max(0, len(sessions)-recent):]
	o.AvgWPM, o.AvgAccuracy = means(last)
	if before := sessions[max(0, len(sessions)-2*recent) : len(sessions)-len(last)]; len(before) > 0 {
		prev, _ := means(before)
		o.Trend = o.AvgWPM - prev
	}
	o.StreakDays = streak(sessions, now)
	return o
}

func means(sessions []Session) (wpm, accuracy float64) {
	if len(sessions) == 0 {
		return 0, 0
	}
	for _, s := range sessions {
		wpm += s.Summary.WPM
		accuracy += s.Summary.Accuracy
	}
	n := float64(len(sessions))
	return wpm / n, accuracy / n
}

// streak counts consecutive calendar days (local time) with at least one
// session, ending today — or yesterday, so a streak survives until midnight.
func streak(sessions []Session, now time.Time) int {
	days := map[time.Time]bool{}
	for _, s := range sessions {
		days[day(s.StartedAt.In(now.Location()))] = true
	}
	d := day(now)
	if !days[d] {
		d = d.AddDate(0, 0, -1)
	}
	n := 0
	for days[d] {
		n++
		d = d.AddDate(0, 0, -1)
	}
	return n
}

func day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
