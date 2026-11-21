// Package symbols provides 220+ zero-bloat rounded 24x24 vector UI symbols for
// MyGo UI and mygo-md3e.
//
// Zero-bloat / dead-code elimination (DCE) guarantee:
//   - No package init() functions.
//   - No package-level slice, map, or registry referencing all icons.
//   - No package-level sync.OnceValue closures.
//   - Each exported symbol is an independent *Symbol variable whose *ui.Path is
//     parsed lazily via a zero-value sync.Once on first use.
//
// Because (*Symbol).Draw has signature func(*ui.Painter, ui.Rect, ui.Color),
// any symbol method value (such as symbols.Home.Draw) passes directly wherever
// md3e.IconDraw is accepted without importing the root md3e package.
package symbols

import (
	"sync"

	"github.com/egoist/mygo/ui"
)

const viewBoxSize float32 = 24

// Symbol is an immutable 24x24 viewBox vector icon whose *ui.Path geometry is
// parsed lazily on first draw and cached for reuse.
type Symbol struct {
	d       string
	evenOdd bool
	once    sync.Once
	cached  *ui.Path
}

// Custom constructs a Symbol from custom 24x24 viewBox SVG path data using the
// non-zero winding fill rule.
func Custom(svgPath string) *Symbol {
	return &Symbol{d: svgPath}
}

// CustomEvenOdd constructs a Symbol from custom 24x24 viewBox SVG path data
// using the even-odd fill rule.
func CustomEvenOdd(svgPath string) *Symbol {
	return &Symbol{d: svgPath, evenOdd: true}
}

// Path returns a fresh *ui.Path in the canonical 24x24 viewBox coordinate
// space. Callers may trim, dash, or transform the returned path without
// mutating shared cached state. Returns nil if s is nil, empty, or invalid.
func (s *Symbol) Path() *ui.Path {
	if s == nil || s.d == "" {
		return nil
	}
	p, err := ui.ParsePath(s.d)
	if err != nil {
		return nil
	}
	if s.evenOdd {
		p.FillRule(ui.EvenOdd)
	}
	return p
}

func (s *Symbol) parsed() *ui.Path {
	if s == nil || s.d == "" {
		return nil
	}
	s.once.Do(func() {
		p, err := ui.ParsePath(s.d)
		if err != nil {
			return
		}
		if s.evenOdd {
			p.FillRule(ui.EvenOdd)
		}
		s.cached = p
	})
	return s.cached
}

func symbolTransform(dst ui.Rect) (ui.Transform, float32, bool) {
	if dst.W <= 0 || dst.H <= 0 {
		return ui.Transform{}, 0, false
	}
	side := min(dst.W, dst.H)
	scale := side / viewBoxSize
	dx := dst.X + (dst.W-side)*0.5
	dy := dst.Y + (dst.H-side)*0.5
	return ui.Scaling(scale, scale).Then(ui.Translation(dx, dy)), scale, true
}

// Draw renders the symbol scaled proportionally and centered inside dst, filled
// with c. Its method value (e.g. symbols.Home.Draw) matches md3e.IconDraw.
func (s *Symbol) Draw(p *ui.Painter, dst ui.Rect, c ui.Color) {
	if p == nil || s == nil || c.A == 0 {
		return
	}
	path := s.parsed()
	t, _, ok := symbolTransform(dst)
	if path == nil || !ok {
		return
	}
	p.Transform(t, 0, 0, func() {
		p.FillPath(path, c)
	})
}

// Stroke renders the symbol outline scaled proportionally and centered inside
// dst with stroke width in DIPs.
func (s *Symbol) Stroke(p *ui.Painter, dst ui.Rect, width float32, c ui.Color) {
	if p == nil || s == nil || width <= 0 || c.A == 0 {
		return
	}
	path := s.parsed()
	t, scale, ok := symbolTransform(dst)
	if path == nil || !ok || scale <= 0 {
		return
	}
	p.Transform(t, 0, 0, func() {
		p.StrokePath(path, width/scale, c)
	})
}
