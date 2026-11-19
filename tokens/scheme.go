package tokens

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Variant is the style of a dynamic color scheme.
type Variant int

const (
	// TonalSpot is the default Material You scheme: low to medium
	// colorfulness with a related tertiary hue.
	TonalSpot Variant = iota
	// Expressive rotates hues far from the source for a bold scheme.
	Expressive
	// Vibrant maxes out the primary palette's chroma.
	Vibrant
	// Neutral is near grayscale.
	Neutral
	// Monochrome is grayscale.
	Monochrome
	// Fidelity keeps the source color in the primary container.
	Fidelity
	// Content keeps the source color, with an analogous tertiary.
	Content
)

// Contrast is the contrast level of a dynamic color scheme.
type Contrast int

const (
	ContrastStandard Contrast = iota // 0
	ContrastMedium                   // 0.5
	ContrastHigh                     // 1
)

// level returns the contrast level as Material's -1..1 scale.
func (c Contrast) level() float64 {
	switch c {
	case ContrastMedium:
		return 0.5
	case ContrastHigh:
		return 1
	default:
		return 0
	}
}

// State layer opacities, composited over a container in the color of its
// content.
const (
	StateHoverOpacity   = 0.08
	StateFocusOpacity   = 0.10
	StatePressedOpacity = 0.10
	StateDraggedOpacity = 0.16
)

// Disabled opacities: the container is filled with OnSurface at
// DisabledContainerOpacity, and its content is drawn at DisabledContentOpacity.
const (
	DisabledContainerOpacity = 0.12
	DisabledContentOpacity   = 0.38
)

// DynamicSchemeOptions configures a dynamic color scheme.
type DynamicSchemeOptions struct {
	// Seed is the source color the scheme derives from.
	Seed ui.Color
	// Dark selects the dark scheme instead of the light one.
	Dark bool
	// Variant is the style of the scheme; the zero value is TonalSpot.
	Variant Variant
	// Contrast is the contrast level; the zero value is ContrastStandard.
	Contrast Contrast
}

// ColorScheme is the set of MD3 color roles a dynamic scheme resolves to.
type ColorScheme struct {
	// Primary.
	Primary, OnPrimary                    ui.Color
	PrimaryContainer, OnPrimaryContainer  ui.Color
	InversePrimary                        ui.Color
	PrimaryFixed, PrimaryFixedDim         ui.Color
	OnPrimaryFixed, OnPrimaryFixedVariant ui.Color

	// Secondary.
	Secondary, OnSecondary                    ui.Color
	SecondaryContainer, OnSecondaryContainer  ui.Color
	SecondaryFixed, SecondaryFixedDim         ui.Color
	OnSecondaryFixed, OnSecondaryFixedVariant ui.Color

	// Tertiary.
	Tertiary, OnTertiary                    ui.Color
	TertiaryContainer, OnTertiaryContainer  ui.Color
	TertiaryFixed, TertiaryFixedDim         ui.Color
	OnTertiaryFixed, OnTertiaryFixedVariant ui.Color

	// Error.
	Error, OnError                   ui.Color
	ErrorContainer, OnErrorContainer ui.Color

	// Surfaces and neutrals.
	Background, OnBackground                    ui.Color
	Surface, SurfaceDim, SurfaceBright          ui.Color
	OnSurface, OnSurfaceVariant                 ui.Color
	SurfaceVariant                              ui.Color
	SurfaceContainerLowest, SurfaceContainerLow ui.Color
	SurfaceContainer, SurfaceContainerHigh      ui.Color
	SurfaceContainerHighest                     ui.Color
	SurfaceTint                                 ui.Color

	// Outline and utility.
	Outline, OutlineVariant          ui.Color
	InverseSurface, InverseOnSurface ui.Color
	Scrim, Shadow                    ui.Color
}

// DynamicScheme is a source color, a variant and a contrast level resolved
// into MD3 color roles, with the six tonal palettes it derives them from.
type DynamicScheme struct {
	Options   DynamicSchemeOptions
	SourceHCT HCT

	PrimaryPalette, SecondaryPalette    *TonalPalette
	TertiaryPalette, NeutralPalette     *TonalPalette
	NeutralVariantPalette, ErrorPalette *TonalPalette

	Colors ColorScheme
}

