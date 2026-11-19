package md3e

import (
	"math"
	"runtime"
	"sync"
	"weak"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
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

// Radius returns the radius in DIPs configured for scale. If the ShapeScale is
// zero-valued, the standard MD3E fallback radius for scale is returned.
func (s ShapeScale) Radius(scale tokens.ShapeScale) float32 {
	if s == (ShapeScale{}) {
		return float32(scale.Radius())
	}
	switch scale {
	case tokens.ShapeNone:
		return s.None
	case tokens.ShapeExtraSmall:
		return s.ExtraSmall
	case tokens.ShapeSmall:
		return s.Small
	case tokens.ShapeMedium:
		return s.Medium
	case tokens.ShapeLarge:
		return s.Large
	case tokens.ShapeLargeIncreased:
		return s.LargeIncreased
	case tokens.ShapeExtraLarge:
		return s.ExtraLarge
	case tokens.ShapeExtraLargeIncreased:
		return s.ExtraLargeIncreased
	case tokens.ShapeExtraExtraLarge:
		return s.ExtraExtraLarge
	case tokens.ShapeFull:
		return s.Full
	default:
		return float32(scale.Radius())
	}
}

// ThemeOptions configures how ThemeFromSeed constructs an MD3E theme.
type ThemeOptions struct {
	Dark         bool
	Variant      tokens.Variant
	Contrast     tokens.Contrast
	Motion       motion.Scheme
	Font         string
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
	Typography   tokens.TypographyScale
	Shapes       ShapeScale
	Contrast     tokens.Contrast
	TextScale    float32
	ReduceMotion bool
	Font         string

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

	ds := tokens.NewDynamicScheme(tokens.DynamicSchemeOptions{
		Seed:     seed,
		Dark:     o.Dark,
		Variant:  o.Variant,
		Contrast: o.Contrast,
	})

	shapes := DefaultShapeScale()
	if o.Shapes != nil {
		shapes = *o.Shapes
	}
	typo := tokens.NewTypographyScale()
	if o.Typography != nil {
		typo = *o.Typography
	}

	scale := o.TextScale
	if scale <= 0 {
		scale = 1
	}

	t := &Theme{
		Dark:         o.Dark,
		Colors:       ds.Colors,
		Motion:       o.Motion,
		Typography:   typo,
		Shapes:       shapes,
		Contrast:     o.Contrast,
		TextScale:    scale,
		ReduceMotion: o.ReduceMotion,
		Font:         o.Font,
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
	if t == nil || t.TextScale <= 0 {
		return 1
	}
	return t.TextScale
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

func (t *Theme) resolvedValue(p ui.Preferences) Theme {
	cp := *t
	if cp.Typography == (tokens.TypographyScale{}) {
		cp.Typography = tokens.NewTypographyScale()
	}
	if cp.Shapes == (ShapeScale{}) {
		cp.Shapes = DefaultShapeScale()
	}
	if cp.Warning.A == 0 || cp.Success.A == 0 {
		applySemanticRoles(&cp)
	}

	if p.HighContrast && cp.Contrast < tokens.ContrastHigh {
		cp.Contrast = tokens.ContrastHigh
		if cp.hasSeed {
			ds := tokens.NewDynamicScheme(tokens.DynamicSchemeOptions{
				Seed:     cp.seed,
				Dark:     cp.Dark,
				Variant:  cp.variant,
				Contrast: tokens.ContrastHigh,
			})
			cp.Colors = ds.Colors
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

	baseScale := cp.TextScale
	if baseScale <= 0 {
		baseScale = 1
	}
	prefScale := p.TextScale
	if prefScale <= 0 {
		prefScale = 1
	}
	cp.TextScale = baseScale * prefScale
	cp.ReduceMotion = cp.ReduceMotion || p.ReduceMotion
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
		Spacing:         4,
		FontSize:        14 * t.EffectiveTextScale(),
		Font:            t.Font,
	}
}

// Weak-reference theme registries prevent memory leaks across multiple windows
// and short-lived ui.Tester runs while preserving pointer stability across rebuilds.
type scopedUIKey struct {
	wp   weak.Pointer[ui.Theme]
	spec Theme
}

var (
	themeRegMu     sync.RWMutex
	themeByUI      = map[weak.Pointer[ui.Theme]]*Theme{}
	themeBySpec    = map[Theme]weak.Pointer[ui.Theme]{}
	defaultThemeMu sync.RWMutex
	defaultThemes  = map[defaultThemeKey]*Theme{}

	stockLightAccent = ui.LightTheme().Accent
	stockDarkAccent  = ui.DarkTheme().Accent
)

type defaultThemeKey struct {
	seed         ui.Color
	dark         bool
	variant      tokens.Variant
	contrast     tokens.Contrast
	textScale    float32
	reduceMotion bool
}

const maxDefaultThemeCache = 32

func registerScopedTheme(resolved Theme) *ui.Theme {
	themeRegMu.Lock()
	defer themeRegMu.Unlock()

	if wp, ok := themeBySpec[resolved]; ok {
		if existing := wp.Value(); existing != nil {
			if cached := themeByUI[wp]; cached != nil {
				return existing
			}
		}
		delete(themeBySpec, resolved)
	}

	uiVal := resolved.buildUITheme()
	uiPtr := &uiVal
	wp := weak.Make(uiPtr)
	stored := resolved
	themeBySpec[stored] = wp
	themeByUI[wp] = &stored

	runtime.AddCleanup(uiPtr, func(k scopedUIKey) {
		themeRegMu.Lock()
		defer themeRegMu.Unlock()
		delete(themeByUI, k.wp)
		if cur, ok := themeBySpec[k.spec]; ok && cur == k.wp {
			delete(themeBySpec, k.spec)
		}
	}, scopedUIKey{wp: wp, spec: stored})

	return uiPtr
}

func lookupScopedTheme(uiT *ui.Theme) (*Theme, bool) {
	if uiT == nil {
		return nil, false
	}
	wp := weak.Make(uiT)
	themeRegMu.RLock()
	t, ok := themeByUI[wp]
	themeRegMu.RUnlock()
	if !ok || t == nil {
		return nil, false
	}
	return t, true
}

func cachedThemeFromSeed(seed ui.Color, dark bool, variant tokens.Variant, prefs ui.Preferences) *Theme {
	if seed.A == 0 {
		seed = DefaultSeedColor
	}
	contrast := tokens.ContrastStandard
	if prefs.HighContrast {
		contrast = tokens.ContrastHigh
	}
	textScale := prefs.TextScale
	if textScale <= 0 {
		textScale = 1
	}
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

	built := ThemeFromSeed(seed, ThemeOptions{
		Dark:         dark,
		Variant:      variant,
		Contrast:     contrast,
		TextScale:    textScale,
		ReduceMotion: prefs.ReduceMotion,
	})

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
func ThemeScope(c *ui.Context, t *Theme, content func()) ui.Element {
	if t == nil {
		t = DefaultTheme(c)
	}
	resolved := t.resolvedValue(c.Preferences())
	uiT := registerScopedTheme(resolved)

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
