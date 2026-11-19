package tokens

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// sampleSeed is the Material baseline purple.
var sampleSeed = ui.Hex("#6750A4")

func TestSurfaceContainerOrder(t *testing.T) {
	tests := []struct {
		name   string
		dark   bool
		expect []float64
	}{
		{"light", false, []float64{100, 96, 94, 92, 90}},
		{"dark", true, []float64{4, 10, 12, 17, 22}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewDynamicScheme(DynamicSchemeOptions{Seed: sampleSeed, Dark: tt.dark}).Colors
			containers := []struct {
				name  string
				color ui.Color
			}{
				{"Lowest", s.SurfaceContainerLowest},
				{"Low", s.SurfaceContainerLow},
				{"Container", s.SurfaceContainer},
				{"High", s.SurfaceContainerHigh},
				{"Highest", s.SurfaceContainerHighest},
			}
			for i, c := range containers {
				tone := HCTFromColor(c.color).Tone
				if math.Abs(tone-tt.expect[i]) > 0.6 {
					t.Errorf("%s tone = %.2f, want %.0f", c.name, tone, tt.expect[i])
				}
				if i > 0 {
					prev := HCTFromColor(containers[i-1].color).Tone
					if tt.dark && tone <= prev {
						t.Errorf("%s tone %.2f is not above %s tone %.2f", c.name, tone, containers[i-1].name, prev)
					}
					if !tt.dark && tone >= prev {
						t.Errorf("%s tone %.2f is not below %s tone %.2f", c.name, tone, containers[i-1].name, prev)
					}
				}
			}
		})
	}
}

func TestFixedRolesAreConstant(t *testing.T) {
	light := NewDynamicScheme(DynamicSchemeOptions{Seed: sampleSeed}).Colors
	dark := NewDynamicScheme(DynamicSchemeOptions{Seed: sampleSeed, Dark: true}).Colors

	fixed := []struct {
		name        string
		light, dark ui.Color
	}{
		{"PrimaryFixed", light.PrimaryFixed, dark.PrimaryFixed},
		{"PrimaryFixedDim", light.PrimaryFixedDim, dark.PrimaryFixedDim},
		{"OnPrimaryFixed", light.OnPrimaryFixed, dark.OnPrimaryFixed},
		{"OnPrimaryFixedVariant", light.OnPrimaryFixedVariant, dark.OnPrimaryFixedVariant},
		{"SecondaryFixed", light.SecondaryFixed, dark.SecondaryFixed},
		{"SecondaryFixedDim", light.SecondaryFixedDim, dark.SecondaryFixedDim},
		{"OnSecondaryFixed", light.OnSecondaryFixed, dark.OnSecondaryFixed},
		{"OnSecondaryFixedVariant", light.OnSecondaryFixedVariant, dark.OnSecondaryFixedVariant},
		{"TertiaryFixed", light.TertiaryFixed, dark.TertiaryFixed},
		{"TertiaryFixedDim", light.TertiaryFixedDim, dark.TertiaryFixedDim},
		{"OnTertiaryFixed", light.OnTertiaryFixed, dark.OnTertiaryFixed},
		{"OnTertiaryFixedVariant", light.OnTertiaryFixedVariant, dark.OnTertiaryFixedVariant},
	}
	for _, f := range fixed {
		if f.light != f.dark {
			t.Errorf("%s differs between light (%v) and dark (%v)", f.name, f.light, f.dark)
		}
	}
}

