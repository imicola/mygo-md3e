package tokens

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestCAM16OfficialVectors checks the CAM16 forward model against the
// reference values of material-color-utilities' hct_test.dart.
func TestCAM16OfficialVectors(t *testing.T) {
	tests := []struct {
		name   string
		color  ui.Color
		j      float64
		chroma float64
		hue    float64
		m      float64
		s      float64
		q      float64
	}{
		{"red", ui.RGB(255, 0, 0), 46.445, 113.357, 27.408, 89.494, 91.889, 105.988},
		{"green", ui.RGB(0, 255, 0), 79.331, 108.410, 142.139, 85.587, 78.604, 138.520},
		{"blue", ui.RGB(0, 0, 255), 25.465, 87.230, 282.788, 68.867, 93.674, 78.481},
		{"white", ui.RGB(255, 255, 255), 100.0, 2.869, 209.492, 2.265, 12.068, 155.521},
		{"black", ui.RGB(0, 0, 0), 0, 0, 0, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cam := CAM16FromColor(tt.color)
			for _, f := range []struct {
				name string
				got  float64
				want float64
			}{
				{"J", cam.J, tt.j},
				{"chroma", cam.Chroma, tt.chroma},
				{"hue", cam.Hue, tt.hue},
				{"M", cam.M, tt.m},
				{"S", cam.S, tt.s},
				{"Q", cam.Q, tt.q},
			} {
				if math.Abs(f.got-f.want) > 0.01 {
					t.Errorf("%s = %.4f, want %.3f", f.name, f.got, f.want)
				}
			}
		})
	}
}

func TestHCTFromReferenceColors(t *testing.T) {
	tests := []struct {
		name   string
		color  ui.Color
		hue    float64
		chroma float64
		tone   float64
	}{
		// The baseline purple's HCT coordinates under Material's default
		// viewing conditions.
		{"baseline purple", ui.Hex("#6750A4"), 298.981, 47.857, 40.083},
		{"white", ui.RGB(255, 255, 255), 209.492, 2.869, 100},
		{"black", ui.RGB(0, 0, 0), 0, 0, 0},
		{"gray", ui.RGB(128, 128, 128), 209.494, 1.896, 53.585},
		{"red", ui.RGB(255, 0, 0), 27.408, 113.357, 53.233},
		{"green", ui.RGB(0, 255, 0), 142.139, 108.410, 87.737},
		{"blue", ui.RGB(0, 0, 255), 282.788, 87.230, 32.303},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HCTFromColor(tt.color)
			if math.Abs(got.Tone-tt.tone) > 0.01 {
				t.Errorf("tone = %.4f, want %.3f", got.Tone, tt.tone)
			}
			if math.Abs(got.Chroma-tt.chroma) > 0.01 {
				t.Errorf("chroma = %.4f, want %.3f", got.Chroma, tt.chroma)
			}
			if delta := math.Abs(SanitizeDegrees(got.Hue - tt.hue)); math.Min(delta, 360-delta) > 0.01 {
				t.Errorf("hue = %.4f, want %.3f", got.Hue, tt.hue)
			}
		})
	}
}

func TestHCTRoundTrip(t *testing.T) {
	colors := []ui.Color{
		ui.Hex("#6750A4"),
		ui.Hex("#D0BCFF"),
		ui.Hex("#EADDFF"),
		ui.RGB(255, 255, 255),
		ui.RGB(0, 0, 0),
		ui.RGB(128, 128, 128),
		ui.RGB(255, 0, 0),
		ui.RGB(0, 255, 0),
		ui.RGB(0, 0, 255),
		ui.RGB(255, 255, 0),
		ui.RGB(0, 255, 255),
		ui.RGB(255, 0, 255),
		ui.Hex("#1D1B20"),
		ui.Hex("#49454F"),
	}
	for _, c := range colors {
		hct := HCTFromColor(c)
		got := hct.ToColor()
		for i, ch := range [][2]uint8{{c.R, got.R}, {c.G, got.G}, {c.B, got.B}} {
			if diff := int(ch[0]) - int(ch[1]); diff < -2 || diff > 2 {
				t.Errorf("%v: channel %d is %d, want %d (HCT %+v)", c, i, ch[1], ch[0], hct)
			}
		}
		if back := HCTFromColor(got); math.Abs(back.Tone-hct.Tone) > 0.5 {
			t.Errorf("%v: tone %v round-trips to %v", c, hct.Tone, back.Tone)
		}
	}
}

