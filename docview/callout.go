package docview

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// CalloutKind selects the semantic style of docview.Callout, matching GitHub's
// five callout types (Note, Tip, Important, Warning, Caution).
type CalloutKind int

const (
	// CalloutNote highlights neutral or informational guidance (PrimaryContainer).
	CalloutNote CalloutKind = iota
	// CalloutTip highlights helpful recommendations or shortcuts (SuccessContainer).
	CalloutTip
	// CalloutImportant highlights key details users should not overlook (TertiaryContainer).
	CalloutImportant
	// CalloutWarning highlights potential pitfalls or breaking behavior (WarningContainer).
	CalloutWarning
	// CalloutCaution highlights destructive or high-risk actions (ErrorContainer).
	CalloutCaution
)

// String returns the canonical display title for k.
func (k CalloutKind) String() string {
	switch k {
	case CalloutTip:
		return "Tip"
	case CalloutImportant:
		return "Important"
	case CalloutWarning:
		return "Warning"
	case CalloutCaution:
		return "Caution"
	default:
		return "Note"
	}
}

// CalloutKindFromAlert converts a markdown parser.Alert value into CalloutKind.
func CalloutKindFromAlert(a parser.Alert) (CalloutKind, bool) {
	switch a {
	case parser.AlertNote:
		return CalloutNote, true
	case parser.AlertTip:
		return CalloutTip, true
	case parser.AlertImportant:
		return CalloutImportant, true
	case parser.AlertWarning:
		return CalloutWarning, true
	case parser.AlertCaution:
		return CalloutCaution, true
	default:
		return CalloutNote, false
	}
}

// CalloutOptions configures docview.Callout.
type CalloutOptions struct {
	// Title overrides the default kind label ("Note", "Tip", etc.).
	Title string
	// Text renders body text when the body callback passed to Callout is nil.
	Text string
	// Icon overrides the default semantic icon for kind.
	Icon md3e.IconDraw
	// ShapeScale selects the MD3E corner token (default tokens.ShapeMedium).
	ShapeScale tokens.ShapeScale
	// Radius overrides the corner radius in DIP when positive and finite.
	Radius float32
	// LiveAlert sets ui.RoleAlert instead of ui.RoleRegion so screen readers
	// announce dynamic callouts immediately.
	LiveAlert bool
	// Label overrides the accessible region label (defaults to the resolved title).
	Label string
}

type calloutPalette struct {
	container   ui.Color
	onContainer ui.Color
	accent      ui.Color
	headerInk   ui.Color
	border      ui.Color
	icon        md3e.IconDraw
	defaultName string
}

// Material Rounded 24x24 paths allocated lazily per symbol via symbols.Custom (zero init).
var (
	calloutTipSymbol       = symbols.Custom("M12 2.5A6.5 6.5 0 0 0 7.7 13.9C8.4 14.6 8.8 15.4 9 16.2H15C15.2 15.4 15.6 14.6 16.3 13.9A6.5 6.5 0 0 0 12 2.5ZM9.5 18A1 1 0 0 0 9.5 20H14.5A1 1 0 0 0 14.5 18ZM10.5 21A1 1 0 0 0 10.5 22.5H13.5A1 1 0 0 0 13.5 21Z")
	calloutImportantSymbol = symbols.CustomEvenOdd("M4 3H20A2 2 0 0 1 22 5V16A2 2 0 0 1 20 18H8L4 21.5V5A2 2 0 0 1 4 3ZM11 6.5V11.5H13V6.5ZM11 13.5V15.5H13V13.5Z")
)

