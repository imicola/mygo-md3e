package main

import (
	"image"
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

// TestAnimatedThemeAndSeedTransitions exercises AnimatedThemeScope across light/dark
// and seed switches, verifying smooth mid-transition interpolation and zero
// lingering frames once settled.
func TestAnimatedThemeAndSeedTransitions(t *testing.T) {
	a, tt := newTester(t)
	darkLum := meanLuma(tt.Image())

	// Toggle Dark -> Light on Overview and verify mid-transition luminance interpolation.
	if err := tt.Click("Dark mode"); err != nil {
		t.Fatalf("toggle Dark mode to light: %v", err)
	}
	if a.dark {
		t.Fatal("a.dark stayed true after clicking Dark mode")
	}
	if !tt.FramePending() {
		t.Fatal("AnimatedThemeScope did not schedule frames on dark -> light switch")
	}
	tt.Advance(48 * time.Millisecond)
	midLum := meanLuma(tt.Image())

	tt.Advance(700 * time.Millisecond)
	if a.currentTheme().Dark {
		t.Fatal("currentTheme().Dark remained true in light mode")
	}
	lightLum := meanLuma(tt.Image())
	if !(darkLum < midLum && midLum < lightLum) {
		t.Fatalf("expected monotonic luminance across transition: dark=%.1f mid=%.1f light=%.1f", darkLum, midLum, lightLum)
	}

	// Switch to Workspace (still indicators) to verify AnimatedThemeScope itself
	// converges to zero pending frames on Seed and Light/Dark switches.
	if err := tt.Click("Workspace"); err != nil {
		t.Fatalf("open Workspace: %v", err)
	}
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("Workspace still has pending frames after settle")
	}

	// Cycle Seed color while in light mode.
	if err := tt.Click("Seed color: Nebula"); err != nil {
		t.Fatalf("cycle seed color: %v", err)
	}
	if a.seedIdx != 1 {
		t.Fatalf("seedIdx = %d, want 1", a.seedIdx)
	}
	if !tt.FramePending() {
		t.Fatal("AnimatedThemeScope did not schedule frames on seed color change")
	}
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("AnimatedThemeScope still has pending frames after seed transition")
	}

	// Toggle Light -> Dark on Workspace and verify clean convergence.
	if err := tt.Click("Dark mode"); err != nil {
		t.Fatalf("toggle Dark mode back to dark: %v", err)
	}
	if !tt.FramePending() {
		t.Fatal("AnimatedThemeScope did not schedule frames on light -> dark switch")
	}
	tt.Advance(700 * time.Millisecond)
	if !a.dark || tt.FramePending() {
		t.Fatalf("expected dark mode settled: dark=%v pending=%v", a.dark, tt.FramePending())
	}
}

