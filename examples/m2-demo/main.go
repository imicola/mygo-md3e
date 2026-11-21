// Command m2-demo is an interactive native MyGo preview of everything
// implemented through milestone M2 (T2.4):
//   - tokens: HCT color space, DynamicScheme (7 variants, 3 contrast levels,
//     light/dark, 45+ semantic color roles), shape & elevation tokens.
//   - motion: MD3E Expressive/Standard spring schemes with exact ζ -> c
//     damping conversion and live spring response plotting.
//   - shape: all 35 built-in MD3E shapes and bidirectional cubic morphing.
//
// Run from the repository root:
//
//	CGO_ENABLED=0 go run ./examples/m2-demo
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

type seedPreset struct {
	name  string
	color ui.Color
}

var seedPresets = []seedPreset{
	{"Purple", ui.Hex("#6750A4")},
	{"Blue", ui.Hex("#0061A4")},
	{"Green", ui.Hex("#2E6B3E")},
	{"Coral", ui.Hex("#BA1A1A")},
	{"Amber", ui.Hex("#8C5000")},
	{"Teal", ui.Hex("#006A6A")},
}

var variantLabels = []string{
	"TonalSpot", "Expressive", "Vibrant", "Neutral", "Monochrome", "Fidelity", "Content",
}

var curatedPairs = [][2]string{
	{"Circle", "SoftBurst"},
	{"Square", "Heart"},
	{"Pentagon", "Sunny"},
	{"Pill", "Cookie9Sided"},
	{"Arrow", "Clover8Leaf"},
	{"PixelTriangle", "Gem"},
	{"Ghostish", "Flower"},
}

type demoState struct {
	// Theme & tokens state.
	seedIdx     int
	variantIdx  int
	darkIdx     int // 0: Light, 1: Dark
	contrastIdx int // 0: Standard, 1: Medium, 2: High

	// Motion state.
	motionSchemeIdx int // 0: Expressive, 1: Standard
	motionTokenIdx  int // 0: FastSpatial, 1: DefaultSpatial, 2: SlowSpatial

	// Shape & morph state.
	shapes      []shape.Shape
	nameIndex   map[string]int
	morphA      int
	morphB      int
	morphTarget float32 // 0 targets morphA, 1 targets morphB
	morph       *shape.Morph
	pairIdx     int
	manualScrub bool
	scrubValue  float64
	lastRaw     float32
}

func newDemoState() *demoState {
	shapes := shape.All()
	nameIndex := make(map[string]int, len(shapes))
	for i, s := range shapes {
		nameIndex[s.Name] = i
	}
	a := nameIndex["Circle"]
	b := nameIndex["SoftBurst"]
	return &demoState{
		shapes:      shapes,
		nameIndex:   nameIndex,
		morphA:      a,
		morphB:      b,
		morphTarget: 1,
		morph:       shape.NewMorph(shapes[a].Polygon(), shapes[b].Polygon()),
		scrubValue:  0.5,
	}
}

func (d *demoState) scheme() *tokens.DynamicScheme {
	return tokens.NewDynamicScheme(tokens.DynamicSchemeOptions{
		Seed:     seedPresets[d.seedIdx].color,
		Dark:     d.darkIdx == 1,
		Variant:  tokens.Variant(d.variantIdx),
		Contrast: tokens.Contrast(d.contrastIdx),
	})
}

func (d *demoState) motionScheme() motion.Scheme {
	if d.motionSchemeIdx == 1 {
		return motion.Standard
	}
	return motion.Expressive
}

func (d *demoState) motionToken() motion.Token {
	switch d.motionTokenIdx {
	case 1:
		return motion.DefaultSpatial
	case 2:
		return motion.SlowSpatial
	default:
		return motion.FastSpatial
	}
}

func (d *demoState) currentShapeIdx() int {
	if d.morphTarget >= 0.5 {
		return d.morphB
	}
	return d.morphA
}

func (d *demoState) morphTo(idx int) {
	if idx < 0 || idx >= len(d.shapes) {
		return
	}
	if d.manualScrub {
		d.morphA = d.currentShapeIdx()
		d.morphB = idx
		d.morph = shape.NewMorph(d.shapes[d.morphA].Polygon(), d.shapes[d.morphB].Polygon())
		d.morphTarget = 1
		return
	}
	cur := d.currentShapeIdx()
	if idx == cur {
		// Re-trigger a bounce from the opposite endpoint.
		d.morphTarget = 1 - d.morphTarget
		return
	}
	// Keep the currently resting endpoint at the current target value so
	// AnimateTo springs smoothly from the visible shape to the newly chosen one.
	if d.morphTarget >= 0.5 {
		d.morphA = idx
		d.morphB = cur
		d.morph = shape.NewMorph(d.shapes[d.morphA].Polygon(), d.shapes[d.morphB].Polygon())
		d.morphTarget = 0
	} else {
		d.morphA = cur
		d.morphB = idx
		d.morph = shape.NewMorph(d.shapes[d.morphA].Polygon(), d.shapes[d.morphB].Polygon())
		d.morphTarget = 1
	}
}

