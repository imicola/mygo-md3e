package md3e

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

// SkeletonShape selects the geometry preset of Skeleton.
type SkeletonShape int

const (
	// SkeletonRect draws a rounded-rectangle placeholder block (default).
	SkeletonRect SkeletonShape = iota
	// SkeletonCircle draws a circular placeholder (for avatars and icon buttons).
	SkeletonCircle
	// SkeletonPill draws a full pill-rounded placeholder (for chips and buttons).
	SkeletonPill
)

// SkeletonOptions configures Skeleton. The zero value is a full-width, 16 DIP
// tall rounded-rectangle skeleton with a subtle tonal pulse.
type SkeletonOptions struct {
	// Shape selects SkeletonRect (default), SkeletonCircle, or SkeletonPill.
	Shape SkeletonShape
	// Width sets a fixed width in DIP. Zero fills available width (except
	// SkeletonCircle, where width defaults to Height).
	Width float32
	// Height sets the block (or single-line) height in DIP. Defaults to 16 DIP
	// (or 40 DIP for SkeletonCircle).
	Height float32
	// Radius overrides the corner radius in DIP when positive and finite.
	Radius float32
	// Lines > 1 renders a vertical stack of text placeholder lines whose final
	// line is shortened to 65% width for natural paragraph rhythm.
	Lines int
	// LineGap sets the vertical spacing between Lines in DIP (default 8 DIP).
	LineGap float32
	// Static disables the tonal pulse animation, asking for zero animation frames.
	Static bool
	// Label optionally names the loading placeholder for assistive technology.
	Label string
}

const skeletonPulsePeriod = 1400 * time.Millisecond

type skeletonClockKey struct{}

type skeletonClock struct {
	start     time.Time
	lastPulse float32
}

// Skeleton builds an MD3E tonal pulsing placeholder block (single rectangle,
// circle, pill, or multi-line text stack). Its pulse animation runs strictly
// inside Draw via p.AnimationFrame(), stops requesting frames when scrolled
// off-screen (p.Visible()), respects ReduceMotion and Static (zero pending
// frames), and is ghost-draw safe when exiting.
func Skeleton(c *ui.Context, opts ...SkeletonOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Skeleton takes at most one SkeletonOptions")
	}
	var o SkeletonOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors

	defaultH := float32(16)
	if o.Shape == SkeletonCircle {
		defaultH = 40
	}
	h := sanitizePositiveF32(o.Height, defaultH)
	w := sanitizeNonNegF32(o.Width)
	if o.Shape == SkeletonCircle && w <= 0 {
		w = h
	}

	lines := o.Lines
	if lines <= 0 {
		lines = 1
	}
	lineGap := sanitizePositiveF32(o.LineGap, 8)
	totalH := h
	if lines > 1 {
		totalH = float32(lines)*h + float32(lines-1)*lineGap
	}

	baseCol := cs.SurfaceContainerHighest
	if baseCol.A == 0 {
		baseCol = cs.SurfaceContainerHigh
	}
	highlightCol := baseCol.Mix(cs.OnSurface, 0.08)

	el := ui.Box(c).Height(totalH).DimDisabled(false)
	if w > 0 {
		el.Width(w).Shrink(0)
	} else {
		el.FillWidth()
	}
	if o.Label != "" {
		el.Role(ui.RoleProgress).Label(o.Label).Range(0, 1, -1)
	}

	clock := ui.Local(el, skeletonClockKey{}, func() skeletonClock {
		return skeletonClock{}
	})

	thReduce := th.ReduceMotion
	static := o.Static
	shapeKind := o.Shape
	customRadius := sanitizeNonNegF32(o.Radius)
	rtl := c.Direction() == ui.RightToLeft

	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
			return
		}
		live := el.Valid()
		reduce := thReduce || static || p.Preferences().ReduceMotion
		visible := rectIntersects(r, p.Visible())

		pulse := float32(0)
		if !reduce {
			if live {
				if visible {
					now := p.Now()
					if clock.start.IsZero() {
						clock.start = now
					}
					elapsed := max(now.Sub(clock.start), 0)
					phase := float64(elapsed%skeletonPulsePeriod) / float64(skeletonPulsePeriod)
					pulse = float32(0.5 - 0.5*math.Cos(2*math.Pi*phase))
					clock.lastPulse = pulse
					p.AnimationFrame()
				}
			} else {
				// Ghost-draw during exit transition: draw frozen pulse without
				// requesting more frames.
				pulse = clock.lastPulse
			}
		}

		col := baseCol.MixOklab(highlightCol, clampF32(pulse, 0, 1))
		if lines == 1 {
			rad := skeletonRadiusFor(shapeKind, r.W, r.H, customRadius)
			p.Fill(r, col, rad)
			return
		}

		rad := skeletonRadiusFor(shapeKind, r.W, h, customRadius)
		for i := range lines {
			lineW := r.W
			if i == lines-1 {
				lineW = r.W * 0.65
			}
			x := r.X
			if rtl {
				x = r.X + (r.W - lineW)
			}
			y := r.Y + float32(i)*(h+lineGap)
			if y >= r.Y+r.H {
				break
			}
			lineH := min(h, r.Y+r.H-y)
			p.Fill(ui.Rect{X: x, Y: y, W: lineW, H: lineH}, col, min(rad, lineH/2))
		}
	})

	return el
}

