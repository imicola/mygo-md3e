package md3e

import (
	"math"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// TestPartialShapeScaleAndIdempotentPreferences checks that a partial
// ShapeScale falls back per field, and that ThemeScope does not multiply
// TextScale twice after WithPreferences already resolved it.
func TestPartialShapeScaleAndIdempotentPreferences(t *testing.T) {
	partial := ShapeScale{Medium: 18}
	if got, want := partial.Radius(tokens.ShapeMedium), float32(18); got != want {
		t.Fatalf("custom ShapeMedium = %v, want %v", got, want)
	}
	for _, scale := range []tokens.ShapeScale{
		tokens.ShapeExtraSmall, tokens.ShapeSmall, tokens.ShapeLarge, tokens.ShapeExtraLarge, tokens.ShapeFull,
	} {
		if got, want := partial.Radius(scale), float32(scale.Radius()); got != want {
			t.Fatalf("partial ShapeScale %v = %v, want the default %v", scale, got, want)
		}
	}

	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{TextScale: 1.2})
	prefs := ui.Preferences{TextScale: 1.5}
	var got *Theme
	ui.NewTesterWith(func(c *ui.Context) {
		ThemeScope(c, th.WithPreferences(prefs), func() {
			got = CurrentTheme(c)
			ui.Box(c).Size(10, 10)
		})
	}, 100, 100, ui.TesterOptions{Preferences: prefs})
	if got == nil {
		t.Fatal("the scoped theme was not resolved")
	}
	if math.Abs(float64(got.EffectiveTextScale()-1.8)) > 1e-4 {
		t.Fatalf("resolved TextScale = %v, want 1.2*1.5=1.8 once", got.EffectiveTextScale())
	}
}

// TestThemeRegistryDropsNaNScopes checks that a theme with a NaN field, which
// can never equal its map key, does not leak registry entries.
func TestThemeRegistryDropsNaNScopes(t *testing.T) {
	runtime.GC()
	uiBaseline := themeRegistryCountForTest()
	specBaseline := themeSpecRegistryCountForTest()

	th := ThemeFromSeed(DefaultSeedColor)
	th.Shapes.Medium = float32(math.NaN())
	for range 6 {
		ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				_ = CurrentTheme(c)
				ui.Box(c).Size(10, 10)
			})
		}, 60, 60)
	}

	for range 12 {
		runtime.GC()
		runtime.Gosched()
		if themeRegistryCountForTest() <= uiBaseline && themeSpecRegistryCountForTest() <= specBaseline {
			break
		}
	}
	if got := themeRegistryCountForTest(); got > uiBaseline {
		t.Fatalf("theme registry leaked NaN entries: count=%d baseline=%d", got, uiBaseline)
	}
	if got := themeSpecRegistryCountForTest(); got > specBaseline {
		t.Fatalf("theme spec index leaked NaN entries: count=%d baseline=%d", got, specBaseline)
	}
}

