package md3e

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

func fmtColorRGBA(c ui.Color) string {
	return fmt.Sprintf("rgba(%d,%d,%d,%d)", c.R, c.G, c.B, c.A)
}

func TestSurfaceStylePlumbingCachesAndScopes(t *testing.T) {
	seed := ui.Hex("#E06D1B")
	tonal := ThemeFromSeed(seed, ThemeOptions{
		Dark:         true,
		Variant:      tokens.Expressive,
		SurfaceStyle: tokens.SurfaceTonal,
	})
	neutral := ThemeFromSeed(seed, ThemeOptions{
		Dark:         true,
		Variant:      tokens.Expressive,
		SurfaceStyle: tokens.SurfaceNeutral,
	})
	if tonal.SurfaceStyle != tokens.SurfaceTonal || neutral.SurfaceStyle != tokens.SurfaceNeutral {
		t.Fatalf("SurfaceStyle mismatch: tonal=%v neutral=%v", tonal.SurfaceStyle, neutral.SurfaceStyle)
	}
	if tonal.Colors.SurfaceContainerLow == neutral.Colors.SurfaceContainerLow {
		t.Fatalf("expected SurfaceNeutral to produce distinct low-chroma SurfaceContainerLow from Expressive SurfaceTonal (%v)", tonal.Colors.SurfaceContainerLow)
	}
	if tonal.Colors.Primary != neutral.Colors.Primary {
		t.Fatalf("SurfaceNeutral must preserve Variant Primary: tonal=%v neutral=%v", tonal.Colors.Primary, neutral.Colors.Primary)
	}
	if invalid := ThemeFromSeed(seed, ThemeOptions{SurfaceStyle: tokens.SurfaceStyle(77)}); invalid.SurfaceStyle != tokens.SurfaceTonal {
		t.Fatalf("out-of-range SurfaceStyle = %v, want SurfaceTonal", invalid.SurfaceStyle)
	}

	// WithPreferences HighContrast round-trip preserves SurfaceStyle and regenerates neutral HC surfaces.
	hcNeutral := neutral.WithPreferences(ui.Preferences{HighContrast: true, TextScale: 1})
	hcTonal := tonal.WithPreferences(ui.Preferences{HighContrast: true, TextScale: 1})
	if hcNeutral.SurfaceStyle != tokens.SurfaceNeutral || hcNeutral.Contrast != tokens.ContrastHigh {
		t.Fatalf("HighContrast WithPreferences dropped SurfaceStyle: %+v", hcNeutral)
	}
	if hcNeutral.Colors.SurfaceContainerHigh == hcTonal.Colors.SurfaceContainerHigh {
		t.Fatalf("HighContrast regeneration collapsed SurfaceNeutral into SurfaceTonal")
	}
	restored := hcNeutral.WithPreferences(ui.Preferences{HighContrast: false, TextScale: 1})
	if restored.SurfaceStyle != tokens.SurfaceNeutral || restored.Colors != neutral.Colors {
		t.Fatalf("WithPreferences round-trip did not restore original SurfaceNeutral colors")
	}

	// Nested ThemeScope + Overlay + AnimatedThemeScope preserve SurfaceStyle.
	var scopeGot, overlayGot *Theme
	active := tonal
	tt := ui.NewTester(func(c *ui.Context) {
		AnimatedThemeScope(c, active, func() {
			scopeGot = CurrentTheme(c)
			ui.Box(c).Size(60, 40)
			ui.Overlay(c, func() {
				overlayGot = CurrentTheme(c)
				ui.Box(c).Size(20, 20)
			})
		}).Fill()
	}, 160, 120)

	if scopeGot == nil || scopeGot.SurfaceStyle != tokens.SurfaceTonal || overlayGot == nil || overlayGot.SurfaceStyle != tokens.SurfaceTonal {
		t.Fatalf("initial scope/overlay SurfaceStyle mismatch: scope=%v overlay=%v", scopeGot, overlayGot)
	}
	active = neutral
	tt.Frame()
	if !tt.FramePending() {
		t.Fatal("AnimatedThemeScope did not animate SurfaceTonal -> SurfaceNeutral transition")
	}
	tt.Advance(400 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("AnimatedThemeScope did not settle after SurfaceStyle transition")
	}
	if scopeGot.SurfaceStyle != tokens.SurfaceNeutral || scopeGot.Colors != neutral.Colors {
		t.Fatalf("AnimatedThemeScope did not converge to SurfaceNeutral")
	}
	if overlayGot.SurfaceStyle != tokens.SurfaceNeutral || overlayGot.Colors != neutral.Colors {
		t.Fatalf("Overlay did not converge to SurfaceNeutral")
	}
}

