package md3e

import (
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"weak"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// DefaultSeedColor is the baseline Material Design 3 primary seed (#6750A4)
// used when no custom seed color or system accent preference is provided.
var DefaultSeedColor = ui.RGBA(0x67, 0x50, 0xA4, 1)

// ShapeScale holds corner radius values in DIPs keyed to MD3E shape tokens.
type ShapeScale struct {
	None                float32
	ExtraSmall          float32
	Small               float32
	Medium              float32
	Large               float32
	LargeIncreased      float32
	ExtraLarge          float32
	ExtraLargeIncreased float32
	ExtraExtraLarge     float32
	Full                float32
}

// DefaultShapeScale returns the standard MD3E shape scale radius values.
func DefaultShapeScale() ShapeScale {
	return ShapeScale{
		None:                float32(tokens.ShapeNone.Radius()),
		ExtraSmall:          float32(tokens.ShapeExtraSmall.Radius()),
		Small:               float32(tokens.ShapeSmall.Radius()),
		Medium:              float32(tokens.ShapeMedium.Radius()),
		Large:               float32(tokens.ShapeLarge.Radius()),
		LargeIncreased:      float32(tokens.ShapeLargeIncreased.Radius()),
		ExtraLarge:          float32(tokens.ShapeExtraLarge.Radius()),
		ExtraLargeIncreased: float32(tokens.ShapeExtraLargeIncreased.Radius()),
		ExtraExtraLarge:     float32(tokens.ShapeExtraExtraLarge.Radius()),
		Full:                float32(tokens.ShapeFull.Radius()),
	}
}

// Radius returns the radius in DIPs configured for scale. Fields left at zero
// fall back to the standard MD3E radius for their own scale, so a partial
// ShapeScale customizes just the steps it sets.
func (s ShapeScale) Radius(scale tokens.ShapeScale) float32 {
	fallback := float32(scale.Radius())
	var v float32
	switch scale {
	case tokens.ShapeNone:
		f := float64(s.None)
		if math.IsNaN(f) || math.IsInf(f, 0) || s.None < 0 {
			return fallback
		}
		return s.None
	case tokens.ShapeExtraSmall:
		v = s.ExtraSmall
	case tokens.ShapeSmall:
		v = s.Small
	case tokens.ShapeMedium:
		v = s.Medium
	case tokens.ShapeLarge:
		v = s.Large
	case tokens.ShapeLargeIncreased:
		v = s.LargeIncreased
	case tokens.ShapeExtraLarge:
		v = s.ExtraLarge
	case tokens.ShapeExtraLargeIncreased:
		v = s.ExtraLargeIncreased
	case tokens.ShapeExtraExtraLarge:
		v = s.ExtraExtraLarge
	case tokens.ShapeFull:
		v = s.Full
	default:
		return fallback
	}
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) || !(v > 0) {
		return fallback
	}
	return v
}

// ThemeOptions configures how ThemeFromSeed constructs an MD3E theme.
type ThemeOptions struct {
	Dark         bool
	Variant      tokens.Variant
	Contrast     tokens.Contrast
	SurfaceStyle tokens.SurfaceStyle
	Motion       motion.Scheme
	Density      tokens.Density
	Font         string
	MonoFont     string
	TextScale    float32
	ReduceMotion bool
	Shapes       *ShapeScale
	Typography   *tokens.TypographyScale
}

// Theme aggregates all Material Design 3 Expressive design tokens and runtime
// accessibility preferences needed to render and animate components cleanly.
type Theme struct {
	Dark         bool
	Colors       tokens.ColorScheme
	SurfaceStyle tokens.SurfaceStyle
	Motion       motion.Scheme
	Density      tokens.Density
	Typography   tokens.TypographyScale
	Shapes       ShapeScale
	Contrast     tokens.Contrast
	TextScale    float32
	ReduceMotion bool
	Font         string
	MonoFont     string

	// Extended semantic status roles used by banners, badges, and ui.Theme bridging.
	Warning            ui.Color
	OnWarning          ui.Color
	WarningContainer   ui.Color
	OnWarningContainer ui.Color
	Success            ui.Color
	OnSuccess          ui.Color
	SuccessContainer   ui.Color
	OnSuccessContainer ui.Color

	// Seed provenance allows c.Preferences().HighContrast to dynamically
	// regenerate contrast-compliant Material roles without guessing colors.
	seed    ui.Color
	hasSeed bool
	variant tokens.Variant

	// followed records the unresolved base theme and the resolved snapshot from
	// the last WithPreferences / ThemeScope call, so repeated preference passes
	// remain idempotent and reversible while preserving manual caller edits.
	followed *followedTheme
}

type resolvedPrefs struct {
	highContrast bool
	textScale    float32
	reduceMotion bool
}

type followedTheme struct {
	from  Theme
	to    Theme
	prefs resolvedPrefs
}

// ThemeFromSeed derives an MD3E Theme from seed using Material HCT palettes.
// At most one ThemeOptions value may be passed; zero options yields a light
// TonalSpot theme using Expressive spring motion.
func ThemeFromSeed(seed ui.Color, opts ...ThemeOptions) *Theme {
	if len(opts) > 1 {
		panic("md3e: ThemeFromSeed takes at most one ThemeOptions")
	}
	var o ThemeOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if seed.A == 0 {
		seed = DefaultSeedColor
	}

	contrast := sanitizeContrast(o.Contrast)
	surfaceStyle := o.SurfaceStyle.Normalize()
	colors := cachedColorScheme(seed, o.Dark, o.Variant, contrast, surfaceStyle)

	shapes := DefaultShapeScale()
	if o.Shapes != nil {
		shapes = sanitizeShapeScale(*o.Shapes)
	}
	typo := tokens.NewTypographyScale()
	if o.Typography != nil {
		typo = sanitizeTypographyScale(*o.Typography)
	}

	scale := sanitizePositiveScale(o.TextScale, 1)

	t := &Theme{
		Dark:         o.Dark,
		Colors:       colors,
		SurfaceStyle: surfaceStyle,
		Motion:       o.Motion,
		Density:      o.Density.Normalize(),
		Typography:   typo,
		Shapes:       shapes,
		Contrast:     contrast,
		TextScale:    scale,
		ReduceMotion: o.ReduceMotion,
		Font:         o.Font,
		MonoFont:     o.MonoFont,
		seed:         seed,
		hasSeed:      true,
		variant:      o.Variant,
	}
	applySemanticRoles(t)
	return t
}

func applySemanticRoles(t *Theme) {
	if t.Dark {
		t.Warning = ui.Hex("#ffb951")
		t.OnWarning = ui.Hex("#472a00")
		t.WarningContainer = ui.Hex("#653e00")
		t.OnWarningContainer = ui.Hex("#ffddb3")
		t.Success = ui.Hex("#7adc77")
		t.OnSuccess = ui.Hex("#00390a")
		t.SuccessContainer = ui.Hex("#005313")
		t.OnSuccessContainer = ui.Hex("#95f990")
		if t.Contrast >= tokens.ContrastHigh {
			t.Warning = ui.Hex("#ffddb3")
			t.OnWarning = ui.Hex("#261400")
			t.WarningContainer = ui.Hex("#583500")
			t.OnWarningContainer = ui.Hex("#fff1df")
			t.Success = ui.Hex("#b8ffb0")
			t.OnSuccess = ui.Hex("#002204")
			t.SuccessContainer = ui.Hex("#004b10")
			t.OnSuccessContainer = ui.Hex("#d4ffd0")
		}
		return
	}
	t.Warning = ui.Hex("#855400")
	t.OnWarning = ui.Hex("#ffffff")
	t.WarningContainer = ui.Hex("#ffddb3")
	t.OnWarningContainer = ui.Hex("#2a1800")
	t.Success = ui.Hex("#146c2e")
	t.OnSuccess = ui.Hex("#ffffff")
	t.SuccessContainer = ui.Hex("#b6f2b3")
	t.OnSuccessContainer = ui.Hex("#002107")
	if t.Contrast >= tokens.ContrastHigh {
		t.Warning = ui.Hex("#543300")
		t.WarningContainer = ui.Hex("#ffd099")
		t.Success = ui.Hex("#004717")
		t.SuccessContainer = ui.Hex("#90ec8d")
	}
}