func TestThemeFromSeedOptionsAndDefaults(t *testing.T) {
	def := ThemeFromSeed(ui.Color{})
	if def.Dark {
		t.Fatal("default ThemeFromSeed should be light")
	}
	if def.Motion != motion.Expressive {
		t.Fatalf("default Motion = %v, want motion.Expressive", def.Motion)
	}
	if def.EffectiveTextScale() != 1 {
		t.Fatalf("default EffectiveTextScale = %v, want 1", def.EffectiveTextScale())
	}
	if def.Colors.Primary != ThemeFromSeed(DefaultSeedColor).Colors.Primary {
		t.Fatal("zero seed color should fall back to DefaultSeedColor")
	}
	if def.Shapes != DefaultShapeScale() {
		t.Fatal("default Shapes should match DefaultShapeScale()")
	}
	if got := def.TypeStyle(tokens.HeadlineMedium, true); got != def.Typography.Style(tokens.HeadlineMedium, true) {
		t.Fatalf("TypeStyle(HeadlineMedium, true) = %+v, want %+v", got, def.Typography.Style(tokens.HeadlineMedium, true))
	}
	if def.ContentColorFor(def.Colors.Primary) != def.Colors.OnPrimary {
		t.Fatal("ContentColorFor(Primary) did not return OnPrimary")
	}
	if def.ContentColorFor(def.Colors.PrimaryContainer) != def.Colors.OnPrimaryContainer {
		t.Fatal("ContentColorFor(PrimaryContainer) did not return OnPrimaryContainer")
	}
	if def.ContentColorFor(def.Warning) != def.OnWarning {
		t.Fatal("ContentColorFor(Warning) did not return OnWarning")
	}
	if def.ContentColorFor(def.Success) != def.OnSuccess {
		t.Fatal("ContentColorFor(Success) did not return OnSuccess")
	}

	customShapes := DefaultShapeScale()
	customShapes.Medium = 18
	darkExp := ThemeFromSeed(ui.Hex("#1b873f"), ThemeOptions{
		Dark:         true,
		Variant:      tokens.Expressive,
		Contrast:     tokens.ContrastHigh,
		Motion:       motion.Standard,
		Font:         "Inter",
		TextScale:    1.25,
		ReduceMotion: true,
		Shapes:       &customShapes,
	})
	if !darkExp.Dark || darkExp.Contrast != tokens.ContrastHigh || darkExp.Motion != motion.Standard {
		t.Fatalf("darkExp metadata mismatch: %+v", darkExp)
	}
	if darkExp.Shapes.Radius(tokens.ShapeMedium) != 18 {
		t.Fatalf("custom ShapeMedium radius = %v, want 18", darkExp.Shapes.Radius(tokens.ShapeMedium))
	}
	if darkExp.Font != "Inter" || darkExp.EffectiveTextScale() != 1.25 || !darkExp.ReduceMotion {
		t.Fatalf("darkExp text/motion options mismatch: Font=%q TextScale=%v ReduceMotion=%v",
			darkExp.Font, darkExp.EffectiveTextScale(), darkExp.ReduceMotion)
	}
	anim := darkExp.Animation(motion.DefaultSpatial)
	if anim.Duration != motion.Snap.Duration {
		t.Fatalf("ReduceMotion Animation.Duration = %v, want %v", anim.Duration, motion.Snap.Duration)
	}
	if relativeLuminance(darkExp.Colors.Surface) >= relativeLuminance(def.Colors.Surface) {
		t.Fatal("dark theme surface luminance should be lower than light theme surface")
	}
}

func TestToUIThemeBridgeAndSemanticMapping(t *testing.T) {
	th := ThemeFromSeed(ui.Hex("#006495"), ThemeOptions{
		Contrast:  tokens.ContrastHigh,
		TextScale: 1.5,
		Font:      "Roboto",
	})
	uiT := th.ToUITheme()
	if uiT == nil {
		t.Fatal("ToUITheme returned nil")
	}
	if uiT.Dark != th.Dark {
		t.Fatalf("uiT.Dark = %v, want %v", uiT.Dark, th.Dark)
	}
	if uiT.Background != th.Colors.Surface {
		t.Fatalf("uiT.Background = %+v, want Surface %+v", uiT.Background, th.Colors.Surface)
	}
	if uiT.Surface != th.Colors.SurfaceContainer {
		t.Fatalf("uiT.Surface = %+v, want SurfaceContainer %+v", uiT.Surface, th.Colors.SurfaceContainer)
	}
	if uiT.Accent != th.Colors.Primary || uiT.AccentText != th.Colors.OnPrimary {
		t.Fatalf("Accent/AccentText mismatch: got (%+v, %+v), want (%+v, %+v)",
			uiT.Accent, uiT.AccentText, th.Colors.Primary, th.Colors.OnPrimary)
	}
	if uiT.Text != th.Colors.OnSurface || uiT.TextMuted != th.Colors.OnSurfaceVariant {
		t.Fatal("Text/TextMuted did not bridge to OnSurface/OnSurfaceVariant")
	}
	if uiT.Border != th.Colors.Outline {
		t.Fatalf("high contrast uiT.Border = %+v, want Outline %+v", uiT.Border, th.Colors.Outline)
	}
	if uiT.Danger != th.Colors.Error || uiT.Warning != th.Warning || uiT.Success != th.Success {
		t.Fatal("semantic Danger/Warning/Success roles did not bridge cleanly")
	}
	if uiT.Inverse != th.Colors.InverseSurface || uiT.InverseText != th.Colors.InverseOnSurface {
		t.Fatal("Inverse/InverseText did not bridge to InverseSurface/InverseOnSurface")
	}
	if uiT.FocusRingWidth != 3 || uiT.FocusRingOffset != 2 {
		t.Fatalf("high-contrast focus ring = (%v, %v), want (3, 2)", uiT.FocusRingWidth, uiT.FocusRingOffset)
	}
	if uiT.FontSize != 21 || uiT.Font != "Roboto" {
		t.Fatalf("FontSize/Font = (%v, %q), want (21, Roboto)", uiT.FontSize, uiT.Font)
	}
	if uiT.Radius != th.Shapes.Small || uiT.Spacing != 4 {
		t.Fatalf("Radius/Spacing = (%v, %v), want (%v, 4)", uiT.Radius, uiT.Spacing, th.Shapes.Small)
	}

	// Mutating returned *ui.Theme must not affect another ToUITheme call.
	uiT.Accent = ui.Hex("#000000")
	if th.UITheme().Accent != th.Colors.Primary {
		t.Fatal("ToUITheme/UITheme must return fresh copies")
	}
}

