package md3e

import (
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

type stubWindowController struct {
	minimized int
	toggled   int
	closed    int
	maximized bool
	pinned    bool
}

func (s *stubWindowController) Minimize()             { s.minimized++ }
func (s *stubWindowController) ToggleMaximize()       { s.toggled++; s.maximized = !s.maximized }
func (s *stubWindowController) Close()                { s.closed++ }
func (s *stubWindowController) IsMaximized() bool     { return s.maximized }
func (s *stubWindowController) SetAlwaysOnTop(v bool) { s.pinned = v }

type stubThemedWindowController struct {
	stubWindowController
	buttonsTheme mygo.ThemeSource
	themeCalls   int
}

func (s *stubThemedWindowController) SetWindowButtonsTheme(src mygo.ThemeSource) {
	s.buttonsTheme = src
	s.themeCalls++
}

func TestWindowTitleBarSafeInsetsControlsAccessibilityAndRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{
		Density:      tokens.DensityDesktop,
		ReduceMotion: true,
	})
	win := &stubWindowController{}
	pinned := false
	pinEvents := 0
	var barR, leadR, trailR ui.Rect

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Fill().Children(func() {
				WindowTitleBar(c, WindowTitleBarOptions{
					Title:              "Workbench",
					Divider:            true,
					Window:             win,
					ShowWindowControls: true,
					Pinned:             &pinned,
					OnPin: func(v bool) {
						pinEvents++
					},
					Leading: func() ui.Element {
						return ui.Box(c).Size(24, 20).DrawOver(func(_ *ui.Painter, r ui.Rect) { leadR = r })
					},
					Trailing: func() ui.Element {
						return ui.Box(c).Size(24, 20).DrawOver(func(_ *ui.Painter, r ui.Rect) { trailR = r })
					},
				}).DrawOver(func(_ *ui.Painter, r ui.Rect) { barR = r })
			})
		})
	}, 640, 200)

	// Default Desktop height is 38 DIP when c.TitleBar() is zero.
	if absF32(barR.H-38) > 0.5 {
		t.Fatalf("default Desktop title bar height = %.1f, want 38", barR.H)
	}

	// Simulate macOS left traffic lights + right system margin taller than default height.
	tt.SetTitleBar(ui.TitleBar{Height: 46, Left: 76, Right: 20, Bottom: 46})
	tt.Frame()
	if absF32(barR.H-46) > 0.5 {
		t.Fatalf("title bar did not grow to c.TitleBar().Height: got %.1f, want 46", barR.H)
	}
	if leadR.X < 76 {
		t.Fatalf("leading slot overlapped macOS left safe inset: x=%.1f, want >= 76", leadR.X)
	}
	if leadR.X >= trailR.X {
		t.Fatalf("LTR leading x (%.1f) should be left of trailing x (%.1f)", leadR.X, trailR.X)
	}

	// Verify banner landmark and every interactive button has a non-empty accessible name.
	if _, ok := tt.AccessByRole(ui.RoleBanner, "Title bar"); !ok {
		t.Fatal("WindowTitleBar missing RoleBanner landmark")
	}
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleButton && info.Name == "" {
			t.Fatalf("unnamed button in WindowTitleBar access tree: %+v", info)
		}
	}

	// Click Pin, Minimize, Maximize (switches label to Restore), and Close.
	pinBtn := buttonInfo(t, tt, ui.RoleButton, "Pin window")
	tt.ClickAt(center(pinBtn.Bounds))
	tt.Frame()
	if !pinned || !win.pinned || pinEvents != 1 {
		t.Fatalf("pin click state = (pinned:%v win:%v events:%d), want (true, true, 1)", pinned, win.pinned, pinEvents)
	}

	minBtn := buttonInfo(t, tt, ui.RoleButton, "Minimize")
	tt.ClickAt(center(minBtn.Bounds))
	tt.Frame()
	if win.minimized != 1 {
		t.Fatalf("win.minimized = %d, want 1", win.minimized)
	}

	maxBtn := buttonInfo(t, tt, ui.RoleButton, "Maximize")
	tt.ClickAt(center(maxBtn.Bounds))
	tt.Frame()
	if win.toggled != 1 || !win.maximized {
		t.Fatalf("maximize state = (toggled:%d maximized:%v), want (1, true)", win.toggled, win.maximized)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Restore"); !ok {
		t.Fatal("maximized window button label did not switch to Restore")
	}

	closeBtn := buttonInfo(t, tt, ui.RoleButton, "Close")
	tt.ClickAt(center(closeBtn.Bounds))
	tt.Frame()
	if win.closed != 1 {
		t.Fatalf("win.closed = %d, want 1", win.closed)
	}

	// RTL reverses logical Leading/Trailing placement inside safe insets.
	tt.SetDirection(ui.RightToLeft)
	tt.Frame()
	if leadR.X <= trailR.X {
		t.Fatalf("RTL leading x (%.1f) should be right of trailing x (%.1f)", leadR.X, trailR.X)
	}
	tt.Move(0, 0)
	tt.Advance(250 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("WindowTitleBar left pending frames after settling")
	}
}