// Callout builds an MD3E semantic callout container with a leading accent bar,
// harmonized container palette, icon, emphasized title, and RTL-aware padding.
func Callout(c *ui.Context, kind CalloutKind, body func(), opts ...CalloutOptions) ui.Element {
	if len(opts) > 1 {
		panic("docview: Callout takes at most one CalloutOptions")
	}
	var o CalloutOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := md3e.CurrentTheme(c)
	pal := calloutPaletteFor(th, kind)
	title := o.Title
	if title == "" {
		title = pal.defaultName
	}
	icon := o.Icon
	if icon == nil {
		icon = pal.icon
	}

	radius := resolveCodeRadius(th, o.ShapeScale, o.Radius)
	padScale := float32(th.Density.PaddingScale())
	padStart := float32(16) * padScale
	padEnd := float32(14) * padScale
	padY := float32(12) * padScale
	stripeW := float32(3.5)
	stripeInsetX := float32(1.5)

	role := ui.RoleRegion
	if o.LiveAlert {
		role = ui.RoleAlert
	}
	accessLabel := o.Label
	if accessLabel == "" {
		accessLabel = title
	}
	rtl := c.Direction() == ui.RightToLeft

	box := ui.Column(c).
		AlignSelf(ui.Stretch).
		Padding(padY, 0, padY, 0).
		PaddingStart(padStart + stripeW + stripeInsetX).
		PaddingEnd(padEnd).
		Gap(6).
		Radius(radius).
		Background(pal.container).
		TextColor(pal.onContainer).
		Border(1, pal.border).
		Clip().
		DimDisabled(false).
		Role(role).
		Label(accessLabel)

	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 || pal.accent.A == 0 {
			return
		}
		x := r.X + stripeInsetX
		if rtl {
			x = r.X + r.W - stripeInsetX - stripeW
		}
		insetY := minF32(8, max(0, (r.H-12)*0.5))
		p.Fill(ui.Rect{X: x, Y: r.Y + insetY, W: stripeW, H: max(4, r.H-2*insetY)}, pal.accent, stripeW*0.5)
	})

	box.Children(func() {
		head := ui.Row(c).AlignItems(ui.Center).Gap(8)
		head.Children(func() {
			docIconBox(c, icon, 18, pal.headerInk)
			md3e.TextWith(c, title, md3e.TextOptions{
				Role:       tokens.TitleSmall,
				Emphasized: true,
				Color:      pal.headerInk,
			})
		})
		if body != nil {
			body()
		} else if o.Text != "" {
			md3e.TextWith(c, o.Text, md3e.TextOptions{
				Role:  tokens.BodyMedium,
				Color: pal.onContainer,
			})
		}
	})
	return box
}

func calloutPaletteFor(th *md3e.Theme, kind CalloutKind) calloutPalette {
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	cs := th.Colors
	var container, onContainer, accent ui.Color
	var icon md3e.IconDraw
	var name string

	switch kind {
	case CalloutTip:
		container = pickColor(th.SuccessContainer, cs.TertiaryContainer)
		onContainer = pickColor(th.OnSuccessContainer, cs.OnTertiaryContainer, th.ContentColorFor(container))
		accent = pickColor(th.Success, cs.Tertiary, onContainer)
		icon = calloutTipSymbol.Draw
		name = "Tip"
	case CalloutImportant:
		container = pickColor(cs.TertiaryContainer, cs.PrimaryContainer)
		onContainer = pickColor(cs.OnTertiaryContainer, cs.OnPrimaryContainer, th.ContentColorFor(container))
		accent = pickColor(cs.Tertiary, cs.Primary, onContainer)
		icon = calloutImportantSymbol.Draw
		name = "Important"
	case CalloutWarning:
		container = pickColor(th.WarningContainer, cs.SecondaryContainer)
		onContainer = pickColor(th.OnWarningContainer, cs.OnSecondaryContainer, th.ContentColorFor(container))
		accent = pickColor(th.Warning, cs.Secondary, onContainer)
		icon = symbols.Warning.Draw
		name = "Warning"
	case CalloutCaution:
		container = pickColor(cs.ErrorContainer, cs.SurfaceContainerHigh)
		onContainer = pickColor(cs.OnErrorContainer, cs.OnSurface, th.ContentColorFor(container))
		accent = pickColor(cs.Error, onContainer)
		icon = symbols.ErrorOutline.Draw
		name = "Caution"
	default:
		container = pickColor(cs.PrimaryContainer, cs.SurfaceContainerHigh)
		onContainer = pickColor(cs.OnPrimaryContainer, cs.OnSurface, th.ContentColorFor(container))
		accent = pickColor(cs.Primary, onContainer)
		icon = symbols.InfoOutline.Draw
		name = "Note"
	}

	if th.Dark && th.Contrast < tokens.ContrastHigh && container.A > 0 && cs.SurfaceContainerLow.A > 0 {
		container = container.MixOklab(cs.SurfaceContainerLow, 0.36)
	}

	headerInk := accent
	if container.A > 0 && tokens.ContrastRatio(headerInk, container) < 4.5 {
		headerInk = onContainer
	}
	border := accent.Alpha(0.28)
	if th.Contrast >= tokens.ContrastHigh {
		border = accent.Alpha(0.75)
	}

	return calloutPalette{
		container:   container,
		onContainer: onContainer,
		accent:      accent,
		headerInk:   headerInk,
		border:      border,
		icon:        icon,
		defaultName: name,
	}
}
