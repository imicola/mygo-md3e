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
	colors := cachedColorScheme(seed, o.Dark, o.Variant, contrast)

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
			t.Success = ui.Hex("#b8ffb0")
			t.OnSuccess = ui.Hex("#002204")
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
			cp.Colors = cachedColorScheme(cp.seed, cp.Dark, cp.variant, tokens.ContrastHigh)
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
	seed     ui.Color
	dark     bool
	variant  tokens.Variant
	contrast tokens.Contrast
}

type defaultThemeKey struct {
	seed         ui.Color
	dark         bool
	variant      tokens.Variant
	contrast     tokens.Contrast
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

func cachedColorScheme(seed ui.Color, dark bool, variant tokens.Variant, contrast tokens.Contrast) tokens.ColorScheme {
	contrast = sanitizeContrast(contrast)
	key := schemeCacheKey{
		seed:     seed,
		dark:     dark,
		variant:  variant,
		contrast: contrast,
	}
	schemeCacheMu.RLock()
	if cs, ok := schemeCache[key]; ok {
		schemeCacheMu.RUnlock()
		return cs
	}
	schemeCacheMu.RUnlock()

	schemeComputeCount.Add(1)
	cs := tokens.NewDynamicScheme(tokens.DynamicSchemeOptions{
		Seed:     seed,
		Dark:     dark,
		Variant:  variant,
		Contrast: contrast,
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
