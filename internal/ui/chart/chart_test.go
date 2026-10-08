package chart

import (
	"math"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func frame(w, h int) Frame {
	return Frame{Width: w, Height: h, Min: 0, Max: 100, LabelWidth: 3, XLabels: []string{"0s", "30s", "60s"}}
}

// checkShape asserts every line of out is exactly width cells wide.
func checkShape(t *testing.T, out string, lines, width int) {
	t.Helper()
	got := strings.Split(out, "\n")
	if len(got) != lines {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), lines, out)
	}
	for i, l := range got {
		if w := lipgloss.Width(l); w != width {
			t.Fatalf("line %d is %d wide, want %d: %q", i, w, width, l)
		}
	}
}

func TestLineShape(t *testing.T) {
	values := []float64{10, 40, 80, math.NaN(), 60, 90, 20}
	c := Line{Frame: frame(30, 6), Fill: true, ShowRef: true, Ref: 50, Marks: []int{2, 5}, MarkLabel: "err"}
	// 6 plot rows + marks row + axis + x labels; gutter is LabelWidth + 2.
	checkShape(t, c.Render(values), 9, 30+5)
}

func TestLineDrawsSomething(t *testing.T) {
	out := Line{Frame: frame(10, 3)}.Render([]float64{0, 100})
	dots := 0
	for _, r := range out {
		if r > 0x2800 && r <= 0x28FF {
			dots++
		}
	}
	if dots < 3 {
		t.Fatalf("expected a diagonal of braille dots, got %d:\n%s", dots, out)
	}
}

func TestBarsShape(t *testing.T) {
	values := []float64{100, 95, math.NaN(), 80, 50}
	c := Bars{Frame: frame(20, 4)}
	checkShape(t, c.Render(values), 6, 20+5)
}

func TestBarsTopRowOnlyForTallBars(t *testing.T) {
	c := Bars{Frame: Frame{Width: 2, Height: 2, Min: 0, Max: 100, LabelWidth: 3}}
	rows := strings.Split(c.Render([]float64{100, 10}), "\n")
	top := []rune(rows[0])
	// Gutter is 5 cells; the 100 bar fills the top row, the 10 bar doesn't.
	if top[5] != '█' || top[6] != ' ' {
		t.Fatalf("top row = %q", rows[0])
	}
}

func TestResample(t *testing.T) {
	got := resample([]float64{1, 3, 5, 7}, 2)
	if got[0] != 2 || got[1] != 6 {
		t.Fatalf("downsample = %v", got)
	}
	got = resample([]float64{1, 2}, 4)
	if got[0] != 1 || got[1] != 1 || got[2] != 2 || got[3] != 2 {
		t.Fatalf("upsample = %v", got)
	}
}

func TestNiceMax(t *testing.T) {
	for in, want := range map[float64]float64{7: 10, 43: 50, 87: 100, 120: 150, 180: 200} {
		if got := NiceMax(in); got != want {
			t.Errorf("NiceMax(%v) = %v, want %v", in, got, want)
		}
	}
}