// TestHCTCorrectness sweeps hues, chromas and tones as material-color-utilities'
// hct_test.dart does: the hue stays close to the request, the chroma never
// grows beyond it, a reduced chroma lands on the gamut boundary, and the tone
// holds.
func TestHCTCorrectness(t *testing.T) {
	for hue := 15.0; hue < 360; hue += 30 {
		for chroma := 0.0; chroma <= 100; chroma += 10 {
			for tone := 20.0; tone <= 80; tone += 10 {
				want := HCT{Hue: hue, Chroma: chroma, Tone: tone}
				c := want.ToColor()
				got := HCTFromColor(c)
				if got.Tone < tone-0.5 || got.Tone > tone+0.5 {
					t.Fatalf("%+v: tone = %.3f", want, got.Tone)
				}
				if got.Chroma > chroma+2.5 {
					t.Fatalf("%+v: chroma = %.3f", want, got.Chroma)
				}
				if got.Chroma < chroma-2.5 {
					if !onGamutBoundary(c) {
						t.Errorf("%+v: chroma reduced to %.3f off the gamut boundary, color %v", want, got.Chroma, c)
					}
				} else if chroma > 0 {
					if delta := math.Abs(SanitizeDegrees(got.Hue - hue)); math.Min(delta, 360-delta) > 4 {
						t.Errorf("%+v: hue = %.3f", want, got.Hue)
					}
				}
			}
		}
	}
}

func TestHCTToneBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		tone  float64
		color ui.Color
	}{
		{"black", 0, ui.RGB(0, 0, 0)},
		{"below black", -10, ui.RGB(0, 0, 0)},
		{"white", 100, ui.RGB(255, 255, 255)},
		{"above white", 110, ui.RGB(255, 255, 255)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HCT{Hue: 200, Chroma: 60, Tone: tt.tone}.ToColor()
			if got != tt.color {
				t.Errorf("tone %v is %v, want %v", tt.tone, got, tt.color)
			}
		})
	}
}

func TestHCTChromaZeroIsGray(t *testing.T) {
	for _, tone := range []float64{10, 25, 50, 75, 90} {
		c := HCT{Hue: 123, Chroma: 0, Tone: tone}.ToColor()
		if c.R != c.G || c.G != c.B {
			t.Errorf("tone %v is %v, want a gray", tone, c)
		}
		if got := HCTFromColor(c).Tone; math.Abs(got-tone) > 0.6 {
			t.Errorf("tone %v round-trips to %.3f", tone, got)
		}
	}
}

func TestHCTHueNormalization(t *testing.T) {
	base := HCT{Hue: 330, Chroma: 50, Tone: 50}.ToColor()
	if got := (HCT{Hue: -30, Chroma: 50, Tone: 50}).ToColor(); got != base {
		t.Errorf("hue -30 is %v, want %v", got, base)
	}
	wrapped := HCT{Hue: 30, Chroma: 50, Tone: 50}.ToColor()
	if got := (HCT{Hue: 390, Chroma: 50, Tone: 50}).ToColor(); got != wrapped {
		t.Errorf("hue 390 is %v, want %v", got, wrapped)
	}
}

func TestHCTOutOfGamutChromaFits(t *testing.T) {
	// Impossibly high chroma at a mid tone is reduced into sRGB, keeping the
	// hue and tone, as Material's HCT solver does.
	want := HCT{Hue: 270, Chroma: 200, Tone: 50}
	got := want.ToColor()
	hct := HCTFromColor(got)
	if hct.Chroma < 40 {
		t.Errorf("chroma = %.1f, want a saturated color", hct.Chroma)
	}
	if math.Abs(hct.Tone-50) > 0.5 {
		t.Errorf("tone = %.2f, want 50", hct.Tone)
	}
	if !onGamutBoundary(got) {
		t.Errorf("%v is not on the gamut boundary", got)
	}
}

func TestCAM16Inverse(t *testing.T) {
	want := ui.Hex("#6750A4")
	cam := CAM16FromColor(want)
	got := ColorFromCAM16(cam.J, cam.Chroma, cam.Hue)
	if diff := channelDiff(got, want); diff > 2 {
		t.Errorf("ColorFromCAM16(%+v) = %v, want %v (off by %d)", cam, got, want, diff)
	}
}

