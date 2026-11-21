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

// TruncateMode selects where the ellipsis appears when text is clipped to a
// maximum line count.
type TruncateMode int

const (
	// TruncateEnd places the ellipsis at the end of the clipped text (default).
	TruncateEnd TruncateMode = iota
	// TruncateMiddle keeps both the start and end of the clipped text, placing
	// the ellipsis in the middle of the last line (ideal for file paths/IDs).
	TruncateMiddle
	// TruncateStart keeps the end of the clipped text, placing the ellipsis at
	// the start of the last line.
	TruncateStart
)

// TextOptions configures TextWith. When TextWith is called without options,
// tokens.BodyMedium is used. When a TextOptions value is passed, Role selects
// the MD3E type scale role (note: the zero value of tokens.TypeRole is
// tokens.DisplayLarge; out-of-range roles fall back to tokens.BodyMedium).
type TextOptions struct {
	// Role selects the MD3E typography scale role.
	Role tokens.TypeRole
	// Emphasized switches to the emphasized weight variant of Role.
	Emphasized bool
	// MaxLines limits the text to at most n lines with ellipsis when positive.
	MaxLines int
	// Truncate chooses end, middle, or start ellipsis placement. Setting
	// TruncateMiddle or TruncateStart without MaxLines applies single-line
	// truncation automatically.
	Truncate TruncateMode
	// Mono switches the text to CurrentTheme(c).MonoFont when configured, or
	// the system "monospace" font family otherwise.
	Mono bool
	// TabularNums enables OpenType tabular lining figures ("tnum") so digits
	// share uniform width across live counters, tables, and metrics.
	TabularNums bool
	// Color overrides the foreground text color when non-transparent (A > 0).
	Color ui.Color
}

// TabularNums enables OpenType tabular lining figures ("tnum") on el so digits
// share a uniform advance width for aligned metrics, timestamps, and tables.
func TabularNums(el ui.Element) ui.Element {
	return el.FontFeatures("tnum")
}

// Text builds a ui.Text element styled with the MD3E typography role and
// optional emphasized weight from CurrentTheme(c). At most one emphasized flag
// may be supplied.
func Text(c *ui.Context, s string, role tokens.TypeRole, emphasized ...bool) ui.Element {
	return ApplyTypeStyle(c, ui.Text(c, s), role, emphasized...)
}

// TextWith builds a ui.Text element styled with MD3E typography, optional
// monospace font family, tabular numerals, line clamping/truncation, and
// foreground color override. At most one TextOptions value may be passed.
func TextWith(c *ui.Context, s string, opts ...TextOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: TextWith takes at most one TextOptions")
	}
	role := tokens.BodyMedium
	var o TextOptions
	if len(opts) == 1 {
		o = opts[0]
		role = o.Role
		if role < 0 || role >= tokens.TypeRoleCount {
			role = tokens.BodyMedium
		}
	}

	th := CurrentTheme(c)
	el := applyTypeStyleWithTheme(th, ui.Text(c, s), role, o.Emphasized)

	if o.Mono {
		if th.MonoFont != "" {
			el.Font(th.MonoFont)
		} else {
			el.Font("monospace")
		}
	}
	if o.TabularNums {
		TabularNums(el)
	}
	if o.MaxLines > 0 {
		el.MaxLines(o.MaxLines)
	}
	switch o.Truncate {
	case TruncateMiddle:
		if o.MaxLines <= 0 {
			el.SingleLine()
		}
		el.TruncateMiddle()
	case TruncateStart:
		if o.MaxLines <= 0 {
			el.SingleLine()
		}
		el.TruncateStart()
	}
	if o.Color.A > 0 {
		el.TextColor(o.Color)
	}
	return el
}
