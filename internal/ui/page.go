package ui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// page shows a body that may be taller than the window: centered when it
// fits, scrollable in a viewport when it doesn't.
type page struct {
	vp            viewport.Model
	body          string
	fits          bool
	width, height int
}

func newPage() page { return page{vp: viewport.New()} }

// set replaces the body, keeping the scroll position.
func (p *page) set(body string, width, height int) {
	offset := p.vp.YOffset()
	p.body, p.width, p.height = body, width, height
	p.fits = lipgloss.Height(body) <= height
	p.vp.SetWidth(width)
	p.vp.SetHeight(height)
	p.vp.SetContent(center(width, body))
	p.vp.SetYOffset(offset)
}

// scroll handles scroll keys, reporting whether msg was one.
func (p *page) scroll(msg tea.KeyPressMsg, k scrollKeys) bool {
	if p.fits {
		return false
	}
	switch {
	case key.Matches(msg, k.Up):
		p.vp.ScrollUp(1)
	case key.Matches(msg, k.Down):
		p.vp.ScrollDown(1)
	case key.Matches(msg, k.HalfUp):
		p.vp.HalfPageUp()
	case key.Matches(msg, k.HalfDown):
		p.vp.HalfPageDown()
	case key.Matches(msg, k.Top):
		p.vp.GotoTop()
	case key.Matches(msg, k.Bottom):
		p.vp.GotoBottom()
	default:
		return false
	}
	return true
}

func (p *page) view() string {
	if p.fits {
		return lipgloss.Place(p.width, p.height, lipgloss.Center, lipgloss.Center, padBlock(p.body))
	}
	return p.vp.View()
}

// indicator shows scroll position when the body overflows.
func (p *page) indicator(t *theme) string {
	if p.fits {
		return ""
	}
	return t.faint.Render(fmt.Sprintf("%.0f%%", 100*p.vp.ScrollPercent())) + "   "
}