// Clone returns a shallow value copy of t so callers can customize fields
// safely without mutating a shared or cached theme.
func (t *Theme) Clone() *Theme {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}

// EffectiveTextScale returns a positive text scale multiplier (1.0 when unset).
func (t *Theme) EffectiveTextScale() float32 {
	if t == nil {
		return 1
	}
	return sanitizePositiveScale(t.TextScale, 1)
}

// Animation resolves token against t.Motion while respecting t.ReduceMotion.
func (t *Theme) Animation(token motion.Token) ui.Animation {
	if t == nil {
		return motion.Resolve(motion.Expressive, token, false)
	}
	return motion.Resolve(t.Motion, token, t.ReduceMotion)
}

// TypeStyle returns the MD3E type style for role (standard or emphasized).
func (t *Theme) TypeStyle(role tokens.TypeRole, emphasized ...bool) tokens.TypeStyle {
	if len(emphasized) > 1 {
		panic("md3e: Theme.TypeStyle takes at most one emphasized bool")
	}
	emph := len(emphasized) == 1 && emphasized[0]
	if t == nil {
		scale := tokens.NewTypographyScale()
		return scale.Style(role, emph)
	}
	typo := t.Typography
	if typo == (tokens.TypographyScale{}) {
		typo = tokens.NewTypographyScale()
	}
	return typo.Style(role, emph)
}

// ContentColorFor returns the pairing On* foreground color for common theme
// background roles, falling back to high-contrast light/dark ink by luminance.
func (t *Theme) ContentColorFor(bg ui.Color) ui.Color {
	if t == nil {
		if relativeLuminance(bg) > 0.35 {
			return ui.Hex("#1d1b20")
		}
		return ui.Hex("#f5eff7")
	}
	cs := t.Colors
	switch bg {
	case cs.Primary:
		return cs.OnPrimary
	case cs.PrimaryContainer:
		return cs.OnPrimaryContainer
	case cs.PrimaryFixed, cs.PrimaryFixedDim:
		return cs.OnPrimaryFixed
	case cs.Secondary:
		return cs.OnSecondary
	case cs.SecondaryContainer:
		return cs.OnSecondaryContainer
	case cs.SecondaryFixed, cs.SecondaryFixedDim:
		return cs.OnSecondaryFixed
	case cs.Tertiary:
		return cs.OnTertiary
	case cs.TertiaryContainer:
		return cs.OnTertiaryContainer
	case cs.TertiaryFixed, cs.TertiaryFixedDim:
		return cs.OnTertiaryFixed
	case cs.Error:
		return cs.OnError
	case cs.ErrorContainer:
		return cs.OnErrorContainer
	case cs.InverseSurface:
		return cs.InverseOnSurface
	case cs.SurfaceVariant:
		return cs.OnSurfaceVariant
	case cs.Background:
		return cs.OnBackground
	case cs.Surface, cs.SurfaceDim, cs.SurfaceBright,
		cs.SurfaceContainerLowest, cs.SurfaceContainerLow,
		cs.SurfaceContainer, cs.SurfaceContainerHigh, cs.SurfaceContainerHighest:
		return cs.OnSurface
	case t.Warning:
		return t.OnWarning
	case t.WarningContainer:
		return t.OnWarningContainer
	case t.Success:
		return t.OnSuccess
	case t.SuccessContainer:
		return t.OnSuccessContainer
	default:
		if bg.A == 0 {
			return cs.OnSurface
		}
		if relativeLuminance(bg) > 0.35 {
			if relativeLuminance(cs.OnSurface) < 0.25 {
				return cs.OnSurface
			}
			return ui.Hex("#1d1b20")
		}
		if relativeLuminance(cs.OnSurface) > 0.65 {
			return cs.OnSurface
		}
		return ui.Hex("#ffffff")
	}
}

func relativeLuminance(c ui.Color) float64 {
	lin := func(u uint8) float64 {
		v := float64(u) / 255.0
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// WithPreferences returns a resolved Theme incorporating desktop accessibility
// preferences (HighContrast, TextScale, ReduceMotion). If t was built via
// ThemeFromSeed and HighContrast is active, the color scheme is regenerated at
// tokens.ContrastHigh cleanly; manual themes strengthen outline visibility.
func (t *Theme) WithPreferences(p ui.Preferences) *Theme {
	if t == nil {
		return DefaultTheme(nil).WithPreferences(p)
	}
	resolved := t.resolvedValue(p)
	return &resolved
}

func sanitizePositiveF32(v, fallback float32) float32 {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) || !(v > 0) {
		return fallback
	}
	return v
}

func isPositiveFiniteF32(v float32) bool {
	f := float64(v)
	return !math.IsNaN(f) && !math.IsInf(f, 0) && v > 0
}

func sanitizePositiveScale(v, fallback float32) float32 {
	return sanitizePositiveF32(v, fallback)
}

func sanitizeNonNegF32(v float32) float32 {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) || v <= 0 {
		return 0
	}
	return v
}

func sanitizeContrast(c tokens.Contrast) tokens.Contrast {
	f := float64(c)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return tokens.ContrastStandard
	}
	if c == 0 {
		return 0
	}
	return c
}

func sanitizeShapeScale(s ShapeScale) ShapeScale {
	if s == (ShapeScale{}) {
		return DefaultShapeScale()
	}
	return ShapeScale{
		None:                sanitizeNonNegF32(s.None),
		ExtraSmall:          sanitizeNonNegF32(s.ExtraSmall),
		Small:               sanitizeNonNegF32(s.Small),
		Medium:              sanitizeNonNegF32(s.Medium),
		Large:               sanitizeNonNegF32(s.Large),
		LargeIncreased:      sanitizeNonNegF32(s.LargeIncreased),
		ExtraLarge:          sanitizeNonNegF32(s.ExtraLarge),
		ExtraLargeIncreased: sanitizeNonNegF32(s.ExtraLargeIncreased),
		ExtraExtraLarge:     sanitizeNonNegF32(s.ExtraExtraLarge),
		Full:                sanitizeNonNegF32(s.Full),
	}
}

func sanitizeTypeStyle(s, fallback tokens.TypeStyle) tokens.TypeStyle {
	if math.IsNaN(s.Size) || math.IsInf(s.Size, 0) || !(s.Size >= 0) {
		s.Size = fallback.Size
	} else if s.Size == 0 {
		s.Size = 0
	}
	if math.IsNaN(s.LineHeight) || math.IsInf(s.LineHeight, 0) || !(s.LineHeight >= 0) {
		s.LineHeight = fallback.LineHeight
	} else if s.LineHeight == 0 {
		s.LineHeight = 0
	}
	if math.IsNaN(s.Tracking) || math.IsInf(s.Tracking, 0) {
		s.Tracking = fallback.Tracking
	} else if s.Tracking == 0 {
		s.Tracking = 0
	}
	return s
}