func TestDefaultThemeFollowsDarkAndPreferences(t *testing.T) {
	var seen []*Theme
	var seenUI []*ui.Theme

	tt := ui.NewTester(func(c *ui.Context) {
		seen = append(seen, CurrentTheme(c))
		seenUI = append(seenUI, c.Theme())
		ui.Box(c).Size(40, 40)
	}, 400, 300)

	light := seen[len(seen)-1]
	if light.Dark {
		t.Fatal("initial DefaultTheme should be light")
	}
	if light.Colors.Primary != ThemeFromSeed(DefaultSeedColor).Colors.Primary {
		t.Fatal("initial DefaultTheme should derive from DefaultSeedColor")
	}
	if light.Contrast != tokens.ContrastStandard || light.EffectiveTextScale() != 1 || light.ReduceMotion {
		t.Fatalf("unexpected baseline default preferences: %+v", light)
	}

	tt.SetDark(true)
	dark := seen[len(seen)-1]
	if !dark.Dark {
		t.Fatal("DefaultTheme did not react to SetDark(true)")
	}
	if dark.Colors.Surface == light.Colors.Surface {
		t.Fatal("dark Surface should differ from light Surface")
	}

	accent := ui.Hex("#e65100")
	tt.SetPreferences(ui.Preferences{
		Accent:       accent,
		HighContrast: true,
		TextScale:    1.5,
		ReduceMotion: true,
	})
	prefTheme := seen[len(seen)-1]
	if !prefTheme.Dark {
		t.Fatal("prefTheme should stay dark after SetDark(true)")
	}
	if prefTheme.Contrast != tokens.ContrastHigh {
		t.Fatalf("prefTheme.Contrast = %v, want ContrastHigh", prefTheme.Contrast)
	}
	if prefTheme.EffectiveTextScale() != 1.5 {
		t.Fatalf("prefTheme.TextScale = %v, want 1.5", prefTheme.EffectiveTextScale())
	}
	if !prefTheme.ReduceMotion {
		t.Fatal("prefTheme.ReduceMotion should be true")
	}
	expectedPrimary := ThemeFromSeed(accent, ThemeOptions{Dark: true, Contrast: tokens.ContrastHigh}).Colors.Primary
	if prefTheme.Colors.Primary != expectedPrimary {
		t.Fatalf("prefTheme Primary = %+v, want %+v derived from Accent preference", prefTheme.Colors.Primary, expectedPrimary)
	}
	if seenUI[len(seenUI)-1] == nil {
		t.Fatal("window ui.Theme unexpectedly nil")
	}
}

