package charts

import (
	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

// Perceptually separated hue offsets (in degrees relative to theme Primary)
// chosen to maximize distinctiveness for slots 3..10 before falling back to
// golden-angle progression.
var seriesHueOffsets = [...]float64{
	145, // emerald/teal complement
	215, // azure/cyan split
	45,  // warm amber/gold
	295, // vivid violet/magenta
	95,  // lime/olive
	180, // direct complement
	255, // indigo
	330, // coral/rose
}

// SeriesColor returns an HCT-harmonized color for series index under th.
//
// Slots 0..2 anchor directly to Primary, Tertiary, and Secondary so single-
// and dual-series charts blend naturally with the active MD3E theme. Higher
// slots rotate cleanly around the HCT hue wheel while enforcing WCAG 2.1
// graphical contrast (>= 3.0:1 in standard/medium contrast, >= 4.5:1 in high
// contrast) against the card surface. Monochrome themes preserve neutral tonal
// separation instead of injecting synthetic chroma.
//
// SeriesColor performs no heap allocations.
func SeriesColor(th *md3e.Theme, index int) ui.Color {
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	if index < 0 {
		index = 0
	}

	cs := th.Colors
	bg := md3e.SurfaceColorForElevation(th, tokens.Level1)
	if bg.A == 0 {
		bg = cs.Surface
	}
	if bg.A == 0 {
		if th.Dark {
			bg = ui.RGB(0x14, 0x12, 0x18)
		} else {
			bg = ui.RGB(0xF7, 0xF2, 0xFA)
		}
	}

	minRatio := 3.0
	if th.Contrast >= tokens.ContrastHigh {
		minRatio = 4.5
	} else if th.Contrast >= tokens.ContrastMedium {
		minRatio = 3.5
	}

	primary := cs.Primary
	if primary.A == 0 {
		primary = md3e.DefaultSeedColor
	}
	primaryHCT := tokens.HCTFromColor(primary)
	monochrome := primaryHCT.Chroma < 4.0

	if monochrome {
		return monochromeSeriesColor(th.Dark, bg, minRatio, index)
	}

	switch index {
	case 0:
		return ensureSeriesContrast(primary, bg, th.Dark, minRatio)
	case 1:
		if cs.Tertiary.A > 0 {
			thct := tokens.HCTFromColor(cs.Tertiary)
			if hueDelta(thct.Hue, primaryHCT.Hue) >= 24 {
				return ensureSeriesContrast(cs.Tertiary, bg, th.Dark, minRatio)
			}
		}
	case 2:
		if cs.Secondary.A > 0 {
			shct := tokens.HCTFromColor(cs.Secondary)
			thHue := primaryHCT.Hue + 60
			if cs.Tertiary.A > 0 {
				thHue = tokens.HCTFromColor(cs.Tertiary).Hue
			}
			if hueDelta(shct.Hue, primaryHCT.Hue) >= 28 && hueDelta(shct.Hue, thHue) >= 24 {
				c := cs.Secondary
				if shct.Chroma < 28 {
					shct.Chroma = 38
					c = shct.ToColor()
				}
				return ensureSeriesContrast(c, bg, th.Dark, minRatio)
			}
		}
	}

	slot := index - 2
	if slot < 0 {
		slot = index
	}
	var offset float64
	if slot < len(seriesHueOffsets) {
		offset = seriesHueOffsets[slot]
	} else {
		offset = float64(slot+1) * 137.50776405003785
	}
	hue := tokens.SanitizeDegrees(primaryHCT.Hue + offset)
	// Avoid landing too close to Tertiary's hue on early slots so adjacent
	// series stay unmistakably distinct.
	if cs.Tertiary.A > 0 && index < 6 {
		thHue := tokens.HCTFromColor(cs.Tertiary).Hue
		if hueDelta(hue, thHue) < 22 {
			hue = tokens.SanitizeDegrees(hue + 28)
		}
	}

	chroma := 52.0
	tone := 44.0
	if th.Dark {
		chroma = 48.0
		tone = 78.0
		if th.Contrast >= tokens.ContrastHigh {
			chroma = 56.0
			tone = 86.0
		}
	} else if th.Contrast >= tokens.ContrastHigh {
		chroma = 60.0
		tone = 30.0
	}
	// Gently stagger tone on wrap-around slots (> 10 series) so even huge
	// charts stay separable.
	if cycle := slot / len(seriesHueOffsets); cycle > 0 {
		delta := float64((cycle%3)-1) * 8.0
		if th.Dark {
			tone = clampF64(tone+delta, 66, 92)
		} else {
			tone = clampF64(tone+delta, 24, 52)
		}
	}

	col := tokens.HCT{Hue: hue, Chroma: chroma, Tone: tone}.ToColor()
	return ensureSeriesContrast(col, bg, th.Dark, minRatio)
}

// SeriesPalette returns n HCT-harmonized series colors derived from th.
func SeriesPalette(th *md3e.Theme, n int) []ui.Color {
	if n <= 0 {
		return nil
	}
	out := make([]ui.Color, n)
	for i := range n {
		out[i] = SeriesColor(th, i)
	}
	return out
}

func monochromeSeriesColor(dark bool, bg ui.Color, minRatio float64, index int) ui.Color {
	var tones [6]float64
	if dark {
		tones = [6]float64{92, 78, 66, 85, 72, 98}
	} else {
		tones = [6]float64{18, 34, 46, 26, 40, 10}
	}
	t := tones[index%len(tones)]
	col := tokens.HCT{Hue: 0, Chroma: 0, Tone: t}.ToColor()
	return ensureSeriesContrast(col, bg, dark, minRatio)
}

func hueDelta(a, b float64) float64 {
	d := tokens.SanitizeDegrees(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// ensureSeriesContrast nudges col's HCT Tone away from bg until its WCAG
// contrast ratio meets minRatio.
func ensureSeriesContrast(col, bg ui.Color, dark bool, minRatio float64) ui.Color {
	if tokens.ContrastRatio(col, bg) >= minRatio {
		return col
	}
	h := tokens.HCTFromColor(col)
	step := -4.0
	if dark {
		step = 4.0
	}
	for range 20 {
		h.Tone = clampF64(h.Tone+step, 2, 98)
		candidate := h.ToColor()
		if tokens.ContrastRatio(candidate, bg) >= minRatio {
			return candidate
		}
		if (dark && h.Tone >= 98) || (!dark && h.Tone <= 2) {
			return candidate
		}
	}
	return h.ToColor()
}
