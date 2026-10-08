package ui

import (
	"context"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ngopalji/typist/internal/content"
	"github.com/ngopalji/typist/internal/typing"
)

type fakeStore struct {
	mu      sync.Mutex
	history []typing.Result
	saved   []typing.Result
}

func (f *fakeStore) Save(_ context.Context, r typing.Result) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, r)
	return int64(len(f.saved)), nil
}

func (f *fakeStore) All(context.Context) ([]typing.Result, error) { return f.history, nil }

// harness drives an App synchronously: it runs commands inline and feeds
// their messages back, except ticks, which tests send by hand.
type harness struct {
	t     *testing.T
	app   *App
	store *fakeStore
	now   time.Time
	quit  bool
}

func newHarness(t *testing.T, start Screen, width, height int) *harness {
	h := &harness{t: t, store: &fakeStore{}, now: time.Date(2026, 9, 28, 20, 0, 0, 0, time.Local)}
	h.app = New(Options{
		Store:  h.store,
		Start:  start,
		DBPath: "/tmp/typist-test.db",
		Now:    func() time.Time { return h.now },
		Rand:   rand.New(rand.NewPCG(1, 2)),
	})
	h.run(h.app.Init())
	h.send(tea.WindowSizeMsg{Width: width, Height: height})
	return h
}

func (h *harness) send(msg tea.Msg) {
	_, cmd := h.app.Update(msg)
	h.run(cmd)
}

func (h *harness) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case nil, tickMsg:
	case tea.BatchMsg:
		for _, c := range msg {
			h.run(c)
		}
	case tea.QuitMsg:
		h.quit = true
	default:
		h.send(msg)
	}
}

func (h *harness) key(s string) {
	var k tea.KeyPressMsg
	switch s {
	case "enter":
		k = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		k = tea.KeyPressMsg{Code: tea.KeyTab}
	case "esc":
		k = tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		k = tea.KeyPressMsg{Code: tea.KeyBackspace}
	default:
		r := []rune(s)[0]
		k = tea.KeyPressMsg{Code: r, Text: s}
	}
	h.send(k)
}

func (h *harness) view() string { return h.app.View().Content }

// text is the view with styling stripped, for matching.
func (h *harness) text() string { return ansi.Strip(h.view()) }

func (h *harness) expectView(parts ...string) {
	h.t.Helper()
	v := h.text()
	for _, p := range parts {
		if !strings.Contains(v, p) {
			h.t.Fatalf("view missing %q:\n%s", p, v)
		}
	}
	if got := lipgloss.Height(v); got != h.app.sh.height {
		h.t.Fatalf("view is %d lines, window is %d", got, h.app.sh.height)
	}
}