func TestNestedThemeScopeAndOverlayInheritance(t *testing.T) {
	outerTheme := ThemeFromSeed(ui.Hex("#6750a4"), ThemeOptions{Dark: false})
	innerTheme := ThemeFromSeed(ui.Hex("#1b873f"), ThemeOptions{Dark: true})

	var (
		outerBefore       *Theme
		outerAfter        *Theme
		innerGot          *Theme
		innerUIGot        *ui.Theme
		overlayGot        *Theme
		overlayUIGot      *ui.Theme
		overlayPaintTheme *ui.Theme
		popoverGot        *Theme
		dialogGot         *Theme
		menuGot           *Theme
		openPopover       = true
		openDialog        = true
		openMenu          = true
	)

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, outerTheme, func() {
			outerBefore = CurrentTheme(c)

			ThemeScope(c, innerTheme, func() {
				innerGot = CurrentTheme(c)
				innerUIGot = c.Theme()

				ui.Overlay(c, func() {
					overlayGot = CurrentTheme(c)
					overlayUIGot = c.Theme()
					ui.Box(c).Size(30, 20).Draw(func(p *ui.Painter, r ui.Rect) {
						overlayPaintTheme = p.Theme()
					})
				})

				anchor := ui.Box(c).Size(60, 24)
				ui.PopoverBase(c, anchor, &openPopover, func(panel ui.Element) {
					popoverGot = CurrentTheme(c)
					panel.Children(func() { ui.Text(c, "pop") })
				})

				ui.DialogBase(c, &openDialog, func(backdrop, panel ui.Element) {
					dialogGot = CurrentTheme(c)
					panel.Children(func() { ui.Text(c, "dlg") })
				})

				menu := ui.MenuBase(c.Key("menu"), &openMenu)
				menu.Trigger.Children(func() { ui.Text(c, "menu") })
				menu.Popup(func(panel ui.Element) {
					menuGot = CurrentTheme(c)
					panel.Children(func() { ui.Text(c, "item") })
				})
			})

			outerAfter = CurrentTheme(c)
		})
	}, 500, 400)

	if outerBefore == nil || outerAfter == nil {
		t.Fatal("outer theme snapshots missing")
	}
	if outerBefore.Dark || outerBefore.Colors.Primary != outerTheme.Colors.Primary {
		t.Fatalf("outerBefore = (Dark:%v Primary:%+v), want Light Purple", outerBefore.Dark, outerBefore.Colors.Primary)
	}
	if outerAfter.Dark || outerAfter.Colors.Primary != outerTheme.Colors.Primary {
		t.Fatalf("outerAfter did not restore cleanly: (Dark:%v Primary:%+v)", outerAfter.Dark, outerAfter.Colors.Primary)
	}
	if innerGot == nil || !innerGot.Dark || innerGot.Colors.Primary != innerTheme.Colors.Primary {
		t.Fatalf("innerGot = (Dark:%v Primary:%+v), want Dark Green", innerGot.Dark, innerGot.Colors.Primary)
	}
	if innerUIGot == nil || !innerUIGot.Dark || innerUIGot.Accent != innerTheme.Colors.Primary {
		t.Fatalf("c.Theme() inside inner ThemeScope did not bridge to inner ui.Theme: %+v", innerUIGot)
	}
	if overlayGot == nil || overlayGot.Colors.Primary != innerTheme.Colors.Primary || !overlayGot.Dark {
		t.Fatalf("CurrentTheme inside ui.Overlay = %+v, want inner Dark Green", overlayGot)
	}
	if overlayUIGot != innerUIGot || overlayPaintTheme != innerUIGot {
		t.Fatalf("Overlay c.Theme()/p.Theme() mismatch: overlayUIGot=%p paint=%p inner=%p",
			overlayUIGot, overlayPaintTheme, innerUIGot)
	}
	for name, got := range map[string]*Theme{
		"PopoverBase": popoverGot,
		"DialogBase":  dialogGot,
		"MenuBase":    menuGot,
	} {
		if got == nil || !got.Dark || got.Colors.Primary != innerTheme.Colors.Primary {
			t.Fatalf("%s CurrentTheme = %+v, want inner Dark Green", name, got)
		}
	}

	// Verify Preferences propagate into explicit ThemeScope AND native UITheme FontSize.
	tt.SetPreferences(ui.Preferences{
		HighContrast: true,
		TextScale:    1.5,
		ReduceMotion: true,
	})
	if innerGot.Contrast != tokens.ContrastHigh || innerGot.EffectiveTextScale() != 1.5 || !innerGot.ReduceMotion {
		t.Fatalf("explicit ThemeScope did not react to Preferences: Contrast=%v TextScale=%v ReduceMotion=%v",
			innerGot.Contrast, innerGot.EffectiveTextScale(), innerGot.ReduceMotion)
	}
	if innerUIGot.FontSize != 21 || innerUIGot.FocusRingWidth != 3 {
		t.Fatalf("bridged UITheme did not scale with Preferences: FontSize=%v FocusRingWidth=%v",
			innerUIGot.FontSize, innerUIGot.FocusRingWidth)
	}
}

