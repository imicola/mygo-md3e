package tokens

// Density controls the vertical rhythm, control heights, padding scale, and
// default navigation rail width across MD3E components. The zero value
// (DensityComfortable) preserves the standard Material Design 3 Expressive
// metrics unchanged.
type Density int

const (
	// DensityComfortable uses standard Material Design 3 Expressive metrics
	// (touch-friendly, default zero value).
	DensityComfortable Density = iota
	// DensityDesktop uses tightened desktop-grade proportions for productivity
	// apps, dashboards, and mouse/keyboard workflows.
	DensityDesktop
	// DensityCompact uses high-density proportions for information-rich panels,
	// inspectors, and tool palettes.
	DensityCompact
)

// DensitySpec holds the resolved metric multipliers and canonical component
// dimensions (in DIP) for one Density tier.
type DensitySpec struct {
	// ControlScale scales standard control heights (1.0 / 0.85 / 0.75).
	ControlScale float64
	// PaddingScale scales horizontal/internal container padding (1.0 / 0.85 / 0.70).
	PaddingScale float64
	// SpacingUnit is the base grid unit in DIP bridged to ui.Theme.Spacing
	// (4 / 3.5 / 3).
	SpacingUnit float64
	// ListOneLineHeight is the minimum height of a one-line list item in DIP.
	ListOneLineHeight float64
	// ListTwoLineHeight is the minimum height of a two-line list item in DIP.
	ListTwoLineHeight float64
	// ListThreeLineHeight is the minimum height of a three-line list item in DIP.
	ListThreeLineHeight float64
	// TabHeight is the height of PrimaryTabs and SecondaryTabs in DIP.
	TabHeight float64
	// TextFieldHeight is the single-line TextField container height in DIP.
	TextFieldHeight float64
	// NavRailExpandedWidth is the default expanded NavigationRail width in DIP.
	NavRailExpandedWidth float64
}

var densitySpecs = [...]DensitySpec{
	DensityComfortable: {
		ControlScale:         1.0,
		PaddingScale:         1.0,
		SpacingUnit:          4,
		ListOneLineHeight:    56,
		ListTwoLineHeight:    72,
		ListThreeLineHeight:  88,
		TabHeight:            48,
		TextFieldHeight:      56,
		NavRailExpandedWidth: 256,
	},
	DensityDesktop: {
		ControlScale:         0.85,
		PaddingScale:         0.85,
		SpacingUnit:          3.5,
		ListOneLineHeight:    44,
		ListTwoLineHeight:    56,
		ListThreeLineHeight:  72,
		TabHeight:            40,
		TextFieldHeight:      44,
		NavRailExpandedWidth: 192,
	},
	DensityCompact: {
		ControlScale:         0.75,
		PaddingScale:         0.70,
		SpacingUnit:          3,
		ListOneLineHeight:    36,
		ListTwoLineHeight:    48,
		ListThreeLineHeight:  60,
		TabHeight:            36,
		TextFieldHeight:      38,
		NavRailExpandedWidth: 180,
	},
}

var desktopButtonSpecs = [...]SizeSpec{
	SizeXS: {Height: 28, RoundRadius: ShapeFull.Radius(), SquareRadius: 10, PressedRadius: 6, TextSize: 13, IconSize: 18, Padding: 10},
	SizeS:  {Height: 34, RoundRadius: ShapeFull.Radius(), SquareRadius: 10, PressedRadius: 7, TextSize: 14, IconSize: 18, Padding: 14},
	SizeM:  {Height: 48, RoundRadius: ShapeFull.Radius(), SquareRadius: 14, PressedRadius: 10, TextSize: 15, IconSize: 22, Padding: 20},
	SizeL:  {Height: 82, RoundRadius: ShapeFull.Radius(), SquareRadius: 24, PressedRadius: 14, TextSize: 22, IconSize: 28, Padding: 40},
	SizeXL: {Height: 116, RoundRadius: ShapeFull.Radius(), SquareRadius: 24, PressedRadius: 14, TextSize: 28, IconSize: 36, Padding: 54},
}

var compactButtonSpecs = [...]SizeSpec{
	SizeXS: {Height: 24, RoundRadius: ShapeFull.Radius(), SquareRadius: 8, PressedRadius: 6, TextSize: 12, IconSize: 16, Padding: 8},
	SizeS:  {Height: 30, RoundRadius: ShapeFull.Radius(), SquareRadius: 8, PressedRadius: 6, TextSize: 13, IconSize: 16, Padding: 12},
	SizeM:  {Height: 42, RoundRadius: ShapeFull.Radius(), SquareRadius: 12, PressedRadius: 8, TextSize: 14, IconSize: 20, Padding: 16},
	SizeL:  {Height: 72, RoundRadius: ShapeFull.Radius(), SquareRadius: 20, PressedRadius: 12, TextSize: 20, IconSize: 26, Padding: 36},
	SizeXL: {Height: 102, RoundRadius: ShapeFull.Radius(), SquareRadius: 20, PressedRadius: 12, TextSize: 26, IconSize: 32, Padding: 48},
}

// Normalize clamps out-of-range Density values safely back to DensityComfortable.
func (d Density) Normalize() Density {
	if d < DensityComfortable || d > DensityCompact {
		return DensityComfortable
	}
	return d
}

// Spec returns the DensitySpec for d, falling back to DensityComfortable for
// out-of-range enum values.
func (d Density) Spec() DensitySpec {
	return densitySpecs[d.Normalize()]
}

// ControlScale returns the control height multiplier for d.
func (d Density) ControlScale() float64 { return d.Spec().ControlScale }

// PaddingScale returns the container padding multiplier for d.
func (d Density) PaddingScale() float64 { return d.Spec().PaddingScale }

// SpacingUnit returns the base grid spacing unit in DIP for d.
func (d Density) SpacingUnit() float64 { return d.Spec().SpacingUnit }

// ListHeights returns the one-, two-, and three-line minimum list item heights
// in DIP for d.
func (d Density) ListHeights() (oneLine, twoLine, threeLine float64) {
	s := d.Spec()
	return s.ListOneLineHeight, s.ListTwoLineHeight, s.ListThreeLineHeight
}

// TabHeight returns the tab bar height in DIP for d.
func (d Density) TabHeight() float64 { return d.Spec().TabHeight }

// TextFieldHeight returns the single-line text field height in DIP for d.
func (d Density) TextFieldHeight() float64 { return d.Spec().TextFieldHeight }

// NavRailExpandedWidth returns the default expanded navigation rail width in
// DIP for d.
func (d Density) NavRailExpandedWidth() float64 { return d.Spec().NavRailExpandedWidth }

// ButtonSpec returns the button SizeSpec adapted to d. DensityComfortable
// returns tier.Spec() unchanged; invalid tiers return the zero SizeSpec.
func (d Density) ButtonSpec(tier SizeTier) SizeSpec {
	if tier < SizeXS || tier > SizeXL {
		return SizeSpec{}
	}
	switch d.Normalize() {
	case DensityDesktop:
		return desktopButtonSpecs[tier]
	case DensityCompact:
		return compactButtonSpecs[tier]
	default:
		return tier.Spec()
	}
}
