// Package chart renders small terminal charts as strings: braille line charts
// for smooth curves and eighth-block bar charts for per-bucket values.
//
// Both share the same frame: a left gutter of y-axis labels, the plot, an
// x-axis, and optional x labels, so charts stacked together line up.
package chart

import (
	"math"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// Frame is the configuration shared by every chart type.
type Frame struct {
	Width, Height int     // plot area, in cells
	Min, Max      float64 // y range
	LabelWidth    int     // width of the y-axis label gutter
	Format        func(float64) string
	XLabels       []string // spread evenly under the plot, left to right
	AxisStyle     lipgloss.Style
	LabelStyle    lipgloss.Style
}

// normalize maps v into [0, 1] of the y range.
func (f Frame) normalize(v float64) float64 {
	if f.Max <= f.Min {
		return 0
	}
	return math.Max(0, math.Min(1, (v-f.Min)/(f.Max-f.Min)))
}

// yLabel returns the axis label for plot row r, or "" if the row is unlabeled.
func (f Frame) yLabel(r int) string {
	format := f.Format
	if format == nil {
		format = func(v float64) string { return strconv.FormatFloat(v, 'f', 0, 64) }
	}
	switch {
	case r == 0:
		return format(f.Max)
	case r == f.Height-1:
		return format(f.Min)
	case f.Height >= 5 && r == f.Height/2:
		return format(f.Min + (f.Max-f.Min)*(1-float64(r)/float64(f.Height-1)))
	}
	return ""
}

// assemble wraps plot rows in the axis frame. extra rows (like event markers)
// go between the plot and the x-axis, and get their own gutter labels.
func (f Frame) assemble(rows []string, extra [][2]string) string {
	var b strings.Builder
	gutter := func(label string, axis string) {
		pad := max(0, f.LabelWidth-lipgloss.Width(label))
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(f.LabelStyle.Render(label))
		b.WriteString(" ")
		b.WriteString(f.AxisStyle.Render(axis))
	}
	for r, row := range rows {
		label := f.yLabel(r)
		axis := "│"
		if label != "" {
			axis = "┤"
		}
		gutter(label, axis)
		b.WriteString(row)
		b.WriteByte('\n')
	}
	for _, e := range extra {
		gutter(e[0], " ")
		b.WriteString(e[1])
		b.WriteByte('\n')
	}
	b.WriteString(strings.Repeat(" ", f.LabelWidth+1))
	b.WriteString(f.AxisStyle.Render("└" + strings.Repeat("─", f.Width)))
	if len(f.XLabels) > 0 {
		b.WriteByte('\n')
		b.WriteString(strings.Repeat(" ", f.LabelWidth+2))
		b.WriteString(f.LabelStyle.Render(spread(f.XLabels, f.Width)))
	}
	return b.String()
}

// spread lays labels out across width: first flush left, last flush right,
// the rest centered at even intervals.
func spread(labels []string, width int) string {
	line := []rune(strings.Repeat(" ", width))
	place := func(s string, at int) {
		rs := []rune(s)
		at = max(0, min(at, width-len(rs)))
		for i, r := range rs {
			if at+i < width {
				line[at+i] = r
			}
		}
	}
	n := len(labels)
	for i, l := range labels {
		switch {
		case i == 0:
			place(l, 0)
		case i == n-1:
			place(l, width-len([]rune(l)))
		default:
			place(l, i*(width-1)/(n-1)-len([]rune(l))/2)
		}
	}
	return string(line)
}

// NiceMax rounds v up to a tidy axis maximum (multiples of 10, 20, 50...).
func NiceMax(v float64) float64 {
	if v <= 10 {
		return 10
	}
	step := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 1.5, 2, 2.5, 5, 10} {
		if s := step * m; v <= s {
			return s
		}
	}
	return math.Ceil(v)
}

// resample fits values into n buckets by averaging (ignoring NaNs). With fewer
// values than buckets, each value is stretched across several buckets.
func resample(values []float64, n int) []float64 {
	out := make([]float64, n)
	for c := range n {
		lo := c * len(values) / n
		hi := max((c+1)*len(values)/n, lo+1)
		sum, count := 0.0, 0
		for _, v := range values[min(lo, len(values)):min(hi, len(values))] {
			if !math.IsNaN(v) {
				sum += v
				count++
			}
		}
		out[c] = math.NaN()
		if count > 0 {
			out[c] = sum / float64(count)
		}
	}
	return out
}
