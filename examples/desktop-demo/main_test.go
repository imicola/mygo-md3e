package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// newTester renders PulseDesk headless and parks the engine, so the
// indeterminate wavy progress and the pulsing status pill stop asking for
// frames and every later convergence check sees a still frame.
func newTester(t *testing.T) (*appState, *ui.Tester) {
	t.Helper()
	a := newApp()
	tt := ui.NewTester(a.view, 1120, 740)
	tt.Frame()
	if !tt.HasText("PulseDesk Engine") {
		t.Fatal("the overview hero card is missing")
	}
	if err := tt.Click("Engine playback"); err != nil {
		t.Fatalf("pause the engine: %v", err)
	}
	tt.Advance(700 * time.Millisecond)
	if a.running {
		t.Fatal("Engine playback did not pause the engine")
	}
	return a, tt
}

// hasField reports whether a named text field is in the accessibility tree.
func hasField(tt *ui.Tester, name string) bool {
	_, ok := tt.AccessByRole(ui.RoleTextField, name)
	return ok
}

// hasProgress reports whether a named progress element is present.
func hasProgress(tt *ui.Tester, name string) bool {
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleProgress && info.Name == name {
			return true
		}
	}
	return false
}

// checkNamedControls fails on an interactive control without an accessible
// name.
func checkNamedControls(t *testing.T, tt *ui.Tester, where string) {
	t.Helper()
	for _, info := range tt.AccessAll() {
		switch info.Role {
		case ui.RoleButton, ui.RoleToggleButton, ui.RoleTab, ui.RoleRadio,
			ui.RoleCheckBox, ui.RoleSwitch, ui.RoleSlider, ui.RoleTextField:
			if info.Name == "" {
				t.Fatalf("%s: a %v at %v has no accessible name", where, info.Role, info.Bounds)
			}
		}
	}
}

// TestDensityProfiles switches the three density profiles from the title bar
// and checks the theme and the title bar height follow. It runs on the
// workspace page, whose still indicators settle between switches.
func TestDensityProfiles(t *testing.T) {
	a, tt := newTester(t)
	if err := tt.Click("Workspace"); err != nil {
		t.Fatalf("open the workspace page: %v", err)
	}
	tt.Advance(700 * time.Millisecond)

	for _, tc := range []struct {
		tab     string
		idx     int
		density tokens.Density
		barH    float32
	}{
		{"Comfort", 0, tokens.DensityComfortable, 44},
		{"Compact", 2, tokens.DensityCompact, 34},
		{"Desktop", 1, tokens.DensityDesktop, 38},
	} {
		if err := tt.Click(tc.tab); err != nil {
			t.Fatalf("click the %s density tab: %v", tc.tab, err)
		}
		tt.Advance(800 * time.Millisecond)
		if a.densityIdx != tc.idx {
			t.Fatalf("density index is %d after %s, want %d", a.densityIdx, tc.tab, tc.idx)
		}
		if got := a.currentTheme().Density; got != tc.density {
			t.Fatalf("%s builds theme density %v, want %v", tc.tab, got, tc.density)
		}
		bar, ok := tt.AccessByRole(ui.RoleBanner, "Title bar")
		if !ok {
			t.Fatalf("%s: the title bar landmark is missing", tc.tab)
		}
		if diff := bar.Bounds.H - tc.barH; diff > 0.5 || diff < -0.5 {
			t.Fatalf("%s title bar height is %.1f, want %.1f", tc.tab, bar.Bounds.H, tc.barH)
		}
		if tt.FramePending() {
			t.Fatalf("%s density keeps asking for frames", tc.tab)
		}
	}
}

