package tokens

// ShapeScale is the MD3 corner radius scale, in DIP.
type ShapeScale int

const (
	ShapeNone ShapeScale = iota
	ShapeExtraSmall
	ShapeSmall
	ShapeMedium
	ShapeLarge
	ShapeLargeIncreased
	ShapeExtraLarge
	ShapeExtraLargeIncreased
	ShapeExtraExtraLarge
	// ShapeFull is a pill or a circle: a radius of half the shorter side.
	ShapeFull
)

// radius returns the shape's corner radius in DIP.
var shapeRadii = [...]float64{
	ShapeNone:                0,
	ShapeExtraSmall:          4,
	ShapeSmall:               8,
	ShapeMedium:              12,
	ShapeLarge:               16,
	ShapeLargeIncreased:      20,
	ShapeExtraLarge:          28,
	ShapeExtraLargeIncreased: 32,
	ShapeExtraExtraLarge:     48,
	ShapeFull:                9999,
}

// Radius returns the corner radius of the shape, in DIP. ShapeFull's 9999
// means half the shorter side: paint it as a pill or a circle.
func (s ShapeScale) Radius() float64 {
	if int(s) < 0 || int(s) >= len(shapeRadii) {
		return 0
	}
	return shapeRadii[s]
}