func sanitizeTypographyScale(ts tokens.TypographyScale) tokens.TypographyScale {
	if ts == (tokens.TypographyScale{}) {
		return tokens.NewTypographyScale()
	}
	def := tokens.NewTypographyScale()
	for i := range ts.Standard {
		ts.Standard[i] = sanitizeTypeStyle(ts.Standard[i], def.Standard[i])
		ts.Emphasized[i] = sanitizeTypeStyle(ts.Emphasized[i], def.Emphasized[i])
	}
	return ts
}

// sanitizeTheme returns a comparable value copy of t free of NaN/Inf floats and
// with followed cleared so it can safely serve as a map key.
func sanitizeTheme(t Theme) Theme {
	t.followed = nil
	t.SurfaceStyle = t.SurfaceStyle.Normalize()
	t.Density = t.Density.Normalize()
	t.Typography = sanitizeTypographyScale(t.Typography)
	t.Shapes = sanitizeShapeScale(t.Shapes)
	t.Contrast = sanitizeContrast(t.Contrast)
	t.TextScale = sanitizePositiveScale(t.TextScale, 1)
	if t.Warning.A == 0 || t.Success.A == 0 {
		applySemanticRoles(&t)
	}
	return t
}

func (t *Theme) rebaseFromFollowed() Theme {
	cur := sanitizeTheme(*t)
	if t.followed == nil {
		return cur
	}
	from := t.followed.from
	to := t.followed.to
	if cur == to {
		return from
	}
	base := from
	if cur.Dark != to.Dark {
		base.Dark = cur.Dark
		base.hasSeed = false
	}
	if cur.Colors != to.Colors {
		base.Colors = cur.Colors
		base.hasSeed = false
	}
	if cur.SurfaceStyle != to.SurfaceStyle {
		base.SurfaceStyle = cur.SurfaceStyle
	}
	if cur.Motion != to.Motion {
		base.Motion = cur.Motion
	}
	if cur.Density != to.Density {
		base.Density = cur.Density
	}
	if cur.Typography != to.Typography {
		base.Typography = cur.Typography
	}
	if cur.Shapes != to.Shapes {
		base.Shapes = cur.Shapes
	}
	if cur.Contrast != to.Contrast {
		base.Contrast = cur.Contrast
	}
	if cur.TextScale != to.TextScale {
		base.TextScale = cur.TextScale
	}
	if cur.ReduceMotion != to.ReduceMotion {
		base.ReduceMotion = cur.ReduceMotion
	}
	if cur.Font != to.Font {
		base.Font = cur.Font
	}
	if cur.MonoFont != to.MonoFont {
		base.MonoFont = cur.MonoFont
	}
	if cur.Warning != to.Warning {
		base.Warning = cur.Warning
	}
	if cur.OnWarning != to.OnWarning {
		base.OnWarning = cur.OnWarning
	}
	if cur.WarningContainer != to.WarningContainer {
		base.WarningContainer = cur.WarningContainer
	}
	if cur.OnWarningContainer != to.OnWarningContainer {
		base.OnWarningContainer = cur.OnWarningContainer
	}
	if cur.Success != to.Success {
		base.Success = cur.Success
	}
	if cur.OnSuccess != to.OnSuccess {
		base.OnSuccess = cur.OnSuccess
	}
	if cur.SuccessContainer != to.SuccessContainer {
		base.SuccessContainer = cur.SuccessContainer
	}
	if cur.OnSuccessContainer != to.OnSuccessContainer {
		base.OnSuccessContainer = cur.OnSuccessContainer
	}
	return base
}

func (t *Theme) resolvedValue(p ui.Preferences) Theme {
	np := resolvedPrefs{
		highContrast: p.HighContrast,
		textScale:    sanitizePositiveScale(p.TextScale, 1),
		reduceMotion: p.ReduceMotion,
	}
	if t.followed != nil && t.followed.prefs == np {
		cur := sanitizeTheme(*t)
		if cur == t.followed.to {
			cur.followed = t.followed
			return cur
		}
	}

	orig := t.rebaseFromFollowed()
	cp := orig
	if np.highContrast && cp.Contrast < tokens.ContrastHigh {
		cp.Contrast = tokens.ContrastHigh
		if cp.hasSeed {
			cp.Colors = cachedColorScheme(cp.seed, cp.Dark, cp.variant, tokens.ContrastHigh, cp.SurfaceStyle)
			applySemanticRoles(&cp)
		} else {
			// Manual theme without a seed: raise outline legibility while keeping custom palette intact.
			if cp.Colors.Outline.A > 0 && cp.Colors.OnSurface.A > 0 {
				cp.Colors.Outline = cp.Colors.Outline.Mix(cp.Colors.OnSurface, 0.5)
				cp.Colors.OutlineVariant = cp.Colors.OutlineVariant.Mix(cp.Colors.OnSurfaceVariant, 0.5)
			}
			applySemanticRoles(&cp)
		}
	}

	baseScale := sanitizePositiveScale(orig.TextScale, 1)
	cp.TextScale = sanitizePositiveScale(baseScale*np.textScale, 1)
	cp.ReduceMotion = orig.ReduceMotion || np.reduceMotion

	toSnap := sanitizeTheme(cp)
	cp = toSnap
	cp.followed = &followedTheme{
		from:  orig,
		to:    toSnap,
		prefs: np,
	}
	return cp
}

// ToUITheme bridges t's Material Design 3 Expressive color roles, metrics, and
// accessibility settings onto a freshly allocated native *ui.Theme so built-in
// MyGo controls, text defaults, and overlays share a cohesive palette.
func (t *Theme) ToUITheme() *ui.Theme {
	if t == nil {
		return DefaultTheme(nil).ToUITheme()
	}
	uiT := t.buildUITheme()
	return &uiT
}

// UITheme returns the native *ui.Theme bridged from t (alias of ToUITheme).
func (t *Theme) UITheme() *ui.Theme {
	return t.ToUITheme()
}

func (t *Theme) buildUITheme() ui.Theme {
	cs := t.Colors
	surface := cs.SurfaceContainer
	if surface.A == 0 {
		surface = cs.Surface
	}
	primary := cs.Primary
	onPrimary := cs.OnPrimary
	onSurface := cs.OnSurface

	border := cs.OutlineVariant
	if t.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
		border = cs.Outline
	}
	focus := cs.Secondary
	if focus.A == 0 {
		focus = primary
	}
	focusWidth := float32(2)
	if t.Contrast >= tokens.ContrastHigh {
		focusWidth = 3
	}
	scrollAlpha := float32(0.32)
	if t.Contrast >= tokens.ContrastHigh {
		scrollAlpha = 0.55
	}
	radius := t.Shapes.Radius(tokens.ShapeSmall)
	if radius <= 0 {
		radius = float32(tokens.ShapeSmall.Radius())
	}

	warning := t.Warning
	if warning.A == 0 {
		if t.Dark {
			warning = ui.Hex("#ffb951")
		} else {
			warning = ui.Hex("#855400")
		}
	}
	success := t.Success
	if success.A == 0 {
		if t.Dark {
			success = ui.Hex("#7adc77")
		} else {
			success = ui.Hex("#146c2e")
		}
	}

	return ui.Theme{
		Dark:            t.Dark,
		Background:      cs.Surface,
		Surface:         surface,
		SurfaceHover:    onSurface.Alpha(tokens.StateHoverOpacity).Over(surface),
		SurfacePressed:  onSurface.Alpha(tokens.StatePressedOpacity).Over(surface),
		Border:          border,
		Text:            onSurface,
		TextMuted:       cs.OnSurfaceVariant,
		Accent:          primary,
		AccentHover:     onPrimary.Alpha(tokens.StateHoverOpacity).Over(primary),
		AccentPressed:   onPrimary.Alpha(tokens.StatePressedOpacity).Over(primary),
		AccentText:      onPrimary,
		InactiveAccent:  cs.SecondaryContainer,
		Danger:          cs.Error,
		Warning:         warning,
		Success:         success,
		Selection:       primary.Alpha(0.24),
		Focus:           focus,
		FocusRingWidth:  focusWidth,
		FocusRingOffset: 2,
		Inverse:         cs.InverseSurface,
		InverseText:     cs.InverseOnSurface,
		Scrollbar:       cs.OnSurfaceVariant.Alpha(scrollAlpha),
		ScrollbarWidth:  6,
		Radius:          radius,
		Spacing:         float32(t.Density.SpacingUnit()),
		FontSize:        14 * t.EffectiveTextScale(),
		Font:            t.Font,
	}
}

