package ui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ngopalji/typist/internal/content"
	"github.com/ngopalji/typist/internal/stats"
	"github.com/ngopalji/typist/internal/typing"
	"github.com/ngopalji/typist/internal/ui/chart"
)

// statsScreen shows all-time history, filterable by mode with h/l.
type statsScreen struct {
	sh     *shared
	keys   statsKeys
	filter int // 0 is every mode; i is content.Modes[i-1]
	metric heatMetric
	page   page
}

func newStatsScreen(sh *shared) *statsScreen {
	return &statsScreen{sh: sh, keys: newStatsKeys(), page: newPage()}
}

func (s *statsScreen) init() tea.Cmd { return nil }

// mode returns the selected filter, "" meaning every mode.
func (s *statsScreen) mode() content.Mode {
	if s.filter == 0 {
		return ""
	}
	return content.Modes[s.filter-1].Mode
}

func (s *statsScreen) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case relayoutMsg:
		s.layout()
	case tea.KeyPressMsg:
		n := len(content.Modes) + 1
		switch {
		case key.Matches(msg, s.keys.Prev):
			s.filter = (s.filter + n - 1) % n
			s.page.vp.GotoTop()
			s.layout()
		case key.Matches(msg, s.keys.Next):
			s.filter = (s.filter + 1) % n
			s.page.vp.GotoTop()
			s.layout()
		case key.Matches(msg, s.keys.Metric):
			s.metric = s.metric.toggle()
			s.layout()
		case key.Matches(msg, s.keys.Back):
			return s, send(goHomeMsg{})
		case key.Matches(msg, s.keys.Quit):
			return s, tea.Quit
		default:
			s.page.scroll(msg, s.keys.scrollKeys)
		}
	}
	return s, nil
}

func (s *statsScreen) layout() {
	s.page.set(s.render(contentWidth(s.sh)), s.sh.width, s.sh.bodyHeight())
	s.keys.setEnabled(!s.page.fits)
}

func (s *statsScreen) view() string {
	return s.sh.frame(s.page.view(), s.page.indicator(s.sh.theme)+s.sh.theme.helpBar(s.keys))
}

func (s *statsScreen) render(w int) string {
	t := s.sh.theme
	head := center(w, t.title.Render("stats")) + "\n\n" + center(w, s.tabs())

	sessions := s.sh.sessionsFor(s.mode())
	switch {
	case s.sh.loadErr != nil:
		return stack(head, center(w, t.bad.Render("couldn't load history: "+s.sh.loadErr.Error())))
	case !s.sh.loaded:
		return stack(head, center(w, t.dim.Render("loading…")))
	case len(sessions) == 0:
		what := "tests"
		if s.mode() != "" {
			what = string(s.mode()) + " tests"
		}
		return stack(head, center(w, t.dim.Render("no "+what+" yet. press b and go type something")))
	}

	breakdown := stats.BreakdownOf(sessions)
	return stack(
		head,
		center(w, s.overview(sessions, w)),
		s.trends(sessions, w),
		t.section("keys · "+s.metric.String()+" · all time", w)+"\n\n"+
			center(w, t.heatmap(breakdown, s.mode(), s.metric))+"\n\n"+
			center(w, t.insights(breakdown, w, 5)),
		t.section("recent", w)+"\n\n"+center(w, s.recent(sessions)),
		center(w, t.faint.Render("data: "+s.sh.dbPath)),
	)
}

func (s *statsScreen) tabs() string {
	t := s.sh.theme
	labels := []string{"all"}
	for _, info := range content.Modes {
		labels = append(labels, info.Title)
	}
	var out []string
	for i, l := range labels {
		if i == s.filter {
			out = append(out, t.pill.Render(" "+l+" "))
		} else {
			out = append(out, t.dim.Render(" "+l+" "))
		}
	}
	return strings.Join(out, " ")
}