func TestFullFlow(t *testing.T) {
	h := newHarness(t, HomeScreen, 120, 45)
	h.expectView("numbers", "letters", "prose", "symbols", "60s", "start")

	// j/k pick the mode, h/l the length; every mode starts on medium.
	h.key("j")
	h.key("l")
	home := h.app.home
	if home.cursor != 1 || home.lengths[1] != typing.Long {
		t.Fatalf("cursor = %d, letters length = %v", home.cursor, home.lengths[1])
	}
	h.key("k")
	h.key("h")
	if home.cursor != 0 || home.lengths[0] != typing.Short || home.lengths[2] != typing.Medium {
		t.Fatalf("after k/h: cursor = %d, lengths = %v", home.cursor, home.lengths)
	}

	h.key("enter")
	ts, ok := h.app.screen.(*typerScreen)
	if !ok || ts.mode != content.Numbers || ts.length != typing.Short {
		t.Fatalf("expected a 15s numbers test, got %T", h.app.screen)
	}
	h.expectView("numbers", "clock starts when you type")

	// Type 40 characters, with one mistake that gets corrected.
	for i := range 40 {
		want := string(ts.test.Rune(ts.test.Cursor()))
		if i == 10 {
			h.key("x")
			h.now = h.now.Add(150 * time.Millisecond)
			h.key("backspace")
			h.now = h.now.Add(150 * time.Millisecond)
		}
		h.key(want)
		h.now = h.now.Add(150 * time.Millisecond)
	}
	h.expectView("left", "wpm", "acc", "errors", "streak")

	// Time runs out: the next tick finishes the test.
	h.now = h.now.Add(30 * time.Second)
	h.send(tickMsg{id: ts.tickID})
	res, ok := h.app.screen.(*resultsScreen)
	if !ok {
		t.Fatalf("expected results, got %T", h.app.screen)
	}
	if len(h.store.saved) != 1 {
		t.Fatalf("saved %d sessions, want 1", len(h.store.saved))
	}
	sum := res.session.Summary
	if sum.Errors != 1 || sum.Corrections != 1 || sum.Correct != 40 {
		t.Fatalf("summary = %+v", sum)
	}
	h.expectView("words per minute", "speed", "accuracy", "weakest keys", "saved")

	h.key("q") // keys still in flight from typing are ignored at first
	if h.quit {
		t.Fatal("results should ignore keys during the grace period")
	}
	h.now = h.now.Add(time.Second)
	h.key("m") // heatmap metric toggles
	if res.metric != heatSpeed {
		t.Fatal("m should switch the heatmap to speed")
	}

	h.key("s")
	st, ok := h.app.screen.(*statsScreen)
	if !ok {
		t.Fatalf("expected stats, got %T", h.app.screen)
	}
	h.expectView("stats", "tests", "one more test", "recent", "/tmp/typist-test.db")
	h.key("l")
	if st.mode() != content.Numbers {
		t.Fatalf("filter = %q, want numbers", st.mode())
	}
	h.key("l")
	h.expectView("no letters tests yet")

	h.key("b")
	if h.app.screen != h.app.home {
		t.Fatalf("b should go home, got %T", h.app.screen)
	}
	h.expectView("best", "1 test ")
	h.key("q")
	if !h.quit {
		t.Fatal("q should quit")
	}
}

func TestTyperRestartAndBack(t *testing.T) {
	h := newHarness(t, HomeScreen, 100, 30)
	h.key("enter")
	ts := h.app.screen.(*typerScreen)
	h.key(string(ts.test.Rune(0)))
	first := ts.test
	h.key("tab")
	if ts.test == first || ts.test.Started() {
		t.Fatal("tab should start a fresh test")
	}
	h.key("esc")
	if h.app.screen != h.app.home {
		t.Fatal("esc should go back home")
	}
	if len(h.store.saved) != 0 {
		t.Fatal("abandoned tests must not be saved")
	}
}

func TestStatsCommandOpensStats(t *testing.T) {
	h := newHarness(t, StatsScreen, 100, 30)
	h.expectView("stats", "no tests yet")
}

// Every screen should render without panicking across window sizes, and fill
// exactly the window height.
func TestScreensRenderAtManySizes(t *testing.T) {
	for _, size := range [][2]int{{50, 16}, {80, 24}, {100, 40}, {200, 60}, {30, 10}} {
		h := newHarness(t, HomeScreen, size[0], size[1])
		h.store.history = nil
		h.view()
		h.key("enter")
		ts := h.app.screen.(*typerScreen)
		for range 30 {
			h.key(string(ts.test.Rune(ts.test.Cursor())))
			h.now = h.now.Add(120 * time.Millisecond)
		}
		h.view()
		h.now = h.now.Add(time.Minute)
		h.send(tickMsg{id: ts.tickID})
		if got := lipgloss.Height(h.view()); got != size[1] {
			t.Errorf("%v results: %d lines", size, got)
		}
		h.now = h.now.Add(time.Second)
		h.key("j")
		h.key("G")
		h.key("s")
		if got := lipgloss.Height(h.view()); got != size[1] {
			t.Errorf("%v stats: %d lines", size, got)
		}
	}
}

func TestWrap(t *testing.T) {
	tt := typing.New(content.Words, time.Minute)
	for _, w := range strings.Fields("the quick brown fox jumps over the lazy dog again and again supercalifragilistic") {
		tt.Append(w)
	}
	lines := wrap(tt, 12)
	next := 0
	for _, l := range lines {
		if l[0] != next || l[1]-l[0] > 13 {
			t.Fatalf("bad line %v in %v", l, lines)
		}
		next = l[1]
	}
	if next != tt.Len() {
		t.Fatalf("lines cover %d of %d runes", next, tt.Len())
	}
}
