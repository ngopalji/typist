package ui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ngopalji/typist/internal/content"
	"github.com/ngopalji/typist/internal/stats"
	"github.com/ngopalji/typist/internal/typing"
)

// homeScreen is the menu: pick a mode with j/k and a length with h/l.
type homeScreen struct {
	sh      *shared
	keys    homeKeys
	cursor  int
	lengths []typing.Length // chosen length per mode; each starts at medium
	samples []string        // preview text per mode
}

func newHomeScreen(sh *shared) *homeScreen {
	h := &homeScreen{
		sh:      sh,
		keys:    newHomeKeys(),
		lengths: make([]typing.Length, len(content.Modes)),
		samples: make([]string, len(content.Modes)),
	}
	for i, info := range content.Modes {
		h.lengths[i] = typing.Medium
		h.samples[i] = content.Sample(info.Mode, sh.rng, 46)
	}
	return h
}

func (h *homeScreen) init() tea.Cmd { return nil }

func (h *homeScreen) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return h, nil
	}
	switch {
	case key.Matches(k, h.keys.Up):
		h.cursor = max(0, h.cursor-1)
	case key.Matches(k, h.keys.Down):
		h.cursor = min(len(content.Modes)-1, h.cursor+1)
	case key.Matches(k, h.keys.Left):
		h.lengths[h.cursor] = max(typing.Short, h.lengths[h.cursor]-1)
	case key.Matches(k, h.keys.Right):
		h.lengths[h.cursor] = min(typing.Long, h.lengths[h.cursor]+1)
	case key.Matches(k, h.keys.Start):
		return h, send(startTestMsg{mode: content.Modes[h.cursor].Mode, length: h.lengths[h.cursor]})
	case key.Matches(k, h.keys.Stats):
		return h, send(goStatsMsg{})
	case key.Matches(k, h.keys.Quit):
		return h, tea.Quit
	}
	return h, nil
}

func (h *homeScreen) view() string {
	t := h.sh.theme
	menu := h.menu()
	width := max(lipgloss.Width(menu), 50)

	logo := smallText("TYPIST", t.palette.cyan, t.palette.accent, t.palette.accent2)
	if h.sh.height >= 30 && h.sh.width >= bigTextWidth("TYPIST")+4 {
		logo = bigText("TYPIST", t.palette.cyan, t.palette.accent, t.palette.accent2)
	}
	width = max(width, lipgloss.Width(logo))

	info := content.Modes[h.cursor]
	preview := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(t.palette.faint).
		Padding(0, 2).
		Render(t.dim.Render(h.samples[h.cursor] + "…"))

	body := stack(
		center(width, logo)+"\n"+center(width, t.dim.Render("type fast · see everything")),
		center(width, menu),
		center(width, t.muted.Render(info.Blurb))+"\n"+center(width, preview),
		center(width, h.summary()),
	)
	return h.sh.frame(body, t.helpBar(h.keys))
}

func (h *homeScreen) menu() string {
	t := h.sh.theme
	var rows []string
	for i, info := range content.Modes {
		selected := i == h.cursor
		marker, name := "  ", t.muted.Render(fmt.Sprintf("%-9s", info.Title))
		if selected {
			marker, name = t.accent.Render("▌ "), t.bold.Render(fmt.Sprintf("%-9s", info.Title))
		}

		var opts []string
		for _, l := range typing.Lengths {
			label := lipgloss.PlaceHorizontal(6, lipgloss.Center, l.Label())
			switch {
			case l == h.lengths[i] && selected:
				opts = append(opts, t.pill.Render(label))
			case l == h.lengths[i]:
				opts = append(opts, t.muted.Render(label))
			case selected:
				opts = append(opts, t.dim.Render(label))
			default:
				opts = append(opts, t.faint.Render(label))
			}
		}
		row := marker + name + "  " + lipgloss.JoinHorizontal(lipgloss.Top, opts...) + "   " + h.best(info.Mode, h.lengths[i])
		rows = append(rows, row)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// best shows the personal best for a mode at a length.
func (h *homeScreen) best(mode content.Mode, l typing.Length) string {
	t := h.sh.theme
	top := 0.0
	for _, s := range h.sh.sessionsFor(mode) {
		if s.Limit == l.Duration() {
			top = max(top, s.Summary.WPM)
		}
	}
	if top == 0 {
		return t.faint.Render("best   –")
	}
	return t.dim.Render("best ") + t.text.Render(fmt.Sprintf("%3.0f", top))
}

func (h *homeScreen) summary() string {
	t := h.sh.theme
	switch {
	case h.sh.loadErr != nil:
		return t.bad.Render("couldn't load history: " + h.sh.loadErr.Error())
	case !h.sh.loaded:
		return t.dim.Render("loading history…")
	case len(h.sh.sessions) == 0:
		return t.dim.Render("your first test is one keypress away")
	}
	o := stats.NewOverview(h.sh.sessions, 10, h.sh.now())
	sep := t.faint.Render("  ·  ")
	out := t.text.Render(fmt.Sprint(o.Tests)) + t.dim.Render(" test"+plural(o.Tests)) + sep +
		t.text.Render(formatDuration(o.TimeTyping)) + t.dim.Render(" typed") + sep +
		t.text.Render(fmt.Sprintf("%.0f", o.AvgWPM)) + t.dim.Render(" wpm lately")
	if o.StreakDays > 1 {
		out += sep + t.warn.Render(fmt.Sprintf("%d day streak", o.StreakDays))
	}
	return out
}
