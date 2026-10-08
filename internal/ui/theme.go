package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// palette is the raw color set. Dark is Tokyo Night; light is its Day variant.
type palette struct {
	fg, muted, dim, faint, surface, onAccent color.Color
	accent, accentSoft, accent2, cyan        color.Color
	good, warn, bad                          color.Color
	heat                                     [heatLevels]color.Color // worst → best
}

const heatLevels = 5

var darkPalette = palette{
	fg:         lipgloss.Color("#C0CAF5"),
	muted:      lipgloss.Color("#A9B1D6"),
	dim:        lipgloss.Color("#565F89"),
	faint:      lipgloss.Color("#3B4261"),
	surface:    lipgloss.Color("#292E42"),
	onAccent:   lipgloss.Color("#1A1B26"),
	accent:     lipgloss.Color("#7AA2F7"),
	accentSoft: lipgloss.Color("#2F3F6B"),
	accent2:    lipgloss.Color("#BB9AF7"),
	cyan:       lipgloss.Color("#7DCFFF"),
	good:       lipgloss.Color("#9ECE6A"),
	warn:       lipgloss.Color("#E0AF68"),
	bad:        lipgloss.Color("#F7768E"),
	heat: [heatLevels]color.Color{
		lipgloss.Color("#F7768E"),
		lipgloss.Color("#FF9E64"),
		lipgloss.Color("#E0AF68"),
		lipgloss.Color("#B9D98A"),
		lipgloss.Color("#9ECE6A"),
	},
}

var lightPalette = palette{
	fg:         lipgloss.Color("#343B58"),
	muted:      lipgloss.Color("#4C505E"),
	dim:        lipgloss.Color("#8990B3"),
	faint:      lipgloss.Color("#C4C8DA"),
	surface:    lipgloss.Color("#DFE1EA"),
	onAccent:   lipgloss.Color("#FFFFFF"),
	accent:     lipgloss.Color("#2E7DE9"),
	accentSoft: lipgloss.Color("#B7CCF2"),
	accent2:    lipgloss.Color("#9854F1"),
	cyan:       lipgloss.Color("#007197"),
	good:       lipgloss.Color("#587539"),
	warn:       lipgloss.Color("#8C6C3E"),
	bad:        lipgloss.Color("#F52A65"),
	heat: [heatLevels]color.Color{
		lipgloss.Color("#F52A65"),
		lipgloss.Color("#E8700A"),
		lipgloss.Color("#C29A2F"),
		lipgloss.Color("#7FA14A"),
		lipgloss.Color("#4E7A2E"),
	},
}

// theme is the palette plus the styles built from it. Screens read styles
// from here rather than constructing their own, so the look stays consistent.
type theme struct {
	palette palette

	text, muted, dim, faint lipgloss.Style
	accent, good, warn, bad lipgloss.Style
	bold, title             lipgloss.Style
	keycap, pill            lipgloss.Style

	// Typing screen.
	pending, typed, wrong, cursor lipgloss.Style
}

func newTheme(isDark bool) *theme {
	p := lightPalette
	if isDark {
		p = darkPalette
	}
	fg := func(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }
	return &theme{
		palette: p,
		text:    fg(p.fg),
		muted:   fg(p.muted),
		dim:     fg(p.dim),
		faint:   fg(p.faint),
		accent:  fg(p.accent),
		good:    fg(p.good),
		warn:    fg(p.warn),
		bad:     fg(p.bad),
		bold:    fg(p.fg).Bold(true),
		title:   fg(p.accent).Bold(true),
		keycap:  fg(p.fg).Background(p.surface).Bold(true).Padding(0, 1),
		pill:    fg(p.onAccent).Background(p.accent).Bold(true),
		pending: fg(p.dim),
		typed:   fg(p.fg),
		wrong:   fg(p.bad).Underline(true),
		cursor:  fg(p.onAccent).Background(p.accent),
	}
}

// heatStyle colors a keycap for heat level 0 (worst) to heatLevels-1 (best);
// a negative level means "no data".
func (t *theme) heatStyle(level int) lipgloss.Style {
	if level < 0 {
		return lipgloss.NewStyle().Foreground(t.palette.dim).Background(t.palette.surface)
	}
	return lipgloss.NewStyle().Foreground(t.palette.onAccent).Background(t.palette.heat[level]).Bold(true)
}

// heatText colors text for a heat level, for numbers next to keys.
func (t *theme) heatText(level int) lipgloss.Style {
	if level < 0 {
		return t.dim
	}
	return lipgloss.NewStyle().Foreground(t.palette.heat[level])
}
