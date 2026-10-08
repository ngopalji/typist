package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/nihaar/typist/internal/content"
	"github.com/nihaar/typist/internal/stats"
	"github.com/nihaar/typist/internal/typing"
)

const (
	tickEvery    = 100 * time.Millisecond
	visibleLines = 3
)

// tickMsg drives the countdown. id ties it to one test so ticks from a
// restarted test are ignored.
type tickMsg struct{ id int }

// typerScreen runs one timed test.
type typerScreen struct {
	sh       *shared
	keys     typerKeys
	mode     content.Mode
	length   typing.Length
	src      content.Source
	test     *typing.Test
	tickID   int
	finished bool
}

func newTyperScreen(sh *shared, mode content.Mode, length typing.Length) *typerScreen {
	src, err := content.New(mode, sh.rng)
	if err != nil {
		panic(err) // modes come from content.Modes, so this is a programming error
	}
	s := &typerScreen{sh: sh, keys: newTyperKeys(), mode: mode, length: length, src: src}
	s.reset()
	return s
}

func (s *typerScreen) init() tea.Cmd { return nil }

// reset starts a fresh test with new text.
func (s *typerScreen) reset() {
	s.test = typing.New(s.mode, s.length.Duration())
	s.tickID++
	s.finished = false
	s.fill()
}

// fill keeps a few lines of text buffered ahead of the cursor.
func (s *typerScreen) fill() {
	for s.test.Buffered() < s.textWidth()*(visibleLines+1) {
		s.test.Append(s.src.Next())
	}
}

func (s *typerScreen) textWidth() int { return max(20, min(s.sh.width-12, 72)) }

func (s *typerScreen) tick() tea.Cmd {
	id := s.tickID
	return tea.Tick(tickEvery, func(time.Time) tea.Msg { return tickMsg{id: id} })
}

func (s *typerScreen) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case relayoutMsg:
		s.fill()
	case tickMsg:
		if msg.id != s.tickID || s.finished {
			return s, nil
		}
		s.test.Tick(s.sh.now())
		if s.test.Done() {
			return s, s.finish()
		}
		return s, s.tick()
	case tea.KeyPressMsg:
		return s, s.handleKey(msg)
	}
	return s, nil
}

func (s *typerScreen) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, s.keys.Back):
		return send(goHomeMsg{})
	case key.Matches(msg, s.keys.Restart):
		s.reset()
		return nil
	}
	if s.finished {
		return nil
	}

	now := s.sh.now()
	wasStarted := s.test.Started()
	switch {
	case key.Matches(msg, s.keys.DeleteWord):
		s.test.DeleteWord(now)
	case key.Matches(msg, s.keys.DeleteChar):
		s.test.Backspace(now)
	case msg.Text != "" && !msg.Mod.Contains(tea.ModCtrl) && !msg.Mod.Contains(tea.ModAlt):
		for _, r := range msg.Text {
			s.test.Type(r, now)
		}
	default:
		return nil
	}
	s.fill()

	switch {
	case s.test.Done():
		return s.finish()
	case !wasStarted && s.test.Started():
		return s.tick()
	}
	return nil
}

func (s *typerScreen) finish() tea.Cmd {
	s.finished = true
	s.tickID++
	return send(testDoneMsg{result: s.test.Snapshot(s.sh.now())})
}

func (s *typerScreen) view() string {
	t := s.sh.theme
	width := s.textWidth()
	info, _ := content.Lookup(s.mode)
	header := t.title.Render(info.Title) + t.dim.Render("  ·  "+s.length.Label())

	body := center(width+1, header) + "\n\n\n" +
		s.renderText(width) + "\n\n\n" +
		s.progressBar(width+1) + "\n" +
		center(width+1, s.statusBar())
	return s.sh.frame(body, t.helpBar(s.keys))
}