func TestThemeScopeNestedUIThemedAndManualThemePreferences(t *testing.T) {
	outerTheme := ThemeFromSeed(ui.Hex("#6750a4"), ThemeOptions{Dark: false})
	customUI := ui.DarkTheme()
	customUI.Accent = ui.Hex("#00bfa5")

	var stockDarkGot, customDarkGot, manualResolved *Theme
	manual := Theme{
		Dark:   false,
		Colors: outerTheme.Colors,
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, outerTheme, func() {
			ui.Themed(c, ui.DarkTheme()).Children(func() {
				stockDarkGot = CurrentTheme(c)
			})
			ui.Themed(c, customUI).Children(func() {
				customDarkGot = CurrentTheme(c)
			})
		})
		ThemeScope(c, &manual, func() {
			manualResolved = CurrentTheme(c)
		})
	}, 400, 300)

	if stockDarkGot == nil || !stockDarkGot.Dark {
		t.Fatalf("nested ui.Themed(DarkTheme) should switch CurrentTheme to dark, got %+v", stockDarkGot)
	}
	if customDarkGot == nil || !customDarkGot.Dark {
		t.Fatal("nested custom ui.Themed should preserve Dark: true")
	}
	wantCustomPrimary := ThemeFromSeed(customUI.Accent, ThemeOptions{Dark: true}).Colors.Primary
	if customDarkGot.Colors.Primary != wantCustomPrimary {
		t.Fatalf("nested custom ui.Themed Accent = %+v, want %+v", customDarkGot.Colors.Primary, wantCustomPrimary)
	}

	tt.SetPreferences(ui.Preferences{HighContrast: true, TextScale: 1.25, ReduceMotion: true})
	if manualResolved.Contrast != tokens.ContrastHigh || manualResolved.EffectiveTextScale() != 1.25 || !manualResolved.ReduceMotion {
		t.Fatalf("manual theme under Preferences = Contrast:%v TextScale:%v ReduceMotion:%v",
			manualResolved.Contrast, manualResolved.EffectiveTextScale(), manualResolved.ReduceMotion)
	}
	if manualResolved.Colors.Primary != outerTheme.Colors.Primary {
		t.Fatal("manual unseeded theme HighContrast must preserve custom Primary color instead of replacing it with seed")
	}
}

func TestThemeScopeWeakRegistryNoMemoryLeak(t *testing.T) {
	runOnce := func(seed ui.Color) {
		th := ThemeFromSeed(seed)
		ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				_ = CurrentTheme(c)
				ui.Box(c).Size(10, 10)
			})
		}, 200, 200)
	}

	runtime.GC()
	baseline := themeRegistryCountForTest()

	for i := range 24 {
		runOnce(ui.RGBA(uint8(40+i*7), 90, 200, 1))
	}

	for range 12 {
		runtime.GC()
		runtime.Gosched()
		if themeRegistryCountForTest() <= baseline {
			break
		}
	}
	if got := themeRegistryCountForTest(); got > baseline {
		t.Fatalf("theme registry leaked entries after scopes dropped: count=%d baseline=%d", got, baseline)
	}
}

