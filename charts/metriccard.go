package charts

import (
	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// MetricProgress configures an optional MD3E progress bar embedded at the
// bottom of MetricCard.
type MetricProgress struct {
	// Value is the progress in 0..1, or < 0 for indeterminate animation.
	Value float64
	// Buffer is an optional 0..1 secondary buffered track value.
	Buffer float64
	// Wavy renders md3e.LinearWavyProgress when true, or md3e.LinearProgress when false.
	Wavy bool
	// Color overrides the active indicator color when A > 0.
	Color ui.Color
	// Thickness overrides the progress stroke thickness in DIP.
	Thickness float32
	// Label names the progress bar for assistive technology.
	Label string
}

// MetricCardOptions configures MetricCard.
type MetricCardOptions struct {
	// Title is the primary metric heading (for example "Download" or "Memory").
	Title string
	// Subtitle is an optional secondary descriptor shown under Title.
	Subtitle string
	// Icon optionally draws a leading 20 DIP icon inside a compact tonal IconBadge.
	Icon md3e.IconDraw
	// IconTone selects the semantic tone of the leading IconBadge (default TonePrimary).
	IconTone md3e.SemanticTone
	// Badge optionally renders a trailing status or delta indicator (for example
	// "Peak 142 MB/s" or "Healthy").
	Badge string
	// BadgeTone selects the semantic color role of Badge.
	BadgeTone md3e.SemanticTone
	// Pulsing renders Badge as a StatusPill with a breathing status dot when true.
	Pulsing bool

	// Value is the large tabular-numeral readout (for example "128.4").
	Value string
	// Unit is an optional muted unit suffix aligned beside Value (for example "MB/s").
	Unit string
	// Secondary is an optional auxiliary metric or peak caption beside/below Value.
	Secondary string
	// ValueColor overrides the foreground color of Value when A > 0.
	ValueColor ui.Color

	// Variant selects CardElevated (default), CardFilled, or CardOutlined.
	Variant md3e.CardVariant
	// ShapeScale selects the MD3E corner token (default tokens.ShapeLargeIncreased
	// for Elevated/Filled and tokens.ShapeLarge for Outlined).
	ShapeScale tokens.ShapeScale
	// Radius overrides the corner radius in DIP when positive and finite.
	Radius float32
	// Hairline draws a subtle 1 DIP tonal highlight border (1.5 DIP in
	// HighContrast). Nil defaults to false.
	Hairline *bool
	// AccentGlow paints a delicate radial ambient wash in the top corner using
	// the card's accent color. Nil defaults to true when Icon or Sparkline is set.
	AccentGlow *bool
	// Clickable makes the card interactive with ripple feedback and pressed corner morph.
	Clickable bool
	// OnClick runs when a clickable card is clicked or activated by keyboard.
	OnClick func()
	// Disabled disables interaction and applies MD3E disabled tokens.
	Disabled bool
	// Label overrides the card's accessible name (defaults to Title + Value).
	Label string

	// Sparkline optionally embeds a full-bleed AreaSparkline flush against the
	// card's rounded bottom corners. Takes precedence over Progress when both are set.
	Sparkline *AreaSparklineOptions
	// Progress optionally embeds an MD3E LinearWavyProgress or LinearProgress bar.
	Progress *MetricProgress
	// Footer optionally renders custom bottom content (such as StackedProgress).
	Footer func()
}

type metricCardCornerKey struct{}

// MetricCard builds an MD3E Expressive dashboard metric card combining an
// optional icon badge, title/subtitle, semantic status badge, large tabular-
// numeral value readout, delicate ambient radial glow, and an optional flush-
// bottom AreaSparkline, wavy progress bar, or custom footer.
func MetricCard(c *ui.Context, opts MetricCardOptions) ui.Element {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	interactive := opts.Clickable || opts.OnClick != nil
	hairline := boolDefault(opts.Hairline, false)

	baseScale := tokens.ShapeLargeIncreased
	if opts.Variant == md3e.CardOutlined {
		baseScale = tokens.ShapeLarge
	}
	if opts.ShapeScale > tokens.ShapeNone && opts.ShapeScale <= tokens.ShapeFull {
		baseScale = opts.ShapeScale
	}
	restRadius := sanitizePositiveF32(opts.Radius, th.Shapes.Radius(baseScale))

	// Density-scaled inner padding & rhythm.
	pad := float32(16)
	headerGap := float32(10)
	sectionGap := float32(10)
	valueRole := tokens.HeadlineSmall
	switch th.Density.Normalize() {
	case tokens.DensityDesktop:
		pad = 14
		headerGap = 8
		sectionGap = 8
	case tokens.DensityCompact:
		pad = 12
		headerGap = 8
		sectionGap = 6
		valueRole = tokens.TitleLarge
	}

	var el ui.Element
	if interactive {
		el = ui.ButtonBase(c).Column().Justify(ui.Start).AlignItems(ui.Stretch).
			MinWidth(0).Clip().DimDisabled(false).FocusRing(false)
	} else {
		el = ui.Column(c).AlignItems(ui.Stretch).MinWidth(0).Clip().DimDisabled(false)
	}
	el.Disabled(opts.Disabled)
	disabled := opts.Disabled || el.IsDisabled()

	bg, border, borderW, elev := metricCardPalette(th, opts.Variant, hairline, disabled)
	radius := restRadius
	if interactive {
		target := restRadius
		if !disabled && el.Pressed() {
			target = restRadius + 4
			if elev > tokens.Level0 && elev < tokens.Level5 {
				elev++
				bg = md3e.SurfaceColorForElevation(th, elev)
			}
		}
		anim := motion.Resolve(th.Motion, motion.FastSpatial, reduce)
		radius = animateF32(el, metricCardCornerKey{}, target, anim, reduce)
	}

	el.Radius(radius)
	if bg.A > 0 {
		el.Background(bg)
	}
	if borderW > 0 && border.A > 0 {
		el.Border(borderW, border)
	}
	if elev > tokens.Level0 {
		md3e.ApplyElevationShadows(th, el, elev)
	}

	ink := cs.OnSurface
	mutedInk := cs.OnSurfaceVariant
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		mutedInk = ink
	}
	el.TextColor(ink)

	// Ambient corner glow: pure, ghost-draw safe, clipped inside card corners.
	wantGlow := boolDefault(opts.AccentGlow, opts.Icon != nil || opts.Sparkline != nil)
	if wantGlow && !disabled {
		glowCol := metricCardAccentColor(th, opts)
		glowAlpha := float32(0.065)
		if th.Dark {
			glowAlpha = 0.095
		}
		drawRadius := radius
		rtl := c.Direction() == ui.RightToLeft
		el.Draw(func(p *ui.Painter, r ui.Rect) {
			if !rectIntersects(r, p.Visible()) {
				return
			}
			cx := float32(0.14)
			if rtl {
				cx = 0.86
			}
			stops := [3]ui.GradientStop{
				{Color: glowCol.Alpha(glowAlpha), At: 0},
				{Color: glowCol.Alpha(glowAlpha * 0.35), At: 0.45},
				{Color: glowCol.Alpha(0), At: 1},
			}
			p.Clip(r, drawRadius, func() {
				p.FillRadialGradient(r, ui.RadialGradient{
					Stops:  stops[:],
					Center: ui.Point{X: cx, Y: 0.12},
					Oklab:  true,
				}, 0)
			})
		})
	}

	if interactive {
		borderInset := float32(0)
		if borderW > 0 && border.A > 0 {
			borderInset = borderW
		}
		md3e.AttachRipple(c, el, md3e.StateLayerOptions{
			Color:       ink,
			Radius:      radius,
			BorderInset: borderInset,
			Disabled:    disabled,
		})
	}

	label := opts.Label
	if label == "" {
		switch {
		case opts.Title != "" && opts.Value != "":
			label = opts.Title + ": " + opts.Value
			if opts.Unit != "" {
				label += " " + opts.Unit
			}
		case opts.Title != "":
			label = opts.Title
		default:
			label = opts.Value
		}
	}
	if label != "" {
		el.Label(label)
	}

	clicked := interactive && el.Clicked()

	hasFlushSparkline := opts.Sparkline != nil
	el.Children(func() {
		// Upper content block with density-aware padding.
		bottomPad := pad
		if hasFlushSparkline {
			bottomPad = max(4, pad*0.45)
		}
		ui.Column(c.Key("body")).FillWidth().MinWidth(0).
			Padding(pad, pad, bottomPad, pad).
			Gap(sectionGap).Children(func() {
			// Header row: optional icon badge + title/subtitle + optional status/tag badge.
			if opts.Icon != nil || opts.Title != "" || opts.Subtitle != "" || opts.Badge != "" {
				ui.Row(c.Key("header")).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(headerGap).Children(func() {
					if opts.Icon != nil {
						badgeVariant := iconBadgeVariantForTone(opts.IconTone)
						iconBoxSize := float32(34)
						iconSize := float32(18)
						if th.Density.Normalize() == tokens.DensityCompact {
							iconBoxSize = 28
							iconSize = 16
						}
						md3e.IconBadge(c.Key("icon"), opts.Icon, md3e.IconBadgeOptions{
							Variant:       badgeVariant,
							ContainerSize: iconBoxSize,
							IconSize:      iconSize,
							ShapeScale:    tokens.ShapeMedium,
						})
					}
					ui.Column(c.Key("titles")).Grow(1).MinWidth(0).Gap(1).Children(func() {
						if opts.Title != "" {
							md3e.TextWith(c, opts.Title, md3e.TextOptions{
								Role:       tokens.TitleSmall,
								Emphasized: true,
								MaxLines:   1,
								Color:      ink,
							})
						}
						if opts.Subtitle != "" {
							md3e.TextWith(c, opts.Subtitle, md3e.TextOptions{
								Role:     tokens.BodySmall,
								MaxLines: 1,
								Color:    mutedInk,
							})
						}
					})
					if opts.Badge != "" {
						if opts.Pulsing {
							md3e.StatusPill(c.Key("badge"), opts.Badge, md3e.StatusPillOptions{
								Tone:    opts.BadgeTone,
								Pulsing: !disabled,
							})
						} else {
							md3e.Tag(c.Key("badge"), opts.Badge, md3e.TagOptions{
								Tone:     opts.BadgeTone,
								Disabled: disabled,
							})
						}
					}
				})
			}

			// Primary value row: large tabular-numeral readout + optional unit & secondary caption.
			if opts.Value != "" || opts.Unit != "" || opts.Secondary != "" {
				valCol := ink
				if !disabled && opts.ValueColor.A > 0 {
					valCol = opts.ValueColor
				}
				ui.Row(c.Key("value-row")).FillWidth().MinWidth(0).AlignItems(ui.Baseline).Justify(ui.SpaceBetween).Gap(8).Wrap().Children(func() {
					ui.Row(c.Key("primary-val")).AlignItems(ui.Baseline).Gap(4).Children(func() {
						if opts.Value != "" {
							md3e.TextWith(c, opts.Value, md3e.TextOptions{
								Role:        valueRole,
								Emphasized:  true,
								TabularNums: true,
								Color:       valCol,
							}).NoWrap()
						}
						if opts.Unit != "" {
							md3e.TextWith(c, opts.Unit, md3e.TextOptions{
								Role:  tokens.LabelMedium,
								Color: mutedInk,
							}).NoWrap()
						}
					})
					if opts.Secondary != "" {
						md3e.TextWith(c, opts.Secondary, md3e.TextOptions{
							Role:        tokens.LabelSmall,
							TabularNums: true,
							Color:       mutedInk,
						}).NoWrap()
					}
				})
			}

			// Embedded progress bar when no flush sparkline is active.
			if !hasFlushSparkline && opts.Progress != nil {
				prog := opts.Progress
				pOpts := md3e.ProgressOptions{
					Thickness: prog.Thickness,
					Buffer:    prog.Buffer,
					Label:     prog.Label,
					Disabled:  disabled,
				}
				if pOpts.Label == "" && opts.Title != "" {
					pOpts.Label = opts.Title + " progress"
				}
				drawProg := func() {
					if prog.Wavy {
						md3e.LinearWavyProgress(c.Key("progress"), prog.Value, pOpts)
					} else {
						md3e.LinearProgress(c.Key("progress"), prog.Value, pOpts)
					}
				}
				if prog.Color.A > 0 && !disabled {
					customTh := *th
					customTh.Colors.Primary = prog.Color
					md3e.ThemeScope(c.Key("prog-theme"), &customTh, drawProg)
				} else {
					drawProg()
				}
			}

			if !hasFlushSparkline && opts.Footer != nil {
				opts.Footer()
			}
		})

		// Edge-to-edge bottom-flush sparkline clipped neatly to the card's inner bottom corners.
		if hasFlushSparkline {
			spOpts := *opts.Sparkline
			if spOpts.FillTopAlpha <= 0 && !spOpts.NoFill {
				spOpts.FillTopAlpha = 0.10
			}
			if spOpts.BottomRadius <= 0 && spOpts.Radii == [4]float32{} {
				innerRad := radius
				if borderW > 0 && border.A > 0 {
					innerRad = max(radius-borderW, 0)
				}
				spOpts.BottomRadius = innerRad
			}
			if spOpts.Height <= 0 {
				spOpts.Height = 46
				if th.Density.Normalize() == tokens.DensityCompact {
					spOpts.Height = 36
				}
			}
			if spOpts.Label == "" && opts.Title != "" {
				spOpts.Label = opts.Title + " trend"
			}
			AreaSparkline(c.Key("sparkline"), spOpts)
		}
	})

	if clicked && opts.OnClick != nil && !disabled {
		opts.OnClick()
	}
	return el
}