// renderText draws the visible window of text: the cursor's line plus one
// line of context above it, so finished lines scroll away.
func (s *typerScreen) renderText(width int) string {
	t := s.sh.theme
	lines := wrap(s.test, width)
	cursor := s.test.Cursor()
	current := 0
	for i, l := range lines {
		if cursor >= l[0] && cursor < l[1] {
			current = i
			break
		}
	}
	first := max(0, current-1)

	const (
		pending = iota
		typed
		wrong
		cursorInk
	)
	inks := []lipgloss.Style{pending: t.pending, typed: t.typed, wrong: t.wrong, cursorInk: t.cursor}

	out := make([]string, visibleLines)
	for row := range visibleLines {
		i := first + row
		if i >= len(lines) {
			out[row] = strings.Repeat(" ", width+1)
			continue
		}
		var b strings.Builder
		var run []rune
		ink := pending
		flush := func() {
			if len(run) > 0 {
				b.WriteString(inks[ink].Render(string(run)))
				run = run[:0]
			}
		}
		for pos := lines[i][0]; pos < lines[i][1]; pos++ {
			r, next := s.test.Rune(pos), pending
			switch {
			case pos == cursor:
				next = cursorInk
			case s.test.State(pos) == typing.Correct:
				next = typed
			case s.test.State(pos) == typing.Incorrect:
				next = wrong
				if r == ' ' {
					r = '·' // make a mistyped space visible
				}
			}
			if next != ink {
				flush()
				ink = next
			}
			run = append(run, r)
		}
		flush()
		// Lines end in a space, so pad to width+1 for a stable block.
		out[row] = b.String() + strings.Repeat(" ", width+1-(lines[i][1]-lines[i][0]))
	}
	return strings.Join(out, "\n")
}

// wrap breaks the test's text into lines of at most width runes, breaking
// after spaces. Each line is a [start, end) range of rune indexes.
func wrap(t *typing.Test, width int) [][2]int {
	var lines [][2]int
	n := t.Len()
	for start := 0; start < n; {
		if n-start <= width {
			lines = append(lines, [2]int{start, n})
			break
		}
		end := start + width
		brk := end // no space: hard-break a very long token
		for i := end; i > start; i-- {
			if t.Rune(i) == ' ' {
				brk = i + 1
				break
			}
		}
		lines = append(lines, [2]int{start, brk})
		start = brk
	}
	return lines
}

func (s *typerScreen) progressBar(width int) string {
	t := s.sh.theme
	now := s.sh.now()
	done := int(math.Round(float64(width) * float64(s.test.Elapsed(now)) / float64(s.test.Limit())))
	return t.accent.Render(strings.Repeat("━", done)) + t.faint.Render(strings.Repeat("━", width-done))
}

// statusBar is the live readout under the text.
func (s *typerScreen) statusBar() string {
	t := s.sh.theme
	now := s.sh.now()
	if !s.test.Started() {
		return t.bold.Render(s.length.Label()) + t.dim.Render("   the clock starts when you type")
	}

	left := int(math.Ceil(s.test.Remaining(now).Seconds()))
	sum := stats.Summarize(s.test.Snapshot(now))
	speed, acc := "–", "–"
	if s.test.Elapsed(now) >= 2*time.Second {
		speed = fmt.Sprintf("%.0f", sum.WPM)
	}
	if sum.Keystrokes > 0 {
		acc = fmt.Sprintf("%.0f%%", 100*sum.Accuracy)
	}

	timeStyle := t.bold
	if left <= 5 {
		timeStyle = t.warn.Bold(true)
	}
	cell := func(value string, style lipgloss.Style, label string) string {
		return style.Render(value) + t.dim.Render(" "+label)
	}
	sep := t.faint.Render("   ·   ")
	return strings.Join([]string{
		cell(fmt.Sprintf("%ds", left), timeStyle, "left"),
		cell(speed, t.bold, "wpm"),
		cell(acc, t.bold, "acc"),
		cell(fmt.Sprint(sum.Errors), t.bad.Bold(true), "errors"),
		cell(fmt.Sprint(s.streak()), t.good.Bold(true), "streak"),
	}, sep)
}

// streak counts correct keystrokes since the last mistake or correction.
func (s *typerScreen) streak() int {
	ks := s.test.Keystrokes()
	n := 0
	for i := len(ks) - 1; i >= 0 && ks[i].Correct(); i-- {
		n++
	}
	return n
}
