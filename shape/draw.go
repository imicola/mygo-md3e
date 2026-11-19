package shape

import (
	"github.com/egoist/mygo/ui"
)

// DrawMorph fills the morph between two shapes at progress, from 0 to 1, in
// dst, in the color c.
func DrawMorph(p *ui.Painter, m *Morph, progress float32, dst ui.Rect, c ui.Color) {
	p.FillPath(m.AsPath(progress, dst), c)
}

// ClipShape runs fn with the painter clipped to the polygon drawn into dst.
// The polygon is normalized, as the built-in shapes are.
func ClipShape(p *ui.Painter, poly *RoundedPolygon, dst ui.Rect, fn func()) {
	p.ClipPath(poly.ToPath(dst), fn)
}