func TestWindowTitleBarPureCallbacksAndDensityHeights(t *testing.T) {
	for _, tc := range []struct {
		density tokens.Density
		wantH   float32
	}{
		{tokens.DensityComfortable, 44},
		{tokens.DensityDesktop, 38},
		{tokens.DensityCompact, 34},
	} {
		th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tc.density})
		var barR ui.Rect
		minCalled, maxCalled, closeCalled := 0, 0, 0
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				WindowTitleBar(c, WindowTitleBarOptions{
					Title:      "App",
					Maximized:  true,
					OnMinimize: func() { minCalled++ },
					OnMaximize: func() { maxCalled++ },
					OnClose:    func() { closeCalled++ },
				}).DrawOver(func(_ *ui.Painter, r ui.Rect) { barR = r })
			})
		}, 480, 120)

		if absF32(barR.H-tc.wantH) > 0.5 {
			t.Fatalf("density %v title bar height = %.1f, want %.1f", tc.density, barR.H, tc.wantH)
		}
		tt.ClickAt(center(buttonInfo(t, tt, ui.RoleButton, "Minimize").Bounds))
		tt.ClickAt(center(buttonInfo(t, tt, ui.RoleButton, "Restore").Bounds))
		tt.ClickAt(center(buttonInfo(t, tt, ui.RoleButton, "Close").Bounds))
		tt.Frame()
		if minCalled != 1 || maxCalled != 1 || closeCalled != 1 {
			t.Fatalf("density %v callbacks = (%d,%d,%d), want (1,1,1)", tc.density, minCalled, maxCalled, closeCalled)
		}
	}
}

func TestWindowTitleBarButtonsThemeSync(t *testing.T) {
	win := &stubThemedWindowController{}
	dark := false
	var customBg ui.Color

	tt := ui.NewTester(func(c *ui.Context) {
		th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{
			Dark:         dark,
			Density:      tokens.DensityDesktop,
			ReduceMotion: true,
		})
		ThemeScope(c, th, func() {
			WindowTitleBar(c, WindowTitleBarOptions{
				Title:      "SyncTest",
				Background: customBg,
				Window:     win,
			})
		})
	}, 640, 120)

	// Zero TitleBar simulates HideWindowButtons: true; SetWindowButtonsTheme must stay untouched.
	if win.themeCalls != 0 {
		t.Fatalf("expected 0 SetWindowButtonsTheme calls when TitleBar is zero, got %d", win.themeCalls)
	}

	// Simulate visible native Windows caption buttons (HideWindowButtons: false).
	tt.SetTitleBar(ui.TitleBar{Height: 32, Right: 138, Bottom: 32})
	tt.Frame()
	if win.buttonsTheme != mygo.ThemeLight || win.themeCalls != 1 {
		t.Fatalf("light mode buttonsTheme = %q (calls=%d), want %q (1)", win.buttonsTheme, win.themeCalls, mygo.ThemeLight)
	}

	// Steady frame with unchanged theme should not re-dispatch SetWindowButtonsTheme.
	tt.Frame()
	if win.themeCalls != 1 {
		t.Fatalf("steady frame triggered redundant SetWindowButtonsTheme call: calls=%d", win.themeCalls)
	}

	// Switching to dark mode updates native buttons to ThemeDark.
	dark = true
	tt.Frame()
	if win.buttonsTheme != mygo.ThemeDark || win.themeCalls != 2 {
		t.Fatalf("dark mode buttonsTheme = %q (calls=%d), want %q (2)", win.buttonsTheme, win.themeCalls, mygo.ThemeDark)
	}

	// Switching back to light mode restores ThemeLight (prevents white-on-light invisible glyphs).
	dark = false
	tt.Frame()
	if win.buttonsTheme != mygo.ThemeLight || win.themeCalls != 3 {
		t.Fatalf("restored light mode buttonsTheme = %q (calls=%d), want %q (3)", win.buttonsTheme, win.themeCalls, mygo.ThemeLight)
	}

	// Custom dark background on a light theme still adapts button polarity to dark background.
	customBg = ui.Hex("#141218")
	tt.Frame()
	if win.buttonsTheme != mygo.ThemeDark || win.themeCalls != 4 {
		t.Fatalf("custom dark bg buttonsTheme = %q (calls=%d), want %q (4)", win.buttonsTheme, win.themeCalls, mygo.ThemeDark)
	}

	// Returning to zero TitleBar (fullscreen / HideWindowButtons) resets sync state without extra call.
	customBg = ui.Color{}
	tt.SetTitleBar(ui.TitleBar{})
	tt.Frame()
	if win.themeCalls != 4 {
		t.Fatalf("zero TitleBar made unexpected SetWindowButtonsTheme call: %d", win.themeCalls)
	}
}

