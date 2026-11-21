package md3e

import (
	"testing"
	"time"

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

func (s *stubWindowController) Minimize()              { s.minimized++ }
func (s *stubWindowController) ToggleMaximize()        { s.toggled++; s.maximized = !s.maximized }
func (s *stubWindowController) Close()                 { s.closed++ }
func (s *stubWindowController) IsMaximized() bool      { return s.maximized }
func (s *stubWindowController) SetAlwaysOnTop(v bool)  { s.pinned = v }

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
