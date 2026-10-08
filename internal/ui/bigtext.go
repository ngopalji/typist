package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// glyphs is a 5-pixel-tall block font covering what the UI draws big: the
// logo and numbers.
var glyphs = map[rune][5]string{
	'T': {"#####", "..#..", "..#..", "..#..", "..#.."},
	'Y': {"#...#", ".#.#.", "..#..", "..#..", "..#.."},
	'P': {"####.", "#...#", "####.", "#....", "#...."},
	'I': {"###", ".#.", ".#.", ".#.", "###"},
	'S': {".####", "#....", ".###.", "....#", "####."},
	'0': {".##.", "#..#", "#..#", "#..#", ".##."},
	'1': {".#.", "##.", ".#.", ".#.", "###"},
	'2': {"###.", "...#", ".##.", "#...", "####"},
	'3': {"###.", "...#", ".##.", "...#", "###."},
	'4': {"#..#", "#..#", "####", "...#", "...#"},
	'5': {"####", "#...", "###.", "...#", "###."},
	'6': {".##.", "#...", "###.", "#..#", ".##."},
	'7': {"####", "...#", "..#.", ".#..", ".#.."},
	'8': {".##.", "#..#", ".##.", "#..#", ".##."},
	'9': {".##.", "#..#", ".###", "...#", ".##."},
	'.': {".", ".", ".", ".", "#"},
	'-': {"...", "...", "###", "...", "..."},
	' ': {"..", "..", "..", "..", ".."},
}

// pixels lays s out as a 5-row bitmap with one blank column between glyphs.
func pixels(s string) [5][]bool {
	var rows [5][]bool
	for i, r := range s {
		g, ok := glyphs[r]
		if !ok {
			continue
		}
		for y := range rows {
			if i > 0 {
				rows[y] = append(rows[y], false)
			}
			for _, c := range g[y] {
				rows[y] = append(rows[y], c == '#')
			}
		}
	}
	return rows
}

// bigTextWidth returns how many cells bigText(s) occupies.
func bigTextWidth(s string) int { return len(pixels(s)[0]) * 2 }

// bigText draws s in the block font. Each pixel is two cells wide so it looks
// square. The gradient stops color the text left to right.
func bigText(s string, stops ...color.Color) string {
	rows := pixels(s)
	width := len(rows[0])
	if width == 0 {
		return ""
	}
	colors := gradient(width, stops...)
	lines := make([]string, len(rows))
	for y, row := range rows {
		var b strings.Builder
		for x, on := range row {
			if on {
				b.WriteString(lipgloss.NewStyle().Foreground(colors[x]).Render("██"))
			} else {
				b.WriteString("  ")
			}
		}
		lines[y] = b.String()
	}
	return strings.Join(lines, "\n")
}

// smallText draws s in the same font at half the size, using half-block
// characters so two pixel rows share one line of text.
func smallText(s string, stops ...color.Color) string {
	rows := pixels(s)
	width := len(rows[0])
	if width == 0 {
		return ""
	}
	colors := gradient(width, stops...)
	at := func(y, x int) bool { return y < len(rows) && rows[y][x] }
	var lines []string
	for y := 0; y < len(rows); y += 2 {
		var b strings.Builder
		for x := range width {
			ch := " "
			switch top, bottom := at(y, x), at(y+1, x); {
			case top && bottom:
				ch = "█"
			case top:
				ch = "▀"
			case bottom:
				ch = "▄"
			}
			b.WriteString(lipgloss.NewStyle().Foreground(colors[x]).Render(ch))
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func gradient(n int, stops ...color.Color) []color.Color {
	if len(stops) == 1 || n == 1 {
		out := make([]color.Color, n)
		for i := range out {
			out[i] = stops[0]
		}
		return out
	}
	return lipgloss.Blend1D(n, stops...)
}
