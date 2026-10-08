package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"

	"github.com/nihaar/typist/internal/content"
	"github.com/nihaar/typist/internal/stats"
)

// Layout helpers.

// padBlock pads every line of s to the same width, so the block keeps its
// left alignment when centered (lipgloss centers line by line).
func padBlock(s string) string {
	lines := strings.Split(s, "\n")
	w := 0
	for _, l := range lines {
		w = max(w, lipgloss.Width(l))
	}
	for i, l := range lines {
		lines[i] = l + strings.Repeat(" ", w-lipgloss.Width(l))
	}
	return strings.Join(lines, "\n")
}

// center horizontally centers a block within width.
func center(width int, s string) string {
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, padBlock(s))
}

// stack joins blocks vertically with a blank line between each.
func stack(blocks ...string) string {
	var kept []string
	for _, b := range blocks {
		if b != "" {
			kept = append(kept, b)
		}
	}
	return strings.Join(kept, "\n\n")
}

// columns lays blocks side by side with gap cells between them if they fit
// within width, and stacks them otherwise.
func columns(width, gap int, blocks ...string) string {
	total := gap * (len(blocks) - 1)
	for _, b := range blocks {
		total += lipgloss.Width(b)
	}
	if total > width {
		return stack(blocks...)
	}
	parts := make([]string, 0, 2*len(blocks))
	for i, b := range blocks {
		if i > 0 {
			parts = append(parts, strings.Repeat(" ", gap))
		}
		parts = append(parts, padBlock(b))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// Chrome shared by every screen.

// helpBar renders a keymap's short help as keycap blocks.
func (t *theme) helpBar(km help.KeyMap) string {
	var items []string
	for _, b := range km.ShortHelp() {
		h := b.Help()
		if !b.Enabled() || h.Key == "" {
			continue
		}
		items = append(items, t.keycap.Render(h.Key)+" "+t.dim.Render(h.Desc))
	}
	return strings.Join(items, "   ")
}

// section renders a titled rule, e.g. "── speed ───────".
func (t *theme) section(title string, width int) string {
	head := t.faint.Render("── ") + t.muted.Render(title) + " "
	return head + t.faint.Render(strings.Repeat("─", max(0, width-lipgloss.Width(head))))
}

// tile renders a small label over a big value.
func (t *theme) tile(label, value string, valueStyle lipgloss.Style) string {
	return t.dim.Render(label) + "\n" + valueStyle.Bold(true).Render(value)
}

// tiles lays tiles out in a row, wrapping onto more rows if needed.
func (t *theme) tiles(width int, tiles ...string) string {
	const gap = 4
	var rows []string
	var row []string
	used := 0
	for _, tl := range tiles {
		w := lipgloss.Width(tl)
		if len(row) > 0 && used+gap+w > width {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
			row, used = nil, 0
		}
		if len(row) > 0 {
			row = append(row, strings.Repeat(" ", gap))
			used += gap
		}
		row = append(row, padBlock(tl))
		used += w
	}
	if len(row) > 0 {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
	}
	return strings.Join(rows, "\n\n")
}

// Key heatmap.

type heatMetric int

const (
	heatAccuracy heatMetric = iota
	heatSpeed
)

func (m heatMetric) String() string {
	if m == heatSpeed {
		return "speed"
	}
	return "accuracy"
}

func (m heatMetric) toggle() heatMetric { return 1 - m }

// Keyboard rows shown in the heatmap, with their stagger (in cells).
type keyRow struct {
	keys   string
	indent int
}

var (
	numberRow  = []keyRow{{"1234567890", 0}}
	letterRows = []keyRow{{"qwertyuiop", 0}, {"asdfghjkl", 1}, {"zxcvbnm", 3}}
	symbolRows = []keyRow{{"`-=[]\\;',./", 0}, {"~!@#$%^&*()_+", 0}, {"{}|:\"<>?", 0}}
	proseRow   = []keyRow{{",.'\";:!?-", 0}}
)

// keyboardFor returns the heatmap layout for a mode; "" means every mode.
func keyboardFor(mode content.Mode) []keyRow {
	switch mode {
	case content.Numbers:
		return numberRow
	case content.Letters, content.Words:
		return letterRows
	case content.Prose:
		return append(append([]keyRow{}, letterRows...), proseRow...)
	case content.Symbols:
		return symbolRows
	}
	rows := append([]keyRow{}, numberRow...)
	rows = append(rows, letterRows...)
	return append(rows, symbolRows...)
}

// keyStat returns the stats for a physical key, folding uppercase letters
// onto their lowercase key.
func keyStat(b *stats.Breakdown, r rune) stats.KeyStat {
	ks := b.Key(r)
	if r >= 'a' && r <= 'z' {
		ks = ks.Merge(b.Key(r - 'a' + 'A'))
	}
	return ks
}

// heatLevel grades a key from 0 (worst) to heatLevels-1 (best), or -1 if
// there is no data. Speed is judged against the typist's own median.
func heatLevel(metric heatMetric, ks stats.KeyStat, median time.Duration) int {
	if metric == heatSpeed {
		lat := ks.AvgLatency()
		if lat == 0 || median == 0 {
			return -1
		}
		return grade(float64(median)/float64(lat), 0.85, 0.95, 1.03, 1.15)
	}
	if ks.Attempts() == 0 {
		return -1
	}
	return grade(ks.Accuracy(), 0.90, 0.94, 0.97, 0.99)
}

// accuracyLevel grades an accuracy value (0–1) on the heat scale.
func accuracyLevel(acc float64) int { return grade(acc, 0.90, 0.94, 0.97, 0.99) }

// grade returns how many ascending thresholds v meets.
func grade(v float64, thresholds ...float64) int {
	level := 0
	for _, th := range thresholds {
		if v >= th {
			level++
		}
	}
	return level
}

// heatmap draws the keyboard for mode, each key colored by metric.
func (t *theme) heatmap(b *stats.Breakdown, mode content.Mode, metric heatMetric) string {
	median := b.MedianLatency()
	var rows []string
	for _, row := range keyboardFor(mode) {
		var cells []string
		for _, r := range row.keys {
			level := heatLevel(metric, keyStat(b, r), median)
			cells = append(cells, t.heatStyle(level).Render(" "+string(r)+" "))
		}
		rows = append(rows, strings.Repeat(" ", row.indent)+strings.Join(cells, " "))
	}
	keys := padBlock(strings.Join(rows, "\n"))
	width := lipgloss.Width(keys)
	bar := min(width, 23)
	space := t.heatStyle(heatLevel(metric, keyStat(b, ' '), median)).
		Render(lipgloss.PlaceHorizontal(bar, lipgloss.Center, "space"))
	keys += "\n" + strings.Repeat(" ", (width-bar)/2) + space
	return padBlock(keys) + "\n\n" + t.heatLegend(metric)
}

func (t *theme) heatLegend(metric heatMetric) string {
	swatch := func(level int) string { return t.heatText(level).Render("■") }
	if metric == heatSpeed {
		var sw []string
		for l := range heatLevels {
			sw = append(sw, swatch(l))
		}
		return t.dim.Render("slower ") + strings.Join(sw, "") + t.dim.Render(" faster  ·  vs your median")
	}
	labels := []string{"<90", "90", "94", "97", "99+"}
	var parts []string
	for l, label := range labels {
		parts = append(parts, swatch(l)+" "+t.dim.Render(label))
	}
	return strings.Join(parts, "  ") + t.dim.Render("  % accuracy")
}

// Key lists.

// keyName renders a rune for display, making whitespace visible.
func keyName(r rune) string {
	if r == ' ' {
		return "␣"
	}
	return string(r)
}

// insights renders the three "where you struggle" lists side by side.
func (t *theme) insights(b *stats.Breakdown, width, minSamples int) string {
	list := func(title string, rows []string, empty string) string {
		if len(rows) == 0 {
			rows = []string{t.dim.Render(empty)}
		}
		return t.muted.Render(title) + "\n" + strings.Join(rows, "\n")
	}

	var weak []string
	for _, ks := range b.Weakest(5, minSamples) {
		level := accuracyLevel(ks.Accuracy())
		weak = append(weak, fmt.Sprintf("%s  %s  %s",
			t.heatStyle(level).Render(" "+keyName(ks.Key)+" "),
			t.heatText(level).Render(fmt.Sprintf("%5.1f%%", 100*ks.Accuracy())),
			t.dim.Render(fmt.Sprintf("%d/%d missed", ks.Misses, ks.Attempts()))))
	}

	var conf []string
	for _, c := range b.Confusions(5) {
		conf = append(conf, fmt.Sprintf("%s %s %s  %s",
			t.bold.Render(keyName(c.Expected)), t.dim.Render("→"), t.bad.Render(keyName(c.Typed)),
			t.dim.Render(fmt.Sprintf("×%d", c.Count))))
	}

	var slow []string
	transitions := b.SlowestTransitions(5, minSamples)
	for _, tr := range transitions {
		ms := tr.Avg.Milliseconds()
		bar := strings.Repeat("▪", int(math.Min(8, math.Max(1, float64(ms)/60))))
		slow = append(slow, fmt.Sprintf("%s%s  %s %s",
			t.bold.Render(keyName(tr.From)), t.bold.Render(keyName(tr.To)),
			t.warn.Render(fmt.Sprintf("%4dms", ms)), t.faint.Render(bar)))
	}

	return columns(width, 6,
		list("weakest keys", weak, "no misses yet"),
		list("typed instead", conf, "nothing mistyped"),
		list("slowest pairs", slow, "not enough data"),
	)
}

// Formatting.

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

func formatAgo(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 2")
}

func pct(v float64) string { return fmt.Sprintf("%.1f%%", 100*v) }

func signed(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("+%.1f", v)
	}
	return fmt.Sprintf("%.1f", v)
}