// NewDynamicScheme resolves opts into a scheme.
func NewDynamicScheme(opts DynamicSchemeOptions) *DynamicScheme {
	source := HCTFromColor(opts.Seed)
	s := &DynamicScheme{Options: opts, SourceHCT: source}
	s.PrimaryPalette, s.SecondaryPalette, s.TertiaryPalette, s.NeutralPalette, s.NeutralVariantPalette =
		variantPalettes(opts.Variant, source)
	s.ErrorPalette = NewTonalPalette(25, 84)
	s.Colors = s.resolve()
	return s
}

// Hues at which the Expressive variant's hue rotations change, and the
// rotations of its secondary and tertiary palettes at those hues.
var (
	expressiveHues               = []float64{0, 21, 51, 121, 151, 191, 271, 321, 360}
	expressiveSecondaryRotations = []float64{45, 95, 45, 20, 45, 90, 45, 45, 45}
	expressiveTertiaryRotations  = []float64{120, 120, 20, 45, 20, 15, 20, 120, 120}

	vibrantHues               = []float64{0, 41, 61, 101, 131, 181, 251, 301, 360}
	vibrantSecondaryRotations = []float64{18, 15, 10, 12, 15, 18, 15, 12, 12}
	vibrantTertiaryRotations  = []float64{35, 30, 20, 25, 30, 35, 30, 25, 25}
)

// variantPalettes returns the five palettes of a variant, as Material's
// scheme classes define them.
func variantPalettes(v Variant, source HCT) (primary, secondary, tertiary, neutral, neutralVariant *TonalPalette) {
	hue := source.Hue
	chroma := source.Chroma
	switch v {
	case Expressive:
		return NewTonalPalette(hue+240, 40),
			NewTonalPalette(rotatedHue(hue, expressiveHues, expressiveSecondaryRotations), 24),
			NewTonalPalette(rotatedHue(hue, expressiveHues, expressiveTertiaryRotations), 32),
			NewTonalPalette(hue+15, 8),
			NewTonalPalette(hue+15, 12)
	case Vibrant:
		return NewTonalPalette(hue, 200),
			NewTonalPalette(rotatedHue(hue, vibrantHues, vibrantSecondaryRotations), 24),
			NewTonalPalette(rotatedHue(hue, vibrantHues, vibrantTertiaryRotations), 32),
			NewTonalPalette(hue, 10),
			NewTonalPalette(hue, 12)
	case Neutral:
		return NewTonalPalette(hue, 12),
			NewTonalPalette(hue, 8),
			NewTonalPalette(hue, 16),
			NewTonalPalette(hue, 2),
			NewTonalPalette(hue, 2)
	case Monochrome:
		return NewTonalPalette(hue, 0),
			NewTonalPalette(hue, 0),
			NewTonalPalette(hue, 0),
			NewTonalPalette(hue, 0),
			NewTonalPalette(hue, 0)
	case Fidelity, Content:
		secondaryChroma := math.Max(chroma-32, chroma*0.5)
		tertiaryHue := hue + 180 // Fidelity complements the source hue.
		if v == Content {
			tertiaryHue = hue + 60 // Content stays analogous to it.
		}
		return NewTonalPalette(hue, chroma),
			NewTonalPalette(hue, secondaryChroma),
			NewTonalPalette(tertiaryHue, chroma),
			NewTonalPalette(hue, chroma/8),
			NewTonalPalette(hue, chroma/8+4)
	default: // TonalSpot
		return NewTonalPalette(hue, 36),
			NewTonalPalette(hue, 16),
			NewTonalPalette(hue+60, 24),
			NewTonalPalette(hue, 6),
			NewTonalPalette(hue, 8)
	}
}

// rotatedHue returns the source hue rotated by the rotation of the hue band it
// falls in, as Material's Expressive and Vibrant schemes do. Bands are half
// open, [hue, next), so every hue belongs to exactly one band and a hue that
// lands on a breakpoint (0, 21, 51, ...) takes the rotation of the band it
// starts rather than falling through unrotated.
func rotatedHue(hue float64, hues, rotations []float64) float64 {
	if len(rotations) == 1 {
		return SanitizeDegrees(hue + rotations[0])
	}
	for i := 0; i+1 < len(hues); i++ {
		if hues[i] <= hue && hue < hues[i+1] {
			return SanitizeDegrees(hue + rotations[i])
		}
	}
	return hue
}