func TestWindowTitleBarNarrowScreenTitleCollapseAndFocusRing(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{
		Dark:    false,
		Density: tokens.DensityDesktop,
	})
	pinned := false
	win := &stubWindowController{}
	var leadR, trailR ui.Rect

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Fill().Children(func() {
				WindowTitleBar(c, WindowTitleBarOptions{
					Title:              "PulseDesk — MD3E Demo",
					Divider:            true,
					Window:             win,
					ShowWindowControls: true,
					Pinned:             &pinned,
					Leading: func() ui.Element {
						return ui.Box(c).Size(76, 24).DrawOver(func(_ *ui.Painter, r ui.Rect) { leadR = r })
					},
					Trailing: func() ui.Element {
						return ui.Box(c).Size(88, 24).DrawOver(func(_ *ui.Painter, r ui.Rect) { trailR = r })
					},
				})
			})
		})
	}, 640, 120)

	if !tt.HasText("PulseDesk — MD3E Demo") {
		t.Fatal("wide 640 DIP title bar did not render centered Title")
	}

	// Threshold 520 DIP still displays centered Title.
	tt.SetSize(520, 120)
	tt.Frame()
	if !tt.HasText("PulseDesk — MD3E Demo") {
		t.Fatal("520 DIP title bar unexpectedly hid centered Title")
	}

	// Below 520 DIP with crowded controls, centered Title collapses gracefully with zero overlap.
	tt.SetSize(420, 120)
	tt.Frame()
	if tt.HasText("PulseDesk — MD3E Demo") {
		t.Fatal("narrow 420 DIP crowded title bar should collapse centered Title to prevent overlap")
	}
	pinBtn := buttonInfo(t, tt, ui.RoleButton, "Pin window")
	minBtn := buttonInfo(t, tt, ui.RoleButton, "Minimize")
	maxBtn := buttonInfo(t, tt, ui.RoleButton, "Maximize")
	closeBtn := buttonInfo(t, tt, ui.RoleButton, "Close")

	if leadR.X+leadR.W > trailR.X+0.5 {
		t.Fatalf("narrow bar leading %+v overlapped trailing %+v", leadR, trailR)
	}
	if trailR.X+trailR.W > pinBtn.Bounds.X+0.5 {
		t.Fatalf("narrow bar trailing %+v overlapped pin %+v", trailR, pinBtn.Bounds)
	}
	if pinBtn.Bounds.X+pinBtn.Bounds.W > minBtn.Bounds.X+0.5 ||
		minBtn.Bounds.X+minBtn.Bounds.W > maxBtn.Bounds.X+0.5 ||
		maxBtn.Bounds.X+maxBtn.Bounds.W > closeBtn.Bounds.X+0.5 ||
		closeBtn.Bounds.X+closeBtn.Bounds.W > 420.5 {
		t.Fatalf("narrow bar window controls overlapped or overflowed 420 DIP: pin=%+v min=%+v max=%+v close=%+v",
			pinBtn.Bounds, minBtn.Bounds, maxBtn.Bounds, closeBtn.Bounds)
	}

	// Uncrowded narrow bar (< 520 DIP with no Leading/Trailing) still shows Title.
	ttUncrowded := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			WindowTitleBar(c, WindowTitleBarOptions{
				Title:              "Compact App",
				ShowWindowControls: true,
			})
		})
	}, 480, 120)
	if !ttUncrowded.HasText("Compact App") {
		t.Fatal("uncrowded 480 DIP title bar should keep Title visible")
	}

	// Keyboard Tab focus onto Pin / Minimize buttons must not draw a raw mygo focus ring.
	focusColor := th.ToUITheme().Focus
	checkNoFocusRing := func(name string, b ui.Rect) {
		t.Helper()
		info := buttonInfo(t, tt, ui.RoleButton, name)
		if !info.Focused {
			t.Fatalf("%s did not receive keyboard focus on Tab: %+v", name, info)
		}
		// Sample 3 DIP outside the button where ui.Painter.FocusRing (offset 2, width 2) paints.
		img := tt.Image()
		pts := [][2]int{
			{int(b.X + b.W/2), max(0, int(b.Y-3))},
			{int(b.X + b.W/2), min(119, int(b.Y+b.H+3))},
			{max(0, int(b.X-3)), int(b.Y + b.H/2)},
			{min(419, int(b.X+b.W+3)), int(b.Y + b.H/2)},
		}
		for _, pt := range pts {
			if px := img.RGBAAt(pt[0], pt[1]); colorMatches(px, focusColor, 12) {
				t.Fatalf("%s painted unexpected raw focus ring pixel %+v at (%d,%d) matching %+v",
					name, px, pt[0], pt[1], focusColor)
			}
		}
	}
	tt.Key(0, ui.KeyTab)
	tt.Advance(250 * time.Millisecond)
	checkNoFocusRing("Pin window", pinBtn.Bounds)
	tt.Key(0, ui.KeyTab)
	tt.Advance(250 * time.Millisecond)
	checkNoFocusRing("Minimize", minBtn.Bounds)

	// Hovering over Close button settles without leaving pending frames once cursor leaves.
	tt.Move(center(closeBtn.Bounds))
	tt.Advance(250 * time.Millisecond)
	tt.Move(0, 0)
	tt.Advance(250 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("WindowTitleBar left pending frames after hover leave settled")
	}
}