func TestWithPreferencesRoundTripAndManualEdits(t *testing.T) {
	seed := ui.Hex("#1E88E5")
	base := ThemeFromSeed(seed)
	highPrefs := ui.Preferences{HighContrast: true, ReduceMotion: true, TextScale: 1.5}
	normalPrefs := ui.Preferences{HighContrast: false, ReduceMotion: false, TextScale: 1}

	high := base.WithPreferences(highPrefs)
	if high.Contrast != tokens.ContrastHigh || !high.ReduceMotion || high.TextScale != 1.5 {
		t.Fatalf("high preferences not applied: contrast=%v rm=%v scale=%v", high.Contrast, high.ReduceMotion, high.TextScale)
	}
	if high.Colors == base.Colors {
		t.Fatal("expected high contrast seeded theme to regenerate colors")
	}

	restored := high.WithPreferences(normalPrefs)
	if restored.Contrast != base.Contrast || restored.ReduceMotion != base.ReduceMotion || restored.TextScale != base.TextScale {
		t.Fatalf("restored theme did not recover baseline prefs: got=(c=%v rm=%v s=%v) want=(c=%v rm=%v s=%v)",
			restored.Contrast, restored.ReduceMotion, restored.TextScale, base.Contrast, base.ReduceMotion, base.TextScale)
	}
	if restored.Colors != base.Colors || restored.Warning != base.Warning || restored.Success != base.Success {
		t.Fatal("restored seeded theme colors/semantic roles did not match baseline")
	}

	// Repeated high -> normal -> high -> normal remains strictly idempotent.
	restoredAgain := restored.WithPreferences(highPrefs).WithPreferences(normalPrefs)
	if restoredAgain.Contrast != base.Contrast || restoredAgain.ReduceMotion != base.ReduceMotion ||
		restoredAgain.TextScale != base.TextScale || restoredAgain.Colors != base.Colors {
		t.Fatal("repeated WithPreferences round-trip was not idempotent")
	}

	// Manual un-seeded theme restores original un-mixed Outline and Contrast.
	manual := &Theme{Colors: base.Colors, Font: "Custom"}
	manualHigh := manual.WithPreferences(highPrefs)
	if manualHigh.Colors.Outline == manual.Colors.Outline {
		t.Fatal("expected manual high-contrast theme to strengthen Outline")
	}
	manualRestored := manualHigh.WithPreferences(normalPrefs)
	if manualRestored.Colors.Outline != manual.Colors.Outline || manualRestored.Contrast != 0 || manualRestored.ReduceMotion {
		t.Fatalf("manual theme did not recover baseline on normal prefs: got c=%v rm=%v", manualRestored.Contrast, manualRestored.ReduceMotion)
	}

	// Explicit post-WithPreferences caller edits survive while untouched fields recover.
	edited := high.Clone()
	edited.Font = "RobotoMono"
	editedRestored := edited.WithPreferences(normalPrefs)
	if editedRestored.Font != "RobotoMono" {
		t.Fatalf("caller Font override lost: got %q", editedRestored.Font)
	}
	if editedRestored.Contrast != base.Contrast || editedRestored.ReduceMotion || editedRestored.TextScale != 1 {
		t.Fatalf("untouched preference fields did not restore after Font edit: c=%v rm=%v s=%v",
			editedRestored.Contrast, editedRestored.ReduceMotion, editedRestored.TextScale)
	}

	// Theme's own explicit strict settings are never relaxed by normal prefs.
	strict := ThemeFromSeed(seed, ThemeOptions{Contrast: tokens.ContrastHigh, ReduceMotion: true, TextScale: 1.25})
	strictAfter := strict.WithPreferences(normalPrefs)
	if strictAfter.Contrast != tokens.ContrastHigh || !strictAfter.ReduceMotion || strictAfter.TextScale != 1.25 {
		t.Fatalf("explicit theme baseline should be preserved: c=%v rm=%v s=%v",
			strictAfter.Contrast, strictAfter.ReduceMotion, strictAfter.TextScale)
	}

	// ThemeScope toggling HighContrast/ReduceMotion dynamically also recovers.
	var seen *Theme
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, base, func() {
			seen = CurrentTheme(c)
			ui.Text(c, "Scope")
		})
	}, 200, 100)

	tt.SetPreferences(highPrefs)
	if seen == nil || seen.Contrast != tokens.ContrastHigh || !seen.ReduceMotion {
		t.Fatalf("ThemeScope failed to apply high prefs: %+v", seen)
	}
	tt.SetPreferences(normalPrefs)
	if seen.Contrast != base.Contrast || seen.ReduceMotion != base.ReduceMotion || seen.Colors != base.Colors {
		t.Fatalf("ThemeScope failed to restore normal prefs: c=%v rm=%v", seen.Contrast, seen.ReduceMotion)
	}
}

func TestThemeScopeHighContrastSteadyRebuildSkipsDynamicScheme(t *testing.T) {
	uniqueSeed := ui.RGBA(0x31, 0x77, 0x99, 1)
	base := ThemeFromSeed(uniqueSeed)
	tick := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, base, func() {
			if tick >= 0 {
				ui.Text(c, "Tick")
			}
		})
	}, 200, 100)

	tt.SetPreferences(ui.Preferences{HighContrast: true, TextScale: 1})
	tt.Frame()
	before := schemeComputeCount.Load()

	for i := 1; i <= 20; i++ {
		tick = i
		tt.Frame()
	}
	if after := schemeComputeCount.Load(); after != before {
		t.Fatalf("steady HighContrast ThemeScope rebuilds recomputed DynamicScheme %d times", after-before)
	}
}