func (d *demoState) nextCuratedPair() {
	d.pairIdx = (d.pairIdx + 1) % len(curatedPairs)
	pair := curatedPairs[d.pairIdx]
	a, okA := d.nameIndex[pair[0]]
	b, okB := d.nameIndex[pair[1]]
	if !okA || !okB {
		return
	}
	d.morphA = a
	d.morphB = b
	d.morph = shape.NewMorph(d.shapes[a].Polygon(), d.shapes[b].Polygon())
	d.morphTarget = 1
}

func (d *demoState) swapMorph() {
	d.morphTarget = 1 - d.morphTarget
}

func uiThemeFromScheme(cs tokens.ColorScheme, dark bool) *ui.Theme {
	var base *ui.Theme
	if dark {
		base = ui.DarkTheme()
	} else {
		base = ui.LightTheme()
	}
	base.Background = cs.Surface
	base.Surface = cs.SurfaceContainerLow
	base.SurfaceHover = cs.SurfaceContainer
	base.SurfacePressed = cs.SurfaceContainerHigh
	base.Border = cs.OutlineVariant
	base.Text = cs.OnSurface
	base.TextMuted = cs.OnSurfaceVariant
	base.Accent = cs.Primary
	base.AccentHover = cs.Primary.Mix(cs.OnPrimary, 0.08)
	base.AccentPressed = cs.Primary.Mix(cs.OnPrimary, 0.14)
	base.AccentText = cs.OnPrimary
	base.InactiveAccent = cs.Secondary
	base.Danger = cs.Error
	base.Selection = cs.PrimaryContainer
	base.Focus = cs.Primary
	base.Inverse = cs.InverseSurface
	base.InverseText = cs.InverseOnSurface
	base.Radius = float32(tokens.ShapeSmall.Radius())
	return base
}

func (d *demoState) view(c *ui.Context) {
	ds := d.scheme()
	cs := ds.Colors
	c.SetTheme(uiThemeFromScheme(cs, d.darkIdx == 1))

	spec := motion.SpecOf(d.motionScheme(), d.motionToken())
	anim := motion.Resolve(d.motionScheme(), d.motionToken(), c.Preferences().ReduceMotion)
	elev := tokens.Level1.Spec()
	shadowColor := cs.Shadow.Alpha(float32(elev.Key.Opacity * 0.45))

	ui.Column(c).Fill().Background(cs.Surface).Padding(16).Gap(12).Children(func() {
		d.headerBar(c, cs, shadowColor)
		ui.Row(c).Grow(1).Gap(14).AlignItems(ui.Stretch).Children(func() {
			d.leftStudio(c, cs, spec, anim, shadowColor)
			d.rightCatalogAndColors(c, ds, cs, shadowColor)
		})
	})
}

func (d *demoState) headerBar(c *ui.Context, cs tokens.ColorScheme, shadow ui.Color) {
	ui.Row(c).Padding(12, 16).Radius(float32(tokens.ShapeLarge.Radius())).
		Background(cs.SurfaceContainer).Border(1, cs.OutlineVariant).
		Shadow(0, 1, 3, 0, shadow).
		AlignItems(ui.Center).Gap(12).Wrap().Children(func() {
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, "MyGo MD3E — M1 & M2 Native Preview").FontSize(18).Bold().TextColor(cs.OnSurface)
			ui.Text(c, "Milestone T2.4 · tokens (HCT/DynamicScheme) + motion (12 Springs) + shape (35 Shapes & Morph)").
				FontSize(12).TextColor(cs.OnSurfaceVariant)
		})
		ui.Segmented(c.Key("appearance"), &d.darkIdx, "Light", "Dark").Label("Appearance")
		ui.Segmented(c.Key("contrast"), &d.contrastIdx, "Std", "Med", "High").Label("Contrast")
		ui.Segmented(c.Key("motion-scheme"), &d.motionSchemeIdx, "Expressive", "Standard").Label("Motion Scheme")
		ui.Segmented(c.Key("motion-token"), &d.motionTokenIdx, "Fast", "Default", "Slow").Label("Motion Speed")
	})
}

