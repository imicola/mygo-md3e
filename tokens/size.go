package tokens

// SizeTier is the size of an Expressive action component, from extra small to
// extra large. Buttons, icon buttons, toggle buttons, button groups and split
// buttons all follow the same tiers.
type SizeTier int

const (
	SizeXS SizeTier = iota
	SizeS
	SizeM
	SizeL
	SizeXL
)

// SizeSpec is the exact geometry of one size tier, in DIP.
type SizeSpec struct {
	// Height is the container's height.
	Height float64
	// RoundRadius is the corner radius of the round shape.
	RoundRadius float64
	// SquareRadius is the corner radius of the square shape.
	SquareRadius float64
	// PressedRadius is the corner radius while pressed.
	PressedRadius float64
	// TextSize is the label's font size.
	TextSize float64
	// IconSize is the side of the tier's icons.
	IconSize float64
	// Padding is the horizontal padding of the standard form.
	Padding float64
}

var sizeSpecs = [...]SizeSpec{
	SizeXS: {Height: 32, RoundRadius: ShapeFull.Radius(), SquareRadius: 12, PressedRadius: 8, TextSize: 14, IconSize: 20, Padding: 12},
	SizeS:  {Height: 40, RoundRadius: ShapeFull.Radius(), SquareRadius: 12, PressedRadius: 8, TextSize: 14, IconSize: 20, Padding: 16},
	SizeM:  {Height: 56, RoundRadius: ShapeFull.Radius(), SquareRadius: 16, PressedRadius: 12, TextSize: 16, IconSize: 24, Padding: 24},
	SizeL:  {Height: 96, RoundRadius: ShapeFull.Radius(), SquareRadius: 28, PressedRadius: 16, TextSize: 24, IconSize: 32, Padding: 48},
	SizeXL: {Height: 136, RoundRadius: ShapeFull.Radius(), SquareRadius: 28, PressedRadius: 16, TextSize: 32, IconSize: 40, Padding: 64},
}

// Spec returns the tier's geometry. A tier outside SizeXS..SizeXL returns the
// zero SizeSpec.
func (t SizeTier) Spec() SizeSpec {
	if t < SizeXS || t > SizeXL {
		return SizeSpec{}
	}
	return sizeSpecs[t]
}

// Height returns the tier's container height in DIP.
func (t SizeTier) Height() float64 { return t.Spec().Height }

// Radius returns the tier's corner radius for its shape, in DIP.
func (t SizeTier) Radius(shape ShapeScale) float64 {
	spec := t.Spec()
	if shape == ShapeFull {
		return spec.RoundRadius
	}
	return spec.SquareRadius
}

// TextSize returns the tier's label size in DIP.
func (t SizeTier) TextSize() float64 { return t.Spec().TextSize }

// IconSize returns the tier's icon side in DIP.
func (t SizeTier) IconSize() float64 { return t.Spec().IconSize }

// Padding returns the tier's horizontal padding in DIP.
func (t SizeTier) Padding() float64 { return t.Spec().Padding }