// TestPageTransitionAndHeroProgressModes exercises PageTransition across all
// three pages and verifies the Hero progress bar modes (two-segment continuous
// stream, continuous 0%->100% + Buffer progression, and Wavy <-> Linear toggle).
func TestPageTransitionAndHeroProgressModes(t *testing.T) {
	a, tt := newTester(t)

	// Paused Stream mode freezes on determinate 68% + 86% buffer.
	if !tt.HasText("68% · buf 86%") {
		t.Fatalf("paused hero card missing determinate buffer readout: %q", tt.Texts())
	}

	// Resume playback: indeterminate Stream mode keeps requesting frames.
	if err := tt.Click("Engine playback"); err != nil {
		t.Fatalf("resume Engine playback: %v", err)
	}
	tt.Advance(160 * time.Millisecond)
	if !a.running || !tt.FramePending() {
		t.Fatalf("resumed stream mode should run continuously: running=%v pending=%v", a.running, tt.FramePending())
	}
	progInfo, ok := accessProgress(tt, "Workspace sync")
	if !ok || progInfo.Now >= 0 {
		t.Fatalf("stream mode Workspace sync progress = (%v, %+v), want indeterminate (<0)", ok, progInfo)
	}

	// Switch to 0->100% + Buffer continuous progression.
	if err := tt.Click("0→100%"); err != nil {
		t.Fatalf("select 0->100%% tab: %v", err)
	}
	tt.Advance(80 * time.Millisecond)
	p1, ok := accessProgress(tt, "Workspace sync")
	if !ok || p1.Now < 0 || p1.Now > 1 {
		t.Fatalf("0->100%% initial progress = (%v, %+v), want 0..1", ok, p1)
	}
	for range 8 {
		tt.Advance(80 * time.Millisecond)
	}
	p2, ok := accessProgress(tt, "Workspace sync")
	if !ok || p2.Now <= p1.Now {
		t.Fatalf("0->100%% progress did not advance: p1=%.3f p2=%.3f", p1.Now, p2.Now)
	}

	// Switch Wavy -> Linear progress bar comparison.
	if err := tt.Click("Linear"); err != nil {
		t.Fatalf("select Linear progress tab: %v", err)
	}
	tt.Advance(120 * time.Millisecond)
	if a.progressStyle != 1 {
		t.Fatalf("progressStyle = %d, want 1 (Linear)", a.progressStyle)
	}
	if _, ok := accessProgress(tt, "Workspace sync"); !ok {
		t.Fatal("Workspace sync missing after switching to Linear")
	}

	// Pause engine: syncPhase must freeze without advancing over time.
	if err := tt.Click("Engine playback"); err != nil {
		t.Fatalf("pause Engine playback: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	frozenA, ok := accessProgress(tt, "Workspace sync")
	if !ok {
		t.Fatal("Workspace sync missing after pause")
	}
	tt.Advance(500 * time.Millisecond)
	frozenB, ok := accessProgress(tt, "Workspace sync")
	if !ok || frozenB.Now != frozenA.Now {
		t.Fatalf("paused progress kept changing: before=%.4f after=%.4f", frozenA.Now, frozenB.Now)
	}

	// Verify PageTransition animates and settles cleanly across Overview -> Workspace -> Components -> Overview.
	for _, step := range []struct {
		nav          string
		want         string
		page         int
		expectSettle bool
	}{
		{"Workspace", "Node workspace", 1, true},
		{"Components", "Component lab", 2, false}, // Components hosts a live pulsing StatusPill until ReduceMotion
		{"Overview", "PulseDesk Engine", 0, false},
	} {
		if err := tt.Click(step.nav); err != nil {
			t.Fatalf("navigate to %s: %v", step.nav, err)
		}
		if a.page != step.page {
			t.Fatalf("a.page = %d after clicking %s, want %d", a.page, step.nav, step.page)
		}
		if !tt.FramePending() {
			t.Fatalf("switching to %s did not trigger PageTransition/NavigationRail frames", step.nav)
		}
		tt.Advance(700 * time.Millisecond)
		if !tt.HasText(step.want) {
			t.Fatalf("page %s missing expected text %q", step.nav, step.want)
		}
		if step.expectSettle && tt.FramePending() {
			t.Fatalf("page %s left pending frames after transition settled", step.nav)
		}
	}

	// Enable ReduceMotion and verify PageTransition settles immediately with zero pending frames.
	a.reduce = true
	if err := tt.Click("Workspace"); err != nil {
		t.Fatalf("navigate to Workspace with ReduceMotion: %v", err)
	}
	if tt.FramePending() {
		t.Fatal("PageTransition with ReduceMotion should swap immediately with zero pending frames")
	}
	if !tt.HasText("Node workspace") {
		t.Fatal("Workspace page missing under ReduceMotion")
	}
}

// TestNarrowScreenAndVisualRefinements verifies narrow title collapse (< 520 DIP),
// workspace scrollbar gutter clearance, Gem ShapeButton visibility, and
// comfortable bottom scroll padding.
func TestNarrowScreenAndVisualRefinements(t *testing.T) {
	_, tt := newTester(t)

	if !tt.HasText("PulseDesk — MD3E Demo") {
		t.Fatal("wide window should show full centered title")
	}

	// Narrow window (420 DIP MinWidth): centered title collapses gracefully while
	// leading mark and compact theme controls fit within [0, 420].
	tt.SetSize(420, 740)
	tt.Frame()
	tt.Advance(500 * time.Millisecond)
	if tt.HasText("PulseDesk — MD3E Demo") {
		t.Fatal("narrow 420 DIP window should collapse crowded centered title")
	}
	if !tt.HasText("PulseDesk") {
		t.Fatal("narrow 420 DIP window should keep leading PulseDesk brand mark")
	}
	for _, btnName := range []string{"Seed color: Nebula", "Dark mode", "Density: Desktop"} {
		info, ok := tt.AccessByRole(ui.RoleButton, btnName)
		if !ok && btnName == "Dark mode" {
			info, ok = tt.AccessByRole(ui.RoleToggleButton, btnName)
		}
		if !ok {
			t.Fatalf("narrow 420 DIP title bar missing control %q", btnName)
		}
		if info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > 420.5 {
			t.Fatalf("narrow 420 DIP control %q out of bounds: %+v", btnName, info.Bounds)
		}
	}

	// Expanding >= 520 DIP restores the centered window title.
	tt.SetSize(1120, 740)
	tt.Frame()
	tt.Advance(500 * time.Millisecond)
	if !tt.HasText("PulseDesk — MD3E Demo") {
		t.Fatal("expanding window back to 1120 DIP did not restore centered title")
	}

	// Workspace node-scroll has >= 14 DIP end padding so scrollbars never overlap ActionTile chevrons/meta.
	if err := tt.Click("Workspace"); err != nil {
		t.Fatalf("open Workspace: %v", err)
	}
	tt.Advance(700 * time.Millisecond)
	loadingSw, ok := tt.AccessByRole(ui.RoleSwitch, "Loading preview")
	if !ok {
		t.Fatal("Loading preview switch missing")
	}
	tile, ok := tt.AccessByRole(ui.RoleButton, "Aurora Cluster")
	if !ok {
		t.Fatal("Aurora Cluster ActionTile missing")
	}
	// SwitchCard has 1 DIP hairline border + Desktop PaddingEnd(14), so SwitchCard outer right edge is loadingSw right + 15.
	switchCardRight := loadingSw.Bounds.X + loadingSw.Bounds.W + 15
	rightClearance := switchCardRight - (tile.Bounds.X + tile.Bounds.W)
	if rightClearance < 13.5 {
		t.Fatalf("node-scroll trailing clearance = %.1f DIP, want >= 14 DIP for scrollbar safety", rightClearance)
	}

	// Components page: Gem ShapeButton has visible outline/surface pixels and bottom has scroll whitespace.
	if err := tt.Click("Components"); err != nil {
		t.Fatalf("open Components: %v", err)
	}
	tt.Advance(700 * time.Millisecond)
	gemBtn, ok := tt.AccessByRole(ui.RoleButton, "Shape gem")
	if !ok {
		t.Fatal("Shape gem button missing")
	}
	img := tt.Image()
	bgCol := img.RGBAAt(max(0, int(gemBtn.Bounds.X-6)), int(gemBtn.Bounds.Y+gemBtn.Bounds.H/2))
	cx, cy := centerOf(gemBtn.Bounds)
	outlineDiffs := 0
	for y := int(gemBtn.Bounds.Y); y <= int(gemBtn.Bounds.Y+gemBtn.Bounds.H); y++ {
		for x := int(gemBtn.Bounds.X); x <= int(gemBtn.Bounds.X+gemBtn.Bounds.W); x++ {
			dx := float32(x) - cx
			dy := float32(y) - cy
			if dx*dx+dy*dy < 11*11 {
				continue // skip centered checkmark icon; check outer polygon perimeter
			}
			px := img.RGBAAt(x, y)
			if absInt(int(px.R)-int(bgCol.R))+absInt(int(px.G)-int(bgCol.G))+absInt(int(px.B)-int(bgCol.B)) > 18 {
				outlineDiffs++
			}
		}
	}
	if outlineDiffs < 12 {
		t.Fatalf("Shape gem outer ring pixel diffs = %d, want >= 12 (visible outline/contour)", outlineDiffs)
	}

	// Scrolling to the bottom leaves comfortable bottom margin below the last switch card.
	tt.Wheel(600, 400, 0, 20000)
	tt.Advance(300 * time.Millisecond)
	sw, ok := tt.AccessByRole(ui.RoleSwitch, "Reduce motion")
	if !ok {
		t.Fatal("Reduce motion switch missing after bottom scroll")
	}
	bottomGap := 740 - (sw.Bounds.Y + sw.Bounds.H)
	if bottomGap < 40 {
		t.Fatalf("bottom scroll gap below Reduce motion = %.1f DIP, want >= 40 DIP", bottomGap)
	}
}

func accessProgress(tt *ui.Tester, name string) (ui.AccessInfo, bool) {
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleProgress && info.Name == name {
			return info, true
		}
	}
	return ui.AccessInfo{}, false
}

func meanLuma(img *image.RGBA) float64 {
	b := img.Bounds()
	var sum float64
	var count int
	for y := b.Min.Y; y < b.Max.Y; y += 4 {
		for x := b.Min.X; x < b.Max.X; x += 4 {
			c := img.RGBAAt(x, y)
			sum += 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// centerOf is the middle of a rect.
func centerOf(r ui.Rect) (float32, float32) {
	return r.X + r.W/2, r.Y + r.H/2
}