func skeletonRadiusFor(kind SkeletonShape, w, h, override float32) float32 {
	if override > 0 {
		return min(override, min(w, h)/2)
	}
	switch kind {
	case SkeletonCircle, SkeletonPill:
		return min(w, h) / 2
	default:
		return min(float32(8), min(w, h)/2)
	}
}

// EmptyStateOptions configures EmptyState.
type EmptyStateOptions struct {
	// Illustration renders a custom illustration or widget above the title.
	// Takes precedence over Icon when non-nil.
	Illustration func()
	// Icon draws a centered expressive IconBadge above the title when
	// Illustration is nil.
	Icon IconDraw
	// IconShape optionally styles the IconBadge with a custom RoundedPolygon
	// (such as shape.SoftBurst() or shape.Cookie9Sided()).
	IconShape *shape.RoundedPolygon
	// Description displays muted supporting copy beneath the title.
	Description string
	// PrimaryActionLabel renders a filled primary action button when non-empty.
	PrimaryActionLabel string
	// OnPrimaryAction runs when the primary action button is clicked.
	OnPrimaryAction func()
	// SecondaryActionLabel renders an outlined secondary action button when non-empty.
	SecondaryActionLabel string
	// OnSecondaryAction runs when the secondary action button is clicked.
	OnSecondaryAction func()
	// Actions optionally renders custom action controls in the centered action row.
	Actions func()
	// Padding sets the outer padding in DIP (defaults to 24, scaled by Density).
	Padding float32
	// Label overrides the accessible group label (defaults to title).
	Label string
}

type (
	emptyPrimaryBtnKey   struct{}
	emptySecondaryBtnKey struct{}
)

