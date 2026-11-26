package main

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

func newWorkbenchTester(t *testing.T, pause bool) (*appState, *ui.Tester) {
	t.Helper()
	a := newApp()
	t.Cleanup(a.media.close)
	tt := ui.NewTester(a.view, 1160, 760)
	tt.Frame()
	if !tt.HasText("Tunnel Core") {
		t.Fatal("workbench monitor hero card is missing")
	}
	if pause {
		if err := tt.Click("Engine toggle"); err != nil {
			t.Fatalf("pause engine: %v", err)
		}
		tt.Advance(800 * time.Millisecond)
		if a.mon.running {
			t.Fatal("Engine toggle did not pause live telemetry")
		}
	}
	return a, tt
}

func centerOf(r ui.Rect) (float32, float32) {
	return r.X + r.W/2, r.Y + r.H/2
}

func checkNamedControls(t *testing.T, tt *ui.Tester, where string) {
	t.Helper()
	for _, info := range tt.AccessAll() {
		switch info.Role {
		case ui.RoleButton, ui.RoleToggleButton, ui.RoleTab, ui.RoleRadio,
			ui.RoleCheckBox, ui.RoleSwitch, ui.RoleSlider, ui.RoleTextField,
			ui.RoleMeter, ui.RoleTable:
			if info.Name == "" {
				t.Fatalf("%s: a %v at %v has no accessible name", where, info.Role, info.Bounds)
			}
		}
	}
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

func TestWorkbenchMonitorLiveTelemetryAndPause(t *testing.T) {
	a, tt := newWorkbenchTester(t, false)

	// Verify warm-orange dark Desktop defaults.
	th := a.currentTheme()
	if !th.Dark || th.Density != tokens.DensityDesktop || seedPresets[a.seedIdx].name != "Amber" {
		t.Fatalf("unexpected defaults: dark=%v density=%v seed=%q", th.Dark, th.Density, seedPresets[a.seedIdx].name)
	}
	if _, ok := tt.AccessByRole(ui.RoleMeter, "Process memory breakdown"); !ok {
		t.Fatal("process memory StackedProgress meter is missing")
	}

	startTicks := a.mon.tickCount
	startSeq := a.mon.downRing.Sequence()

	// Advance virtual clock across two 500ms telemetry ticks.
	tt.Advance(1100 * time.Millisecond)
	if a.mon.tickCount <= startTicks {
		t.Fatalf("live telemetry did not advance: start=%d now=%d", startTicks, a.mon.tickCount)
	}
	if a.mon.downRing.Sequence() <= startSeq {
		t.Fatalf("downRing sequence did not advance: start=%d now=%d", startSeq, a.mon.downRing.Sequence())
	}

	// Pause via cookie MorphIconButton and verify zero pending frames once settled.
	if err := tt.Click("Engine toggle"); err != nil {
		t.Fatalf("pause engine: %v", err)
	}
	tt.Advance(800 * time.Millisecond)
	if a.mon.running {
		t.Fatal("Engine toggle did not pause monitor")
	}
	if !tt.HasText("Engine Paused") {
		t.Fatal("paused status text is missing")
	}
	if tt.FramePending() {
		t.Fatal("paused workbench monitor still requests animation frames")
	}

	frozenTicks := a.mon.tickCount
	tt.Advance(1500 * time.Millisecond)
	if a.mon.tickCount != frozenTicks {
		t.Fatalf("paused telemetry kept ticking: before=%d after=%d", frozenTicks, a.mon.tickCount)
	}

	// Resume playback.
	if err := tt.Click("Engine toggle"); err != nil {
		t.Fatalf("resume engine: %v", err)
	}
	tt.Advance(600 * time.Millisecond)
	if !a.mon.running || a.mon.tickCount <= frozenTicks {
		t.Fatalf("resumed telemetry did not advance: running=%v ticks=%d->%d", a.mon.running, frozenTicks, a.mon.tickCount)
	}
}

func TestWorkbenchDataTable10KAnd100K(t *testing.T) {
	a, tt := newWorkbenchTester(t, true)

	if len(a.mon.filtered) != scale10K {
		t.Fatalf("initial connection count = %d, want %d", len(a.mon.filtered), scale10K)
	}
	if _, ok := tt.AccessByRole(ui.RoleTable, "Connections virtual table"); !ok {
		t.Fatal("Connections DataTable is missing")
	}

	// Switch dataset scale to 100,000 rows and verify deep virtual scrolling.
	if err := tt.Click("100K rows"); err != nil {
		t.Fatalf("switch to 100K rows: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if len(a.mon.filtered) != scale100K {
		t.Fatalf("100K filtered count = %d, want %d", len(a.mon.filtered), scale100K)
	}

	// Scroll page down via outer margin so DataTable does not capture the wheel,
	// then scroll DataTable itself to row 50,000.
	tt.Wheel(205, 120, 0, 500)
	tt.Advance(400 * time.Millisecond)
	a.mon.connTableState.ScrollTo(50_000, ui.Start)
	tt.Frame()
	wantHost := a.mon.hostAt(50_000)
	if wantHost == "" || !tt.HasText(wantHost) {
		t.Fatalf("scrolled 100K row 50,000 host %q is not rendered", wantHost)
	}

	// Sort by Latency ascending, then descending.
	latHdr, ok := tt.AccessByRole(ui.RoleColumnHeader, "Latency")
	if !ok {
		t.Fatal("Latency column header missing")
	}
	tt.ClickAt(centerOf(latHdr.Bounds))
	tt.Advance(200 * time.Millisecond)
	if a.mon.connSort.Column != "Latency" || a.mon.connSort.Descending {
		t.Fatalf("first Latency click sort = %+v, want ascending Latency", a.mon.connSort)
	}
	lat0 := a.mon.conns[a.mon.filtered[0]].latency10
	lat1 := a.mon.conns[a.mon.filtered[1]].latency10
	if lat0 > lat1 {
		t.Fatalf("ascending Latency sort invalid: %d > %d", lat0, lat1)
	}

	latHdr, _ = tt.AccessByRole(ui.RoleColumnHeader, "Latency")
	tt.ClickAt(centerOf(latHdr.Bounds))
	tt.Advance(200 * time.Millisecond)
	if a.mon.connSort.Column != "Latency" || !a.mon.connSort.Descending {
		t.Fatalf("second Latency click sort = %+v, want descending Latency", a.mon.connSort)
	}
	d0 := a.mon.conns[a.mon.filtered[0]].latency10
	d1 := a.mon.conns[a.mon.filtered[1]].latency10
	if d0 < d1 {
		t.Fatalf("descending Latency sort invalid: %d < %d", d0, d1)
	}

	// Filter via TextField and FilterChip, then Reset.
	searchField, ok := tt.AccessByRole(ui.RoleTextField, "Filter host, process, or rule")
	if !ok {
		t.Fatal("ledger search field missing")
	}
	tt.ClickAt(centerOf(searchField.Bounds))
	tt.Type("github")
	tt.Advance(300 * time.Millisecond)
	if len(a.mon.filtered) == 0 || len(a.mon.filtered) >= scale100K {
		t.Fatalf("search 'github' yielded %d rows, want subset of 100K", len(a.mon.filtered))
	}
	for i := range min(20, len(a.mon.filtered)) {
		rec := &a.mon.conns[a.mon.filtered[i]]
		rowText := strings.ToLower(a.mon.hostPool[rec.hostIdx] + " " + ruleLabels[rec.ruleIdx])
		if !strings.Contains(rowText, "github") {
			t.Fatalf("filtered row %d (%q) does not match 'github'", i, rowText)
		}
	}

	if err := tt.Click("Proxy"); err != nil {
		t.Fatalf("click Proxy filter chip: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !a.mon.filterRoute[routeProxy] {
		t.Fatal("Proxy filter chip did not toggle on")
	}

	if err := tt.Click("Reset"); err != nil {
		t.Fatalf("click Reset filters: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.mon.query != "" || len(a.mon.filtered) != scale100K {
		t.Fatalf("Reset left query=%q count=%d, want empty & %d", a.mon.query, len(a.mon.filtered), scale100K)
	}

	// Select first visible row, scroll outer page to bottom so inspector strip is
	// inside the viewport, and verify Copy host writes the clipboard.
	a.mon.connTableState.ScrollTo(0, ui.Start)
	tt.Frame()
	firstHost := a.mon.hostAt(0)
	rowInfo, ok := tt.AccessByRole(ui.RoleRow, firstHost)
	if !ok {
		t.Fatalf("first connection row %q missing from accessibility tree", firstHost)
	}
	tt.ClickAt(centerOf(rowInfo.Bounds))
	tt.Advance(200 * time.Millisecond)
	if a.mon.selectedRow != 0 {
		t.Fatalf("selectedRow = %d after clicking first row, want 0", a.mon.selectedRow)
	}
	tt.Wheel(205, 120, 0, 20000)
	tt.Advance(400 * time.Millisecond)
	if err := tt.Click("Copy host"); err != nil {
		t.Fatalf("click Copy host: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if got := tt.Clipboard(); got != firstHost {
		t.Fatalf("clipboard = %q, want %q", got, firstHost)
	}

	// Switch to DNS Queries mode and verify 100K DNS table.
	if err := tt.Click("DNS Queries"); err != nil {
		t.Fatalf("switch to DNS Queries: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if a.mon.tableMode != modeDNS || len(a.mon.filtered) != scale100K {
		t.Fatalf("DNS mode=%d count=%d, want mode=%d count=%d", a.mon.tableMode, len(a.mon.filtered), modeDNS, scale100K)
	}
	if _, ok := tt.AccessByRole(ui.RoleTable, "DNS queries virtual table"); !ok {
		t.Fatal("DNS DataTable is missing")
	}

	// Scroll back to top and trigger Run GC button.
	tt.Wheel(205, 120, 0, -20000)
	tt.Advance(400 * time.Millisecond)
	beforeGC := a.mon.mem.numGC
	if err := tt.Click("Run GC"); err != nil {
		t.Fatalf("click Run GC: %v", err)
	}
	if a.mon.mem.numGC <= beforeGC {
		t.Fatalf("Run GC did not advance numGC: before=%d after=%d", beforeGC, a.mon.mem.numGC)
	}
	// Advance past the earlier "Copied ..." snackbar so the queued GC toast appears.
	tt.Advance(4200 * time.Millisecond)
	if !tt.HasText("GC complete") {
		t.Fatal("Run GC did not trigger confirmation snackbar")
	}
	tt.Advance(4500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("snackbar dismissal left pending frames")
	}
}

func TestWorkbenchBreakpointsThemeAndRTL(t *testing.T) {
	a, tt := newWorkbenchTester(t, true)

	// 1. Expanded (1160x760): expanded NavigationRail + centered title.
	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Workbench sections")
	if !ok || nav.Bounds.W < 150 {
		t.Fatalf("expanded nav = (%v, %+v), want expanded rail", ok, nav.Bounds)
	}
	if !tt.HasText("MyGo MD3E Workbench — Monitor") {
		t.Fatal("expanded title bar missing full centered title")
	}

	// 2. Medium (720x760): collapsed rail.
	tt.SetSize(720, 760)
	tt.Advance(500 * time.Millisecond)
	nav, _ = tt.AccessByRole(ui.RoleNavigation, "Workbench sections")
	if nav.Bounds.W > 120 {
		t.Fatalf("medium nav width = %.1f, want collapsed rail", nav.Bounds.W)
	}

	// 3. Compact (480x760): bottom NavigationBar + collapsed centered title.
	tt.SetSize(480, 760)
	tt.Advance(500 * time.Millisecond)
	nav, _ = tt.AccessByRole(ui.RoleNavigation, "Workbench sections")
	if nav.Bounds.W < 400 || nav.Bounds.H < 60 {
		t.Fatalf("compact nav bounds = %+v, want bottom NavigationBar", nav.Bounds)
	}
	if tt.HasText("MyGo MD3E Workbench — Monitor") {
		t.Fatal("compact 480 DIP window should collapse centered title")
	}
	if !tt.HasText("Workbench") {
		t.Fatal("compact window should retain leading brand mark")
	}

	// 4. Open Preferences AlertDialog, change Density/Variant/ReduceMotion/RTL.
	if err := tt.Click("Preferences"); err != nil {
		t.Fatalf("open Preferences dialog: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, "Workbench preferences"); !ok {
		t.Fatal("Preferences AlertDialog did not open")
	}
	checkNamedControls(t, tt, "preferences dialog")

	if err := tt.Click("Compact"); err != nil {
		t.Fatalf("select Compact density: %v", err)
	}
	if err := tt.Click("Vibrant"); err != nil {
		t.Fatalf("select Vibrant variant: %v", err)
	}
	if err := tt.Click("Reduce motion"); err != nil {
		t.Fatalf("enable Reduce motion: %v", err)
	}
	if err := tt.Click("Right-to-left (RTL)"); err != nil {
		t.Fatalf("enable RTL: %v", err)
	}
	if err := tt.Click("Done"); err != nil {
		t.Fatalf("close Preferences dialog: %v", err)
	}
	tt.Advance(500 * time.Millisecond)

	if a.densityIdx != 2 || a.variantIdx != 2 || !a.reduce || !a.rtl {
		t.Fatalf("preferences state mismatch: density=%d variant=%d reduce=%v rtl=%v",
			a.densityIdx, a.variantIdx, a.reduce, a.rtl)
	}

	bar, ok := tt.AccessByRole(ui.RoleBanner, "Title bar")
	if !ok || bar.Bounds.H < 33.5 || bar.Bounds.H > 34.5 {
		t.Fatalf("Compact title bar height = %+v, want ~34 DIP", bar.Bounds)
	}

	// Return to Expanded and verify RTL NavigationRail is placed on the right side.
	tt.SetSize(1160, 760)
	tt.Advance(400 * time.Millisecond)
	nav, ok = tt.AccessByRole(ui.RoleNavigation, "Workbench sections")
	if !ok || nav.Bounds.X < 700 {
		t.Fatalf("RTL expanded rail at X=%.1f, want right edge", nav.Bounds.X)
	}

	// ReduceMotion + paused monitor leaves zero pending frames.
	if tt.FramePending() {
		t.Fatal("ReduceMotion + paused workbench left pending frames")
	}

	// Switch Dark -> Light and verify canvas luminance rises.
	darkLuma := meanLuma(tt.Image())
	if err := tt.Click("Dark mode"); err != nil {
		t.Fatalf("switch to Light mode: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	lightLuma := meanLuma(tt.Image())
	if lightLuma <= darkLuma+30 {
		t.Fatalf("light mode luma (%.1f) did not increase meaningfully over dark mode (%.1f)", lightLuma, darkLuma)
	}
}

func TestWorkbenchAccessibility(t *testing.T) {
	a, tt := newWorkbenchTester(t, true)
	checkNamedControls(t, tt, "monitor connections")

	if err := tt.Click("DNS Queries"); err != nil {
		t.Fatalf("switch to DNS Queries: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	checkNamedControls(t, tt, "monitor dns")

	// Verify ReduceMotion on non-paused transitions settles cleanly on mode switch.
	a.reduce = true
	if err := tt.Click("Connections"); err != nil {
		t.Fatalf("switch back to Connections: %v", err)
	}
	tt.Advance(100 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("ReduceMotion left pending frames after tab switch")
	}
}