func metricCardPalette(th *md3e.Theme, v md3e.CardVariant, hairline, disabled bool) (bg, border ui.Color, borderW float32, elev tokens.ElevationLevel) {
	cs := th.Colors
	if disabled {
		if v == md3e.CardOutlined {
			return cs.Surface, cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)), 1, tokens.Level0
		}
		return cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)), ui.Transparent, 0, tokens.Level0
	}
	hc := th.Contrast >= tokens.ContrastHigh
	bw := float32(1)
	if hc {
		bw = 1.5
	}
	switch v {
	case md3e.CardOutlined:
		bg = cs.Surface
		border = cs.OutlineVariant
		if hc && cs.Outline.A > 0 {
			border = cs.Outline
		}
		borderW = bw
	default: // CardElevated & CardFilled share borderless tonal Low / E0
		bg = cs.SurfaceContainerLow
		if bg.A == 0 {
			bg = md3e.SurfaceColorForElevation(th, tokens.Level1)
		}
	}
	if hairline && border.A == 0 {
		borderW = bw
		if hc && cs.Outline.A > 0 {
			border = cs.Outline
		} else if th.Dark {
			border = cs.OutlineVariant.Mix(cs.OnSurface, 0.15).Alpha(0.42)
		} else {
			border = cs.OutlineVariant.Alpha(0.45)
		}
	} else if hc && border.A == 0 {
		borderW = 1.5
		border = cs.Outline
		if border.A == 0 {
			border = cs.OnSurface
		}
	}
	return bg, border, borderW, elev
}

func metricCardAccentColor(th *md3e.Theme, opts MetricCardOptions) ui.Color {
	if opts.Sparkline != nil && opts.Sparkline.Color.A > 0 {
		return opts.Sparkline.Color
	}
	if opts.Progress != nil && opts.Progress.Color.A > 0 {
		return opts.Progress.Color
	}
	if opts.ValueColor.A > 0 {
		return opts.ValueColor
	}
	switch opts.IconTone {
	case md3e.ToneTertiary:
		return th.Colors.Tertiary
	case md3e.ToneSecondary:
		return th.Colors.Secondary
	case md3e.ToneError:
		return th.Colors.Error
	default:
		return th.Colors.Primary
	}
}

func iconBadgeVariantForTone(tone md3e.SemanticTone) md3e.IconBadgeVariant {
	switch tone {
	case md3e.ToneTertiary:
		return md3e.IconBadgeTertiaryTonal
	case md3e.ToneError:
		return md3e.IconBadgeErrorTonal
	case md3e.ToneNeutral:
		return md3e.IconBadgeTonal
	default:
		return md3e.IconBadgePrimaryTonal
	}
}