func TestContrastLevels(t *testing.T) {
	for _, dark := range []bool{false, true} {
		name := "light"
		if dark {
			name = "dark"
		}
		t.Run(name, func(t *testing.T) {
			standard := NewDynamicScheme(DynamicSchemeOptions{Seed: sampleSeed, Dark: dark, Contrast: ContrastStandard}).Colors
			high := NewDynamicScheme(DynamicSchemeOptions{Seed: sampleSeed, Dark: dark, Contrast: ContrastHigh}).Colors

			onPrimary := ContrastRatio(standard.OnPrimary, standard.Primary)
			onPrimaryHigh := ContrastRatio(high.OnPrimary, high.Primary)
			if onPrimary < 4.5 {
				t.Errorf("standard OnPrimary on Primary is %.2f:1, want at least 4.5", onPrimary)
			}
			if onPrimaryHigh <= onPrimary {
				t.Errorf("high contrast OnPrimary on Primary is %.2f:1, want above standard's %.2f", onPrimaryHigh, onPrimary)
			}
			if onPrimaryHigh < 7 {
				t.Errorf("high contrast OnPrimary on Primary is %.2f:1, want at least 7", onPrimaryHigh)
			}

			onSurface := ContrastRatio(standard.OnSurface, standard.Surface)
			onSurfaceHigh := ContrastRatio(high.OnSurface, high.Surface)
			if onSurface < 4.5 {
				t.Errorf("standard OnSurface on Surface is %.2f:1, want at least 4.5", onSurface)
			}
			if onSurfaceHigh <= onSurface {
				t.Errorf("high contrast OnSurface on Surface is %.2f:1, want above standard's %.2f", onSurfaceHigh, onSurface)
			}
			if onSurfaceHigh < 7 {
				t.Errorf("high contrast OnSurface on Surface is %.2f:1, want at least 7", onSurfaceHigh)
			}
		})
	}
}

func TestSchemeVariants(t *testing.T) {
	variants := []Variant{TonalSpot, Expressive, Vibrant, Neutral, Monochrome, Fidelity, Content}
	for _, v := range variants {
		for _, dark := range []bool{false, true} {
			for _, contrast := range []Contrast{ContrastStandard, ContrastMedium, ContrastHigh} {
				s := NewDynamicScheme(DynamicSchemeOptions{
					Seed: sampleSeed, Dark: dark, Variant: v, Contrast: contrast,
				})
				roles := []struct {
					name  string
					color ui.Color
				}{
					{"Primary", s.Colors.Primary},
					{"OnPrimary", s.Colors.OnPrimary},
					{"PrimaryContainer", s.Colors.PrimaryContainer},
					{"OnPrimaryContainer", s.Colors.OnPrimaryContainer},
					{"InversePrimary", s.Colors.InversePrimary},
					{"PrimaryFixed", s.Colors.PrimaryFixed},
					{"PrimaryFixedDim", s.Colors.PrimaryFixedDim},
					{"OnPrimaryFixed", s.Colors.OnPrimaryFixed},
					{"OnPrimaryFixedVariant", s.Colors.OnPrimaryFixedVariant},
					{"Secondary", s.Colors.Secondary},
					{"OnSecondary", s.Colors.OnSecondary},
					{"SecondaryContainer", s.Colors.SecondaryContainer},
					{"OnSecondaryContainer", s.Colors.OnSecondaryContainer},
					{"Tertiary", s.Colors.Tertiary},
					{"OnTertiary", s.Colors.OnTertiary},
					{"TertiaryContainer", s.Colors.TertiaryContainer},
					{"OnTertiaryContainer", s.Colors.OnTertiaryContainer},
					{"Error", s.Colors.Error},
					{"OnError", s.Colors.OnError},
					{"ErrorContainer", s.Colors.ErrorContainer},
					{"OnErrorContainer", s.Colors.OnErrorContainer},
					{"Background", s.Colors.Background},
					{"OnBackground", s.Colors.OnBackground},
					{"Surface", s.Colors.Surface},
					{"SurfaceDim", s.Colors.SurfaceDim},
					{"SurfaceBright", s.Colors.SurfaceBright},
					{"OnSurface", s.Colors.OnSurface},
					{"OnSurfaceVariant", s.Colors.OnSurfaceVariant},
					{"SurfaceVariant", s.Colors.SurfaceVariant},
					{"SurfaceContainerLowest", s.Colors.SurfaceContainerLowest},
					{"SurfaceContainerLow", s.Colors.SurfaceContainerLow},
					{"SurfaceContainer", s.Colors.SurfaceContainer},
					{"SurfaceContainerHigh", s.Colors.SurfaceContainerHigh},
					{"SurfaceContainerHighest", s.Colors.SurfaceContainerHighest},
					{"SurfaceTint", s.Colors.SurfaceTint},
					{"Outline", s.Colors.Outline},
					{"OutlineVariant", s.Colors.OutlineVariant},
					{"InverseSurface", s.Colors.InverseSurface},
					{"InverseOnSurface", s.Colors.InverseOnSurface},
					{"Scrim", s.Colors.Scrim},
					{"Shadow", s.Colors.Shadow},
				}
				for _, role := range roles {
					if role.color.A != 255 {
						t.Errorf("%v dark=%v contrast=%v: %s is not opaque: %v", v, dark, contrast, role.name, role.color)
					}
				}
				if ContrastRatio(s.Colors.OnPrimary, s.Colors.Primary) < 3 {
					t.Errorf("%v dark=%v contrast=%v: OnPrimary on Primary is %.2f:1", v, dark, contrast,
						ContrastRatio(s.Colors.OnPrimary, s.Colors.Primary))
				}
			}
		}
	}
}