// TestAdaptiveBreakpoints checks the scaffold and the list/detail pane across
// Compact, Medium and Expanded.
func TestAdaptiveBreakpoints(t *testing.T) {
	a, tt := newTester(t)

	// Expanded: expanded rail, both panes.
	tt.SetSize(1100, 740)
	tt.Frame()
	tt.Advance(500 * time.Millisecond)
	nav, ok := tt.AccessByRole(ui.RoleNavigation, "PulseDesk sections")
	if !ok {
		t.Fatal("the expanded shell has no named navigation landmark")
	}
	if nav.Bounds.W < 150 {
		t.Fatalf("expanded rail width is %.1f, want an expanded rail", nav.Bounds.W)
	}
	if err := tt.Click("Workspace"); err != nil {
		t.Fatalf("open the workspace page: %v", err)
	}
	tt.Advance(700 * time.Millisecond)
	if !tt.HasText("Node workspace") {
		t.Fatal("the workspace page did not open")
	}
	if !hasField(tt, "Search nodes") || !tt.HasText("Profile intensity") {
		t.Fatal("expanded list and detail panes are not both showing")
	}

	// Medium: collapsed rail, both panes still.
	tt.SetSize(720, 740)
	tt.Frame()
	tt.Advance(500 * time.Millisecond)
	nav, _ = tt.AccessByRole(ui.RoleNavigation, "PulseDesk sections")
	if nav.Bounds.W > 120 {
		t.Fatalf("medium rail width is %.1f, want a collapsed rail", nav.Bounds.W)
	}
	if !hasField(tt, "Search nodes") || !tt.HasText("Profile intensity") {
		t.Fatal("medium list and detail panes are not both showing")
	}

	// Compact: bottom navigation bar and a single pane.
	tt.SetSize(480, 740)
	tt.Frame()
	tt.Advance(500 * time.Millisecond)
	nav, ok = tt.AccessByRole(ui.RoleNavigation, "PulseDesk sections")
	if !ok {
		t.Fatal("the compact shell has no named navigation landmark")
	}
	if nav.Bounds.W < 400 || nav.Bounds.H < 60 {
		t.Fatalf("compact navigation is %v, want the bottom bar across the window", nav.Bounds)
	}
	if !hasField(tt, "Search nodes") {
		t.Fatal("the compact workspace should start on the list pane")
	}
	if tt.HasText("Profile intensity") {
		t.Fatal("the compact workspace shows both panes")
	}

	if err := tt.Click("Aurora Cluster"); err != nil {
		t.Fatalf("select a node in the compact list: %v", err)
	}
	tt.Advance(700 * time.Millisecond)
	if hasField(tt, "Search nodes") {
		t.Fatal("selecting a compact node did not switch to the detail pane")
	}
	if !tt.HasText("Profile intensity") {
		t.Fatal("the compact detail pane did not show")
	}

	if err := tt.Click("Back to list"); err != nil {
		t.Fatalf("go back to the compact list: %v", err)
	}
	tt.Advance(700 * time.Millisecond)
	if !hasField(tt, "Search nodes") {
		t.Fatal("Back to list did not return to the list pane")
	}

	// Expanded again: the rail widens and both panes return.
	tt.SetSize(1100, 740)
	tt.Frame()
	tt.Advance(500 * time.Millisecond)
	nav, _ = tt.AccessByRole(ui.RoleNavigation, "PulseDesk sections")
	if nav.Bounds.W < 150 {
		t.Fatalf("returning to expanded kept a %.1f rail", nav.Bounds.W)
	}
	if !hasField(tt, "Search nodes") || !tt.HasText("Profile intensity") {
		t.Fatal("returning to expanded did not restore both panes")
	}
	if a.page != 1 {
		t.Fatalf("the shell forgot the workspace page: page %d", a.page)
	}
}