// TestColorFromCAM16ClampsNegativeInputs checks that a negative lightness or
// chroma resolves to black rather than to a NaN through the square root.
func TestColorFromCAM16ClampsNegativeInputs(t *testing.T) {
	if got := ColorFromCAM16(-10, -20, 200); got != ui.RGB(0, 0, 0) {
		t.Errorf("negative lightness and chroma is %v, want black", got)
	}
	if got := ColorFromCAM16(0, 60, 200); got != ui.RGB(0, 0, 0) {
		t.Errorf("zero lightness is %v, want black", got)
	}
	if got := ColorFromCAM16(100, 500, 200); got.A != 255 {
		t.Errorf("over-saturated color is %v, want an opaque color", got)
	}
}

// TestInverseAdaptResponseLimit checks that the response at and beyond the
// model's ±400 limit stays finite instead of dividing by zero.
func TestInverseAdaptResponseLimit(t *testing.T) {
	for _, component := range []float64{400, 500, -400, -500} {
		got := inverseAdaptResponse(component, 1)
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Errorf("inverseAdaptResponse(%v) = %v, want a finite response", component, got)
		}
	}
}

func TestLabConversion(t *testing.T) {
	colors := []ui.Color{
		ui.Hex("#6750A4"), ui.RGB(0, 0, 0), ui.RGB(255, 255, 255),
		ui.RGB(255, 0, 0), ui.RGB(0, 128, 0), ui.RGB(0, 0, 255),
	}
	for _, c := range colors {
		xyz := XYZFromColor(c)
		l, a, b := LabFromXYZ(xyz)
		back := XYZFromLab(l, a, b)
		for i := range xyz {
			if math.Abs(back[i]-xyz[i]) > 1e-6 {
				t.Errorf("%v: XYZ %v round-trips to %v", c, xyz, back)
			}
		}
	}
}

func TestLstarYRoundTrip(t *testing.T) {
	for tone := 0.0; tone <= 100; tone += 5 {
		if got := LstarFromY(YFromLstar(tone)); math.Abs(got-tone) > 1e-9 {
			t.Errorf("tone %v round-trips to %v", tone, got)
		}
	}
	if got := YFromLstar(50); math.Abs(got-18.418) > 0.001 {
		t.Errorf("Y of tone 50 is %v, want 18.418", got)
	}
}

func TestContrastRatio(t *testing.T) {
	white := ui.RGB(255, 255, 255)
	black := ui.RGB(0, 0, 0)
	if got := ContrastRatio(white, black); math.Abs(got-21) > 0.01 {
		t.Errorf("white on black is %.3f, want 21", got)
	}
	if got := ContrastRatio(black, white); math.Abs(got-21) > 0.01 {
		t.Errorf("black on white is %.3f, want 21", got)
	}
	if got := ContrastRatio(white, white); math.Abs(got-1) > 1e-9 {
		t.Errorf("white on white is %.3f, want 1", got)
	}

	purple := ui.Hex("#6750A4")
	if got := ContrastRatio(purple, white); math.Abs(got-6.46) > 0.1 {
		t.Errorf("purple on white is %.3f, want about 6.46", got)
	}
	if got := ContrastRatio(purple, black); math.Abs(got-3.25) > 0.1 {
		t.Errorf("purple on black is %.3f, want about 3.25", got)
	}
}

func TestContrastTone(t *testing.T) {
	if got := ContrastTone(40, 4.5); got != 100 {
		t.Errorf("on tone 40 at 4.5:1 is %v, want 100", got)
	}
	if got := ContrastTone(90, 4.5); got != 0 {
		t.Errorf("on tone 90 at 4.5:1 is %v, want 0", got)
	}
	// Neither extreme reaches 7:1 against this tone; the greater contrast wins.
	if got := ContrastTone(40, 7); got != 100 {
		t.Errorf("on tone 40 at 7:1 is %v, want 100", got)
	}
	for _, tone := range []float64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100} {
		got := ContrastTone(tone, 4.5)
		if ratio := ContrastRatioOfTones(tone, got); ratio < 4.4 {
			t.Errorf("tone %v on %v is %.2f:1", got, tone, ratio)
		}
	}
}

// onGamutBoundary reports whether a color has a channel at 0 or 255.
func onGamutBoundary(c ui.Color) bool {
	return c.R == 0 || c.R == 255 || c.G == 0 || c.G == 255 || c.B == 0 || c.B == 255
}

func channelDiff(a, b ui.Color) int {
	diff := absInt(int(a.R) - int(b.R))
	diff = max(diff, absInt(int(a.G)-int(b.G)))
	return max(diff, absInt(int(a.B)-int(b.B)))
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