func TestMonochromeIsGray(t *testing.T) {
	s := NewDynamicScheme(DynamicSchemeOptions{Seed: sampleSeed, Variant: Monochrome}).Colors
	for _, c := range []ui.Color{s.Primary, s.Secondary, s.Tertiary, s.Surface, s.OnSurface} {
		if c.R != c.G || c.G != c.B {
			t.Errorf("%v is not gray", c)
		}
	}
}

func TestExpressiveRotatesPrimaryHue(t *testing.T) {
	tonal := NewDynamicScheme(DynamicSchemeOptions{Seed: sampleSeed, Variant: TonalSpot})
	expressive := NewDynamicScheme(DynamicSchemeOptions{Seed: sampleSeed, Variant: Expressive})
	sourceHue := HCTFromColor(sampleSeed).Hue
	wantExpressive := SanitizeDegrees(sourceHue + 240)
	if delta := math.Abs(SanitizeDegrees(HCTFromColor(expressive.Colors.Primary).Hue - wantExpressive)); math.Min(delta, 360-delta) > 2 {
		t.Errorf("Expressive primary hue is %.2f, want %.2f", HCTFromColor(expressive.Colors.Primary).Hue, wantExpressive)
	}
	if tonal.Colors.Primary == expressive.Colors.Primary {
		t.Error("Expressive and TonalSpot resolve the same primary")
	}
	if s := expressive.PrimaryPalette; math.Abs(s.Chroma()-40) > 1e-9 {
		t.Errorf("Expressive primary chroma = %v, want 40", s.Chroma())
	}
}

func TestTonalPaletteCache(t *testing.T) {
	p := NewTonalPalette(300, 40)
	first := p.Tone(40)
	if second := p.Tone(40); second != first {
		t.Errorf("tone 40 came back as %v and %v", first, second)
	}
	if p.Tone(0) != ui.RGB(0, 0, 0) {
		t.Errorf("tone 0 is %v, want black", p.Tone(0))
	}
	if p.Tone(100) != ui.RGB(255, 255, 255) {
		t.Errorf("tone 100 is %v, want white", p.Tone(100))
	}
	if hct := HCTFromColor(p.Tone(40)); math.Abs(hct.Hue-300) > 0.5 || math.Abs(hct.Chroma-40) > 0.5 {
		t.Errorf("tone 40 is %+v, want hue 300 and chroma 40", hct)
	}

	fromColor := PaletteFromColor(sampleSeed)
	got := HCTFromColor(sampleSeed)
	if math.Abs(fromColor.Hue()-got.Hue) > 1e-9 || math.Abs(fromColor.Chroma()-got.Chroma) > 1e-9 {
		t.Errorf("palette from %v is %v/%v, want %v/%v", sampleSeed, fromColor.Hue(), fromColor.Chroma(), got.Hue, got.Chroma)
	}
}

func TestStateLayerOpacities(t *testing.T) {
	cases := []struct {
		name string
		got  float64
		want float64
	}{
		{"hover", StateHoverOpacity, 0.08},
		{"focus", StateFocusOpacity, 0.10},
		{"pressed", StatePressedOpacity, 0.10},
		{"dragged", StateDraggedOpacity, 0.16},
		{"disabled container", DisabledContainerOpacity, 0.12},
		{"disabled content", DisabledContentOpacity, 0.38},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s opacity = %v, want %v", c.name, c.got, c.want)
		}
	}
}
