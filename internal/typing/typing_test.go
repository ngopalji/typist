package typing

import (
	"testing"
	"time"

	"github.com/nihaar/typist/internal/content"
)

var t0 = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func at(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }

func newTest(text string) *Test {
	t := New(content.Numbers, 30*time.Second)
	t.Append(text)
	return t
}

func TestTypeTracksCorrectAndIncorrect(t *testing.T) {
	tt := newTest("12 34")
	tt.Type('1', at(0))
	tt.Type('3', at(100))

	if got := tt.Cursor(); got != 2 {
		t.Fatalf("cursor = %d, want 2", got)
	}
	if tt.State(0) != Correct || tt.State(1) != Incorrect || tt.State(2) != Pending {
		t.Fatalf("states = %v %v %v", tt.State(0), tt.State(1), tt.State(2))
	}
	ks := tt.Keystrokes()
	if len(ks) != 2 || ks[0].At != 0 || ks[1].At != 100*time.Millisecond {
		t.Fatalf("keystrokes = %+v", ks)
	}
	if !ks[0].Correct() || !ks[1].Wrong() || ks[1].Expected != '2' || ks[1].Typed != '3' {
		t.Fatalf("keystroke correctness wrong: %+v", ks)
	}
}

func TestClockStartsOnFirstKeystroke(t *testing.T) {
	tt := newTest("123")
	if tt.Started() || tt.Elapsed(at(5000)) != 0 {
		t.Fatal("test should not start before a keystroke")
	}
	tt.Type('1', at(1000))
	if got := tt.Elapsed(at(3000)); got != 2*time.Second {
		t.Fatalf("elapsed = %v, want 2s", got)
	}
	if got := tt.Remaining(at(3000)); got != 28*time.Second {
		t.Fatalf("remaining = %v, want 28s", got)
	}
}

func TestBackspaceAndDeleteWord(t *testing.T) {
	tt := newTest("12 34 56")
	for i, r := range "12 3x" {
		tt.Type(r, at(i*100))
	}
	tt.Backspace(at(600))
	if tt.Cursor() != 4 {
		t.Fatalf("cursor after backspace = %d, want 4", tt.Cursor())
	}
	tt.DeleteWord(at(700))
	if tt.Cursor() != 3 {
		t.Fatalf("cursor after delete word = %d, want 3", tt.Cursor())
	}
	tt.DeleteWord(at(800)) // skips the trailing space, then deletes "12"
	if tt.Cursor() != 0 {
		t.Fatalf("cursor after second delete word = %d, want 0", tt.Cursor())
	}
	backspaces := 0
	for _, k := range tt.Keystrokes() {
		if k.Kind == KindBackspace {
			backspaces++
		}
	}
	if backspaces != 5 {
		t.Fatalf("recorded %d backspaces, want 5", backspaces)
	}
	tt.Backspace(at(900)) // nothing left to delete: ignored
	if len(tt.Keystrokes()) != 10 {
		t.Fatalf("backspace at 0 should not be recorded")
	}
}

func TestTimeLimitFinishesAndDropsLateKeys(t *testing.T) {
	tt := newTest("1234")
	tt.Type('1', at(0))
	tt.Tick(at(29_999))
	if tt.Done() {
		t.Fatal("finished early")
	}
	tt.Type('2', at(30_000))
	if !tt.Done() {
		t.Fatal("key after the limit should finish the test")
	}
	if tt.Cursor() != 1 {
		t.Fatalf("late key was accepted: cursor = %d", tt.Cursor())
	}
	r := tt.Snapshot(at(45_000))
	if r.Duration != 30*time.Second || r.Target != "1" || r.Typed != "1" || !r.StartedAt.Equal(t0) {
		t.Fatalf("result = %+v", r)
	}
}

func TestCannotTypePastBuffer(t *testing.T) {
	tt := newTest("1")
	tt.Type('1', at(0))
	tt.Type('2', at(10))
	if tt.Cursor() != 1 || len(tt.Keystrokes()) != 1 {
		t.Fatal("typed past the end of the buffer")
	}
	tt.Append("2")
	if tt.Buffered() != 2 || tt.Rune(1) != ' ' {
		t.Fatalf("append should add a separating space; buffered = %d", tt.Buffered())
	}
}

func TestFormatLimit(t *testing.T) {
	for l, want := range map[Length]string{Short: "30s", Medium: "60s", Long: "120s"} {
		if got := l.Label(); got != want {
			t.Errorf("%v label = %q, want %q", l, got, want)
		}
	}
}