// TestInteractionsToys exercises the morphing toggle, a switch card, the pill
// filters, the skeleton preview and the empty state.
func TestInteractions(t *testing.T) {
	a, tt := newTester(t)

	// MorphIconButton: pause -> play -> pause.
	if err := tt.Click("Engine playback"); err != nil {
		t.Fatalf("play the engine: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if !a.running {
		t.Fatal("Engine playback did not resume the engine")
	}
	if err := tt.Click("Engine playback"); err != nil {
		t.Fatalf("pause the engine again: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if a.running {
		t.Fatal("Engine playback did not pause the engine again")
	}

	// SwitchCard on the overview.
	if err := tt.Click("Focus mode"); err != nil {
		t.Fatalf("toggle Focus mode: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if !a.focusMode {
		t.Fatal("the Focus mode switch card did not turn on")
	}

	if err := tt.Click("Workspace"); err != nil {
		t.Fatalf("open the workspace page: %v", err)
	}
	tt.Advance(700 * time.Millisecond)

	// Search with no match shows the empty state; its reset restores the list.
	field, ok := tt.AccessByRole(ui.RoleTextField, "Search nodes")
	if !ok {
		t.Fatal("the node search field is missing")
	}
	tt.ClickAt(centerOf(field.Bounds))
	tt.Advance(100 * time.Millisecond)
	tt.Type("zzz")
	tt.Advance(500 * time.Millisecond)
	if a.query != "zzz" {
		t.Fatalf("the search field holds %q, want zzz", a.query)
	}
	if !tt.HasText("No nodes match") {
		t.Fatal("an unmatched search does not show the empty state")
	}
	if err := tt.Click("Reset filter"); err != nil {
		t.Fatalf("reset the filter: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if a.query != "" {
		t.Fatalf("Reset filter left the query at %q", a.query)
	}
	if !tt.HasText("Aurora Cluster") {
		t.Fatal("resetting the filter did not restore the list")
	}

	// Skeleton preview switch.
	if err := tt.Click("Loading preview"); err != nil {
		t.Fatalf("enable the loading preview: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if !a.skeleton {
		t.Fatal("the loading preview switch did not turn on")
	}
	if !hasProgress(tt, "Loading node avatar") {
		t.Fatal("the skeleton placeholders did not show")
	}
	if err := tt.Click("Loading preview"); err != nil {
		t.Fatalf("disable the loading preview: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if a.skeleton {
		t.Fatal("the loading preview switch did not turn off")
	}

	// PillTabs filter with counts.
	if err := tt.Click("Idle"); err != nil {
		t.Fatalf("filter to idle nodes: %v", err)
	}
	tt.Advance(900 * time.Millisecond)
	if a.tab != 2 {
		t.Fatalf("the filter tab is %d, want 2", a.tab)
	}
	if !tt.HasText("Comet Worker") {
		t.Fatal("the idle filter does not show Comet Worker")
	}
	if tt.HasText("Beacon Relay") {
		t.Fatal("the idle filter still shows the active Beacon Relay")
	}
}

// TestReduceMotionAndAccessibility turns ReduceMotion on and checks the demo
// settles into still frames with named, roled interactive controls.
func TestReduceMotionAndAccessibility(t *testing.T) {
	a, tt := newTester(t)

	if err := tt.Click("Components"); err != nil {
		t.Fatalf("open the component lab: %v", err)
	}
	tt.Advance(700 * time.Millisecond)
	if !tt.HasText("Component lab") {
		t.Fatal("the component lab did not open")
	}

	// Reduce motion is the last section: scroll the page to its bottom.
	tt.Wheel(600, 400, 0, 20000)
	tt.Advance(400 * time.Millisecond)
	if err := tt.Click("Reduce motion"); err != nil {
		t.Fatalf("turn Reduce motion on: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if !a.reduce {
		t.Fatal("the Reduce motion switch card did not turn on")
	}
	if !a.currentTheme().ReduceMotion {
		t.Fatal("the theme does not carry ReduceMotion")
	}

	tt.Advance(1500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("ReduceMotion left animations asking for frames")
	}

	checkNamedControls(t, tt, "component lab")
	if _, ok := tt.AccessByRole(ui.RoleSwitch, "Reduce motion"); !ok {
		t.Fatal("the Reduce motion control is not a named switch")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Shape heart"); !ok {
		t.Fatal("a shape button is not a named button")
	}
	if _, ok := tt.AccessByRole(ui.RoleTab, "Compact"); !ok {
		t.Fatal("the density tabs are not named tabs")
	}
}

// centerOf is the middle of a rect.
func centerOf(r ui.Rect) (float32, float32) {
	return r.X + r.W/2, r.Y + r.H/2
}
