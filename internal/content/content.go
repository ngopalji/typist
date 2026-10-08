// Package content generates the practice text a typing test is made of.
//
// Every mode is an endless Source of tokens (a digit group, a word, a whole
// passage). The typing engine keeps pulling tokens until it has enough text
// buffered, so a test can run for any length of time without running out.
package content

import (
	"fmt"
	"math/rand/v2"
)

// Mode identifies a kind of practice text. Values are persisted with every
// saved session, so existing ones must never be renamed.
type Mode string

const (
	Numbers Mode = "numbers"
	Letters Mode = "letters"
	Words   Mode = "words"
	Prose   Mode = "prose"
	Symbols Mode = "symbols"
)

// Info describes a mode for display.
type Info struct {
	Mode  Mode
	Title string
	Blurb string
}

// Modes lists every mode in menu order.
var Modes = []Info{
	{Numbers, "numbers", "digit groups, 0–9"},
	{Letters, "letters", "random letter clusters, a–z"},
	{Words, "words", "the most common english words"},
	{Prose, "prose", "real sentences with capitals and punctuation"},
	{Symbols, "symbols", "brackets, operators, and shifted keys"},
}

// Lookup returns the display info for m.
func Lookup(m Mode) (Info, bool) {
	for _, info := range Modes {
		if info.Mode == m {
			return info, true
		}
	}
	return Info{}, false
}

// Source yields an endless stream of tokens. Tokens never have leading or
// trailing whitespace; callers join them with single spaces.
type Source interface {
	Next() string
}

// SourceFunc adapts a plain function to Source.
type SourceFunc func() string

func (f SourceFunc) Next() string { return f() }

// New returns the Source for mode m, drawing randomness from rng.
func New(m Mode, rng *rand.Rand) (Source, error) {
	switch m {
	case Numbers:
		return groups(rng, []rune("0123456789"), 2, 5), nil
	case Letters:
		return groups(rng, []rune("abcdefghijklmnopqrstuvwxyz"), 2, 6), nil
	case Words:
		return pick(rng, wordList), nil
	case Prose:
		return shuffled(rng, passages), nil
	case Symbols:
		return symbols(rng), nil
	}
	return nil, fmt.Errorf("unknown mode %q", m)
}

// Sample returns roughly n characters of text for m, for previews.
func Sample(m Mode, rng *rand.Rand, n int) string {
	src, err := New(m, rng)
	if err != nil {
		return ""
	}
	var out []rune
	for len(out) < n {
		if len(out) > 0 {
			out = append(out, ' ')
		}
		out = append(out, []rune(src.Next())...)
	}
	return string(out[:n])
}

// groups emits random clusters of runes from alphabet, between min and max
// runes long.
func groups(rng *rand.Rand, alphabet []rune, min, max int) Source {
	return SourceFunc(func() string {
		g := make([]rune, min+rng.IntN(max-min+1))
		for i := range g {
			g[i] = alphabet[rng.IntN(len(alphabet))]
		}
		return string(g)
	})
}

// pick emits uniformly random items, never the same one twice in a row.
func pick(rng *rand.Rand, items []string) Source {
	last := -1
	return SourceFunc(func() string {
		i := rng.IntN(len(items))
		if i == last && len(items) > 1 {
			i = (i + 1) % len(items)
		}
		last = i
		return items[i]
	})
}

// shuffled emits every item once in random order, then reshuffles.
func shuffled(rng *rand.Rand, items []string) Source {
	var order []int
	return SourceFunc(func() string {
		if len(order) == 0 {
			order = rng.Perm(len(items))
		}
		i := order[0]
		order = order[1:]
		return items[i]
	})
}