// curve returns the value of a contrast curve at the scheme's contrast level:
// normal at 0, medium at 0.5 and high at 1.
func curve(normal, medium, high, level float64) float64 {
	switch {
	case level <= 0:
		return normal
	case level < 0.5:
		return lerp(normal, medium, level/0.5)
	case level < 1:
		return lerp(medium, high, (level-0.5)/0.5)
	default:
		return high
	}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// resolve computes every role's tone and color.
func (s *DynamicScheme) resolve() ColorScheme {
	dark := s.Options.Dark
	level := s.Options.Contrast.level()
	mono := s.Options.Variant == Monochrome
	fidelity := s.Options.Variant == Fidelity || s.Options.Variant == Content

	// The surfaces, whose tones the other roles are measured against.
	surfaceDimTone := 6.0
	if !dark {
		surfaceDimTone = curve(87, 80, 75, level)
	}
	surfaceBrightTone := 98.0
	if dark {
		surfaceBrightTone = curve(24, 29, 34, level)
	}
	highestSurface := surfaceDimTone
	if dark {
		highestSurface = surfaceBrightTone
	}

	lowestTone, lowTone := 100.0, curve(96, 96, 95, level)
	containerTone, highTone := curve(94, 92, 90, level), curve(92, 88, 85, level)
	highestContainerTone := curve(90, 84, 80, level)
	if dark {
		lowestTone = curve(4, 2, 0, level)
		lowTone = curve(10, 11, 12, level)
		containerTone = curve(12, 16, 20, level)
		highTone = curve(17, 21, 25, level)
		highestContainerTone = curve(22, 26, 30, level)
	}

	surfaceTone := 98.0
	if dark {
		surfaceTone = 6
	}

	var c ColorScheme

	// Primary.
	primaryInitial := 40.0
	switch {
	case mono && dark:
		primaryInitial = 100
	case mono:
		primaryInitial = 0
	case dark:
		primaryInitial = 80
	}
	primaryTone := s.tone(primaryInitial, highestSurface, 4.5, 7, 7, true)

	primaryContainerInitial := 90.0
	switch {
	case fidelity:
		primaryContainerInitial = s.SourceHCT.Tone
	case mono && dark:
		primaryContainerInitial = 85
	case mono:
		primaryContainerInitial = 25
	case dark:
		primaryContainerInitial = 30
	}
	primaryContainerTone := s.tone(primaryContainerInitial, highestSurface, 1, 3, 4.5, true)

	onPrimaryInitial := 100.0
	switch {
	case mono && dark:
		onPrimaryInitial = 10
	case mono:
		onPrimaryInitial = 90
	case dark:
		onPrimaryInitial = 20
	}

	onPrimaryContainerInitial := 30.0
	switch {
	case fidelity:
		onPrimaryContainerInitial = foregroundTone(primaryContainerTone, 4.5)
	case mono && dark:
		onPrimaryContainerInitial = 0
	case mono:
		onPrimaryContainerInitial = 100
	case dark:
		onPrimaryContainerInitial = 90
	}

	c.Primary = s.PrimaryPalette.Tone(primaryTone)
	c.OnPrimary = s.PrimaryPalette.Tone(s.tone(onPrimaryInitial, primaryTone, 7, 11, 21, false))
	c.PrimaryContainer = s.PrimaryPalette.Tone(primaryContainerTone)
	c.OnPrimaryContainer = s.PrimaryPalette.Tone(s.tone(onPrimaryContainerInitial, primaryContainerTone, 4.5, 7, 11, false))
	c.InversePrimary = s.PrimaryPalette.Tone(s.tone(lerp(80, 40, boolFloat(dark)), surfaceTone, 4.5, 7, 7, false))
	c.PrimaryFixed = s.PrimaryPalette.Tone(90)
	c.PrimaryFixedDim = s.PrimaryPalette.Tone(80)
	c.OnPrimaryFixed = s.PrimaryPalette.Tone(10)
	c.OnPrimaryFixedVariant = s.PrimaryPalette.Tone(30)

	// Secondary.
	secondaryInitial := 40.0
	if dark {
		secondaryInitial = 80
	}
	secondaryTone := s.tone(secondaryInitial, highestSurface, 4.5, 7, 7, true)

	secondaryContainerInitial := 90.0
	if dark {
		secondaryContainerInitial = 30
	} else if mono {
		secondaryContainerInitial = 85
	}
	if fidelity {
		secondaryContainerInitial = findDesiredChromaByTone(s.SecondaryPalette, secondaryContainerInitial, !dark)
	}
	secondaryContainerTone := s.tone(secondaryContainerInitial, highestSurface, 1, 3, 4.5, true)

	onSecondaryInitial := 100.0
	if mono && dark {
		onSecondaryInitial = 10
	} else if dark {
		onSecondaryInitial = 20
	}

	onSecondaryContainerInitial := 30.0
	if mono && dark {
		onSecondaryContainerInitial = 90
	} else if mono {
		onSecondaryContainerInitial = 10
	} else if fidelity {
		onSecondaryContainerInitial = foregroundTone(secondaryContainerTone, 4.5)
	} else if dark {
		onSecondaryContainerInitial = 90
	}

	c.Secondary = s.SecondaryPalette.Tone(secondaryTone)
	c.OnSecondary = s.SecondaryPalette.Tone(s.tone(onSecondaryInitial, secondaryTone, 7, 11, 21, false))
	c.SecondaryContainer = s.SecondaryPalette.Tone(secondaryContainerTone)
	c.OnSecondaryContainer = s.SecondaryPalette.Tone(s.tone(onSecondaryContainerInitial, secondaryContainerTone, 4.5, 7, 11, false))
	c.SecondaryFixed = s.SecondaryPalette.Tone(90)
	c.SecondaryFixedDim = s.SecondaryPalette.Tone(80)
	c.OnSecondaryFixed = s.SecondaryPalette.Tone(10)
	c.OnSecondaryFixedVariant = s.SecondaryPalette.Tone(30)

	// Tertiary.
	tertiaryInitial := 40.0
	switch {
	case mono && dark:
		tertiaryInitial = 90
	case mono:
		tertiaryInitial = 25
	case dark:
		tertiaryInitial = 80
	}
	tertiaryTone := s.tone(tertiaryInitial, highestSurface, 4.5, 7, 7, true)

	tertiaryContainerInitial := 90.0
	switch {
	case fidelity:
		tertiaryContainerInitial = s.SourceHCT.Tone
	case mono && dark:
		tertiaryContainerInitial = 60
	case mono:
		tertiaryContainerInitial = 49
	case dark:
		tertiaryContainerInitial = 30
	}
	tertiaryContainerTone := s.tone(tertiaryContainerInitial, highestSurface, 1, 3, 4.5, true)

	onTertiaryInitial := 100.0
	switch {
	case mono && dark:
		onTertiaryInitial = 10
	case mono:
		onTertiaryInitial = 90
	case dark:
		onTertiaryInitial = 20
	}

	onTertiaryContainerInitial := 30.0
	switch {
	case fidelity:
		onTertiaryContainerInitial = foregroundTone(tertiaryContainerTone, 4.5)
	case mono && dark:
		onTertiaryContainerInitial = 0
	case mono:
		onTertiaryContainerInitial = 100
	case dark:
		onTertiaryContainerInitial = 90
	}

	c.Tertiary = s.TertiaryPalette.Tone(tertiaryTone)
	c.OnTertiary = s.TertiaryPalette.Tone(s.tone(onTertiaryInitial, tertiaryTone, 7, 11, 21, false))
	c.TertiaryContainer = s.TertiaryPalette.Tone(tertiaryContainerTone)
	c.OnTertiaryContainer = s.TertiaryPalette.Tone(s.tone(onTertiaryContainerInitial, tertiaryContainerTone, 4.5, 7, 11, false))
	c.TertiaryFixed = s.TertiaryPalette.Tone(90)
	c.TertiaryFixedDim = s.TertiaryPalette.Tone(80)
	c.OnTertiaryFixed = s.TertiaryPalette.Tone(10)
	c.OnTertiaryFixedVariant = s.TertiaryPalette.Tone(30)

	// Error.
	errorInitial := 40.0
	if dark {
		errorInitial = 80
	}
	errorTone := s.tone(errorInitial, highestSurface, 4.5, 7, 7, true)
	errorContainerTone := s.tone(lerp(90, 30, boolFloat(dark)), highestSurface, 1, 3, 4.5, true)
	onErrorInitial := 100.0
	if dark {
		onErrorInitial = 20
	}
	onErrorContainerInitial := 30.0
	if dark {
		onErrorContainerInitial = 90
	}

	c.Error = s.ErrorPalette.Tone(errorTone)
	c.OnError = s.ErrorPalette.Tone(s.tone(onErrorInitial, errorTone, 7, 11, 21, false))
	c.ErrorContainer = s.ErrorPalette.Tone(errorContainerTone)
	c.OnErrorContainer = s.ErrorPalette.Tone(s.tone(onErrorContainerInitial, errorContainerTone, 4.5, 7, 11, false))

	// Background and surfaces.
	c.Background = s.NeutralPalette.Tone(surfaceTone)
	c.Surface = s.NeutralPalette.Tone(surfaceTone)
	c.SurfaceDim = s.NeutralPalette.Tone(surfaceDimTone)
	c.SurfaceBright = s.NeutralPalette.Tone(surfaceBrightTone)
	c.SurfaceContainerLowest = s.NeutralPalette.Tone(lowestTone)
	c.SurfaceContainerLow = s.NeutralPalette.Tone(lowTone)
	c.SurfaceContainer = s.NeutralPalette.Tone(containerTone)
	c.SurfaceContainerHigh = s.NeutralPalette.Tone(highTone)
	c.SurfaceContainerHighest = s.NeutralPalette.Tone(highestContainerTone)
	c.SurfaceVariant = s.NeutralVariantPalette.Tone(lerp(90, 30, boolFloat(dark)))
	c.SurfaceTint = s.PrimaryPalette.Tone(lerp(40, 80, boolFloat(dark)))

	onBackgroundInitial := 10.0
	if dark {
		onBackgroundInitial = 90
	}
	c.OnBackground = s.NeutralPalette.Tone(s.tone(onBackgroundInitial, surfaceTone, 3, 4.5, 7, false))

	onSurfaceInitial := 10.0
	if dark {
		onSurfaceInitial = 90
	}
	c.OnSurface = s.NeutralPalette.Tone(s.tone(onSurfaceInitial, highestSurface, 7, 11, 21, false))

	onSurfaceVariantInitial := 30.0
	if dark {
		onSurfaceVariantInitial = 80
	}
	c.OnSurfaceVariant = s.NeutralVariantPalette.Tone(s.tone(onSurfaceVariantInitial, highestSurface, 4.5, 7, 11, false))

	// Outline and utility.
	outlineInitial := 50.0
	if dark {
		outlineInitial = 60
	}
	c.Outline = s.NeutralVariantPalette.Tone(s.tone(outlineInitial, highestSurface, 3, 4.5, 7, false))

	outlineVariantInitial := 80.0
	if dark {
		outlineVariantInitial = 30
	}
	c.OutlineVariant = s.NeutralVariantPalette.Tone(s.tone(outlineVariantInitial, highestSurface, 1, 3, 4.5, false))

	inverseSurfaceTone := lerp(20, 90, boolFloat(dark))
	c.InverseSurface = s.NeutralPalette.Tone(inverseSurfaceTone)
	c.InverseOnSurface = s.NeutralPalette.Tone(s.tone(lerp(95, 20, boolFloat(dark)), inverseSurfaceTone, 7, 11, 21, false))
	c.Scrim = s.NeutralPalette.Tone(0)
	c.Shadow = s.NeutralPalette.Tone(0)

	return c
}

// tone resolves a role's tone: the initial tone when it already reaches the
// role's contrast curve against bg, and otherwise the nearest tone that does.
// Background roles are kept out of the 50..60 awkward zone.
func (s *DynamicScheme) tone(initial, bg, normal, medium, high float64, isBackground bool) float64 {
	desired := curve(normal, medium, high, s.Options.Contrast.level())
	answer := initial
	if ContrastRatioOfTones(bg, answer) < desired {
		answer = foregroundTone(bg, desired)
	}
	if isBackground && 50 <= answer && answer < 60 {
		if ContrastRatioOfTones(49, bg) >= desired {
			answer = 49
		} else {
			answer = 60
		}
	}
	return answer
}

// foregroundTone returns a tone with about ratio contrast against bgTone,
// preferring a light foreground below tone 60 and a dark one above it, as
// Material's DynamicColor.foregroundTone does.
func foregroundTone(bgTone, ratio float64) float64 {
	lighter := lighterToneUnsafe(bgTone, ratio)
	darker := darkerToneUnsafe(bgTone, ratio)
	lighterRatio := ContrastRatioOfTones(lighter, bgTone)
	darkerRatio := ContrastRatioOfTones(darker, bgTone)
	if tonePrefersLightForeground(bgTone) {
		// When neither tone reaches the ratio, the closer one still wins,
		// unless they are all but equal.
		negligible := math.Abs(lighterRatio-darkerRatio) < 0.1 && lighterRatio < ratio && darkerRatio < ratio
		if lighterRatio >= ratio || lighterRatio >= darkerRatio || negligible {
			return lighter
		}
		return darker
	}
	if darkerRatio >= ratio || darkerRatio >= lighterRatio {
		return darker
	}
	return lighter
}

// tonePrefersLightForeground reports whether people prefer a light foreground
// on this tone: tones below 60.
func tonePrefersLightForeground(tone float64) bool {
	return math.Round(tone) < 60
}

// lighterToneUnsafe returns a tone of at least tone that reaches ratio against
// it, or 100 when none does.
func lighterToneUnsafe(tone, ratio float64) float64 {
	if t := lighterTone(tone, ratio); t >= 0 {
		return t
	}
	return 100
}

// darkerToneUnsafe returns a tone of at most tone that reaches ratio against
// it, or 0 when none does.
func darkerToneUnsafe(tone, ratio float64) float64 {
	if t := darkerTone(tone, ratio); t >= 0 {
		return t
	}
	return 0
}

// lighterTone returns a tone of at least tone that reaches ratio against it,
// or -1 when none does.
func lighterTone(tone, ratio float64) float64 {
	if tone < 0 || tone > 100 {
		return -1
	}
	darkY := YFromLstar(tone)
	lightY := ratio*(darkY+5) - 5
	real := ratioOfYs(lightY, darkY)
	if real < ratio && math.Abs(real-ratio) > 0.04 {
		return -1
	}
	// Darken slightly, so that gamut mapping keeps the ratio.
	v := LstarFromY(lightY) + 0.4
	if v < 0 || v > 100 {
		return -1
	}
	return v
}

// darkerTone returns a tone of at most tone that reaches ratio against it, or
// -1 when none does.
func darkerTone(tone, ratio float64) float64 {
	if tone < 0 || tone > 100 {
		return -1
	}
	lightY := YFromLstar(tone)
	darkY := (lightY+5)/ratio - 5
	real := ratioOfYs(lightY, darkY)
	if real < ratio && math.Abs(real-ratio) > 0.04 {
		return -1
	}
	// Darken slightly, so that gamut mapping keeps the ratio.
	v := LstarFromY(darkY) - 0.4
	if v < 0 || v > 100 {
		return -1
	}
	return v
}

// ratioOfYs returns the WCAG contrast ratio of two luminances in 0..100.
func ratioOfYs(y1, y2 float64) float64 {
	if y1 < y2 {
		y1, y2 = y2, y1
	}
	return (y1 + 5) / (y2 + 5)
}

// findDesiredChromaByTone walks from tone toward the tone where palette's
// chroma peaks, so that a fidelity scheme's container keeps its color.
func findDesiredChromaByTone(palette *TonalPalette, tone float64, byDecreasingTone bool) float64 {
	answer := tone
	closest := palette.HCT(tone)
	if closest.Chroma >= palette.Chroma() {
		return answer
	}
	step := 1.0
	if byDecreasingTone {
		step = -1
	}
	chromaPeak := closest.Chroma
	for range 100 {
		answer += step
		if answer < 0 || answer > 100 {
			break
		}
		candidate := palette.HCT(answer)
		if chromaPeak > candidate.Chroma {
			break
		}
		if math.Abs(candidate.Chroma-palette.Chroma()) < 0.4 {
			break
		}
		if math.Abs(candidate.Chroma-palette.Chroma()) < math.Abs(closest.Chroma-palette.Chroma()) {
			closest = candidate
		}
		chromaPeak = math.Max(chromaPeak, candidate.Chroma)
	}
	return answer
}

// boolFloat returns 1 when b is true, and 0 otherwise.
func boolFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