func (d *demoState) leftStudio(c *ui.Context, cs tokens.ColorScheme, spec motion.Spec, anim ui.Animation, shadow ui.Color) {
	ui.Column(c).Width(400).Gap(12).Children(func() {
		// Hero Shape Morph Card.
		ui.Column(c).Padding(16).Gap(12).Radius(float32(tokens.ShapeExtraLarge.Radius())).
			Background(cs.SurfaceContainerLow).Border(1, cs.OutlineVariant).
			Shadow(0, 2, 6, 0, shadow).Children(func() {
			fromName := d.shapes[d.morphA].Name
			toName := d.shapes[d.morphB].Name
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, "Shape Morph Studio").FontSize(15).Bold().TextColor(cs.OnSurface)
					ui.Textf(c, "%s  ↔  %s", fromName, toName).FontSize(12).TextColor(cs.Primary)
				})
				ui.Box(c).Padding(4, 10).Radius(float32(tokens.ShapeFull.Radius())).
					Background(cs.SecondaryContainer).Children(func() {
					ui.Textf(c, "Target: %s", d.shapes[d.currentShapeIdx()].Name).
						FontSize(11).Bold().TextColor(cs.OnSecondaryContainer)
				})
			})

			hero := ui.Box(c.Key("hero-morph")).Height(236).Radius(float32(tokens.ShapeLarge.Radius())).
				Background(cs.SurfaceContainerLowest).Border(1, cs.OutlineVariant)

			raw := hero.AnimateTo("morph", d.morphTarget, anim)
			if d.manualScrub {
				raw = float32(d.scrubValue)
			}
			d.lastRaw = raw
			clamped := max(float32(0), min(float32(1), raw))
			overshoot := float32(0)
			if !d.manualScrub {
				if d.morphTarget >= 0.5 && raw > 1 {
					overshoot = raw - 1
				} else if d.morphTarget < 0.5 && raw < 0 {
					overshoot = -raw
				}
			}

			hero.Draw(func(p *ui.Painter, r ui.Rect) {
				side := min(r.W, r.H) * 0.72
				scale := 1 + min(float32(0.14), overshoot*0.6)
				s := side * scale
				cx, cy := r.X+r.W/2, r.Y+r.H/2
				halo := ui.Rect{X: cx - side*0.56, Y: cy - side*0.56, W: side * 1.12, H: side * 1.12}
				p.Fill(halo, cs.PrimaryContainer.Alpha(0.45), halo.W/2)

				dst := ui.Rect{X: cx - s/2, Y: cy - s/2, W: s, H: s}
				shape.DrawMorph(p, d.morph, clamped, dst, cs.Primary)
			})

			// Progress & overshoot gauge.
			ui.Box(c.Key("progress-gauge")).Height(20).Draw(func(p *ui.Painter, r ui.Rect) {
				track := ui.Rect{X: r.X, Y: r.Y + 7, W: r.W * 0.85, H: 6}
				p.Fill(track, cs.SurfaceContainerHighest, 3)
				// Draw 1.0 reference tick.
				p.Line(track.X+track.W, r.Y+3, track.X+track.W, r.Y+17, 1.5, cs.Outline)
				fillW := max(float32(0), track.W*raw)
				fillColor := cs.Primary
				if raw > 1.005 || raw < -0.005 {
					fillColor = cs.Tertiary
				}
				if fillW > 0 {
					p.Fill(ui.Rect{X: track.X, Y: track.Y, W: min(fillW, r.W), H: 6}, fillColor, 3)
				}
				label := ui.Span{
					Text:   fmt.Sprintf("t=%.2f", raw),
					Size:   11,
					Weight: 600,
					Color:  cs.OnSurfaceVariant,
				}
				w, h := p.MeasureText(0, label)
				p.RichText(r.X+r.W-w, r.Y+(r.H-h)/2, 0, label)
			})

			// Morph actions.
			ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				if ui.PrimaryButton(c.Key("swap-morph"), "Morph A ↔ B").Clicked() {
					d.swapMorph()
				}
				if ui.Button(c.Key("next-pair"), "Next Pair").Clicked() {
					d.nextCuratedPair()
				}
				ui.Checkbox(c.Key("manual-scrub"), &d.manualScrub, "Scrub")
			})

			if d.manualScrub {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "0").FontSize(11).TextColor(cs.OnSurfaceVariant)
					ui.Slider(c.Key("scrub-slider"), &d.scrubValue, 0, 1).Grow(1).Label("Morph Progress")
					ui.Text(c, "1").FontSize(11).TextColor(cs.OnSurfaceVariant)
				})
			}
		})

		// Motion Physics Curve Card.
		ui.Column(c).Grow(1).Padding(16).Gap(10).Radius(float32(tokens.ShapeLarge.Radius())).
			Background(cs.SurfaceContainerLow).Border(1, cs.OutlineVariant).
			Shadow(0, 1, 4, 0, shadow).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Spring Physics Curve").FontSize(14).Bold().TextColor(cs.OnSurface).Grow(1)
				ui.Textf(c, "%s · %s", spec.Scheme, spec.Token).FontSize(12).TextColor(cs.Tertiary)
			})
			ui.Textf(c, "ζ = %.1f   k = %.0f   c = 2ζ√k = %.2f   Settle ≈ %dms",
				spec.DampingRatio, spec.Stiffness, spec.Damping(), spec.SettleTime().Milliseconds()).
				FontSize(11).TextColor(cs.OnSurfaceVariant)

			ui.Box(c.Key("spring-curve")).Grow(1).MinHeight(110).Radius(float32(tokens.ShapeMedium.Radius())).
				Background(cs.SurfaceContainerLowest).Border(1, cs.OutlineVariant).
				Draw(func(p *ui.Painter, r ui.Rect) {
					drawSpringCurve(p, r, spec, cs)
				})
		})
	})
}

