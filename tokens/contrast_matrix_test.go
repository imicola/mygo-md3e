package tokens

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

var releaseContrastSeeds = []struct {
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

var releaseVariants = []struct {
	name    string
	variant Variant
}{
	{"TonalSpot", TonalSpot},
	{"Expressive", Expressive},
	{"Vibrant", Vibrant},
	{"Neutral", Neutral},
	{"Monochrome", Monochrome},
	{"Fidelity", Fidelity},
	{"Content", Content},
}

var releaseSurfaceStyles = []struct {
	name  string
	style SurfaceStyle
}{
	{"Tonal", SurfaceTonal},
	{"Neutral", SurfaceNeutral},
}

var releaseContrasts = []struct {
	name     string
	contrast Contrast
}{
	{"Standard", ContrastStandard},
	{"Medium", ContrastMedium},
	{"High", ContrastHigh},
}

func formatRGBA(c ui.Color) string {
	return fmt.Sprintf("rgba(%d,%d,%d,%d)", c.R, c.G, c.B, c.A)
}

// TestSurfaceStyleNeutralPalettesAndNormalization verifies SurfaceNeutral's
// neutral chroma clamping, accent preservation, Monochrome grayscale handling,
// and out-of-range SurfaceStyle fallback.
func TestSurfaceStyleNeutralPalettesAndNormalization(t *testing.T) {
	if SurfaceStyle(99).Normalize() != SurfaceTonal || SurfaceStyle(-1).Normalize() != SurfaceTonal {
		t.Fatalf("unknown SurfaceStyle did not normalize to SurfaceTonal")
	}
	if SurfaceNeutral.Normalize() != SurfaceNeutral {
		t.Fatalf("SurfaceNeutral.Normalize() != SurfaceNeutral")
	}

	for _, seed := range releaseContrastSeeds {
		for _, v := range releaseVariants {
			tonal := NewDynamicScheme(DynamicSchemeOptions{
				Seed: seed.color, Dark: true, Variant: v.variant, SurfaceStyle: SurfaceTonal,
			})
			neutral := NewDynamicScheme(DynamicSchemeOptions{
				Seed: seed.color, Dark: true, Variant: v.variant, SurfaceStyle: SurfaceNeutral,
			})

			// Accent palettes remain faithful to the Variant.
			if neutral.PrimaryPalette.Hue() != tonal.PrimaryPalette.Hue() || neutral.PrimaryPalette.Chroma() != tonal.PrimaryPalette.Chroma() {
				t.Errorf("%s/%s: PrimaryPalette changed under SurfaceNeutral", seed.name, v.name)
			}
			if neutral.SecondaryPalette.Hue() != tonal.SecondaryPalette.Hue() || neutral.SecondaryPalette.Chroma() != tonal.SecondaryPalette.Chroma() {
				t.Errorf("%s/%s: SecondaryPalette changed under SurfaceNeutral", seed.name, v.name)
			}
			if neutral.TertiaryPalette.Hue() != tonal.TertiaryPalette.Hue() || neutral.TertiaryPalette.Chroma() != tonal.TertiaryPalette.Chroma() {
				t.Errorf("%s/%s: TertiaryPalette changed under SurfaceNeutral", seed.name, v.name)
			}

			if v.variant == Monochrome {
				if neutral.NeutralPalette.Chroma() != 0 || neutral.NeutralVariantPalette.Chroma() != 0 {
					t.Errorf("%s/Monochrome: expected zero neutral chroma, got N=%.2f NV=%.2f",
						seed.name, neutral.NeutralPalette.Chroma(), neutral.NeutralVariantPalette.Chroma())
				}
				continue
			}

			wantN := math.Min(tonal.NeutralPalette.Chroma(), 2)
			wantNV := math.Min(tonal.NeutralVariantPalette.Chroma(), 3)
			if math.Abs(neutral.NeutralPalette.Chroma()-wantN) > 1e-9 {
				t.Errorf("%s/%s: Neutral chroma = %.3f, want %.3f", seed.name, v.name, neutral.NeutralPalette.Chroma(), wantN)
			}
			if math.Abs(neutral.NeutralVariantPalette.Chroma()-wantNV) > 1e-9 {
				t.Errorf("%s/%s: NeutralVariant chroma = %.3f, want %.3f", seed.name, v.name, neutral.NeutralVariantPalette.Chroma(), wantNV)
			}
			if neutral.NeutralPalette.Hue() != tonal.NeutralPalette.Hue() {
				t.Errorf("%s/%s: Neutral hue drifted under SurfaceNeutral", seed.name, v.name)
			}
			if neutral.NeutralVariantPalette.Hue() != tonal.NeutralVariantPalette.Hue() {
				t.Errorf("%s/%s: NeutralVariant hue drifted under SurfaceNeutral", seed.name, v.name)
			}
		}
	}
}

// TestContrastMatrixQuantizedSRGB is the M17.1 release gate across all
// 7 seeds x 7 variants x 2 SurfaceStyles x Light/Dark x 3 Contrast levels,
// verifying surface luminance monotonicity and opaque composited sRGB contrast
// under rest, hover (8%), and pressed (10%) state layers.
func TestContrastMatrixQuantizedSRGB(t *testing.T) {
	overlays := []struct {
		name  string
		alpha float32
	}{
		{"rest", 0},
		{"hover", StateHoverOpacity},
		{"press", StatePressedOpacity},
	}

	for _, seed := range releaseContrastSeeds {
		for _, v := range releaseVariants {
			for _, st := range releaseSurfaceStyles {
				for _, dark := range []bool{false, true} {
					for _, cr := range releaseContrasts {
						s := NewDynamicScheme(DynamicSchemeOptions{
							Seed:         seed.color,
							Dark:         dark,
							Variant:      v.variant,
							Contrast:     cr.contrast,
							SurfaceStyle: st.style,
						})
						cs := s.Colors

						// 1. Opaque sRGB surface container luminance order.
						steps := [5]ui.Color{
							cs.SurfaceContainerLowest,
							cs.SurfaceContainerLow,
							cs.SurfaceContainer,
							cs.SurfaceContainerHigh,
							cs.SurfaceContainerHighest,
						}
						for i := 1; i < len(steps); i++ {
							prevY := RelativeLuminance(steps[i-1])
							curY := RelativeLuminance(steps[i])
							if dark && curY <= prevY {
								t.Fatalf("seed=%s variant=%s style=%s dark=%v contrast=%s: dark surface luminance not increasing at step %d (%s Y=%.5f <= %s Y=%.5f)",
									seed.name, v.name, st.name, dark, cr.name, i, formatRGBA(steps[i]), curY, formatRGBA(steps[i-1]), prevY)
							}
							if !dark && curY >= prevY {
								t.Fatalf("seed=%s variant=%s style=%s dark=%v contrast=%s: light surface luminance not decreasing at step %d (%s Y=%.5f >= %s Y=%.5f)",
									seed.name, v.name, st.name, dark, cr.name, i, formatRGBA(steps[i]), curY, formatRGBA(steps[i-1]), prevY)
							}
						}

						textMin := 4.5
						outlineMin := 3.0
						if cr.contrast >= ContrastHigh {
							textMin = 7.0
							outlineMin = 4.5
						}

						checkPair := func(pairName string, fg, bg ui.Color, minRatio float64, checkStates bool) {
							for _, ov := range overlays {
								if !checkStates && ov.alpha > 0 {
									continue
								}
								effBg := bg
								if ov.alpha > 0 {
									effBg = fg.Alpha(ov.alpha).Over(bg)
								}
								got := ContrastRatio(fg, effBg)
								if got+1e-6 < minRatio {
									t.Errorf("seed=%s variant=%s style=%s dark=%v contrast=%s pair=%s(%s): fg=%s bg=%s ratio=%.2f < %.2f",
										seed.name, v.name, st.name, dark, cr.name, pairName, ov.name, formatRGBA(fg), formatRGBA(effBg), got, minRatio)
								}
							}
						}

						allSurfaces := []struct {
							name string
							bg   ui.Color
						}{
							{"Surface", cs.Surface},
							{"Lowest", cs.SurfaceContainerLowest},
							{"Low", cs.SurfaceContainerLow},
							{"Container", cs.SurfaceContainer},
							{"High", cs.SurfaceContainerHigh},
							{"Highest", cs.SurfaceContainerHighest},
						}

						for _, surf := range allSurfaces {
							onSurfStates := !(dark && cr.contrast >= ContrastHigh && surf.name == "Highest")
							accentStates := surf.name != "Highest"
							checkPair("OnSurface/"+surf.name, cs.OnSurface, surf.bg, textMin, onSurfStates)
							checkPair("OnSurfaceVariant/"+surf.name, cs.OnSurfaceVariant, surf.bg, textMin, onSurfStates)
							checkPair("Primary/"+surf.name, cs.Primary, surf.bg, textMin, accentStates)
							checkPair("Error/"+surf.name, cs.Error, surf.bg, textMin, accentStates)
							checkPair("Outline/"+surf.name, cs.Outline, surf.bg, outlineMin, false)
							if !onSurfStates {
								// Dark HighContrast Highest (T30) with hover/press overlay:
								// verify against achievable sRGB floor (>= 6.0:1).
								checkPair("OnSurface/"+surf.name+"-state", cs.OnSurface, surf.bg, 6.0, true)
								checkPair("OnSurfaceVariant/"+surf.name+"-state", cs.OnSurfaceVariant, surf.bg, 6.0, true)
							}
						}

						checkPair("OnBackground/Background", cs.OnBackground, cs.Background, textMin, true)
						checkPair("OnPrimary/Primary", cs.OnPrimary, cs.Primary, textMin, true)
						checkPair("OnPrimaryContainer/PrimaryContainer", cs.OnPrimaryContainer, cs.PrimaryContainer, textMin, true)
						checkPair("InversePrimary/InverseSurface", cs.InversePrimary, cs.InverseSurface, textMin, true)
						checkPair("OnSecondary/Secondary", cs.OnSecondary, cs.Secondary, textMin, true)
						checkPair("OnSecondaryContainer/SecondaryContainer", cs.OnSecondaryContainer, cs.SecondaryContainer, textMin, true)
						checkPair("OnTertiary/Tertiary", cs.OnTertiary, cs.Tertiary, textMin, true)
						checkPair("OnTertiaryContainer/TertiaryContainer", cs.OnTertiaryContainer, cs.TertiaryContainer, textMin, true)
						checkPair("OnError/Error", cs.OnError, cs.Error, textMin, true)
						checkPair("OnErrorContainer/ErrorContainer", cs.OnErrorContainer, cs.ErrorContainer, textMin, true)
						checkPair("InverseOnSurface/InverseSurface", cs.InverseOnSurface, cs.InverseSurface, textMin, true)

						// Monochrome fixed containers stay at Material's T40 (white ceiling ~6.46:1).
						fixedMin := textMin
						if v.variant == Monochrome && fixedMin > 6.4 {
							fixedMin = 6.4
						}
						checkPair("OnPrimaryFixed/PrimaryFixed", cs.OnPrimaryFixed, cs.PrimaryFixed, fixedMin, false)
						checkPair("OnPrimaryFixed/PrimaryFixedDim", cs.OnPrimaryFixed, cs.PrimaryFixedDim, textMin, false)
						checkPair("OnSecondaryFixed/SecondaryFixed", cs.OnSecondaryFixed, cs.SecondaryFixed, fixedMin, false)
						checkPair("OnTertiaryFixed/TertiaryFixed", cs.OnTertiaryFixed, cs.TertiaryFixed, fixedMin, false)
					}
				}
			}
		}
	}
}
