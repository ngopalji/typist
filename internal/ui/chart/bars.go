package chart

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

var eighths = []rune(" ▁▂▃▄▅▆▇█")

// Bars is a column chart drawn with eighth blocks, one bucket per cell, so
// each column has eight levels of resolution per row.
type Bars struct {
	Frame
	Color func(v float64) lipgloss.Style // style for a bar of value v
}

// Render fits values into the plot width and draws one bar per column. NaN
// values leave a gap.
func (c Bars) Render(values []float64) string {
	if c.Width <= 0 || c.Height <= 0 {
		return ""
	}
	cols := resample(values, c.Width)
	levels := make([]int, len(cols))
	styles := make([]lipgloss.Style, len(cols))
	for i, v := range cols {
		if math.IsNaN(v) {
			levels[i] = -1
			continue
		}
		// Always show at least a sliver so a value at the floor is visible.
		levels[i] = max(1, int(math.Round(c.normalize(v)*float64(c.Height*8))))
		if c.Color != nil {
			styles[i] = c.Color(v)
		}
	}

	// When a few values are stretched across many columns, leave a gap
	// column between neighbors so each reads as its own bar.
	gap := make([]bool, len(cols))
	if n := len(values); n > 0 && c.Width/n >= 3 {
		for i := range gap {
			gap[i] = (i+1)*n/c.Width != i*n/c.Width
		}
	}

	rows := make([]string, c.Height)
	for r := range c.Height {
		var b strings.Builder
		floor := (c.Height - 1 - r) * 8
		for i, level := range levels {
			if n := level - floor; level >= 0 && n > 0 && !gap[i] {
				b.WriteString(styles[i].Render(string(eighths[min(n, 8)])))
			} else {
				b.WriteByte(' ')
			}
		}
		rows[r] = b.String()
	}
	return c.assemble(rows, nil)
}