func TestTopSpecularReflectionDarkLightHighContrastAndGhost(t *testing.T) {
	darkStd := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Dark: true, SurfaceStyle: tokens.SurfaceNeutral})
	darkHC := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Dark: true, Contrast: tokens.ContrastHigh, SurfaceStyle: tokens.SurfaceNeutral})
	lightStd := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Dark: false, SurfaceStyle: tokens.SurfaceNeutral})

	renderSurface := func(th *Theme, elev tokens.ElevationLevel) (topPx, midPx, cornerOutPx ui.Color) {
		img := ui.Render(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Box(c).Size(120, 80).Background(th.Colors.Surface).Padding(10).Children(func() {
					Surface(c, SurfaceOptions{
						Elevation: elev,
						Color:     th.Colors.SurfaceContainerHigh,
						Radius:    16,
					}).Size(100, 60)
				})
			})
		}, 120, 80, 1)
		top := img.RGBAAt(60, 10)
		mid := img.RGBAAt(60, 30)
		out := img.RGBAAt(11, 11)
		return ui.RGB(top.R, top.G, top.B), ui.RGB(mid.R, mid.G, mid.B), ui.RGB(out.R, out.G, out.B)
	}

	// Dark E2/E3: top row is subtly brighter than interior, clipped inside corner radius.
	for _, elev := range []tokens.ElevationLevel{tokens.Level2, tokens.Level3} {
		topPx, midPx, cornerOut := renderSurface(darkStd, elev)
		if tokens.RelativeLuminance(topPx) <= tokens.RelativeLuminance(midPx) {
			t.Fatalf("dark E%d top specular did not brighten top inner row: top=%s mid=%s", elev, fmtColorRGBA(topPx), fmtColorRGBA(midPx))
		}
		// 3.5% white over dark SurfaceContainerHigh adds ~7..9 sRGB code points (subtle, never harsh white).
		if int(topPx.R)-int(midPx.R) > 20 {
			t.Fatalf("dark E%d top specular too strong: top=%s mid=%s", elev, fmtColorRGBA(topPx), fmtColorRGBA(midPx))
		}
		if cornerOut == topPx {
			t.Fatalf("dark E%d top specular leaked outside 16dp rounded corner: out=%s top=%s", elev, fmtColorRGBA(cornerOut), fmtColorRGBA(topPx))
		}
	}

	// Dark E1, Dark HighContrast E3, and Light E3 must NOT paint top specular reflection.
	for _, tc := range []struct {
		name string
		th   *Theme
		elev tokens.ElevationLevel
	}{
		{"dark-E1", darkStd, tokens.Level1},
		{"dark-HC-E3", darkHC, tokens.Level3},
		{"light-E3", lightStd, tokens.Level3},
	} {
		topPx, midPx, _ := renderSurface(tc.th, tc.elev)
		if topPx != midPx {
			t.Fatalf("%s unexpectedly painted top specular: top=%s mid=%s", tc.name, fmtColorRGBA(topPx), fmtColorRGBA(midPx))
		}
	}

	// Ghost-draw safety during ElementTransition exit.
	show := true
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, darkStd, func() {
			ui.Column(c).Size(140, 100).Children(func() {
				if show {
					Surface(c.Key("spec-ghost"), SurfaceOptions{
						Elevation:  tokens.Level3,
						ShapeScale: tokens.ShapeLarge,
					}).Size(100, 60).Transition(ui.ElementTransition{
						Duration: 120 * time.Millisecond,
						Exit:     &ui.Motion{Opacity: 0},
					})
				}
			})
		})
	}, 140, 100)
	show = false
	tt.Frame()
	tt.Advance(250 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("exiting E3 Surface with top specular left pending frames")
	}
}