func (s *statsScreen) overview(sessions []stats.Session, w int) string {
	t := s.sh.theme
	o := stats.NewOverview(sessions, 10, s.sh.now())
	trendStyle := t.good
	if o.Trend < 0 {
		trendStyle = t.bad
	}
	tiles := []string{
		t.tile("tests", fmt.Sprint(o.Tests), t.text),
		t.tile("time typing", formatDuration(o.TimeTyping), t.text),
		t.tile("best", fmt.Sprintf("%.0f wpm", o.BestWPM), t.accent),
		t.tile("last 10 avg", fmt.Sprintf("%.0f wpm", o.AvgWPM), t.text),
		t.tile("accuracy", pct(o.AvgAccuracy), t.heatText(accuracyLevel(o.AvgAccuracy))),
	}
	if o.Trend != 0 {
		tiles = append(tiles, t.tile("trend", signed(o.Trend)+" wpm", trendStyle))
	}
	tiles = append(tiles,
		t.tile("streak", fmt.Sprintf("%d day%s", o.StreakDays, plural(o.StreakDays)), t.warn),
		t.tile("keystrokes", fmt.Sprint(o.Keystrokes), t.text),
	)
	return t.tiles(w, tiles...)
}

// trends charts speed and accuracy per test for the most recent tests.
func (s *statsScreen) trends(sessions []stats.Session, w int) string {
	t := s.sh.theme
	if len(sessions) < 2 {
		return t.section("trends", w) + "\n\n" + center(w, t.dim.Render("take one more test to start seeing trends"))
	}
	wide := w >= 96
	chartW := w
	if wide {
		chartW = (w - 6) / 2
	}
	shown := sessions[max(0, len(sessions)-(chartW-5)*2):]

	wpm := make([]float64, len(shown))
	acc := make([]float64, len(shown))
	peak, avgWPM, avgAcc := 0.0, 0.0, 0.0
	for i, sess := range shown {
		wpm[i], acc[i] = sess.Summary.WPM, 100*sess.Summary.Accuracy
		peak = max(peak, wpm[i])
		avgWPM += wpm[i] / float64(len(shown))
		avgAcc += acc[i] / float64(len(shown))
	}
	xLabels := []string{formatAgo(shown[0].StartedAt, s.sh.now()), "latest"}

	speed := chart.Line{
		Frame:     chartFrame(t, chartW, 7, 0, chart.NiceMax(peak*1.1), xLabels),
		LineStyle: t.accent,
		Fill:      true,
		FillStyle: lipgloss.NewStyle().Foreground(t.palette.accentSoft),
		ShowRef:   true,
		Ref:       avgWPM,
		RefStyle:  t.faint,
	}
	accuracy := chart.Line{
		Frame:     chartFrame(t, chartW, 7, accuracyFloor(acc), 100, xLabels),
		LineStyle: t.good,
		ShowRef:   true,
		Ref:       avgAcc,
		RefStyle:  t.faint,
	}

	title := fmt.Sprintf("last %d tests", len(shown))
	left := t.section("speed · "+title, chartW) + "\n" + speed.Render(wpm)
	right := t.section("accuracy · "+title, chartW) + "\n" + accuracy.Render(acc)
	if wide {
		return lipgloss.JoinHorizontal(lipgloss.Top, padBlock(left), "      ", padBlock(right))
	}
	return left + "\n\n" + right
}

// recent lists the latest tests, newest first.
func (s *statsScreen) recent(sessions []stats.Session) string {
	t := s.sh.theme
	latest := slices.Clone(sessions[max(0, len(sessions)-8):])
	slices.Reverse(latest)
	var rows []string
	for _, sess := range latest {
		rows = append(rows, strings.Join([]string{
			t.dim.Render(fmt.Sprintf("%-12s", sess.StartedAt.Format("Jan 2 15:04"))),
			t.muted.Render(fmt.Sprintf("%-8s", sess.Mode)),
			t.dim.Render(fmt.Sprintf("%4s", typing.FormatLimit(sess.Limit))),
			t.bold.Render(fmt.Sprintf("%4.0f", sess.Summary.WPM)) + t.dim.Render(" wpm"),
			t.heatText(accuracyLevel(sess.Summary.Accuracy)).Render(fmt.Sprintf("%6s", pct(sess.Summary.Accuracy))),
		}, "   "))
	}
	return strings.Join(rows, "\n")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