// Weak-reference theme registries prevent memory leaks across multiple windows
// and short-lived ui.Tester runs while preserving pointer stability across rebuilds.
type scopedUIKey struct {
	wp weak.Pointer[ui.Theme]
	id uint64
}

// scopedThemeEntry tags a registry entry with its registration id, so cleanup
// removes exactly the entry it registered even if the address is reused.
type scopedThemeEntry struct {
	theme *Theme
	spec  Theme
	wp    weak.Pointer[ui.Theme]
	id    uint64
}

var (
	themeRegMu     sync.RWMutex
	themeByUI      = map[weak.Pointer[ui.Theme]]scopedThemeEntry{}
	themeBySpec    = map[Theme]scopedThemeEntry{}
	themeSeq       uint64
	defaultThemeMu sync.RWMutex
	defaultThemes  = map[defaultThemeKey]*Theme{}

	stockLightAccent = ui.LightTheme().Accent
	stockDarkAccent  = ui.DarkTheme().Accent
)

type schemeCacheKey struct {
	seed         ui.Color
	dark         bool
	variant      tokens.Variant
	contrast     tokens.Contrast
	surfaceStyle tokens.SurfaceStyle
}

type defaultThemeKey struct {
	seed         ui.Color
	dark         bool
	variant      tokens.Variant
	contrast     tokens.Contrast
	surfaceStyle tokens.SurfaceStyle
	textScale    float32
	reduceMotion bool
}

const (
	maxDefaultThemeCache = 32
	maxSchemeCache       = 64
)

var (
	schemeCacheMu      sync.RWMutex
	schemeCache        = map[schemeCacheKey]tokens.ColorScheme{}
	schemeComputeCount atomic.Uint64
)

func cachedColorScheme(seed ui.Color, dark bool, variant tokens.Variant, contrast tokens.Contrast, surfaceStyle tokens.SurfaceStyle) tokens.ColorScheme {
	contrast = sanitizeContrast(contrast)
	surfaceStyle = surfaceStyle.Normalize()
	key := schemeCacheKey{
		seed:         seed,
		dark:         dark,
		variant:      variant,
		contrast:     contrast,
		surfaceStyle: surfaceStyle,
	}
	schemeCacheMu.RLock()
	if cs, ok := schemeCache[key]; ok {
		schemeCacheMu.RUnlock()
		return cs
	}
	schemeCacheMu.RUnlock()

	schemeComputeCount.Add(1)
	cs := tokens.NewDynamicScheme(tokens.DynamicSchemeOptions{
		Seed:         seed,
		Dark:         dark,
		Variant:      variant,
		Contrast:     contrast,
		SurfaceStyle: surfaceStyle,
	}).Colors

	schemeCacheMu.Lock()
	defer schemeCacheMu.Unlock()
	if cs, ok := schemeCache[key]; ok {
		return cs
	}
	if len(schemeCache) >= maxSchemeCache {
		for k := range schemeCache {
			delete(schemeCache, k)
			if len(schemeCache) < maxSchemeCache/2 {
				break
			}
		}
	}
	schemeCache[key] = cs
	return cs
}

func registerScopedTheme(resolved Theme, tracked *Theme) *ui.Theme {
	spec := sanitizeTheme(resolved)

	themeRegMu.Lock()
	defer themeRegMu.Unlock()

	if e, ok := themeBySpec[spec]; ok {
		if existing := e.wp.Value(); existing != nil {
			if cached, ok := themeByUI[e.wp]; ok && cached.id == e.id && cached.theme != nil {
				return existing
			}
		}
		delete(themeBySpec, spec)
	}

	uiVal := spec.buildUITheme()
	uiPtr := &uiVal
	wp := weak.Make(uiPtr)
	stored := spec
	if tracked != nil {
		stored = *tracked
	}
	themeSeq++
	id := themeSeq
	themeBySpec[spec] = scopedThemeEntry{wp: wp, id: id}
	themeByUI[wp] = scopedThemeEntry{theme: &stored, spec: spec, id: id}

	// All float fields in spec are sanitized (finite and non-NaN), so O(1)
	// direct key deletion with generation checks cleanly reclaims stale entries.
	runtime.AddCleanup(uiPtr, func(k scopedUIKey) {
		themeRegMu.Lock()
		defer themeRegMu.Unlock()
		if cur, ok := themeByUI[k.wp]; ok && cur.id == k.id {
			delete(themeByUI, k.wp)
			if live, ok := themeBySpec[cur.spec]; ok && live.id == k.id {
				delete(themeBySpec, cur.spec)
			}
		}
	}, scopedUIKey{wp: wp, id: id})

	return uiPtr
}

func lookupScopedTheme(uiT *ui.Theme) (*Theme, bool) {
	if uiT == nil {
		return nil, false
	}
	wp := weak.Make(uiT)
	themeRegMu.RLock()
	e, ok := themeByUI[wp]
	themeRegMu.RUnlock()
	if !ok || e.theme == nil {
		return nil, false
	}
	return e.theme, true
}

func registerThemeAlias(uiT *ui.Theme, th *Theme) {
	if uiT == nil || th == nil {
		return
	}
	wp := weak.Make(uiT)
	themeRegMu.RLock()
	if e, ok := themeByUI[wp]; ok && e.theme == th {
		themeRegMu.RUnlock()
		return
	}
	themeRegMu.RUnlock()

	themeRegMu.Lock()
	defer themeRegMu.Unlock()
	if e, ok := themeByUI[wp]; ok {
		if e.theme == th {
			return
		}
		e.theme = th
		themeByUI[wp] = e
		return
	}
	themeSeq++
	id := themeSeq
	themeByUI[wp] = scopedThemeEntry{theme: th, id: id}
	runtime.AddCleanup(uiT, func(k scopedUIKey) {
		themeRegMu.Lock()
		defer themeRegMu.Unlock()
		if cur, ok := themeByUI[k.wp]; ok && cur.id == k.id {
			delete(themeByUI, k.wp)
		}
	}, scopedUIKey{wp: wp, id: id})
}

func cachedThemeFromSeed(seed ui.Color, dark bool, variant tokens.Variant, prefs ui.Preferences) *Theme {
	if seed.A == 0 {
		seed = DefaultSeedColor
	}
	contrast := tokens.ContrastStandard
	if prefs.HighContrast {
		contrast = tokens.ContrastHigh
	}
	textScale := sanitizePositiveScale(prefs.TextScale, 1)
	key := defaultThemeKey{
		seed:         seed,
		dark:         dark,
		variant:      variant,
		contrast:     contrast,
		textScale:    textScale,
		reduceMotion: prefs.ReduceMotion,
	}

	defaultThemeMu.RLock()
	if hit, ok := defaultThemes[key]; ok {
		defaultThemeMu.RUnlock()
		return hit
	}
	defaultThemeMu.RUnlock()

	base := ThemeFromSeed(seed, ThemeOptions{
		Dark:    dark,
		Variant: variant,
	})
	resolved := base.resolvedValue(prefs)
	built := &resolved

	defaultThemeMu.Lock()
	defer defaultThemeMu.Unlock()
	if hit, ok := defaultThemes[key]; ok {
		return hit
	}
	if len(defaultThemes) >= maxDefaultThemeCache {
		for k := range defaultThemes {
			delete(defaultThemes, k)
			if len(defaultThemes) < maxDefaultThemeCache/2 {
				break
			}
		}
	}
	defaultThemes[key] = built
	return built
}

