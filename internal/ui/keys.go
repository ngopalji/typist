package ui

import "charm.land/bubbles/v2/key"

// Each screen has its own keymap. Keymaps implement help.KeyMap, which is what
// the help bar renders from, so bindings and their hints never drift apart.

var quitKey = key.NewBinding(key.WithKeys("ctrl+c"))

type homeKeys struct {
	Up, Down, Left, Right, Start, Stats, Quit key.Binding
}

func newHomeKeys() homeKeys {
	return homeKeys{
		Up:    key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("j/k", "mode")),
		Down:  key.NewBinding(key.WithKeys("j", "down")),
		Left:  key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/l", "length")),
		Right: key.NewBinding(key.WithKeys("l", "right")),
		Start: key.NewBinding(key.WithKeys("enter", "space"), key.WithHelp("enter", "start")),
		Stats: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stats")),
		Quit:  key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
	}
}

func (k homeKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Left, k.Start, k.Stats, k.Quit}
}
func (k homeKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// typerKeys can't use letters: every printable key is input while typing.
type typerKeys struct {
	Restart, Back, DeleteChar, DeleteWord key.Binding
}

func newTyperKeys() typerKeys {
	return typerKeys{
		Restart:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "restart")),
		Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		DeleteChar: key.NewBinding(key.WithKeys("backspace")),
		DeleteWord: key.NewBinding(
			key.WithKeys("ctrl+w", "alt+backspace", "ctrl+backspace", "ctrl+h"),
			key.WithHelp("ctrl+w", "delete word"),
		),
	}
}

func (k typerKeys) ShortHelp() []key.Binding { return []key.Binding{k.Restart, k.Back} }
func (k typerKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp(), {k.DeleteWord}}
}

// scrollKeys are shared by every scrollable page.
type scrollKeys struct {
	Up, Down, HalfUp, HalfDown, Top, Bottom key.Binding
}

func newScrollKeys() scrollKeys {
	return scrollKeys{
		Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("j/k", "scroll")),
		Down:     key.NewBinding(key.WithKeys("j", "down")),
		HalfUp:   key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
		HalfDown: key.NewBinding(key.WithKeys("ctrl+d", "pgdown")),
		Top:      key.NewBinding(key.WithKeys("g", "home")),
		Bottom:   key.NewBinding(key.WithKeys("G", "end")),
	}
}

func (k *scrollKeys) setEnabled(on bool) {
	for _, b := range []*key.Binding{&k.Up, &k.Down, &k.HalfUp, &k.HalfDown, &k.Top, &k.Bottom} {
		b.SetEnabled(on)
	}
}

type resultsKeys struct {
	scrollKeys
	Again, Back, Stats, Metric, Quit key.Binding
}

func newResultsKeys() resultsKeys {
	return resultsKeys{
		scrollKeys: newScrollKeys(),
		Again:      key.NewBinding(key.WithKeys("enter", "r"), key.WithHelp("enter", "again")),
		Back:       key.NewBinding(key.WithKeys("b", "esc"), key.WithHelp("b", "back")),
		Stats:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stats")),
		Metric:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "heatmap")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

func (k resultsKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Again, k.Back, k.Stats, k.Metric, k.Up, k.Quit}
}
func (k resultsKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

type statsKeys struct {
	scrollKeys
	Prev, Next, Metric, Back, Quit key.Binding
}

func newStatsKeys() statsKeys {
	return statsKeys{
		scrollKeys: newScrollKeys(),
		Prev:       key.NewBinding(key.WithKeys("h", "left", "shift+tab"), key.WithHelp("h/l", "mode")),
		Next:       key.NewBinding(key.WithKeys("l", "right", "tab")),
		Metric:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "heatmap")),
		Back:       key.NewBinding(key.WithKeys("b", "esc"), key.WithHelp("b", "back")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

func (k statsKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Prev, k.Up, k.Metric, k.Back, k.Quit}
}
func (k statsKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }
