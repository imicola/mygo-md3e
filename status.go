package md3e

import (
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

// SemanticTone selects a harmonized MD3E semantic color role used by Tag,
// StatusPill, Avatar, and Banner.
type SemanticTone int

const (
	// ToneNeutral uses tonal surface colors (SurfaceContainerHighest / OnSurfaceVariant).
	ToneNeutral SemanticTone = iota
	// TonePrimary uses PrimaryContainer / OnPrimaryContainer.
	TonePrimary
	// ToneSecondary uses SecondaryContainer / OnSecondaryContainer.
	ToneSecondary
	// ToneTertiary uses TertiaryContainer / OnTertiaryContainer.
	ToneTertiary
	// ToneError uses ErrorContainer / OnErrorContainer.
	ToneError
	// ToneWarning uses an HCT-derived amber tonal pair adapted to Light/Dark/Contrast.
	ToneWarning
	// ToneSuccess uses an HCT-derived emerald tonal pair adapted to Light/Dark/Contrast.
	ToneSuccess
)

type semanticColors struct {
	container   ui.Color
	onContainer ui.Color
	accent      ui.Color
	border      ui.Color
}

func semanticColorsFor(th *Theme, tone SemanticTone) semanticColors {
	cs := th.Colors
	highContrast := th.Contrast >= tokens.ContrastHigh
	switch tone {
	case TonePrimary:
		return semanticColors{
			container:   cs.PrimaryContainer,
			onContainer: cs.OnPrimaryContainer,
			accent:      cs.Primary,
			border:      cs.Primary,
		}
	case ToneSecondary:
		return semanticColors{
			container:   cs.SecondaryContainer,
			onContainer: cs.OnSecondaryContainer,
			accent:      cs.Secondary,
			border:      cs.Secondary,
		}
	case ToneTertiary:
		return semanticColors{
			container:   cs.TertiaryContainer,
			onContainer: cs.OnTertiaryContainer,
			accent:      cs.Tertiary,
			border:      cs.Tertiary,
		}
	case ToneError:
		return semanticColors{
			container:   cs.ErrorContainer,
			onContainer: cs.OnErrorContainer,
			accent:      cs.Error,
			border:      cs.Error,
		}
	case ToneWarning:
		if th.WarningContainer.A > 0 && th.OnWarningContainer.A > 0 {
			border := th.Warning
			if highContrast && cs.Outline.A > 0 {
				border = cs.Outline
			}
			return semanticColors{
				container:   th.WarningContainer,
				onContainer: th.OnWarningContainer,
				accent:      th.Warning,
				border:      border,
			}
		}
		return hctSemanticPair(th.Dark, highContrast, 68, 64)
	case ToneSuccess:
		if th.SuccessContainer.A > 0 && th.OnSuccessContainer.A > 0 {
			border := th.Success
			if highContrast && cs.Outline.A > 0 {
				border = cs.Outline
			}
			return semanticColors{
				container:   th.SuccessContainer,
				onContainer: th.OnSuccessContainer,
				accent:      th.Success,
				border:      border,
			}
		}
		return hctSemanticPair(th.Dark, highContrast, 145, 56)
	default:
		border := cs.OutlineVariant
		container := cs.SurfaceContainerHighest
		if highContrast {
			if cs.Outline.A > 0 {
				border = cs.Outline
			}
			if th.Dark && cs.SurfaceContainerHigh.A > 0 {
				// SurfaceContainerHighest sits at T30 in dark HighContrast where
				// a 10% pressed white state layer caps pure white below 7:1;
				// stepping down to SurfaceContainerHigh (T25) preserves >= 7:1
				// across rest, hover, and pressed states.
				container = cs.SurfaceContainerHigh
			}
		}
		return semanticColors{
			container:   container,
			onContainer: cs.OnSurfaceVariant,
			accent:      cs.Outline,
			border:      border,
		}
	}
}

func hctSemanticPair(dark, highContrast bool, hue, chroma float64) semanticColors {
	if dark {
		cTone, fgTone, accTone := 26.0, 92.0, 76.0
		if highContrast {
			cTone, fgTone, accTone = 24.0, 98.0, 86.0
		}
		return semanticColors{
			container:   tokens.HCT{Hue: hue, Chroma: chroma * 0.72, Tone: cTone}.ToColor(),
			onContainer: tokens.HCT{Hue: hue, Chroma: chroma * 0.45, Tone: fgTone}.ToColor(),
			accent:      tokens.HCT{Hue: hue, Chroma: chroma, Tone: accTone}.ToColor(),
			border:      tokens.HCT{Hue: hue, Chroma: chroma * 0.8, Tone: 64}.ToColor(),
		}
	}
	cTone, fgTone, accTone := 90.0, 20.0, 42.0
	if highContrast {
		cTone, fgTone, accTone = 94.0, 10.0, 32.0
	}
	return semanticColors{
		container:   tokens.HCT{Hue: hue, Chroma: chroma * 0.65, Tone: cTone}.ToColor(),
		onContainer: tokens.HCT{Hue: hue, Chroma: chroma * 0.85, Tone: fgTone}.ToColor(),
		accent:      tokens.HCT{Hue: hue, Chroma: chroma, Tone: accTone}.ToColor(),
		border:      tokens.HCT{Hue: hue, Chroma: chroma * 0.8, Tone: 50}.ToColor(),
	}
}

func rectIntersects(a, b ui.Rect) bool {
	return a.W > 0 && a.H > 0 && b.W > 0 && b.H > 0 &&
		a.X < b.X+b.W && a.X+a.W > b.X &&
		a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

// TagVariant selects the container treatment of Tag.
type TagVariant int

const (
	// TagTonal fills a compact tonal container (default).
	TagTonal TagVariant = iota
	// TagOutlined draws a crisp 1 DIP border around a transparent container.
	TagOutlined
)

// TagOptions configures Tag. The zero value is a compact, non-interactive
// neutral tonal tag.
type TagOptions struct {
	// Variant selects TagTonal (default) or TagOutlined.
	Variant TagVariant
	// Tone selects the semantic color role (Neutral, Primary, Secondary,
	// Tertiary, Error, Warning, Success).
	Tone SemanticTone
	// LeadingIcon draws an optional 14 DIP icon before the label.
	LeadingIcon IconDraw
	// OnClick makes the tag interactive and invokes OnClick on click/Enter/Space.
	OnClick func()
	// OnClose adds a trailing dismiss button invoking OnClose.
	OnClose func()
	// CloseLabel overrides the close button's accessible name ("Remove <text>").
	CloseLabel string
	// Label overrides the tag's accessible name.
	Label string
	// Disabled disables interaction and applies MD3E disabled color tokens.
	Disabled bool
}

type (
	tagBodyKey  struct{}
	tagCloseKey struct{}
)

// Tag builds a compact MD3E informational tag supporting tonal and outlined
// treatments, semantic color roles (Primary, Secondary, Tertiary, Error,
// Warning, Success, Neutral), optional leading icon, and optional click or
// close actions.
func Tag(c *ui.Context, text string, opts ...TagOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Tag takes at most one TagOptions")
	}
	var o TagOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	h, padX, iconSz, gap := tagMetricsForDensity(th.Density)
	radius := float32(8)
	if h < 22 {
		radius = 6
	}

	sem := semanticColorsFor(th, o.Tone)
	bg := sem.container
	fg := sem.onContainer
	var border ui.Color
	borderW := float32(0)
	if o.Variant == TagOutlined {
		bg = ui.Transparent
		border = sem.border
		borderW = outlineBorderWidth(th)
		if o.Tone == ToneNeutral {
			fg = cs.OnSurfaceVariant
		} else {
			fg = sem.accent
		}
	} else if hcW, hcCol := highContrastBorder(th); hcW > 0 {
		borderW = hcW
		border = hcCol
	}

	name := o.Label
	if name == "" {
		name = text
	}

	// Simple clickable tag without separate close button: single ButtonBase.
	if o.OnClick != nil && o.OnClose == nil {
		el := ui.ButtonBase(c).
			DimDisabled(false).
			FocusRing(false).
			Height(h).
			Shrink(0).
			PaddingStart(padX).
			PaddingEnd(padX).
			Gap(gap).
			Radius(radius).
			AlignItems(ui.Center).
			Justify(ui.Center)
		disabled := o.Disabled || el.IsDisabled()
		el.Disabled(o.Disabled)
		if disabled {
			fg = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
			if o.Variant == TagOutlined {
				border = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
				borderW = 1
			} else {
				bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
				border = ui.Transparent
				borderW = 0
			}
		}
		if bg.A > 0 {
			el.Background(bg)
		}
		if borderW > 0 && border.A > 0 {
			el.Border(borderW, border)
		}
		el.TextColor(fg)
		if name != "" {
			el.Label(name)
		}
		if !disabled && el.Clicked() {
			o.OnClick()
		}
		borderInset := float32(0)
		if borderW > 0 && border.A > 0 {
			borderInset = borderW
		}
		attachRippleWithContext(c, th, el, StateLayerOptions{
			Color:       fg,
			Radius:      radius,
			BorderInset: borderInset,
			Disabled:    disabled,
		})
		el.Children(func() {
			if o.LeadingIcon != nil {
				drawIconBox(c, o.LeadingIcon, iconSz, fg)
			}
			if text != "" {
				txt := applyTypeStyleWithTheme(th, ui.Text(c, text).NoWrap(), tokens.LabelMedium)
				txt.TextColor(fg)
			}
		})
		return el
	}

	// Static tag or tag with close action.
	row := ui.Row(c).
		DimDisabled(false).
		Height(h).
		Shrink(0).
		Radius(radius).
		AlignItems(ui.Center)
	disabled := o.Disabled || row.IsDisabled()
	row.Disabled(o.Disabled)
	if disabled {
		fg = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		if o.Variant == TagOutlined {
			border = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
			borderW = 1
		} else {
			bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
			border = ui.Transparent
			borderW = 0
		}
	}
	if bg.A > 0 {
		row.Background(bg)
	}
	if borderW > 0 && border.A > 0 {
		row.Border(borderW, border)
	}
	row.TextColor(fg)
	if o.OnClose == nil {
		row.PaddingStart(padX).PaddingEnd(padX).Gap(gap)
		if name != "" {
			row.Role(ui.RoleText).Label(name)
		}
		row.Children(func() {
			if o.LeadingIcon != nil {
				drawIconBox(c, o.LeadingIcon, iconSz, fg)
			}
			if text != "" {
				txt := applyTypeStyleWithTheme(th, ui.Text(c, text).NoWrap(), tokens.LabelMedium)
				txt.TextColor(fg)
			}
		})
		return row
	}

	// Closeable tag: keep body and close button as sibling controls so clicking
	// close never triggers OnClick and both remain keyboard accessible.
	rtl := c.Direction() == ui.RightToLeft
	row.PaddingEnd(max(padX-4, 4)).Gap(2)
	if o.OnClick == nil {
		row.PaddingStart(padX)
	}
	row.Children(func() {
		if o.OnClick != nil {
			bodyRadii := [4]float32{radius, 0, 0, radius}
			if rtl {
				bodyRadii = [4]float32{0, radius, radius, 0}
			}
			borderInset := float32(0)
			if borderW > 0 && border.A > 0 {
				borderInset = borderW
			}
			body := ui.ButtonBase(c.Key(tagBodyKey{})).
				DimDisabled(false).
				FocusRing(false).
				Height(h).
				PaddingStart(padX).
				PaddingEnd(4).
				Gap(gap).
				Radius(bodyRadii[0], bodyRadii[1], bodyRadii[2], bodyRadii[3]).
				AlignItems(ui.Center).
				Disabled(disabled).
				TextColor(fg)
			if name != "" {
				body.Label(name)
			}
			if !disabled && body.Clicked() {
				o.OnClick()
			}
			attachRippleWithContext(c, th, body, StateLayerOptions{
				Color:       fg,
				Radii:       bodyRadii,
				HasRadii:    true,
				BorderInset: borderInset,
				Disabled:    disabled,
			})
			body.Children(func() {
				if o.LeadingIcon != nil {
					drawIconBox(c, o.LeadingIcon, iconSz, fg)
				}
				if text != "" {
					txt := applyTypeStyleWithTheme(th, ui.Text(c, text).NoWrap(), tokens.LabelMedium)
					txt.TextColor(fg)
				}
			})
		} else {
			if o.LeadingIcon != nil {
				drawIconBox(c, o.LeadingIcon, iconSz, fg)
			}
			if text != "" {
				txt := applyTypeStyleWithTheme(th, ui.Text(c, text).NoWrap(), tokens.LabelMedium)
				txt.TextColor(fg)
			}
		}

		closeSide := max(h-6, 16)
		closeLabel := o.CloseLabel
		if closeLabel == "" {
			if text != "" {
				closeLabel = "Remove " + text
			} else {
				closeLabel = "Remove"
			}
		}
		closeBtn := ui.ButtonBase(c.Key(tagCloseKey{})).
			DimDisabled(false).
			FocusRing(false).
			Size(closeSide, closeSide).
			Shrink(0).
			Radius(closeSide / 2).
			Label(closeLabel).
			Disabled(disabled)
		if !disabled && closeBtn.Clicked() {
			o.OnClose()
		}
		attachRippleWithContext(c, th, closeBtn, StateLayerOptions{
			Color:    fg,
			Radius:   closeSide / 2,
			Disabled: disabled,
		})
		closeBtn.Children(func() {
			drawIconBox(c, mdicon.Close.Draw, min(iconSz, closeSide-4), fg)
		})
	})

	return row
}

func tagMetricsForDensity(d tokens.Density) (height, padX, iconSize, gap float32) {
	switch d.Normalize() {
	case tokens.DensityCompact:
		return 20, 6, 12, 4
	case tokens.DensityDesktop:
		return 22, 8, 13, 5
	default:
		return 26, 10, 14, 6
	}
}

// StatusPillOptions configures StatusPill.
type StatusPillOptions struct {
	// Tone selects the semantic color role (Neutral, Primary, Secondary,
	// Tertiary, Error, Warning, Success).
	Tone SemanticTone
	// Pulsing enables a gentle breathing halo around the status dot. The pulse
	// runs strictly inside Draw via p.AnimationFrame(), pauses automatically
	// when scrolled off-screen (p.Visible()), respects ReduceMotion, and is
	// ghost-draw safe.
	Pulsing bool
	// Outlined draws a 1 DIP semantic border on a transparent background
	// instead of a tonal pill fill.
	Outlined bool
	// DotColor optionally overrides the indicator dot color.
	DotColor ui.Color
	// Label overrides the accessible status name (defaults to text).
	Label string
}

const statusPulsePeriod = 1600 * time.Millisecond

type statusPulseClockKey struct{}

type statusPulseClock struct {
	start     time.Time
	lastPhase float32
}

// StatusPill builds a compact MD3E status pill with a leading indicator dot
// (static or smoothly pulsing) and status label.
func StatusPill(c *ui.Context, text string, opts ...StatusPillOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: StatusPill takes at most one StatusPillOptions")
	}
	var o StatusPillOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	h, padX, _, gap := tagMetricsForDensity(th.Density)
	radius := h / 2

	sem := semanticColorsFor(th, o.Tone)
	bg := sem.container
	fg := sem.onContainer
	var border ui.Color
	borderW := float32(0)
	if o.Outlined {
		bg = ui.Transparent
		border = sem.border
		borderW = outlineBorderWidth(th)
		if o.Tone == ToneNeutral {
			fg = cs.OnSurfaceVariant
		} else {
			fg = sem.accent
		}
	} else if hcW, hcCol := highContrastBorder(th); hcW > 0 {
		borderW = hcW
		border = hcCol
	}
	dotCol := o.DotColor
	if dotCol.A == 0 {
		dotCol = sem.accent
		if dotCol.A == 0 {
			dotCol = fg
		}
	}

	pill := ui.Row(c).
		DimDisabled(false).
		Height(h).
		Shrink(0).
		PaddingStart(max(padX-3, 5)).
		PaddingEnd(padX).
		Gap(max(gap-1, 4)).
		Radius(radius).
		AlignItems(ui.Center).
		Role(ui.RoleStatus)
	if bg.A > 0 {
		pill.Background(bg)
	}
	if borderW > 0 && border.A > 0 {
		pill.Border(borderW, border)
	}
	pill.TextColor(fg)
	label := o.Label
	if label == "" {
		label = text
	}
	if label != "" {
		pill.Label(label)
	}

	thReduce := th.ReduceMotion
	pulsing := o.Pulsing
	dotBoxSide := float32(14)
	if h <= 20 {
		dotBoxSide = 12
	}

	pill.Children(func() {
		dotEl := ui.Box(c).Size(dotBoxSide, dotBoxSide).Shrink(0)
		clock := ui.Local(dotEl, statusPulseClockKey{}, func() statusPulseClock {
			return statusPulseClock{}
		})
		dotEl.Draw(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			live := dotEl.Valid()
			reduce := thReduce || p.Preferences().ReduceMotion
			visible := rectIntersects(r, p.Visible())

			phase := clock.lastPhase
			showRing := false
			if pulsing && !reduce {
				if live {
					if visible {
						now := p.Now()
						if clock.start.IsZero() {
							clock.start = now
						}
						elapsed := max(now.Sub(clock.start), 0)
						phase = float32(elapsed%statusPulsePeriod) / float32(statusPulsePeriod)
						clock.lastPhase = phase
						showRing = true
						p.AnimationFrame()
					}
				} else {
					// Ghost-draw during exit transition: render frozen phase
					// without requesting further frames.
					showRing = true
				}
			}

			cx := r.X + r.W/2
			cy := r.Y + r.H/2
			coreR := min(r.W, r.H) * 0.24
			if coreR < 3 {
				coreR = 3
			}
			if showRing {
				// Crisp hollow expanding radar ring + gentle core breathe.
				inv := 1 - phase
				eased := 1 - inv*inv*inv
				strokeW := float32(1.6) - 0.8*eased
				maxR := min(r.W, r.H)*0.5 - strokeW*0.5
				ringR := coreR + (maxR-coreR)*eased
				alpha := float32(0.70) * (1 - eased)
				if alpha > 0.01 && ringR > 0.5 {
					p.Stroke(ui.Rect{
						X: cx - ringR,
						Y: cy - ringR,
						W: ringR * 2,
						H: ringR * 2,
					}, dotCol.Alpha(alpha), ringR, strokeW)
				}
				breathe := 1 + 0.08*float32(math.Sin(2*math.Pi*float64(phase)))
				coreR *= breathe
			}
			p.Fill(ui.Rect{
				X: cx - coreR,
				Y: cy - coreR,
				W: coreR * 2,
				H: coreR * 2,
			}, dotCol, coreR)
		})

		if text != "" {
			txt := applyTypeStyleWithTheme(th, ui.Text(c, text).NoWrap(), tokens.LabelMedium)
			txt.TextColor(fg)
		}
	})

	return pill
}

// AvatarStatus selects an optional presence indicator dot on Avatar.
type AvatarStatus int

const (
	// AvatarStatusNone shows no status indicator dot (default).
	AvatarStatusNone AvatarStatus = iota
	// AvatarStatusOnline shows a green online dot.
	AvatarStatusOnline
	// AvatarStatusBusy shows an error/red busy dot.
	AvatarStatusBusy
	// AvatarStatusAway shows an amber away dot.
	AvatarStatusAway
	// AvatarStatusOffline shows a muted neutral offline dot.
	AvatarStatusOffline
)

// AvatarOptions configures Avatar.
type AvatarOptions struct {
	// Text provides the user name or initials displayed when Image is nil.
	Text string
	// Image optionally renders a bitmap or SVG scaled with ui.Cover inside the
	// avatar's circular or custom RoundedPolygon clip.
	Image ui.ImageSource
	// Size is the avatar diameter/side in DIP (defaults to 40, scaled by Density).
	Size float32
	// Shape optionally clips the avatar container and image to a custom
	// *shape.RoundedPolygon instead of a circle.
	Shape *shape.RoundedPolygon
	// Tone selects the semantic tonal background when rendering initials
	// (defaults to TonePrimary).
	Tone SemanticTone
	// HasTone distinguishes ToneNeutral explicitly from the zero value.
	HasTone bool
	// ContainerColor overrides the background color.
	ContainerColor ui.Color
	// ContentColor overrides the initials ink color.
	ContentColor ui.Color
	// Status optionally draws an RTL-aware bottom-end presence dot.
	Status AvatarStatus
	// StatusRingColor overrides the cutout halo behind the status dot
	// (defaults to cs.Surface so callers can match card container surfaces).
	StatusRingColor ui.Color
	// Label overrides the accessible name (RoleImage); defaults to Text.
	Label string
}

// Avatar builds an MD3E user/entity avatar supporting initials text, bitmap/SVG
// ui.ImageSource (painted with ui.Cover), circular or custom *shape.RoundedPolygon
// clipping, and an optional RTL-aware online/busy/away/offline status dot.
func Avatar(c *ui.Context, opts ...AvatarOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Avatar takes at most one AvatarOptions")
	}
	var o AvatarOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	defaultSide := float32(40 * th.Density.ControlScale())
	side := sanitizePositiveF32(o.Size, defaultSide)

	tone := o.Tone
	if !o.HasTone && o.Tone == ToneNeutral {
		tone = TonePrimary
	}
	sem := semanticColorsFor(th, tone)
	bg := sem.container
	fg := sem.onContainer
	if o.ContainerColor.A > 0 {
		bg = o.ContainerColor
	}
	if o.ContentColor.A > 0 {
		fg = o.ContentColor
	}

	el := ui.Box(c).
		Size(side, side).
		Shrink(0).
		DimDisabled(false).
		AlignItems(ui.Center).
		Justify(ui.Center).
		TextColor(fg)
	label := o.Label
	if label == "" {
		label = o.Text
	}
	if label != "" {
		el.Role(ui.RoleImage).Label(label)
	}

	poly := validShapePolygon(o.Shape, nil)
	var scratch *ui.Path
	if poly != nil {
		scratch = localDrawPath(el)
	}
	photoRim := cs.OnSurface.Alpha(0.10)
	imgSrc := o.Image
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
			return
		}
		if poly != nil {
			path := poly.WriteTo(scratch, r)
			if bg.A > 0 {
				p.FillPath(path, bg)
			}
			if imgSrc != nil {
				p.ClipPath(path, func() {
					p.Image(imgSrc, r, ui.Cover)
				})
				p.StrokePath(poly.WriteTo(scratch, insetRect(r, 0.5)), 1, photoRim)
			}
			return
		}
		rad := min(r.W, r.H) / 2
		if bg.A > 0 {
			p.Fill(r, bg, rad)
		}
		if imgSrc != nil {
			p.Clip(r, rad, func() {
				p.Image(imgSrc, r, ui.Cover)
			})
			p.Stroke(r, photoRim, rad, 1)
		}
	})

	if imgSrc == nil && o.Text != "" {
		initials := avatarInitials(o.Text)
		if initials != "" {
			fontSz := clampF32(side*0.40, 10, 32)
			style := typeStyleOfWithTheme(th, tokens.TitleMedium)
			el.Children(func() {
				txt := ui.Text(c, initials).
					NoWrap().
					FontSize(fontSz).
					FixedLineHeight(fontSz * 1.1).
					FontWeight(500).
					TextColor(fg)
				if style.Font != "" {
					txt.Font(style.Font)
				}
			})
		}
	}

	if o.Status != AvatarStatusNone {
		dotCol := avatarStatusColor(th, o.Status)
		ringCol := o.StatusRingColor
		if ringCol.A == 0 {
			ringCol = cs.Surface
		}
		if ringCol.A == 0 {
			ringCol = ui.RGB(255, 255, 255)
		}
		rtl := c.Direction() == ui.RightToLeft
		el.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			dotD := clampF32(min(r.W, r.H)*0.28, 8, 16)
			ringW := float32(2)
			outerR := dotD/2 + ringW
			// Inset center by outerR so the ring stays strictly within r without clipping.
			cx := r.X + r.W - outerR
			if rtl {
				cx = r.X + outerR
			}
			cy := r.Y + r.H - outerR
			p.Fill(ui.Rect{X: cx - outerR, Y: cy - outerR, W: outerR * 2, H: outerR * 2}, ringCol, outerR)
			p.Fill(ui.Rect{X: cx - dotD/2, Y: cy - dotD/2, W: dotD, H: dotD}, dotCol, dotD/2)
		})
	}

	return el
}

func avatarStatusColor(th *Theme, st AvatarStatus) ui.Color {
	switch st {
	case AvatarStatusOnline:
		return semanticColorsFor(th, ToneSuccess).accent
	case AvatarStatusBusy:
		return semanticColorsFor(th, ToneError).accent
	case AvatarStatusAway:
		return semanticColorsFor(th, ToneWarning).accent
	default:
		return th.Colors.Outline
	}
}

func avatarInitials(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return ""
	}
	if len(fields) >= 2 {
		r1, _ := utf8.DecodeRuneInString(fields[0])
		r2, _ := utf8.DecodeRuneInString(fields[len(fields)-1])
		return strings.ToUpper(string([]rune{r1, r2}))
	}
	runes := []rune(fields[0])
	if len(runes) == 1 {
		return strings.ToUpper(string(runes))
	}
	if unicode.Is(unicode.Han, runes[0]) {
		return string(runes[:1])
	}
	limit := min(len(runes), 2)
	return strings.ToUpper(string(runes[:limit]))
}
