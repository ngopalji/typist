package ui

import (
	"fmt"
	"math"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ngopalji/typist/internal/content"
	"github.com/ngopalji/typist/internal/stats"
	"github.com/ngopalji/typist/internal/typing"
	"github.com/ngopalji/typist/internal/ui/chart"
)

// inputGrace is how long the results screen ignores keys after a test ends,
// so keystrokes still in flight from typing don't trigger shortcuts.
const inputGrace = 750 * time.Millisecond

// resultsScreen breaks down the test that just finished.
type resultsScreen struct {
	sh        *shared
	shownAt   time.Time
	keys      resultsKeys
	session   stats.Session
	timeline  []stats.Point
	breakdown *stats.Breakdown
	metric    heatMetric
	page      page
}

func newResultsScreen(sh *shared, s stats.Session) *resultsScreen {
	return &resultsScreen{
		sh:        sh,
		shownAt:   sh.now(),
		keys:      newResultsKeys(),
		session:   s,
		timeline:  stats.Timeline(s.Result),
		breakdown: stats.BreakdownOf([]stats.Session{s}),
		page:      newPage(),
	}
}

func (r *resultsScreen) init() tea.Cmd { return nil }

func (r *resultsScreen) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case relayoutMsg:
		r.layout()
	case tea.KeyPressMsg:
		if r.sh.now().Sub(r.shownAt) < inputGrace {
			return r, nil
		}
		switch {
		case key.Matches(msg, r.keys.Again):
			return r, send(startTestMsg{mode: r.session.Mode, length: typing.LengthOf(r.session.Limit)})
		case key.Matches(msg, r.keys.Back):
			return r, send(goHomeMsg{})
		case key.Matches(msg, r.keys.Stats):
			return r, send(goStatsMsg{})
		case key.Matches(msg, r.keys.Metric):
			r.metric = r.metric.toggle()
			r.layout()
		case key.Matches(msg, r.keys.Quit):
			return r, tea.Quit
		default:
			r.page.scroll(msg, r.keys.scrollKeys)
		}
	}
	return r, nil
}

func (r *resultsScreen) layout() {
	r.page.set(r.render(contentWidth(r.sh)), r.sh.width, r.sh.bodyHeight())
	r.keys.setEnabled(!r.page.fits)
}

func (r *resultsScreen) view() string {
	return r.sh.frame(r.page.view(), r.page.indicator(r.sh.theme)+r.sh.theme.helpBar(r.keys))
}

// contentWidth is the width results and stats pages lay out to.
func contentWidth(sh *shared) int { return max(40, min(sh.width-6, 104)) }

func (r *resultsScreen) render(w int) string {
	t := r.sh.theme
	info, _ := content.Lookup(r.session.Mode)
	header := t.title.Render(info.Title) +
		t.dim.Render("  ·  "+typing.FormatLimit(r.session.Limit)+"  ·  "+formatAgo(r.session.StartedAt, r.sh.now()))
	if r.isBest() {
		header += "   " + lipgloss.NewStyle().Foreground(t.palette.onAccent).Background(t.palette.good).
			Bold(true).Padding(0, 1).Render("new personal best")
	}

	return stack(
		center(w, header),
		r.hero(w),
		r.charts(w),
		t.section("keys · "+r.metric.String(), w)+"\n\n"+
			center(w, t.heatmap(r.breakdown, r.session.Mode, r.metric))+"\n\n"+
			center(w, t.insights(r.breakdown, w, 2)),
		center(w, r.saveStatus()),
	)
}

// hero is the big WPM number and the headline stats beside it.
func (r *resultsScreen) hero(w int) string {
	t := r.sh.theme
	sum := r.session.Summary
	wpm := fmt.Sprintf("%.0f", sum.WPM)
	big := bigText(wpm, t.palette.accent, t.palette.accent2)
	big += "\n" + lipgloss.PlaceHorizontal(bigTextWidth(wpm), lipgloss.Center, t.dim.Render("words per minute"))

	tiles := []string{
		t.tile("accuracy", pct(sum.Accuracy), t.heatText(accuracyLevel(sum.Accuracy))),
		t.tile("raw", fmt.Sprintf("%.0f", sum.RawWPM), t.text),
		t.tile("peak", fmt.Sprintf("%.0f", sum.PeakWPM), t.text),
		t.tile("consistency", fmt.Sprintf("%.0f%%", 100*sum.Consistency), t.text),
		t.tile("errors", fmt.Sprint(sum.Errors), t.bad),
		t.tile("fixed", fmt.Sprint(sum.Corrections), t.warn),
		t.tile("keys", fmt.Sprint(sum.Keystrokes), t.text),
	}
	if avg, ok := r.priorAverage(); ok {
		style := t.good
		if sum.WPM < avg {
			style = t.bad
		}
		tiles = append(tiles, t.tile("vs your avg", signed(sum.WPM-avg), style))
	}

	const gap = 8
	side := w - lipgloss.Width(big) - gap
	if side < 36 {
		return center(w, big) + "\n\n" + center(w, t.tiles(w, tiles...))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Center, padBlock(big), "        ", t.tiles(min(side, 60), tiles...))
	return center(w, row)
}