// DefaultTheme derives and caches the MD3E Theme matching c's Dark appearance
// and OS/window Preferences (Accent, HighContrast, TextScale, ReduceMotion).
// Passing nil returns a standard light baseline theme.
func DefaultTheme(c *ui.Context) *Theme {
	if c == nil {
		return cachedThemeFromSeed(DefaultSeedColor, false, tokens.TonalSpot, ui.Preferences{TextScale: 1})
	}
	prefs := c.Preferences()
	seed := prefs.Accent
	if seed.A == 0 {
		seed = DefaultSeedColor
	}
	dark := false
	if uiT := c.Theme(); uiT != nil {
		dark = uiT.Dark
	}
	return cachedThemeFromSeed(seed, dark, tokens.TonalSpot, prefs)
}

// ThemeScope establishes t (resolved against c.Preferences()) as the active
// MD3E theme for content, simultaneously bridging into ui.Themed so native MyGo
// controls, nested scopes, and overlays launched inside content inherit the
// exact same palette, typography metrics, and accessibility preferences.
//
// A ThemeScope takes the height its content needs, as a ui.Themed column
// does. When it wraps a whole window and its content fills or scrolls the
// window (a ui.Column with Fill(), a ui.Scroll with Grow(1)), chain Fill()
// on the returned element so those sizes resolve against the window rather
// than growing without bound:
//
//	md3e.ThemeScope(c, th, func() {
//		ui.Column(c).Fill().Children(app)
//	}).Fill()
func ThemeScope(c *ui.Context, t *Theme, content func()) ui.Element {
	if t == nil {
		t = DefaultTheme(c)
	}
	resolved := t.resolvedValue(c.Preferences())
	uiT := registerScopedTheme(resolved, &resolved)

	el := ui.Themed(c, uiT)
	if content == nil {
		return el
	}
	return el.Children(content)
}

// CurrentTheme returns the active *Theme for c. Inside ThemeScope or an overlay
// (ui.Overlay, PopoverBase, DialogBase, MenuBase) launched from a ThemeScope, it
// retrieves that scope's resolved theme; if a nested ui.Themed overrides c.Theme(),
// it adapts automatically to Dark/Accent changes; otherwise it falls back to
// DefaultTheme(c).
func CurrentTheme(c *ui.Context) *Theme {
	if c == nil {
		return DefaultTheme(nil)
	}
	uiT := c.Theme()
	if scoped, ok := lookupScopedTheme(uiT); ok {
		return scoped
	}
	prefs := c.Preferences()
	if uiT != nil && prefs.Accent.A == 0 && uiT.Accent.A > 0 && uiT.Accent != stockLightAccent && uiT.Accent != stockDarkAccent {
		return cachedThemeFromSeed(uiT.Accent, uiT.Dark, tokens.TonalSpot, prefs)
	}
	return DefaultTheme(c)
}

func clampLerpT(t float32) float32 {
	if math.IsNaN(float64(t)) || t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t
}

func mixRoleColor(a, b ui.Color, t float32) ui.Color {
	if a.A == 0 && b.A == 0 {
		return ui.Color{}
	}
	if a.A == 0 {
		a = b
	} else if b.A == 0 {
		b = a
	}
	m := a.Mix(b, t)
	if m.A == 0 {
		m.A = 1
	}
	return m
}

