package tokens

import (
	"math"
	"testing"
)

func TestShapeScale(t *testing.T) {
	tests := []struct {
		shape  ShapeScale
		radius float64
	}{
		{ShapeNone, 0},
		{ShapeExtraSmall, 4},
		{ShapeSmall, 8},
		{ShapeMedium, 12},
		{ShapeLarge, 16},
		{ShapeLargeIncreased, 20},
		{ShapeExtraLarge, 28},
		{ShapeExtraLargeIncreased, 32},
		{ShapeExtraExtraLarge, 48},
		{ShapeFull, 9999},
	}
	if len(tests) != int(ShapeFull)+1 {
		t.Fatalf("%d shapes, want %d", len(tests), int(ShapeFull)+1)
	}
	for i, tt := range tests {
		if got := tt.shape.Radius(); got != tt.radius {
			t.Errorf("shape %d radius = %v, want %v", tt.shape, got, tt.radius)
		}
		if i > 0 {
			if prev := tests[i-1].shape.Radius(); tt.radius <= prev {
				t.Errorf("shape %d radius %v does not grow from %v", tt.shape, tt.radius, prev)
			}
		}
	}
	if got := ShapeScale(99).Radius(); got != 0 {
		t.Errorf("unknown shape radius = %v, want 0", got)
	}
}

func TestSizeTiers(t *testing.T) {
	tests := []struct {
		tier SizeTier
		want SizeSpec
	}{
		{SizeXS, SizeSpec{Height: 32, RoundRadius: 9999, SquareRadius: 12, PressedRadius: 8, TextSize: 14, IconSize: 20, Padding: 12}},
		{SizeS, SizeSpec{Height: 40, RoundRadius: 9999, SquareRadius: 12, PressedRadius: 8, TextSize: 14, IconSize: 20, Padding: 16}},
		{SizeM, SizeSpec{Height: 56, RoundRadius: 9999, SquareRadius: 16, PressedRadius: 12, TextSize: 16, IconSize: 24, Padding: 24}},
		{SizeL, SizeSpec{Height: 96, RoundRadius: 9999, SquareRadius: 28, PressedRadius: 16, TextSize: 24, IconSize: 32, Padding: 48}},
		{SizeXL, SizeSpec{Height: 136, RoundRadius: 9999, SquareRadius: 28, PressedRadius: 16, TextSize: 32, IconSize: 40, Padding: 64}},
	}
	for i, tt := range tests {
		got := tt.tier.Spec()
		if got != tt.want {
			t.Errorf("tier %d = %+v, want %+v", tt.tier, got, tt.want)
		}
		if i > 0 {
			if prev := tests[i-1].want.Height; tt.want.Height <= prev {
				t.Errorf("tier %d height %v does not grow from %v", tt.tier, tt.want.Height, prev)
			}
		}
		for name, v := range map[string]float64{
			"height": got.Height, "square radius": got.SquareRadius, "pressed radius": got.PressedRadius,
			"text size": got.TextSize, "icon size": got.IconSize, "padding": got.Padding,
		} {
			if v <= 0 {
				t.Errorf("tier %d %s = %v, want positive", tt.tier, name, v)
			}
		}
	}
	if got := SizeTier(99).Spec(); got != (SizeSpec{}) {
		t.Errorf("unknown tier = %+v, want the zero spec", got)
	}
	if got := SizeM.Radius(ShapeFull); got != 9999 {
		t.Errorf("SizeM round radius = %v, want 9999", got)
	}
	if got := SizeM.Radius(ShapeMedium); got != 16 {
		t.Errorf("SizeM square radius = %v, want 16", got)
	}
}