func drawSpringCurve(p *ui.Painter, r ui.Rect, spec motion.Spec, cs tokens.ColorScheme) {
	padX, padY := float32(14), float32(14)
	w := r.W - 2*padX
	h := r.H - 2*padY
	if w <= 10 || h <= 10 {
		return
	}
	// Map progress y in [0, 1.2] to plot coordinates.
	yOf := func(val float64) float32 {
		norm := val / 1.2
		return r.Y + padY + h*float32(1-norm)
	}
	// Baseline (0) and target (1.0) reference lines.
	y0 := yOf(0)
	y1 := yOf(1)
	p.Line(r.X+padX, y0, r.X+padX+w, y0, 1, cs.OutlineVariant)
	p.Line(r.X+padX, y1, r.X+padX+w, y1, 1, cs.Outline.Alpha(0.6))

	opts := spec.SpringOptions()
	const samples = 90
	const windowMS = 750.0
	var path ui.Path
	peak := 0.0
	for i := 0; i <= samples; i++ {
		frac := float64(i) / samples
		t := time.Duration(frac*windowMS) * time.Millisecond
		v := opts.Progress(t)
		if v > peak {
			peak = v
		}
		px := r.X + padX + w*float32(frac)
		py := yOf(v)
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
	p.StrokePath(&path, 2.5, cs.Tertiary)

	tag := ui.Span{
		Text:   fmt.Sprintf("peak %.3f (0–750ms)", peak),
		Size:   10,
		Weight: 600,
		Color:  cs.OnSurfaceVariant,
	}
	tw, _ := p.MeasureText(0, tag)
	p.RichText(r.X+r.W-padX-tw, r.Y+6, 0, tag)
}

func (d *demoState) rightCatalogAndColors(c *ui.Context, ds *tokens.DynamicScheme, cs tokens.ColorScheme, shadow ui.Color) {
	ui.Column(c).Grow(1).Gap(12).Children(func() {
		// 35 Shapes Catalog Card.
		ui.Column(c).Grow(1).Padding(12, 16).Gap(8).Radius(float32(tokens.ShapeLarge.Radius())).
			Background(cs.SurfaceContainerLow).Border(1, cs.OutlineVariant).
			Shadow(0, 1, 4, 0, shadow).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "35 Material 3 Expressive Shapes (Click any shape to morph)").
					FontSize(14).Bold().TextColor(cs.OnSurface).Grow(1)
				ui.Textf(c, "%d shapes", len(d.shapes)).FontSize(12).TextColor(cs.OnSurfaceVariant)
			})
			ui.Scroll(c.Key("shapes-scroll")).Grow(1).Children(func() {
				ui.Row(c).Wrap().Gap(6).Padding(2).Children(func() {
					activeIdx := d.currentShapeIdx()
					for i, s := range d.shapes {
						isTarget := i == activeIdx
						isOther := (i == d.morphA || i == d.morphB) && !isTarget

						bg := cs.SurfaceContainer
						border := cs.OutlineVariant
						fg := cs.Primary
						textCol := cs.OnSurfaceVariant
						if isTarget {
							bg = cs.PrimaryContainer
							border = cs.Primary
							fg = cs.OnPrimaryContainer
							textCol = cs.OnPrimaryContainer
						} else if isOther {
							bg = cs.SecondaryContainer.Alpha(0.55)
							border = cs.Secondary
						}

						tile := ui.ButtonBase(c.Key(s.Name)).Width(84).Height(70).
							Column().Center().Padding(4).Gap(4).
							Radius(float32(tokens.ShapeMedium.Radius())).
							Background(bg).Border(1, border).
							Label("Morph to " + s.Name)
						if tile.Clicked() {
							d.morphTo(i)
						}
						tile.Children(func() {
							poly := s.Polygon()
							col := fg
							ui.Box(c).Size(36, 36).Draw(func(p *ui.Painter, r ui.Rect) {
								p.FillPath(poly.ToPath(r), col)
							})
							ui.Text(c, s.Name).FontSize(9.5).TextColor(textCol)
						})
					}
				})
			})
		})

		// Dynamic Color & HCT Card.
		d.colorRolesCard(c, ds, cs, shadow)
	})
}