func contrastRatio(a, b ui.Color) float64 {
	la := relativeLuminance(a)
	lb := relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func minContrastAcross(fg ui.Color, bgs ...ui.Color) float64 {
	best := math.Inf(1)
	for _, bg := range bgs {
		if bg.A == 0 {
			continue
		}
		if r := contrastRatio(fg, bg); r < best {
			best = r
		}
	}
	if math.IsInf(best, 1) {
		return 21
	}
	return best
}

func targetPairContrast(highContrast bool, fromFg, toFg ui.Color, fromBgs, toBgs []ui.Color) float64 {
	targetRatio := 4.5
	if highContrast {
		targetRatio = 7.0
	}
	if len(fromBgs) > 0 && fromFg.A > 0 {
		if r := minContrastAcross(fromFg, fromBgs...); r < targetRatio {
			targetRatio = r
		}
	}
	if len(toBgs) > 0 && toFg.A > 0 {
		if r := minContrastAcross(toFg, toBgs...); r < targetRatio {
			targetRatio = r
		}
	}
	return targetRatio
}

func maxBlackWhiteContrast(bgs ...ui.Color) float64 {
	black := ui.RGB(0, 0, 0)
	white := ui.RGB(255, 255, 255)
	return max(minContrastAcross(black, bgs...), minContrastAcross(white, bgs...))
}

// ensurePairContrast adjusts an interpolated foreground role when mixing crosses
// background luminance mid-transition (e.g. Light <-> Dark at t ≈ 0.5), keeping
// text and icons legible across all paired backgrounds.
func ensurePairContrast(fg, fromFg, toFg ui.Color, bgs, fromBgs, toBgs []ui.Color, highContrast bool) ui.Color {
	if fg.A == 0 {
		return fg
	}
	targetRatio := targetPairContrast(highContrast, fromFg, toFg, fromBgs, toBgs)
	if targetRatio <= 1 || minContrastAcross(fg, bgs...) >= targetRatio {
		return fg
	}

	alpha := fg.A
	bestEnd := ui.Color{}
	bestEndRatio := 0.0
	if fromFg.A > 0 {
		bestEnd = fromFg
		bestEndRatio = minContrastAcross(fromFg, bgs...)
	}
	if toFg.A > 0 {
		if r := minContrastAcross(toFg, bgs...); r > bestEndRatio {
			bestEnd = toFg
			bestEndRatio = r
		}
	}

	if bestEnd.A > 0 && bestEndRatio >= targetRatio {
		lo, hi := float32(0), float32(1)
		best := bestEnd
		for range 12 {
			mid := (lo + hi) * 0.5
			cand := fg.Mix(bestEnd, mid)
			if minContrastAcross(cand, bgs...) >= targetRatio {
				best = cand
				hi = mid
			} else {
				lo = mid
			}
		}
		best.A = alpha
		return best
	}

	start := fg
	if bestEnd.A > 0 && bestEndRatio > minContrastAcross(fg, bgs...) {
		start = bestEnd
	}
	black := ui.Color{R: 0, G: 0, B: 0, A: alpha}
	white := ui.Color{R: 255, G: 255, B: 255, A: alpha}
	extreme := black
	extremeRatio := minContrastAcross(black, bgs...)
	if wr := minContrastAcross(white, bgs...); wr > extremeRatio {
		extreme = white
		extremeRatio = wr
	}
	if extremeRatio < targetRatio {
		targetRatio = extremeRatio
	}

	lo, hi := float32(0), float32(1)
	best := extreme
	for range 12 {
		mid := (lo + hi) * 0.5
		cand := start.Mix(extreme, mid)
		if minContrastAcross(cand, bgs...) >= targetRatio {
			best = cand
			hi = mid
		} else {
			lo = mid
		}
	}
	best.A = alpha
	return best
}

// ensureSinglePairWithBg snaps (fg, bg) together to the closer endpoint when a
// high-contrast Light <-> Dark transition passes through a mid-gray where
// neither black nor white can meet targetRatio (e.g. 7:1).
func ensureSinglePairWithBg(fg, bg, fromFg, fromBg, toFg, toBg ui.Color, t float32, highContrast bool) (ui.Color, ui.Color) {
	targetRatio := targetPairContrast(highContrast, fromFg, toFg, []ui.Color{fromBg}, []ui.Color{toBg})
	if highContrast && targetRatio > 4.5 && maxBlackWhiteContrast(bg) < targetRatio {
		if t >= 0.5 {
			return toFg, toBg
		}
		return fromFg, fromBg
	}
	return ensurePairContrast(fg, fromFg, toFg, []ui.Color{bg}, []ui.Color{fromBg}, []ui.Color{toBg}, highContrast), bg
}

func lerpColorScheme(a, b tokens.ColorScheme, t float32, highContrast bool) tokens.ColorScheme {
	cs := tokens.ColorScheme{
		Primary:                 mixRoleColor(a.Primary, b.Primary, t),
		OnPrimary:               mixRoleColor(a.OnPrimary, b.OnPrimary, t),
		PrimaryContainer:        mixRoleColor(a.PrimaryContainer, b.PrimaryContainer, t),
		OnPrimaryContainer:      mixRoleColor(a.OnPrimaryContainer, b.OnPrimaryContainer, t),
		InversePrimary:          mixRoleColor(a.InversePrimary, b.InversePrimary, t),
		PrimaryFixed:            mixRoleColor(a.PrimaryFixed, b.PrimaryFixed, t),
		PrimaryFixedDim:         mixRoleColor(a.PrimaryFixedDim, b.PrimaryFixedDim, t),
		OnPrimaryFixed:          mixRoleColor(a.OnPrimaryFixed, b.OnPrimaryFixed, t),
		OnPrimaryFixedVariant:   mixRoleColor(a.OnPrimaryFixedVariant, b.OnPrimaryFixedVariant, t),
		Secondary:               mixRoleColor(a.Secondary, b.Secondary, t),
		OnSecondary:             mixRoleColor(a.OnSecondary, b.OnSecondary, t),
		SecondaryContainer:      mixRoleColor(a.SecondaryContainer, b.SecondaryContainer, t),
		OnSecondaryContainer:    mixRoleColor(a.OnSecondaryContainer, b.OnSecondaryContainer, t),
		SecondaryFixed:          mixRoleColor(a.SecondaryFixed, b.SecondaryFixed, t),
		SecondaryFixedDim:       mixRoleColor(a.SecondaryFixedDim, b.SecondaryFixedDim, t),
		OnSecondaryFixed:        mixRoleColor(a.OnSecondaryFixed, b.OnSecondaryFixed, t),
		OnSecondaryFixedVariant: mixRoleColor(a.OnSecondaryFixedVariant, b.OnSecondaryFixedVariant, t),
		Tertiary:                mixRoleColor(a.Tertiary, b.Tertiary, t),
		OnTertiary:              mixRoleColor(a.OnTertiary, b.OnTertiary, t),
		TertiaryContainer:       mixRoleColor(a.TertiaryContainer, b.TertiaryContainer, t),
		OnTertiaryContainer:     mixRoleColor(a.OnTertiaryContainer, b.OnTertiaryContainer, t),
		TertiaryFixed:           mixRoleColor(a.TertiaryFixed, b.TertiaryFixed, t),
		TertiaryFixedDim:        mixRoleColor(a.TertiaryFixedDim, b.TertiaryFixedDim, t),
		OnTertiaryFixed:         mixRoleColor(a.OnTertiaryFixed, b.OnTertiaryFixed, t),
		OnTertiaryFixedVariant:  mixRoleColor(a.OnTertiaryFixedVariant, b.OnTertiaryFixedVariant, t),
		Error:                   mixRoleColor(a.Error, b.Error, t),
		OnError:                 mixRoleColor(a.OnError, b.OnError, t),
		ErrorContainer:          mixRoleColor(a.ErrorContainer, b.ErrorContainer, t),
		OnErrorContainer:        mixRoleColor(a.OnErrorContainer, b.OnErrorContainer, t),
		Background:              mixRoleColor(a.Background, b.Background, t),
		OnBackground:            mixRoleColor(a.OnBackground, b.OnBackground, t),
		Surface:                 mixRoleColor(a.Surface, b.Surface, t),
		SurfaceDim:              mixRoleColor(a.SurfaceDim, b.SurfaceDim, t),
		SurfaceBright:           mixRoleColor(a.SurfaceBright, b.SurfaceBright, t),
		OnSurface:               mixRoleColor(a.OnSurface, b.OnSurface, t),
		OnSurfaceVariant:        mixRoleColor(a.OnSurfaceVariant, b.OnSurfaceVariant, t),
		SurfaceVariant:          mixRoleColor(a.SurfaceVariant, b.SurfaceVariant, t),
		SurfaceContainerLowest:  mixRoleColor(a.SurfaceContainerLowest, b.SurfaceContainerLowest, t),
		SurfaceContainerLow:     mixRoleColor(a.SurfaceContainerLow, b.SurfaceContainerLow, t),
		SurfaceContainer:        mixRoleColor(a.SurfaceContainer, b.SurfaceContainer, t),
		SurfaceContainerHigh:    mixRoleColor(a.SurfaceContainerHigh, b.SurfaceContainerHigh, t),
		SurfaceContainerHighest: mixRoleColor(a.SurfaceContainerHighest, b.SurfaceContainerHighest, t),
		SurfaceTint:             mixRoleColor(a.SurfaceTint, b.SurfaceTint, t),
		Outline:                 mixRoleColor(a.Outline, b.Outline, t),
		OutlineVariant:          mixRoleColor(a.OutlineVariant, b.OutlineVariant, t),
		InverseSurface:          mixRoleColor(a.InverseSurface, b.InverseSurface, t),
		InverseOnSurface:        mixRoleColor(a.InverseOnSurface, b.InverseOnSurface, t),
		Scrim:                   mixRoleColor(a.Scrim, b.Scrim, t),
		Shadow:                  mixRoleColor(a.Shadow, b.Shadow, t),
	}

	cs.OnPrimary, cs.Primary = ensureSinglePairWithBg(cs.OnPrimary, cs.Primary, a.OnPrimary, a.Primary, b.OnPrimary, b.Primary, t, highContrast)
	cs.OnPrimaryContainer, cs.PrimaryContainer = ensureSinglePairWithBg(cs.OnPrimaryContainer, cs.PrimaryContainer, a.OnPrimaryContainer, a.PrimaryContainer, b.OnPrimaryContainer, b.PrimaryContainer, t, highContrast)
	cs.OnPrimaryFixed = ensurePairContrast(cs.OnPrimaryFixed, a.OnPrimaryFixed, b.OnPrimaryFixed,
		[]ui.Color{cs.PrimaryFixed, cs.PrimaryFixedDim}, []ui.Color{a.PrimaryFixed, a.PrimaryFixedDim}, []ui.Color{b.PrimaryFixed, b.PrimaryFixedDim}, highContrast)
	cs.OnPrimaryFixedVariant = ensurePairContrast(cs.OnPrimaryFixedVariant, a.OnPrimaryFixedVariant, b.OnPrimaryFixedVariant,
		[]ui.Color{cs.PrimaryFixed, cs.PrimaryFixedDim}, []ui.Color{a.PrimaryFixed, a.PrimaryFixedDim}, []ui.Color{b.PrimaryFixed, b.PrimaryFixedDim}, highContrast)

	cs.OnSecondary, cs.Secondary = ensureSinglePairWithBg(cs.OnSecondary, cs.Secondary, a.OnSecondary, a.Secondary, b.OnSecondary, b.Secondary, t, highContrast)
	cs.OnSecondaryContainer, cs.SecondaryContainer = ensureSinglePairWithBg(cs.OnSecondaryContainer, cs.SecondaryContainer, a.OnSecondaryContainer, a.SecondaryContainer, b.OnSecondaryContainer, b.SecondaryContainer, t, highContrast)
	cs.OnSecondaryFixed = ensurePairContrast(cs.OnSecondaryFixed, a.OnSecondaryFixed, b.OnSecondaryFixed,
		[]ui.Color{cs.SecondaryFixed, cs.SecondaryFixedDim}, []ui.Color{a.SecondaryFixed, a.SecondaryFixedDim}, []ui.Color{b.SecondaryFixed, b.SecondaryFixedDim}, highContrast)
	cs.OnSecondaryFixedVariant = ensurePairContrast(cs.OnSecondaryFixedVariant, a.OnSecondaryFixedVariant, b.OnSecondaryFixedVariant,
		[]ui.Color{cs.SecondaryFixed, cs.SecondaryFixedDim}, []ui.Color{a.SecondaryFixed, a.SecondaryFixedDim}, []ui.Color{b.SecondaryFixed, b.SecondaryFixedDim}, highContrast)

	cs.OnTertiary, cs.Tertiary = ensureSinglePairWithBg(cs.OnTertiary, cs.Tertiary, a.OnTertiary, a.Tertiary, b.OnTertiary, b.Tertiary, t, highContrast)
	cs.OnTertiaryContainer, cs.TertiaryContainer = ensureSinglePairWithBg(cs.OnTertiaryContainer, cs.TertiaryContainer, a.OnTertiaryContainer, a.TertiaryContainer, b.OnTertiaryContainer, b.TertiaryContainer, t, highContrast)
	cs.OnTertiaryFixed = ensurePairContrast(cs.OnTertiaryFixed, a.OnTertiaryFixed, b.OnTertiaryFixed,
		[]ui.Color{cs.TertiaryFixed, cs.TertiaryFixedDim}, []ui.Color{a.TertiaryFixed, a.TertiaryFixedDim}, []ui.Color{b.TertiaryFixed, b.TertiaryFixedDim}, highContrast)
	cs.OnTertiaryFixedVariant = ensurePairContrast(cs.OnTertiaryFixedVariant, a.OnTertiaryFixedVariant, b.OnTertiaryFixedVariant,
		[]ui.Color{cs.TertiaryFixed, cs.TertiaryFixedDim}, []ui.Color{a.TertiaryFixed, a.TertiaryFixedDim}, []ui.Color{b.TertiaryFixed, b.TertiaryFixedDim}, highContrast)

	cs.OnError, cs.Error = ensureSinglePairWithBg(cs.OnError, cs.Error, a.OnError, a.Error, b.OnError, b.Error, t, highContrast)
	cs.OnErrorContainer, cs.ErrorContainer = ensureSinglePairWithBg(cs.OnErrorContainer, cs.ErrorContainer, a.OnErrorContainer, a.ErrorContainer, b.OnErrorContainer, b.ErrorContainer, t, highContrast)

	surfaces := []ui.Color{
		cs.Surface, cs.SurfaceContainerLowest, cs.SurfaceContainerLow,
		cs.SurfaceContainer, cs.SurfaceContainerHigh, cs.SurfaceContainerHighest,
	}
	aSurfaces := []ui.Color{
		a.Surface, a.SurfaceContainerLowest, a.SurfaceContainerLow,
		a.SurfaceContainer, a.SurfaceContainerHigh, a.SurfaceContainerHighest,
	}
	bSurfaces := []ui.Color{
		b.Surface, b.SurfaceContainerLowest, b.SurfaceContainerLow,
		b.SurfaceContainer, b.SurfaceContainerHigh, b.SurfaceContainerHighest,
	}
	surfTarget := targetPairContrast(highContrast, a.OnSurface, b.OnSurface, aSurfaces, bSurfaces)
	if highContrast && surfTarget > 4.5 && maxBlackWhiteContrast(surfaces...) < surfTarget {
		src := a
		if t >= 0.5 {
			src = b
		}
		cs.Background = src.Background
		cs.OnBackground = src.OnBackground
		cs.Surface = src.Surface
		cs.SurfaceDim = src.SurfaceDim
		cs.SurfaceBright = src.SurfaceBright
		cs.SurfaceVariant = src.SurfaceVariant
		cs.SurfaceContainerLowest = src.SurfaceContainerLowest
		cs.SurfaceContainerLow = src.SurfaceContainerLow
		cs.SurfaceContainer = src.SurfaceContainer
		cs.SurfaceContainerHigh = src.SurfaceContainerHigh
		cs.SurfaceContainerHighest = src.SurfaceContainerHighest
		cs.OnSurface = src.OnSurface
		cs.OnSurfaceVariant = src.OnSurfaceVariant
		cs.Outline = src.Outline
		cs.OutlineVariant = src.OutlineVariant
	} else {
		cs.OnBackground = ensurePairContrast(cs.OnBackground, a.OnBackground, b.OnBackground,
			[]ui.Color{cs.Background}, []ui.Color{a.Background}, []ui.Color{b.Background}, highContrast)
		cs.OnSurface = ensurePairContrast(cs.OnSurface, a.OnSurface, b.OnSurface,
			surfaces, aSurfaces, bSurfaces, highContrast)
		cs.OnSurfaceVariant = ensurePairContrast(cs.OnSurfaceVariant, a.OnSurfaceVariant, b.OnSurfaceVariant,
			surfaces, aSurfaces, bSurfaces, highContrast)
	}
	cs.InverseOnSurface, cs.InverseSurface = ensureSinglePairWithBg(
		cs.InverseOnSurface, cs.InverseSurface,
		a.InverseOnSurface, a.InverseSurface,
		b.InverseOnSurface, b.InverseSurface,
		t, highContrast,
	)

	return cs
}

// LerpTheme interpolates between from and to at fraction t (clamped to [0, 1],
// with NaN treated as 0). All 49 ColorScheme roles and extended Warning/Success
// semantic roles are smoothly mixed while preserving usable foreground contrast
// and non-transparent colors across mid-transition luminance crossings.
// TextScale interpolates linearly, while discrete fields (Dark, SurfaceStyle,
// Motion, Density, Shapes, Typography, Contrast, ReduceMotion, Font, MonoFont)
// switch at t >= 0.5. Intermediate themes clear seed provenance so they never
// pollute schemeCache.
func LerpTheme(from, to *Theme, t float32) *Theme {
	if from == nil {
		from = DefaultTheme(nil)
	}
	if to == nil {
		to = DefaultTheme(nil)
	}
	t = clampLerpT(t)

	sa := sanitizeTheme(*from)
	sb := sanitizeTheme(*to)

	if t == 0 {
		out := sa
		if from.followed != nil {
			f := *from.followed
			out.followed = &f
		}
		return &out
	}
	if t == 1 {
		out := sb
		if to.followed != nil {
			f := *to.followed
			out.followed = &f
		}
		return &out
	}

	out := sa
	if t >= 0.5 {
		out.Dark = sb.Dark
		out.SurfaceStyle = sb.SurfaceStyle
		out.Motion = sb.Motion
		out.Density = sb.Density
		out.Typography = sb.Typography
		out.Shapes = sb.Shapes
		out.Contrast = sb.Contrast
		out.ReduceMotion = sb.ReduceMotion
		out.Font = sb.Font
		out.MonoFont = sb.MonoFont
	}
	highContrast := sa.Contrast >= tokens.ContrastHigh && sb.Contrast >= tokens.ContrastHigh
	out.TextScale = sanitizePositiveScale(sa.TextScale+(sb.TextScale-sa.TextScale)*t, 1)
	out.Colors = lerpColorScheme(sa.Colors, sb.Colors, t, highContrast)

	out.Warning = mixRoleColor(sa.Warning, sb.Warning, t)
	out.OnWarning, out.Warning = ensureSinglePairWithBg(mixRoleColor(sa.OnWarning, sb.OnWarning, t), out.Warning, sa.OnWarning, sa.Warning, sb.OnWarning, sb.Warning, t, highContrast)
	out.WarningContainer = mixRoleColor(sa.WarningContainer, sb.WarningContainer, t)
	out.OnWarningContainer, out.WarningContainer = ensureSinglePairWithBg(mixRoleColor(sa.OnWarningContainer, sb.OnWarningContainer, t), out.WarningContainer, sa.OnWarningContainer, sa.WarningContainer, sb.OnWarningContainer, sb.WarningContainer, t, highContrast)

	out.Success = mixRoleColor(sa.Success, sb.Success, t)
	out.OnSuccess, out.Success = ensureSinglePairWithBg(mixRoleColor(sa.OnSuccess, sb.OnSuccess, t), out.Success, sa.OnSuccess, sa.Success, sb.OnSuccess, sb.Success, t, highContrast)
	out.SuccessContainer = mixRoleColor(sa.SuccessContainer, sb.SuccessContainer, t)
	out.OnSuccessContainer, out.SuccessContainer = ensureSinglePairWithBg(mixRoleColor(sa.OnSuccessContainer, sb.OnSuccessContainer, t), out.SuccessContainer, sa.OnSuccessContainer, sa.SuccessContainer, sb.OnSuccessContainer, sb.SuccessContainer, t, highContrast)

	// Intermediate blended themes do not belong to a single seed, preventing
	// schemeCache entries from ever being generated during transitions.
	out.seed = ui.Color{}
	out.hasSeed = false
	out.variant = 0
	out.followed = nil

	if from.followed != nil && to.followed != nil && from.followed.prefs == to.followed.prefs {
		unscaled := out
		ba := sanitizePositiveScale(from.followed.from.TextScale, 1)
		bb := sanitizePositiveScale(to.followed.from.TextScale, 1)
		unscaled.TextScale = sanitizePositiveScale(ba+(bb-ba)*t, 1)
		if t >= 0.5 {
			unscaled.Contrast = to.followed.from.Contrast
			unscaled.ReduceMotion = to.followed.from.ReduceMotion
		} else {
			unscaled.Contrast = from.followed.from.Contrast
			unscaled.ReduceMotion = from.followed.from.ReduceMotion
		}
		out.followed = &followedTheme{
			from:  unscaled,
			to:    out,
			prefs: from.followed.prefs,
		}
	}

	return &out
}

// ThemeTransitionOptions configures how AnimatedThemeScope transitions between
// themes when color or appearance signatures change.
type ThemeTransitionOptions struct {
	// Token selects the MD3E effects spring driving the transition. The zero
	// value defaults to motion.DefaultEffects (~230ms critically damped, zero
	// overshoot); motion.FastEffects and motion.SlowEffects are also supported.
	Token motion.Token
}

type themeTransitionKey struct{}
type themeTransitionProgressKey struct{}

type themeVisualSig struct {
	dark               bool
	contrast           tokens.Contrast
	surfaceStyle       tokens.SurfaceStyle
	colors             tokens.ColorScheme
	warning            ui.Color
	onWarning          ui.Color
	warningContainer   ui.Color
	onWarningContainer ui.Color
	success            ui.Color
	onSuccess          ui.Color
	successContainer   ui.Color
	onSuccessContainer ui.Color
	textScale          float32
}

func visualSigOf(t Theme) themeVisualSig {
	return themeVisualSig{
		dark:               t.Dark,
		contrast:           t.Contrast,
		surfaceStyle:       t.SurfaceStyle,
		colors:             t.Colors,
		warning:            t.Warning,
		onWarning:          t.OnWarning,
		warningContainer:   t.WarningContainer,
		onWarningContainer: t.OnWarningContainer,
		success:            t.Success,
		onSuccess:          t.OnSuccess,
		successContainer:   t.SuccessContainer,
		onSuccessContainer: t.OnSuccessContainer,
		textScale:          t.TextScale,
	}
}

type themeTransitionState struct {
	initialized bool
	from        Theme
	to          Theme
	targetSig   themeVisualSig
}

// AnimatedThemeScope wraps ThemeScope and smoothly animates colors and
// TextScale whenever target's visual signature changes (such as Light <-> Dark,
// seed color, or contrast switches). Descendants and ui.Overlay popups inherit
// the interpolated theme frame by frame until the spring settles cleanly. When
// ReduceMotion is enabled on either target or c.Preferences(), changes jump
// immediately to target with zero pending animation frames.
func AnimatedThemeScope(c *ui.Context, target *Theme, fn func(), opts ...ThemeTransitionOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: AnimatedThemeScope takes at most one ThemeTransitionOptions")
	}
	var opt ThemeTransitionOptions
	if len(opts) == 1 {
		opt = opts[0]
	}
	if c == nil {
		return ui.Element{}
	}
	if target == nil {
		target = DefaultTheme(c)
	}

	token := motion.DefaultEffects
	if opt.Token >= motion.FastEffects && opt.Token <= motion.SlowEffects {
		token = opt.Token
	}

	prefs := c.Preferences()
	resolvedTarget := target.resolvedValue(prefs)
	reduceMotion := prefs.ReduceMotion || resolvedTarget.ReduceMotion
	sig := visualSigOf(resolvedTarget)

	scheme := resolvedTarget.Motion
	if scheme != motion.Expressive && scheme != motion.Standard {
		scheme = motion.Expressive
	}
	anim := ui.Animation{
		Spring: motion.Spring(scheme, token),
		Oklab:  true,
	}

	host := ui.Column(c)
	st := ui.Local(host, themeTransitionKey{}, func() themeTransitionState {
		return themeTransitionState{}
	})

	var current *Theme
	if !st.initialized {
		st.initialized = true
		st.from = resolvedTarget
		st.to = resolvedTarget
		st.targetSig = sig
		host.Jump(themeTransitionProgressKey{}, 1)
		cp := resolvedTarget
		current = &cp
	} else if reduceMotion {
		st.from = resolvedTarget
		st.to = resolvedTarget
		st.targetSig = sig
		host.Jump(themeTransitionProgressKey{}, 1)
		cp := resolvedTarget
		current = &cp
	} else if sig != st.targetSig {
		prevT := host.AnimateTo(themeTransitionProgressKey{}, 1, anim)
		snap := LerpTheme(&st.from, &st.to, prevT)
		st.from = *snap
		st.to = resolvedTarget
		st.targetSig = sig
		host.Jump(themeTransitionProgressKey{}, 0)
		t := host.AnimateTo(themeTransitionProgressKey{}, 1, anim)
		current = LerpTheme(&st.from, &st.to, t)
	} else {
		t := host.AnimateTo(themeTransitionProgressKey{}, 1, anim)
		if host.Settled(themeTransitionProgressKey{}) || t >= 1 {
			st.from = resolvedTarget
			st.to = resolvedTarget
			cp := resolvedTarget
			current = &cp
		} else {
			current = LerpTheme(&st.from, &st.to, t)
		}
	}

	host.Children(func() {
		ThemeScope(c, current, fn).Fill()
	})
	return host
}

func themeRegistryCountForTest() int {
	themeRegMu.RLock()
	defer themeRegMu.RUnlock()
	return len(themeByUI)
}

func themeSpecRegistryCountForTest() int {
	themeRegMu.RLock()
	defer themeRegMu.RUnlock()
	return len(themeBySpec)
}