func TestNaNPreferencesDoNotBloatThemeCaches(t *testing.T) {
	nan := float32(math.NaN())
	infPos := float32(math.Inf(1))
	infNeg := float32(math.Inf(-1))

	th := ThemeFromSeed(ui.Hex("#6750A4"), ThemeOptions{
		Contrast:  tokens.Contrast(math.NaN()),
		TextScale: nan,
		Shapes:    &ShapeScale{ExtraSmall: nan, Large: infPos, Medium: 12},
	})
	if got := th.EffectiveTextScale(); got != 1 {
		t.Fatalf("EffectiveTextScale with NaN = %v, want 1", got)
	}
	if got := th.Shapes.Radius(tokens.ShapeExtraSmall); got != float32(tokens.ShapeExtraSmall.Radius()) {
		t.Fatalf("ShapeScale NaN radius = %v, want fallback %v", got, tokens.ShapeExtraSmall.Radius())
	}

	_ = cachedThemeFromSeed(DefaultSeedColor, false, tokens.TonalSpot, ui.Preferences{TextScale: 1})
	defaultThemeMu.RLock()
	defBefore := len(defaultThemes)
	defaultThemeMu.RUnlock()

	for range 20 {
		_ = cachedThemeFromSeed(DefaultSeedColor, false, tokens.TonalSpot, ui.Preferences{TextScale: nan})
		_ = cachedThemeFromSeed(DefaultSeedColor, false, tokens.TonalSpot, ui.Preferences{TextScale: infPos})
		_ = cachedThemeFromSeed(DefaultSeedColor, false, tokens.TonalSpot, ui.Preferences{TextScale: infNeg})
	}
	defaultThemeMu.RLock()
	defAfter := len(defaultThemes)
	defaultThemeMu.RUnlock()
	if defAfter != defBefore {
		t.Fatalf("defaultThemes cache bloated on NaN/Inf TextScale: before=%d after=%d", defBefore, defAfter)
	}

	runtime.GC()
	baseSpec := themeSpecRegistryCountForTest()
	for range 20 {
		func() {
			tmp := ui.NewTester(func(c *ui.Context) {
				dirty := &Theme{
					Contrast:  tokens.Contrast(math.NaN()),
					TextScale: nan,
					Shapes:    ShapeScale{Small: nan},
				}
				ThemeScope(c, dirty, func() {
					ui.Text(c, "nan")
				})
			}, 200, 100)
			tmp.SetPreferences(ui.Preferences{TextScale: nan})
		}()
	}
	if gotSpec := themeSpecRegistryCountForTest(); gotSpec > baseSpec+1 {
		t.Fatalf("themeBySpec bloated on NaN scopes: before=%d after=%d", baseSpec, gotSpec)
	}
	for range 25 {
		runtime.GC()
		runtime.Gosched()
		if themeRegistryCountForTest() <= baseSpec && themeSpecRegistryCountForTest() <= baseSpec {
			break
		}
	}
	if gotSpec := themeSpecRegistryCountForTest(); gotSpec > baseSpec {
		t.Fatalf("O(1) cleanup left stale spec entries: count=%d baseline=%d", gotSpec, baseSpec)
	}
}

func TestThemeDensityAndMonoFontPropagation(t *testing.T) {
	// Zero options remain Comfortable with empty MonoFont and Spacing == 4.
	def := ThemeFromSeed(DefaultSeedColor)
	if def.Density != tokens.DensityComfortable || def.MonoFont != "" {
		t.Fatalf("default Density/MonoFont = (%v, %q), want (Comfortable, empty)", def.Density, def.MonoFont)
	}
	if got := def.ToUITheme().Spacing; got != 4 {
		t.Fatalf("default ui.Theme.Spacing = %v, want 4", got)
	}

	// Out-of-range Density in ThemeOptions falls back safely to Comfortable.
	invalid := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tokens.Density(99)})
	if invalid.Density != tokens.DensityComfortable {
		t.Fatalf("out-of-range Density = %v, want DensityComfortable", invalid.Density)
	}

	// Desktop/Compact Density and MonoFont survive WithPreferences, manual edits,
	// and nested ThemeScope + Overlay inheritance.
	desk := ThemeFromSeed(ui.Hex("#2e7d32"), ThemeOptions{
		Density:  tokens.DensityDesktop,
		MonoFont: "JetBrains Mono",
	})
	if desk.Density != tokens.DensityDesktop || desk.MonoFont != "JetBrains Mono" {
		t.Fatalf("desk Density/MonoFont = (%v, %q)", desk.Density, desk.MonoFont)
	}
	if got := desk.ToUITheme().Spacing; got != float32(tokens.DensityDesktop.SpacingUnit()) {
		t.Fatalf("desk ui.Theme.Spacing = %v, want %v", got, tokens.DensityDesktop.SpacingUnit())
	}

	high := desk.WithPreferences(ui.Preferences{HighContrast: true, TextScale: 1.25})
	if high.Density != tokens.DensityDesktop || high.MonoFont != "JetBrains Mono" {
		t.Fatalf("WithPreferences dropped Density/MonoFont: (%v, %q)", high.Density, high.MonoFont)
	}
	edited := high.Clone()
	edited.Density = tokens.DensityCompact
	edited.MonoFont = "Cascadia Code"
	restored := edited.WithPreferences(ui.Preferences{TextScale: 1})
	if restored.Density != tokens.DensityCompact || restored.MonoFont != "Cascadia Code" {
		t.Fatalf("caller edits to Density/MonoFont lost on restore: (%v, %q)", restored.Density, restored.MonoFont)
	}

	var outerGot, innerGot, overlayGot *Theme
	ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, desk, func() {
			outerGot = CurrentTheme(c)
			ThemeScope(c, restored, func() {
				innerGot = CurrentTheme(c)
				ui.Overlay(c, func() {
					overlayGot = CurrentTheme(c)
					ui.Box(c).Size(10, 10)
				})
			})
		})
	}, 200, 160)

	if outerGot == nil || outerGot.Density != tokens.DensityDesktop || outerGot.MonoFont != "JetBrains Mono" {
		t.Fatalf("outer ThemeScope Density/MonoFont = %+v", outerGot)
	}
	if innerGot == nil || innerGot.Density != tokens.DensityCompact || innerGot.MonoFont != "Cascadia Code" {
		t.Fatalf("inner ThemeScope Density/MonoFont = %+v", innerGot)
	}
	if overlayGot == nil || overlayGot.Density != tokens.DensityCompact || overlayGot.MonoFont != "Cascadia Code" {
		t.Fatalf("overlay ThemeScope Density/MonoFont = %+v", overlayGot)
	}
}