// charts draws speed and accuracy over the course of the test, side by side
// when there's room.
func (r *resultsScreen) charts(w int) string {
	t := r.sh.theme
	seconds := int(r.session.Limit.Seconds())
	xLabels := []string{"0s", fmt.Sprintf("%ds", seconds/2), fmt.Sprintf("%ds", seconds)}

	wide := w >= 96
	chartW, accHeight := w, 4
	if wide {
		chartW, accHeight = (w-6)/2, 8
	}

	wpm := make([]float64, len(r.timeline))
	acc := make([]float64, len(r.timeline))
	var marks []int
	peak := r.session.Summary.WPM
	for i, p := range r.timeline {
		wpm[i], acc[i] = p.WPM, 100*p.Accuracy
		peak = max(peak, p.WPM)
		if p.Errors > 0 {
			marks = append(marks, i)
		}
	}

	speed := chart.Line{
		Frame:     chartFrame(t, chartW, 7, 0, chart.NiceMax(peak*1.1), xLabels),
		LineStyle: t.accent,
		Fill:      true,
		FillStyle: lipgloss.NewStyle().Foreground(t.palette.accentSoft),
		ShowRef:   true,
		Ref:       r.session.Summary.WPM,
		RefStyle:  t.faint,
		Marks:     marks,
		MarkLabel: "err",
		MarkStyle: t.bad,
	}
	accuracy := chart.Bars{
		Frame: chartFrame(t, chartW, accHeight, accuracyFloor(acc), 100, xLabels),
		Color: func(v float64) lipgloss.Style { return t.heatText(accuracyLevel(v / 100)) },
	}

	left := t.section("speed · wpm", chartW) + "\n" + speed.Render(wpm)
	right := t.section("accuracy · %", chartW) + "\n" + accuracy.Render(acc)
	if wide {
		return lipgloss.JoinHorizontal(lipgloss.Top, padBlock(left), "      ", padBlock(right))
	}
	return left + "\n\n" + right
}

// chartFrame returns chart axes sized so the whole chart is width cells wide.
func chartFrame(t *theme, width, height int, lo, hi float64, xLabels []string) chart.Frame {
	const labelWidth = 3
	return chart.Frame{
		Width: width - labelWidth - 2, Height: height,
		Min: lo, Max: hi,
		LabelWidth: labelWidth,
		XLabels:    xLabels,
		AxisStyle:  t.faint,
		LabelStyle: t.dim,
	}
}

// accuracyFloor picks the bottom of the accuracy axis: a round number below
// the worst value, so differences near 100% stay visible.
func accuracyFloor(values []float64) float64 {
	lo := 90.0
	for _, v := range values {
		if !math.IsNaN(v) {
			lo = min(lo, v)
		}
	}
	return max(0, math.Floor(lo/10)*10-10)
}

// prior returns earlier sessions of the same mode and length.
func (r *resultsScreen) prior() []stats.Session {
	var out []stats.Session
	for _, s := range r.sh.sessions {
		if s.Mode == r.session.Mode && s.Limit == r.session.Limit && !s.StartedAt.Equal(r.session.StartedAt) {
			out = append(out, s)
		}
	}
	return out
}

func (r *resultsScreen) priorAverage() (float64, bool) {
	prior := r.prior()
	if len(prior) == 0 {
		return 0, false
	}
	o := stats.NewOverview(prior, 10, r.sh.now())
	return o.AvgWPM, true
}

func (r *resultsScreen) isBest() bool {
	prior := r.prior()
	if len(prior) == 0 {
		return false
	}
	return r.session.Summary.WPM > stats.NewOverview(prior, 10, r.sh.now()).BestWPM
}

func (r *resultsScreen) saveStatus() string {
	t := r.sh.theme
	switch r.sh.save {
	case saving:
		return t.dim.Render("saving…")
	case saved:
		return t.good.Render("✓") + t.dim.Render(" saved")
	case saveFailed:
		return t.bad.Render("not saved: " + r.sh.saveErr.Error())
	}
	return ""
}
