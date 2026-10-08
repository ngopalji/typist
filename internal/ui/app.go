// Package ui is typist's terminal interface, built on Bubble Tea.
//
// App is the root model. It owns shared state and routes between screens
// (home, typing, results, stats), each of which is a small model of its own.
// Screens never talk to each other directly: they emit navigation messages
// that App handles, and all I/O happens in commands, off the UI loop.
package ui

import (
	"context"
	"math/rand/v2"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/nihaar/typist/internal/content"
	"github.com/nihaar/typist/internal/stats"
	"github.com/nihaar/typist/internal/typing"
)

// Store is the persistence the UI needs. *store.Store satisfies it.
type Store interface {
	Save(ctx context.Context, r typing.Result) (int64, error)
	All(ctx context.Context) ([]typing.Result, error)
}

// Screen picks what the app opens on.
type Screen int

const (
	HomeScreen Screen = iota
	StatsScreen
)

// Options configures an App.
type Options struct {
	Store  Store
	Start  Screen
	DBPath string           // shown on the stats screen
	Now    func() time.Time // defaults to time.Now; override in tests
	Rand   *rand.Rand       // defaults to a randomly seeded source
}

// saveState tracks persisting the most recent result.
type saveState int

const (
	saveIdle saveState = iota
	saving
	saved
	saveFailed
)

// shared is state every screen reads. Only App mutates it.
type shared struct {
	theme         *theme
	width, height int
	sessions      []stats.Session // all history, oldest first
	loaded        bool
	loadErr       error
	save          saveState
	saveErr       error
	dbPath        string
	now           func() time.Time
	rng           *rand.Rand
}

// screen is one page of the app.
type screen interface {
	init() tea.Cmd
	update(tea.Msg) (screen, tea.Cmd)
	view() string
}

// Messages screens and commands send to App.
type (
	startTestMsg struct {
		mode   content.Mode
		length typing.Length
	}
	testDoneMsg struct{ result typing.Result }
	goHomeMsg   struct{}
	goStatsMsg  struct{}
	historyMsg  struct {
		sessions []stats.Session
		err      error
	}
	savedMsg struct {
		session stats.Session
		err     error
	}
	// relayoutMsg tells the current screen that size, theme, or data changed.
	relayoutMsg struct{}
)

func send(msg tea.Msg) tea.Cmd { return func() tea.Msg { return msg } }

// App is the root Bubble Tea model.
type App struct {
	sh     *shared
	store  Store
	home   *homeScreen // kept so the menu remembers its selection
	screen screen
}

// New returns the root model.
func New(opts Options) *App {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Rand == nil {
		opts.Rand = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	sh := &shared{
		theme:  newTheme(true),
		dbPath: opts.DBPath,
		now:    opts.Now,
		rng:    opts.Rand,
	}
	a := &App{sh: sh, store: opts.Store, home: newHomeScreen(sh)}
	a.screen = a.home
	if opts.Start == StatsScreen {
		a.screen = newStatsScreen(sh)
	}
	return a
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, a.loadHistory(), a.screen.init())
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, quitKey) {
			return a, tea.Quit
		}
	case tea.WindowSizeMsg:
		a.sh.width, a.sh.height = msg.Width, msg.Height
		return a, a.relayout()
	case tea.BackgroundColorMsg:
		a.sh.theme = newTheme(msg.IsDark())
		return a, a.relayout()

	case historyMsg:
		a.sh.loaded, a.sh.loadErr, a.sh.sessions = true, msg.err, msg.sessions
		return a, a.relayout()
	case savedMsg:
		if msg.err != nil {
			a.sh.save, a.sh.saveErr = saveFailed, msg.err
		} else {
			a.sh.save = saved
			a.sh.sessions = append(a.sh.sessions, msg.session)
		}
		return a, a.relayout()

	case startTestMsg:
		return a, a.show(newTyperScreen(a.sh, msg.mode, msg.length))
	case testDoneMsg:
		session := stats.Analyze(msg.result)
		a.sh.save, a.sh.saveErr = saving, nil
		return a, tea.Batch(a.show(newResultsScreen(a.sh, session)), a.saveSession(session))
	case goHomeMsg:
		return a, a.show(a.home)
	case goStatsMsg:
		return a, a.show(newStatsScreen(a.sh))
	}

	var cmd tea.Cmd
	a.screen, cmd = a.screen.update(msg)
	return a, cmd
}

func (a *App) View() tea.View {
	var body string
	switch {
	case a.sh.width == 0:
		// Wait for the first WindowSizeMsg before drawing anything.
	case a.sh.width < minWidth || a.sh.height < minHeight:
		body = lipgloss.Place(a.sh.width, a.sh.height, lipgloss.Center, lipgloss.Center,
			a.sh.theme.dim.Render("make the window a little bigger"))
	default:
		body = a.screen.view()
	}
	v := tea.NewView(body)
	v.AltScreen = true
	v.WindowTitle = "typist"
	return v
}

const minWidth, minHeight = 50, 16

// show switches to s and lays it out for the current window.
func (a *App) show(s screen) tea.Cmd {
	a.screen = s
	return tea.Batch(s.init(), a.relayout())
}

func (a *App) relayout() tea.Cmd {
	var cmd tea.Cmd
	a.screen, cmd = a.screen.update(relayoutMsg{})
	return cmd
}

func (a *App) loadHistory() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		results, err := a.store.All(ctx)
		sessions := make([]stats.Session, len(results))
		for i, r := range results {
			sessions[i] = stats.Analyze(r)
		}
		return historyMsg{sessions: sessions, err: err}
	}
}

func (a *App) saveSession(s stats.Session) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := a.store.Save(ctx, s.Result)
		return savedMsg{session: s, err: err}
	}
}

// frame lays out a screen: body centered in the window, help bar pinned to
// the bottom.
func (sh *shared) frame(body, footer string) string {
	h := max(0, sh.height-2)
	top := lipgloss.Place(sh.width, h, lipgloss.Center, lipgloss.Center, padBlock(body))
	footer = ansi.Truncate(footer, sh.width-2, "…")
	return top + "\n\n" + lipgloss.PlaceHorizontal(sh.width, lipgloss.Center, footer)
}

// bodyHeight is the space frame gives a screen's body.
func (sh *shared) bodyHeight() int { return max(0, sh.height-2) }

// sessionsFor returns history for one mode, or all of it for mode "".
func (sh *shared) sessionsFor(mode content.Mode) []stats.Session {
	if mode == "" {
		return sh.sessions
	}
	var out []stats.Session
	for _, s := range sh.sessions {
		if s.Mode == mode {
			out = append(out, s)
		}
	}
	return out
}
