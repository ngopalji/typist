package content

import (
	"math/rand/v2"
	"strings"
	"testing"
	"unicode"
)

func TestEveryModeProducesTokens(t *testing.T) {
	for _, info := range Modes {
		t.Run(string(info.Mode), func(t *testing.T) {
			src, err := New(info.Mode, rand.New(rand.NewPCG(1, 2)))
			if err != nil {
				t.Fatal(err)
			}
			for range 200 {
				tok := src.Next()
				if tok == "" || strings.TrimSpace(tok) != tok {
					t.Fatalf("bad token %q", tok)
				}
			}
		})
	}
}

func TestModesUseTheirCharacterSets(t *testing.T) {
	check := func(m Mode, ok func(rune) bool) {
		src, _ := New(m, rand.New(rand.NewPCG(3, 4)))
		for range 100 {
			for _, r := range src.Next() {
				if !ok(r) {
					t.Fatalf("%s produced %q", m, r)
				}
			}
		}
	}
	check(Numbers, unicode.IsDigit)
	check(Letters, unicode.IsLower)
	check(Symbols, func(r rune) bool { return !unicode.IsSpace(r) && r < 128 })
}

func TestSourcesAreDeterministic(t *testing.T) {
	a := Sample(Words, rand.New(rand.NewPCG(9, 9)), 80)
	b := Sample(Words, rand.New(rand.NewPCG(9, 9)), 80)
	if a != b || len([]rune(a)) != 80 {
		t.Fatalf("samples differ or wrong length:\n%q\n%q", a, b)
	}
}

func TestEmbeddedData(t *testing.T) {
	if len(wordList) < 100 || len(passages) < 20 {
		t.Fatalf("words = %d, passages = %d", len(wordList), len(passages))
	}
	if _, err := New("nope", rand.New(rand.NewPCG(1, 1))); err == nil {
		t.Fatal("unknown mode should error")
	}
}