func TestTextWithAndTabularNums(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{MonoFont: "Fira Code"})
	var defR, singleR, middleR, startR, tabR ui.Rect

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Width(120).Gap(4).Children(func() {
				TextWith(c, "Default body text").DrawOver(func(_ *ui.Painter, r ui.Rect) { defR = r })
				TextWith(c, "A very long file path /usr/local/share/mygo-md3e/typography.go", TextOptions{
					Role:     tokens.BodySmall,
					MaxLines: 1,
					Truncate: TruncateEnd,
				}).DrawOver(func(_ *ui.Painter, r ui.Rect) { singleR = r })
				TextWith(c, "A very long file path /usr/local/share/mygo-md3e/typography.go", TextOptions{
					Role:     tokens.BodySmall,
					Truncate: TruncateMiddle,
					Mono:     true,
				}).DrawOver(func(_ *ui.Painter, r ui.Rect) { middleR = r })
				TextWith(c, "A very long file path /usr/local/share/mygo-md3e/typography.go", TextOptions{
					Role:     tokens.BodySmall,
					MaxLines: 1,
					Truncate: TruncateStart,
					Color:    th.Colors.Primary,
				}).DrawOver(func(_ *ui.Painter, r ui.Rect) { startR = r })
				TabularNums(TextWith(c, "123,456.78 ms", TextOptions{
					Role:        tokens.LabelLarge,
					Emphasized:  true,
					TabularNums: true,
				})).DrawOver(func(_ *ui.Painter, r ui.Rect) { tabR = r })
				// Out-of-range Role falls back safely to BodyMedium (non-zero height).
				TextWith(c, "Fallback role", TextOptions{Role: tokens.TypeRole(99)})
			})
		})
	}, 240, 220)

	for name, r := range map[string]ui.Rect{
		"default": defR,
		"single":  singleR,
		"middle":  middleR,
		"start":   startR,
		"tabular": tabR,
	} {
		if r.W <= 0 || r.H <= 0 {
			t.Fatalf("TextWith(%s) laid out with empty rect %+v", name, r)
		}
	}
	if middleR.W > 120.5 || startR.W > 120.5 || singleR.W > 120.5 {
		t.Fatalf("truncated texts overflowed 120 DIP container: single=%v middle=%v start=%v",
			singleR.W, middleR.W, startR.W)
	}
	// Single-line BodySmall should not wrap taller than one line (~16 DIP).
	if middleR.H > 22 || singleR.H > 22 || startR.H > 22 {
		t.Fatalf("single-line truncated text wrapped unexpectedly: single=%v middle=%v start=%v",
			singleR.H, middleR.H, startR.H)
	}
	if tt.FramePending() {
		t.Fatal("static TextWith should not leave frames pending")
	}

	// Passing more than one TextOptions must panic.
	defer func() {
		if recover() == nil {
			t.Fatal("expected TextWith with 2 options to panic")
		}
	}()
	ui.NewTester(func(c *ui.Context) {
		TextWith(c, "panic", TextOptions{}, TextOptions{})
	}, 100, 50)
}
