package tokens

// ElevationLevel is one of the six MD3 elevation levels.
type ElevationLevel int

const (
	Level0 ElevationLevel = iota
	Level1
	Level2
	Level3
	Level4
	Level5
)

// Shadow is one shadow of an elevation, in DIP.
type Shadow struct {
	// OffsetY is the vertical offset, downward.
	OffsetY float64
	// Blur is the blur radius.
	Blur float64
	// Spread grows (or, when negative, shrinks) the shadow.
	Spread float64
	// Opacity is the shadow's alpha, 0..1.
	Opacity float64
}

// ElevationSpec is the geometry of an elevation: its height above the
// surface, its key and ambient shadows, and the share of the surface tint it
// mixes into the surface's background.
type ElevationSpec struct {
	// HeightDP is the elevation above the surface, in DIP.
	HeightDP float64
	// Key is the shadow cast by a light straight above the surface.
	Key Shadow
	// Ambient is the softer shadow all around.
	Ambient Shadow
	// TintRatio is the share of SurfaceTint mixed into the background, 0..1.
	TintRatio float64
}

var elevationSpecs = [...]ElevationSpec{
	Level0: {HeightDP: 0},
	Level1: {
		HeightDP:  1,
		Key:       Shadow{OffsetY: 1, Blur: 2, Opacity: 0.30},
		Ambient:   Shadow{OffsetY: 1, Blur: 3, Opacity: 0.15},
		TintRatio: 0.05,
	},
	Level2: {
		HeightDP:  3,
		Key:       Shadow{OffsetY: 1, Blur: 3, Opacity: 0.30},
		Ambient:   Shadow{OffsetY: 2, Blur: 6, Opacity: 0.15},
		TintRatio: 0.08,
	},
	Level3: {
		HeightDP:  6,
		Key:       Shadow{OffsetY: 3, Blur: 5, Opacity: 0.30},
		Ambient:   Shadow{OffsetY: 3, Blur: 10, Opacity: 0.15},
		TintRatio: 0.11,
	},
	Level4: {
		HeightDP:  8,
		Key:       Shadow{OffsetY: 4, Blur: 5, Opacity: 0.30},
		Ambient:   Shadow{OffsetY: 4, Blur: 10, Opacity: 0.15},
		TintRatio: 0.12,
	},
	Level5: {
		HeightDP:  12,
		Key:       Shadow{OffsetY: 6, Blur: 7, Opacity: 0.30},
		Ambient:   Shadow{OffsetY: 6, Blur: 12, Opacity: 0.15},
		TintRatio: 0.14,
	},
}

// Spec returns the level's elevation. A level outside Level0..Level5 returns
// the zero ElevationSpec.
func (l ElevationLevel) Spec() ElevationSpec {
	if l < Level0 || l > Level5 {
		return ElevationSpec{}
	}
	return elevationSpecs[l]
}

// Height returns the level's elevation above the surface, in DIP.
func (l ElevationLevel) Height() float64 { return l.Spec().HeightDP }

// TintRatio returns the share of SurfaceTint the level mixes into the
// background, 0..1.
func (l ElevationLevel) TintRatio() float64 { return l.Spec().TintRatio }