func TestTypographyScale(t *testing.T) {
	if TypeRoleCount != 15 {
		t.Fatalf("%d type roles, want 15", TypeRoleCount)
	}
	scale := NewTypographyScale()
	styles := 0
	for role := TypeRole(0); role < TypeRoleCount; role++ {
		standard := scale.Style(role, false)
		emphasized := scale.Style(role, true)
		for _, s := range []TypeStyle{standard, emphasized} {
			styles++
			if s.Size <= 0 {
				t.Errorf("role %d size = %v, want positive", role, s.Size)
			}
			if s.LineHeight < s.Size {
				t.Errorf("role %d line height %v is below its size %v", role, s.LineHeight, s.Size)
			}
			if s.Weight < 100 || s.Weight > 900 || s.Weight%100 != 0 {
				t.Errorf("role %d weight = %v, want a hundred between 100 and 900", role, s.Weight)
			}
			if math.IsNaN(s.Tracking) || math.IsInf(s.Tracking, 0) {
				t.Errorf("role %d tracking = %v", role, s.Tracking)
			}
		}
		if emphasized.Weight <= standard.Weight {
			t.Errorf("role %d emphasized weight %d is not above standard's %d", role, emphasized.Weight, standard.Weight)
		}
		if emphasized.Size != standard.Size || emphasized.LineHeight != standard.LineHeight || emphasized.Tracking != standard.Tracking {
			t.Errorf("role %d emphasized form changes geometry: %+v vs %+v", role, emphasized, standard)
		}
	}
	if styles != 30 {
		t.Errorf("%d styles, want 30", styles)
	}

	if got := scale.Style(DisplayLarge, false); got != (TypeStyle{Size: 57, LineHeight: 64, Weight: 400, Tracking: -0.25}) {
		t.Errorf("DisplayLarge = %+v", got)
	}
	if got := scale.Style(TitleMedium, true); got != (TypeStyle{Size: 16, LineHeight: 24, Weight: 600, Tracking: 0.15}) {
		t.Errorf("emphasized TitleMedium = %+v", got)
	}
	if got := scale.Style(LabelSmall, false); got != (TypeStyle{Size: 11, LineHeight: 16, Weight: 500, Tracking: 0.5}) {
		t.Errorf("LabelSmall = %+v", got)
	}
	if got := scale.Style(TypeRole(99), false); got != (TypeStyle{}) {
		t.Errorf("unknown role = %+v, want the zero style", got)
	}
}

func TestElevationLevels(t *testing.T) {
	tests := []struct {
		level  ElevationLevel
		height float64
		tint   float64
	}{
		{Level0, 0, 0},
		{Level1, 1, 0.05},
		{Level2, 3, 0.08},
		{Level3, 6, 0.11},
		{Level4, 8, 0.12},
		{Level5, 12, 0.14},
	}
	if len(tests) != int(Level5)+1 {
		t.Fatalf("%d levels, want %d", len(tests), int(Level5)+1)
	}
	for i, tt := range tests {
		spec := tt.level.Spec()
		if spec.HeightDP != tt.height {
			t.Errorf("level %d height = %v, want %v", tt.level, spec.HeightDP, tt.height)
		}
		if spec.TintRatio != tt.tint {
			t.Errorf("level %d tint = %v, want %v", tt.level, spec.TintRatio, tt.tint)
		}
		if got := tt.level.Height(); got != tt.height {
			t.Errorf("level %d Height() = %v, want %v", tt.level, got, tt.height)
		}
		if got := tt.level.TintRatio(); got != tt.tint {
			t.Errorf("level %d TintRatio() = %v, want %v", tt.level, got, tt.tint)
		}
		if i == 0 {
			if spec.Key != (Shadow{}) || spec.Ambient != (Shadow{}) {
				t.Errorf("level 0 casts shadows: %+v / %+v", spec.Key, spec.Ambient)
			}
			continue
		}
		prev := tests[i-1].level.Spec()
		if spec.HeightDP <= prev.HeightDP {
			t.Errorf("level %d height %v does not grow from %v", tt.level, spec.HeightDP, prev.HeightDP)
		}
		if spec.TintRatio < prev.TintRatio {
			t.Errorf("level %d tint %v falls from %v", tt.level, spec.TintRatio, prev.TintRatio)
		}
		if spec.Key.OffsetY < prev.Key.OffsetY || spec.Ambient.OffsetY < prev.Ambient.OffsetY {
			t.Errorf("level %d offset falls back", tt.level)
		}
		if spec.Key.Blur < prev.Key.Blur || spec.Ambient.Blur < prev.Ambient.Blur {
			t.Errorf("level %d blur falls back", tt.level)
		}
		if spec.Key.Opacity != 0.30 || spec.Ambient.Opacity != 0.15 {
			t.Errorf("level %d opacities = %v/%v, want 0.30/0.15", tt.level, spec.Key.Opacity, spec.Ambient.Opacity)
		}
	}
	if got := ElevationLevel(99).Spec(); got != (ElevationSpec{}) {
		t.Errorf("unknown level = %+v, want the zero spec", got)
	}
}