func (d *demoState) colorRolesCard(c *ui.Context, ds *tokens.DynamicScheme, cs tokens.ColorScheme, shadow ui.Color) {
	ui.Column(c).Padding(16).Gap(10).Radius(float32(tokens.ShapeLarge.Radius())).
		Background(cs.SurfaceContainerLow).Border(1, cs.OutlineVariant).
		Shadow(0, 1, 4, 0, shadow).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(10).Wrap().Children(func() {
			ui.Text(c, "HCT & DynamicScheme").FontSize(14).Bold().TextColor(cs.OnSurface).Grow(1)
			// Seed pills.
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				for i, sp := range seedPresets {
					selected := i == d.seedIdx
					border := cs.OutlineVariant
					bw := float32(1)
					if selected {
						border = cs.OnSurface
						bw = 2
					}
					btn := ui.ButtonBase(c.Key("seed-"+sp.name)).Size(26, 26).
						Radius(13).Background(sp.color).Border(bw, border).
						Label("Seed " + sp.name)
					if btn.Clicked() {
						d.seedIdx = i
					}
				}
			})
		})

		ui.Segmented(c.Key("variant"), &d.variantIdx, variantLabels...).Label("Scheme Variant")

		seedHCT := ds.SourceHCT
		ratio := tokens.ContrastRatio(cs.Primary, cs.OnPrimary)
		ui.Textf(c, "Seed %s · HCT(H=%.1f°, C=%.1f, T=%.1f) · Primary/OnPrimary contrast %.2f:1",
			seedPresets[d.seedIdx].name, seedHCT.Hue, seedHCT.Chroma, seedHCT.Tone, ratio).
			FontSize(11).TextColor(cs.OnSurfaceVariant)

		// Semantic color role swatches.
		type swatch struct {
			name   string
			bg, fg ui.Color
		}
		swatches := []swatch{
			{"Primary", cs.Primary, cs.OnPrimary},
			{"PrimaryCont", cs.PrimaryContainer, cs.OnPrimaryContainer},
			{"Secondary", cs.Secondary, cs.OnSecondary},
			{"SecondaryCont", cs.SecondaryContainer, cs.OnSecondaryContainer},
			{"Tertiary", cs.Tertiary, cs.OnTertiary},
			{"TertiaryCont", cs.TertiaryContainer, cs.OnTertiaryContainer},
			{"Error", cs.Error, cs.OnError},
			{"SurfLowest", cs.SurfaceContainerLowest, cs.OnSurface},
			{"SurfCont", cs.SurfaceContainer, cs.OnSurface},
			{"SurfHighest", cs.SurfaceContainerHighest, cs.OnSurface},
		}
		ui.Row(c).Wrap().Gap(6).Children(func() {
			for _, sw := range swatches {
				ui.Column(c).Width(94).Height(44).Padding(6, 8).Center().
					Radius(float32(tokens.ShapeSmall.Radius())).
					Background(sw.bg).Border(1, cs.OutlineVariant.Alpha(0.5)).
					Children(func() {
						ui.Text(c, sw.name).FontSize(10).Bold().TextColor(sw.fg)
						ui.Textf(c, "#%02X%02X%02X", sw.bg.R, sw.bg.G, sw.bg.B).FontSize(9).TextColor(sw.fg.Alpha(0.85))
					})
			}
		})
	})
}

func main() {
	d := newDemoState()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "MyGo MD3E — M1 & M2 Preview (T2.4)",
			Width:   1240,
			Height:  820,
			Content: ui.View(d.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