// TestContrastMatrixComponentsAndLerpTheme is the root-package M17.1 release
// contrast matrix covering all 7 seeds x 7 variants x 2 SurfaceStyles x Light/Dark
// x 3 Contrasts plus LerpTheme interpolation at t = 0, 0.25, 0.5, 0.75, 1.
func TestContrastMatrixComponentsAndLerpTheme(t *testing.T) {
	seeds := []struct {
		name  string
		color ui.Color
	}{
		{"Amber", ui.Hex("#E06D1B")},
		{"Nebula", ui.Hex("#6750A4")},
		{"Harbor", ui.Hex("#0061A4")},
		{"Emerald", ui.Hex("#2E6B3E")},
		{"Gray", ui.Hex("#808080")},
		{"Black", ui.Hex("#000000")},
		{"White", ui.Hex("#FFFFFF")},
	}
	variants := []tokens.Variant{
		tokens.TonalSpot, tokens.Expressive, tokens.Vibrant,
		tokens.Neutral, tokens.Monochrome, tokens.Fidelity, tokens.Content,
	}
	styles := []tokens.SurfaceStyle{tokens.SurfaceTonal, tokens.SurfaceNeutral}
	contrasts := []tokens.Contrast{tokens.ContrastStandard, tokens.ContrastMedium, tokens.ContrastHigh}
	overlays := []float32{0, tokens.StateHoverOpacity, tokens.StatePressedOpacity}

	checkComposited := func(ctx, control string, fg, bg ui.Color, minRatio float64, withStates bool) {
		for _, a := range overlays {
			if !withStates && a > 0 {
				continue
			}
			effBg := bg
			if a > 0 {
				effBg = fg.Alpha(a).Over(bg)
			}
			if r := tokens.ContrastRatio(fg, effBg); r+1e-6 < minRatio {
				t.Errorf("%s [%s overlay=%.2f]: fg=%s bg=%s ratio=%.2f < %.2f",
					ctx, control, a, fmtColorRGBA(fg), fmtColorRGBA(effBg), r, minRatio)
			}
		}
	}

	// 1. Static theme & component pairing matrix (including ambient glow worst-case composite).
	for _, seed := range seeds {
		for _, v := range variants {
			for _, st := range styles {
				for _, dark := range []bool{false, true} {
					for _, cr := range contrasts {
						th := ThemeFromSeed(seed.color, ThemeOptions{
							Dark:         dark,
							Variant:      v,
							Contrast:     cr,
							SurfaceStyle: st,
						})
						cs := th.Colors
						ctx := fmt.Sprintf("seed=%s var=%d style=%d dark=%v contrast=%d", seed.name, v, st, dark, cr)
						textMin := 4.5
						if cr >= tokens.ContrastHigh {
							textMin = 7.0
						}

						// Theme extended Warning / Success roles.
						checkComposited(ctx, "Warning", th.OnWarning, th.Warning, textMin, true)
						checkComposited(ctx, "WarningContainer", th.OnWarningContainer, th.WarningContainer, textMin, true)
						checkComposited(ctx, "Success", th.OnSuccess, th.Success, textMin, true)
						checkComposited(ctx, "SuccessContainer", th.OnSuccessContainer, th.SuccessContainer, textMin, true)

						// Button variants (Filled, Tonal, Elevated, Outlined/Text on SurfaceContainerLow).
						for _, bv := range []ButtonVariant{ButtonFilled, ButtonTonal, ButtonElevated, ButtonOutlined, ButtonText} {
							pal := buttonPaletteOf(th, bv, false)
							bg := pal.container
							if bg.A == 0 {
								bg = cs.SurfaceContainerLow
							}
							checkComposited(ctx, fmt.Sprintf("Button(%d)", bv), pal.content, bg, textMin, true)
						}

						// MetricCard / ambient radial glow worst-case composite (peak alpha 0.095 dark / 0.065 light).
						glowAlpha := float32(0.065)
						if dark {
							glowAlpha = 0.095
						}
						glowBg := cs.Primary.Alpha(glowAlpha).Over(cs.SurfaceContainerLow)
						checkComposited(ctx, "MetricCardGlow/OnSurface", cs.OnSurface, glowBg, textMin, true)
						checkComposited(ctx, "MetricCardGlow/OnSurfaceVariant", cs.OnSurfaceVariant, glowBg, textMin, true)

						// DataTable cells & selection contrast.
						cells := dataTableDefaultCellsStyle(th)
						checkComposited(ctx, "DataTable/CellSelect", cells.CellText, cells.Cell, textMin, false)
						stripeBg := cells.Stripe.Over(cs.SurfaceContainerLow)
						checkComposited(ctx, "DataTable/Stripe", cs.OnSurface, stripeBg, textMin, false)
					}
				}
			}
		}
	}

	// 2. LerpTheme interpolation matrix at t = 0, 0.25, 0.5, 0.75, 1 across Light<->Dark,
	//    Seed<->Seed, Tonal<->Neutral, and HighContrast<->HighContrast.
	lerpCases := []struct {
		name string
		a, b *Theme
	}{
		{
			"AmberDarkNeutral->NebulaLightTonal",
			ThemeFromSeed(ui.Hex("#E06D1B"), ThemeOptions{Dark: true, Variant: tokens.Expressive, SurfaceStyle: tokens.SurfaceNeutral}),
			ThemeFromSeed(ui.Hex("#6750A4"), ThemeOptions{Dark: false, Variant: tokens.TonalSpot, SurfaceStyle: tokens.SurfaceTonal}),
		},
		{
			"HarborLightHC->EmeraldDarkHC",
			ThemeFromSeed(ui.Hex("#0061A4"), ThemeOptions{Dark: false, Contrast: tokens.ContrastHigh, SurfaceStyle: tokens.SurfaceNeutral}),
			ThemeFromSeed(ui.Hex("#2E6B3E"), ThemeOptions{Dark: true, Contrast: tokens.ContrastHigh, SurfaceStyle: tokens.SurfaceTonal}),
		},
		{
			"MonoDarkHC->WhiteLightHC",
			ThemeFromSeed(ui.Hex("#808080"), ThemeOptions{Dark: true, Variant: tokens.Monochrome, Contrast: tokens.ContrastHigh}),
			ThemeFromSeed(ui.Hex("#FFFFFF"), ThemeOptions{Dark: false, Variant: tokens.Fidelity, Contrast: tokens.ContrastHigh}),
		},
		{
			"BlackDarkTonal->AmberDarkNeutral",
			ThemeFromSeed(ui.Hex("#000000"), ThemeOptions{Dark: true, Variant: tokens.Content, SurfaceStyle: tokens.SurfaceTonal}),
			ThemeFromSeed(ui.Hex("#E06D1B"), ThemeOptions{Dark: true, Variant: tokens.Vibrant, SurfaceStyle: tokens.SurfaceNeutral}),
		},
	}

	for _, lc := range lerpCases {
		minRatio := 4.5
		if lc.a.Contrast >= tokens.ContrastHigh && lc.b.Contrast >= tokens.ContrastHigh {
			minRatio = 7.0
		}
		for _, frac := range []float32{0, 0.25, 0.5, 0.75, 1} {
			mid := LerpTheme(lc.a, lc.b, frac)
			cs := mid.Colors
			ctx := fmt.Sprintf("lerp=%s t=%.2f", lc.name, frac)
			for _, surf := range []struct {
				name string
				bg   ui.Color
			}{
				{"Surface", cs.Surface},
				{"Lowest", cs.SurfaceContainerLowest},
				{"Low", cs.SurfaceContainerLow},
				{"Container", cs.SurfaceContainer},
				{"High", cs.SurfaceContainerHigh},
				{"Highest", cs.SurfaceContainerHighest},
			} {
				checkComposited(ctx, "OnSurface/"+surf.name, cs.OnSurface, surf.bg, minRatio, false)
				checkComposited(ctx, "OnSurfaceVariant/"+surf.name, cs.OnSurfaceVariant, surf.bg, minRatio, false)
			}
			checkComposited(ctx, "OnPrimary/Primary", cs.OnPrimary, cs.Primary, minRatio, false)
			checkComposited(ctx, "OnPrimaryContainer/PrimaryContainer", cs.OnPrimaryContainer, cs.PrimaryContainer, minRatio, false)
			checkComposited(ctx, "OnSecondary/Secondary", cs.OnSecondary, cs.Secondary, minRatio, false)
			checkComposited(ctx, "OnTertiary/Tertiary", cs.OnTertiary, cs.Tertiary, minRatio, false)
			checkComposited(ctx, "OnError/Error", cs.OnError, cs.Error, minRatio, false)
			checkComposited(ctx, "InverseOnSurface/InverseSurface", cs.InverseOnSurface, cs.InverseSurface, minRatio, false)
			checkComposited(ctx, "OnWarning/Warning", mid.OnWarning, mid.Warning, minRatio, false)
			checkComposited(ctx, "OnSuccess/Success", mid.OnSuccess, mid.Success, minRatio, false)
		}
	}
}