// EmptyState builds a centered MD3E empty-state placeholder with an optional
// illustration or expressive shaped IconBadge, title, supporting description,
// and primary/secondary action buttons.
func EmptyState(c *ui.Context, title string, opts ...EmptyStateOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: EmptyState takes at most one EmptyStateOptions")
	}
	var o EmptyStateOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	pad := sanitizePositiveF32(o.Padding, float32(24*th.Density.PaddingScale()))

	col := ui.Column(c).
		FillWidth().
		Padding(pad).
		Gap(12).
		AlignItems(ui.Center).
		Justify(ui.Center).
		DimDisabled(false)
	label := o.Label
	if label == "" {
		label = title
	}
	if label != "" {
		col.Role(ui.RoleGroup).Label(label)
	}

	hasActions := o.PrimaryActionLabel != "" || o.SecondaryActionLabel != "" || o.Actions != nil

	col.Children(func() {
		if o.Illustration != nil {
			o.Illustration()
		} else if o.Icon != nil {
			IconBadge(c, o.Icon, IconBadgeOptions{
				Variant:       IconBadgePrimaryTonal,
				ContainerSize: float32(64 * th.Density.ControlScale()),
				IconSize:      float32(30 * th.Density.ControlScale()),
				Shape:         o.IconShape,
				ShapeScale:    tokens.ShapeExtraLarge,
			})
		}

		if title != "" {
			txt := applyTypeStyleWithTheme(th, ui.Text(c, title).TextAlign(ui.Center), tokens.TitleLarge)
			txt.TextColor(cs.OnSurface)
		}

		if o.Description != "" {
			desc := applyTypeStyleWithTheme(th, ui.Text(c, o.Description).TextAlign(ui.Center), tokens.BodyMedium)
			desc.MaxWidth(360).TextColor(cs.OnSurfaceVariant)
		}

		if hasActions {
			ui.Row(c).Gap(8).Margin(4, 0, 0, 0).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				if o.SecondaryActionLabel != "" {
					sec := Button(c.Key(emptySecondaryBtnKey{}), o.SecondaryActionLabel, ButtonOptions{
						Variant: ButtonOutlined,
						Size:    tokens.SizeS,
					})
					if sec.Clicked() && o.OnSecondaryAction != nil {
						o.OnSecondaryAction()
					}
				}
				if o.PrimaryActionLabel != "" {
					pri := Button(c.Key(emptyPrimaryBtnKey{}), o.PrimaryActionLabel, ButtonOptions{
						Variant: ButtonFilled,
						Size:    tokens.SizeS,
					})
					if pri.Clicked() && o.OnPrimaryAction != nil {
						o.OnPrimaryAction()
					}
				}
				if o.Actions != nil {
					o.Actions()
				}
			})
		}
	})

	return col
}

// BannerAction defines an inline action button displayed inside a Banner.
type BannerAction struct {
	Label   string
	OnClick func()
}

// BannerOptions configures Banner, a non-modal top-of-page notification bar.
type BannerOptions struct {
	// Tone selects the semantic color role (Neutral uses SurfaceContainerLow
	// with hairline outline; Primary/Secondary/Tertiary/Error/Warning/Success
	// use their harmonized tonal container pair).
	Tone SemanticTone
	// Icon draws an optional 20 DIP leading icon.
	Icon IconDraw
	// Title optionally renders a short emphasized heading above the message.
	Title string
	// Actions displays up to 2 inline action buttons.
	Actions []BannerAction
	// Open optionally controls visibility. When non-nil and *Open is false,
	// Banner renders a zero-size box; clicking the dismiss button sets *Open to
	// false before invoking OnDismiss.
	Open *bool
	// Dismissible shows a trailing close button (automatically enabled when
	// OnDismiss is non-nil).
	Dismissible bool
	// DismissLabel overrides the close button's accessible label ("Dismiss").
	DismissLabel string
	// OnDismiss runs when the user activates the dismiss button.
	OnDismiss func()
	// Label overrides the accessible region name.
	Label string
}

type (
	bannerActionKey struct{ i int }
	bannerCloseKey  struct{}
)

