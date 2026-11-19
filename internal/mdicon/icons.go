// Package mdicon provides built-in Material Design 24x24 vector icons used by
// md3e components cleanly without init() hooks or global icon registries so
// applications only link the specific icons they reference.
package mdicon

import (
	"sync"

	"github.com/egoist/mygo/ui"
)

const viewBoxSize float32 = 24

// Icon represents an immutable 24x24 viewBox vector icon whose path geometry is
// parsed lazily on first draw.
type Icon struct {
	d       string
	evenOdd bool
	once    sync.Once
	cached  *ui.Path
}

// Path returns a fresh *ui.Path in the canonical 24x24 viewBox coordinate space.
// Callers may trim, transform, or dash the returned path without mutating shared state.
func (ic *Icon) Path() *ui.Path {
	if ic == nil || ic.d == "" {
		return nil
	}
	p := ui.MustParsePath(ic.d)
	if ic.evenOdd {
		p.FillRule(ui.EvenOdd)
	}
	return p
}

func (ic *Icon) parsed() *ui.Path {
	if ic == nil || ic.d == "" {
		return nil
	}
	ic.once.Do(func() {
		ic.cached = ui.MustParsePath(ic.d)
		if ic.evenOdd {
			ic.cached.FillRule(ui.EvenOdd)
		}
	})
	return ic.cached
}

func iconTransform(dst ui.Rect) (ui.Transform, float32, bool) {
	if dst.W <= 0 || dst.H <= 0 {
		return ui.Transform{}, 0, false
	}
	side := min(dst.W, dst.H)
	scale := side / viewBoxSize
	dx := dst.X + (dst.W-side)*0.5
	dy := dst.Y + (dst.H-side)*0.5
	t := ui.Scaling(scale, scale).Then(ui.Translation(dx, dy))
	return t, scale, true
}

// Draw renders the icon scaled proportionally inside dst filled with c.
func (ic *Icon) Draw(p *ui.Painter, dst ui.Rect, c ui.Color) {
	if p == nil || ic == nil || c.A == 0 {
		return
	}
	path := ic.parsed()
	t, _, ok := iconTransform(dst)
	if path == nil || !ok {
		return
	}
	p.Transform(t, 0, 0, func() {
		p.FillPath(path, c)
	})
}

// Stroke renders the icon outline scaled proportionally inside dst with stroke width in DIPs.
func (ic *Icon) Stroke(p *ui.Painter, dst ui.Rect, width float32, c ui.Color) {
	if p == nil || ic == nil || width <= 0 || c.A == 0 {
		return
	}
	path := ic.parsed()
	t, scale, ok := iconTransform(dst)
	if path == nil || !ok || scale <= 0 {
		return
	}
	p.Transform(t, 0, 0, func() {
		p.StrokePath(path, width/scale, c)
	})
}

// Draw is a convenience helper painting ic scaled inside dst with c when ic is non-nil.
func Draw(p *ui.Painter, ic *Icon, dst ui.Rect, c ui.Color) {
	if ic != nil {
		ic.Draw(p, dst, c)
	}
}

// Independent exported icon instances (never grouped into a global slice/map or
// initialized via init(), preserving per-icon dead-code elimination).
var (
	Check = &Icon{
		d: "M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z",
	}
	Minus = &Icon{
		d: "M5 11h14v2H5z",
	}
	Close = &Icon{
		d: "M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z",
	}
	Clear = &Icon{
		d: "M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z",
	}
	ChevronDown = &Icon{
		d: "M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z",
	}
	ChevronUp = &Icon{
		d: "M7.41 15.41L12 10.83l4.59 4.58L18 14l-6-6-6 6 1.41 1.41z",
	}
	ChevronLeft = &Icon{
		d: "M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z",
	}
	ChevronRight = &Icon{
		d: "M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z",
	}
	ArrowBack = &Icon{
		d: "M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z",
	}
	ArrowForward = &Icon{
		d: "M12 4l-1.41 1.41L16.17 11H4v2h12.17l-5.58 5.59L12 20l8-8z",
	}
	Add = &Icon{
		d: "M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z",
	}
	Edit = &Icon{
		d: "M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34a.996.996 0 0 0-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z",
	}
	Search = &Icon{
		d: "M15.5 14h-.79l-.28-.27A6.471 6.471 0 0 0 16 9.5 6.5 6.5 0 1 0 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z",
	}
	Menu = &Icon{
		d: "M3 18h18v-2H3v2zm0-5h18v-2H3v2zm0-7v2h18V6H3z",
	}
	MoreVert = &Icon{
		d: "M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z",
	}
	MoreHoriz = &Icon{
		d: "M6 10c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm12 0c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm-6 0c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z",
	}
	Visibility = &Icon{
		d:       "M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z",
		evenOdd: true,
	}
	VisibilityOff = &Icon{
		d: "M12 7c2.76 0 5 2.24 5 5 0 .65-.13 1.26-.36 1.83l2.92 2.92c1.51-1.26 2.7-2.89 3.43-4.75-1.73-4.39-6-7.5-11-7.5-1.4 0-2.74.25-3.98.7l2.16 2.16C10.74 7.13 11.35 7 12 7zM2 4.27l2.28 2.28.46.46C3.08 8.3 1.78 10.02 1 12c1.73 4.39 6 7.5 11 7.5 1.55 0 3.03-.3 4.38-.84l.42.42L19.73 22 21 20.73 3.27 3 2 4.27zM7.53 9.8l1.55 1.55c-.05.21-.08.43-.08.65 0 1.66 1.34 3 3 3 .22 0 .44-.03.65-.08l1.55 1.55c-.67.33-1.41.53-2.2.53-2.76 0-5-2.24-5-5 0-.79.2-1.53.53-2.2zm4.31-.78l3.15 3.15.02-.16c0-1.66-1.34-3-3-3l-.17.01z",
	}
	Calendar = &Icon{
		d: "M20 3h-1V1h-2v2H7V1H5v2H4c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 18H4V8h16v13z",
	}
	Schedule = &Icon{
		d: schedulePathData,
	}
	Clock = &Icon{
		d: schedulePathData,
	}
	ErrorOutline = &Icon{
		d: "M11 15h2v2h-2zm0-8h2v6h-2zm.99-5C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z",
	}
	Star = &Icon{
		d: "M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z",
	}
)

const schedulePathData = "M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z"
