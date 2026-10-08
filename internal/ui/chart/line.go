package chart

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

// Line is a braille line chart. Each cell holds a 2×4 grid of dots, so curves
// get twice the horizontal and four times the vertical resolution of text.
type Line struct {
	Frame
	LineStyle lipgloss.Style

	Fill      bool // shade the area under the curve
	FillStyle lipgloss.Style

	ShowRef  bool // draw a dashed reference line at Ref, e.g. an average
	Ref      float64
	RefStyle lipgloss.Style

	Marks     []int // indexes into values to flag on a row under the plot
	MarkLabel string
	MarkStyle lipgloss.Style
}

// Render draws values evenly spaced across the plot. NaN values break the line.
func (c Line) Render(values []float64) string {
	w, h := c.Width, c.Height
	if w <= 0 || h <= 0 {
		return ""
	}
	dotW, dotH := w*2, h*4
	line, fill := newGrid(w, h), newGrid(w, h)
	top := make([]int, dotW) // highest line dot in each dot column, -1 if none
	for i := range top {
		top[i] = -1
	}

	xOf := func(i int) int {
		if len(values) <= 1 {
			return 0
		}
		return int(math.Round(float64(i) * float64(dotW-1) / float64(len(values)-1)))
	}
	yOf := func(v float64) int {
		return dotH - 1 - int(math.Round(c.normalize(v)*float64(dotH-1)))
	}
	plot := func(x, y int) {
		line.set(x, y)
		if top[x] < 0 || y < top[x] {
			top[x] = y
		}
	}

	var px, py int
	connected := false
	for i, v := range values {
		if math.IsNaN(v) {
			connected = false
			continue
		}
		x, y := xOf(i), yOf(v)
		if connected {
			bresenham(px, py, x, y, plot)
		} else {
			plot(x, y)
		}
		px, py, connected = x, y, true
	}
	if c.Fill {
		for x, t := range top {
			for y := t + 1; t >= 0 && y < dotH; y++ {
				fill.set(x, y)
			}
		}
	}
	refRow := -1
	if c.ShowRef {
		refRow = yOf(c.Ref) / 4
	}

	const (
		blank = iota
		lineInk
		fillInk
		refInk
	)
	rows := make([]string, h)
	for r := range h {
		p := painter{styles: []lipgloss.Style{lineInk: c.LineStyle, fillInk: c.FillStyle, refInk: c.RefStyle}}
		for col := range w {
			switch {
			case line.at(col, r) != 0:
				p.put(lineInk, braille(line.at(col, r)))
			case fill.at(col, r) != 0:
				p.put(fillInk, braille(fill.at(col, r)))
			case r == refRow:
				p.put(refInk, '┄')
			default:
				p.put(blank, ' ')
			}
		}
		rows[r] = p.String()
	}

	var extra [][2]string
	if len(c.Marks) > 0 {
		cells := []rune(strings.Repeat(" ", w))
		for _, i := range c.Marks {
			if i >= 0 && i < len(values) {
				cells[min(xOf(i)/2, w-1)] = '•'
			}
		}
		extra = append(extra, [2]string{c.MarkLabel, c.MarkStyle.Render(string(cells))})
	}
	return c.assemble(rows, extra)
}

// dotBits maps a dot's (row, column) within a cell to its braille bit.
var dotBits = [4][2]uint8{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

func braille(bits uint8) rune { return rune(0x2800 + int(bits)) }

// grid is a canvas of braille cells addressed by dot coordinates.
type grid struct {
	w     int
	cells []uint8
}

func newGrid(w, h int) grid { return grid{w: w, cells: make([]uint8, w*h)} }

func (g grid) set(x, y int)          { g.cells[(y/4)*g.w+x/2] |= dotBits[y%4][x%2] }
func (g grid) at(col, row int) uint8 { return g.cells[row*g.w+col] }

func bresenham(x0, y0, x1, y1 int, plot func(x, y int)) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		plot(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// painter batches runs of same-styled runes so each run is styled once.
type painter struct {
	styles []lipgloss.Style // indexed by ink; ink 0 is unstyled
	b      strings.Builder
	ink    int
	buf    []rune
}

func (p *painter) put(ink int, r rune) {
	if ink != p.ink {
		p.flush()
		p.ink = ink
	}
	p.buf = append(p.buf, r)
}

func (p *painter) flush() {
	if len(p.buf) == 0 {
		return
	}
	if p.ink == 0 {
		p.b.WriteString(string(p.buf))
	} else {
		p.b.WriteString(p.styles[p.ink].Render(string(p.buf)))
	}
	p.buf = p.buf[:0]
}

func (p *painter) String() string {
	p.flush()
	return p.b.String()
}
