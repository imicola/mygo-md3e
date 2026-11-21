package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// TypeStyleMetrics holds resolved typographic measurements in DIPs after
// applying the active Theme scale and accessibility TextScale.
type TypeStyleMetrics struct {
	Size       float32
	LineHeight float32
	Weight     int
	Tracking   float32
	Font       string
}

// TypeStyleOf resolves the MD3E TypeStyleMetrics for role (standard or
// emphasized) from CurrentTheme(c), scaled by the theme's EffectiveTextScale.
func TypeStyleOf(c *ui.Context, role tokens.TypeRole, emphasized ...bool) TypeStyleMetrics {
	return typeStyleOfWithTheme(CurrentTheme(c), role, emphasized...)
}

func typeStyleOfWithTheme(th *Theme, role tokens.TypeRole, emphasized ...bool) TypeStyleMetrics {
	if th == nil {
		th = DefaultTheme(nil)
	}
	ts := th.TypeStyle(role, emphasized...)
	scale := th.EffectiveTextScale()
	return TypeStyleMetrics{
		Size:       float32(ts.Size) * scale,
		LineHeight: float32(ts.LineHeight) * scale,
		Weight:     ts.Weight,
		Tracking:   float32(ts.Tracking) * scale,
		Font:       th.Font,
	}
}

// ApplyTypeStyle applies the MD3E typographic scale tokens for role (size,
// weight, fixed DIP line height, letter spacing, and optional font family) to el
// without altering el's foreground TextColor so parent Surface inheritance works.
func ApplyTypeStyle(c *ui.Context, el ui.Element, role tokens.TypeRole, emphasized ...bool) ui.Element {
	return applyTypeStyleWithTheme(CurrentTheme(c), el, role, emphasized...)
}

func applyTypeStyleWithTheme(th *Theme, el ui.Element, role tokens.TypeRole, emphasized ...bool) ui.Element {
	m := typeStyleOfWithTheme(th, role, emphasized...)
	el.FontSize(m.Size).
		FontWeight(m.Weight).
		FixedLineHeight(m.LineHeight).
		LetterSpacing(m.Tracking)
	if m.Font != "" {
		el.Font(m.Font)
	}
	return el
}

// Text builds a ui.Text element styled with the MD3E typography role and
// optional emphasized weight from CurrentTheme(c). At most one emphasized flag
// may be supplied.
func Text(c *ui.Context, s string, role tokens.TypeRole, emphasized ...bool) ui.Element {
	return ApplyTypeStyle(c, ui.Text(c, s), role, emphasized...)
}