// Banner builds an MD3E non-modal notification banner supporting semantic color
// roles, optional leading icon and title, 1–2 inline action buttons, optional
// dismiss button, and *bool visibility binding. Error and Warning banners use
// ui.RoleAlert; other tones use ui.RoleStatus.
func Banner(c *ui.Context, message string, opts ...BannerOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Banner takes at most one BannerOptions")
	}
	var o BannerOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Open != nil && !*o.Open {
		return ui.Box(c).Size(0, 0)
	}

	th := CurrentTheme(c)
	cs := th.Colors
	sem := semanticColorsFor(th, o.Tone)

	bg := sem.container
	fg := sem.onContainer
	iconInk := sem.accent
	border := sem.border.Alpha(0.35)
	if o.Tone == ToneNeutral {
		bg = cs.SurfaceContainerLow
		if bg.A == 0 {
			bg = SurfaceColorForElevation(th, tokens.Level1)
		}
		fg = cs.OnSurface
		iconInk = cs.OnSurfaceVariant
		border = resolveHairlineColor(th, ui.Color{})
	}
	if iconInk.A == 0 {
		iconInk = fg
	}

	padX := float32(16 * th.Density.PaddingScale())
	padY := float32(12 * th.Density.PaddingScale())
	radius := th.Shapes.Radius(tokens.ShapeMedium)

	role := ui.RoleStatus
	if o.Tone == ToneError || o.Tone == ToneWarning {
		role = ui.RoleAlert
	}

	row := ui.Row(c).
		FillWidth().
		Padding(padY, 0, padY, 0).
		PaddingStart(padX).
		PaddingEnd(padX).
		Gap(12).
		AlignItems(ui.Center).
		Radius(radius).
		Background(bg).
		TextColor(fg).
		Role(role).
		DimDisabled(false)
	if border.A > 0 {
		row.Border(1, border)
	}
	label := o.Label
	if label == "" {
		if o.Title != "" {
			label = o.Title + ": " + message
		} else {
			label = message
		}
	}
	if label != "" {
		row.Label(label)
	}

	actions := o.Actions
	if len(actions) > 2 {
		actions = actions[:2]
	}
	showDismiss := o.Dismissible || o.OnDismiss != nil

	row.Children(func() {
		if o.Icon != nil {
			drawIconBox(c, o.Icon, 20, iconInk)
		}

		ui.Column(c).Grow(1).Gap(2).Children(func() {
			if o.Title != "" {
				t := applyTypeStyleWithTheme(th, ui.Text(c, o.Title), tokens.TitleSmall)
				t.TextColor(fg)
			}
			if message != "" {
				m := applyTypeStyleWithTheme(th, ui.Text(c, message), tokens.BodyMedium)
				m.TextColor(fg)
			}
		})

		if len(actions) > 0 {
			ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
				for i, act := range actions {
					if act.Label == "" {
						continue
					}
					btn := ui.ButtonBase(c.Key(bannerActionKey{i: i})).
						DimDisabled(false).
						FocusRing(false).
						Height(32).
						PaddingX(10).
						Radius(16).
						AlignItems(ui.Center).
						Justify(ui.Center).
						Label(act.Label).
						TextColor(iconInk)
					if btn.Clicked() && act.OnClick != nil {
						act.OnClick()
					}
					attachRippleWithContext(c, th, btn, StateLayerOptions{
						Color:  iconInk,
						Radius: 16,
					})
					btn.Children(func() {
						txt := applyTypeStyleWithTheme(th, ui.Text(c, act.Label).NoWrap(), tokens.LabelLarge)
						txt.TextColor(iconInk)
					})
				}
			})
		}

		if showDismiss {
			dismissLabel := o.DismissLabel
			if dismissLabel == "" {
				dismissLabel = "Dismiss"
			}
			closeBtn := ui.ButtonBase(c.Key(bannerCloseKey{})).
				DimDisabled(false).
				FocusRing(false).
				Size(32, 32).
				Shrink(0).
				Radius(16).
				Label(dismissLabel)
			if closeBtn.Clicked() {
				if o.Open != nil {
					*o.Open = false
				}
				if o.OnDismiss != nil {
					o.OnDismiss()
				}
			}
			attachRippleWithContext(c, th, closeBtn, StateLayerOptions{
				Color:  fg,
				Radius: 16,
			})
			closeBtn.Children(func() {
				drawIconBox(c, mdicon.Close.Draw, 18, fg)
			})
		}
	})

	return row
}
