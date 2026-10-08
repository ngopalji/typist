package stats

import (
	"cmp"
	"slices"
	"time"

	"github.com/ngopalji/typist/internal/typing"
)

// maxLatency discards gaps longer than this from speed stats: at that point
// the user paused, and the gap says nothing about the key itself.
const maxLatency = 2 * time.Second

// KeyStat is how one key performed whenever it was the expected character.
type KeyStat struct {
	Key     rune
	Hits    int // correct keystrokes
	Misses  int // wrong keystrokes
	latency time.Duration
	samples int
}

// Attempts returns how many times the key was typed at.
func (k KeyStat) Attempts() int { return k.Hits + k.Misses }

// Accuracy returns the share of attempts that were correct, 0–1.
func (k KeyStat) Accuracy() float64 {
	if k.Attempts() == 0 {
		return 0
	}
	return float64(k.Hits) / float64(k.Attempts())
}

// AvgLatency returns the mean time to reach this key from the previous one,
// counting only clean, uninterrupted keystrokes. Zero means no samples.
func (k KeyStat) AvgLatency() time.Duration {
	if k.samples == 0 {
		return 0
	}
	return k.latency / time.Duration(k.samples)
}

// Merge combines two keys' stats (e.g. 't' and 'T' onto one physical key).
func (k KeyStat) Merge(o KeyStat) KeyStat {
	k.Hits += o.Hits
	k.Misses += o.Misses
	k.latency += o.latency
	k.samples += o.samples
	return k
}

// Confusion counts how often Typed was entered when Expected was wanted.
type Confusion struct {
	Expected, Typed rune
	Count           int
}

// Transition is the average time to go from one key to the next.
type Transition struct {
	From, To rune
	Count    int
	Avg      time.Duration
}

type pair [2]rune

type transitionAcc struct {
	total time.Duration
	count int
}

// Breakdown accumulates per-key analytics across one or many tests.
type Breakdown struct {
	keys        map[rune]*KeyStat
	confusions  map[pair]int
	transitions map[pair]*transitionAcc
}

// NewBreakdown returns an empty Breakdown.
func NewBreakdown() *Breakdown {
	return &Breakdown{
		keys:        map[rune]*KeyStat{},
		confusions:  map[pair]int{},
		transitions: map[pair]*transitionAcc{},
	}
}

// Add folds one test's keystrokes into the breakdown.
func (b *Breakdown) Add(strokes []typing.Keystroke) {
	for i, k := range strokes {
		if k.Kind != typing.KindChar {
			continue
		}
		ks := b.key(k.Expected)
		if k.Wrong() {
			ks.Misses++
			b.confusions[pair{k.Expected, k.Typed}]++
			continue
		}
		ks.Hits++

		// Speed only counts clean flow: the previous keystroke was a correct
		// character immediately before this one.
		if i == 0 {
			continue
		}
		prev := strokes[i-1]
		gap := k.At - prev.At
		if !prev.Correct() || prev.Pos != k.Pos-1 || gap > maxLatency {
			continue
		}
		ks.latency += gap
		ks.samples++
		t := b.transitions[pair{prev.Expected, k.Expected}]
		if t == nil {
			t = &transitionAcc{}
			b.transitions[pair{prev.Expected, k.Expected}] = t
		}
		t.total += gap
		t.count++
	}
}

func (b *Breakdown) key(r rune) *KeyStat {
	ks := b.keys[r]
	if ks == nil {
		ks = &KeyStat{Key: r}
		b.keys[r] = ks
	}
	return ks
}

// Key returns the stats for r (zero if it was never expected).
func (b *Breakdown) Key(r rune) KeyStat {
	if ks := b.keys[r]; ks != nil {
		return *ks
	}
	return KeyStat{Key: r}
}

// Keys returns every key seen, in rune order.
func (b *Breakdown) Keys() []KeyStat {
	out := make([]KeyStat, 0, len(b.keys))
	for _, ks := range b.keys {
		out = append(out, *ks)
	}
	slices.SortFunc(out, func(a, b KeyStat) int { return cmp.Compare(a.Key, b.Key) })
	return out
}

// MedianLatency returns the median of every key's average latency, a baseline
// for judging which keys are slow for this particular typist.
func (b *Breakdown) MedianLatency() time.Duration {
	var ls []time.Duration
	for _, ks := range b.keys {
		if l := ks.AvgLatency(); l > 0 {
			ls = append(ls, l)
		}
	}
	if len(ls) == 0 {
		return 0
	}
	slices.Sort(ls)
	return ls[len(ls)/2]
}

// Weakest returns up to n keys with the lowest accuracy, among keys with at
// least minAttempts attempts and at least one miss.
func (b *Breakdown) Weakest(n, minAttempts int) []KeyStat {
	var out []KeyStat
	for _, ks := range b.keys {
		if ks.Attempts() >= minAttempts && ks.Misses > 0 {
			out = append(out, *ks)
		}
	}
	slices.SortFunc(out, func(a, b KeyStat) int {
		return cmp.Or(
			cmp.Compare(a.Accuracy(), b.Accuracy()),
			cmp.Compare(b.Misses, a.Misses),
			cmp.Compare(a.Key, b.Key),
		)
	})
	return out[:min(n, len(out))]
}

// Confusions returns up to n of the most frequent wrong substitutions.
func (b *Breakdown) Confusions(n int) []Confusion {
	out := make([]Confusion, 0, len(b.confusions))
	for p, c := range b.confusions {
		out = append(out, Confusion{Expected: p[0], Typed: p[1], Count: c})
	}
	slices.SortFunc(out, func(a, b Confusion) int {
		return cmp.Or(
			cmp.Compare(b.Count, a.Count),
			cmp.Compare(a.Expected, b.Expected),
			cmp.Compare(a.Typed, b.Typed),
		)
	})
	return out[:min(n, len(out))]
}

// SlowestTransitions returns up to n key-to-key transitions with the highest
// average latency, among those seen at least minCount times.
func (b *Breakdown) SlowestTransitions(n, minCount int) []Transition {
	var out []Transition
	for p, t := range b.transitions {
		if t.count >= minCount {
			out = append(out, Transition{From: p[0], To: p[1], Count: t.count, Avg: t.total / time.Duration(t.count)})
		}
	}
	slices.SortFunc(out, func(a, b Transition) int {
		return cmp.Or(
			cmp.Compare(b.Avg, a.Avg),
			cmp.Compare(a.From, b.From),
			cmp.Compare(a.To, b.To),
		)
	})
	return out[:min(n, len(out))]
}
