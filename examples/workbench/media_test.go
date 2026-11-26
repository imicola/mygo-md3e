package main

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func openMediaStudio(t *testing.T, pauseMonitor bool) (*appState, *ui.Tester) {
	t.Helper()
	a, tt := newWorkbenchTester(t, pauseMonitor)
	t.Cleanup(a.media.close)

	if err := tt.Click("Media"); err != nil {
		t.Fatalf("navigate to Media page: %v", err)
	}
	tt.Advance(900 * time.Millisecond)
	if a.page != 1 {
		t.Fatalf("a.page = %d after clicking Media, want 1", a.page)
	}
	if !tt.HasText("Media & Gallery Studio") {
		t.Fatal("Media & Gallery Studio header missing")
	}
	return a, tt
}

func clickInsideRegion(t *testing.T, tt *ui.Tester, regionName string, role ui.Role, targetName string) {
	t.Helper()
	reg, ok := tt.AccessByRole(ui.RoleRegion, regionName)
	if !ok {
		t.Fatalf("region %q missing from accessibility tree", regionName)
	}
	for _, info := range tt.AccessAll() {
		if info.Role != role || info.Name != targetName {
			continue
		}
		cx, cy := centerOf(info.Bounds)
		if cx >= reg.Bounds.X && cx <= reg.Bounds.X+reg.Bounds.W &&
			cy >= reg.Bounds.Y && cy <= reg.Bounds.Y+reg.Bounds.H {
			tt.ClickAt(cx, cy)
			return
		}
	}
	t.Fatalf("control (%v, %q) not found inside region %q (%+v)", role, targetName, regionName, reg.Bounds)
}

func samplePixelInRegion(t *testing.T, tt *ui.Tester, regionName string, rx, ry float32) color.RGBA {
	t.Helper()
	reg, ok := tt.AccessByRole(ui.RoleRegion, regionName)
	if !ok {
		t.Fatalf("region %q missing", regionName)
	}
	px := int(reg.Bounds.X + reg.Bounds.W*rx)
	py := int(reg.Bounds.Y + reg.Bounds.H*ry)
	return tt.Image().RGBAAt(px, py)
}

func TestWorkbenchMediaNavigationAndZeroIdleFrames(t *testing.T) {
	// Even when Monitor started in live telemetry mode, switching to Media stops
	// off-screen telemetry timers so the idle Media Studio produces 0 pending frames.
	a, tt := openMediaStudio(t, false)
	if !tt.HasText("MyGo MD3E Workbench — Media") {
		t.Fatal("WindowTitleBar did not update to Media title")
	}
	if !tt.HasText("Dynamic Cover Seed") || !tt.HasText("Expressive Video Stage") || !tt.HasText("Expressive Artwork Gallery") || !tt.HasText("Play Queue") {
		t.Fatal("expected all Media Studio sections and Play Queue sidebar")
	}
	if tt.FramePending() {
		t.Fatal("idle Media Studio should settle to FramePending() == false")
	}
	checkNamedControls(t, tt, "media studio")

	// Switching back to Monitor resumes live telemetry cleanly.
	if err := tt.Click("Monitor"); err != nil {
		t.Fatalf("navigate back to Monitor: %v", err)
	}
	tt.Advance(600 * time.Millisecond)
	if a.page != 0 || !tt.HasText("MyGo MD3E Workbench — Monitor") {
		t.Fatalf("did not return cleanly to Monitor page: page=%d", a.page)
	}
	ticksBefore := a.mon.tickCount
	tt.Advance(1100 * time.Millisecond)
	if a.mon.tickCount <= ticksBefore {
		t.Fatalf("Monitor live telemetry did not resume upon returning: before=%d after=%d", ticksBefore, a.mon.tickCount)
	}
}

func TestWorkbenchMediaDynamicSeedFromImageAndQueue(t *testing.T) {
	a, tt := openMediaStudio(t, true)

	// Verify all 6 procedural artworks extracted non-empty distinct seeds and 4-color palettes.
	seenSeeds := make(map[ui.Color]bool, len(a.media.artworks))
	for i, art := range a.media.artworks {
		if art.seed.A == 0 {
			t.Fatalf("artwork %d (%s) extracted empty SeedFromImage", i, art.title)
		}
		if len(art.palette) < 2 {
			t.Fatalf("artwork %d (%s) extracted too few PaletteFromImage colors: %d", i, art.title, len(art.palette))
		}
		seenSeeds[art.seed] = true
	}
	if len(seenSeeds) < 4 {
		t.Fatalf("expected diverse SeedFromImage colors across artworks, got only %d distinct", len(seenSeeds))
	}

	// Capture AudioPlayerCard dynamic pixel color on Solar Flare (warm amber),
	// then select Abyssal Tide (cool cyan) from Play Queue and verify AudioPlayerCard
	// re-themes dynamically via DynamicColor + SeedFromImage.
	solarBtnPx := func() color.RGBA {
		reg, ok := tt.AccessByRole(ui.RoleRegion, "Audio player")
		if !ok {
			t.Fatal("Audio player region missing")
		}
		for _, info := range tt.AccessAll() {
			if info.Role == ui.RoleToggleButton && info.Name == "Play" {
				cx, cy := centerOf(info.Bounds)
				if cx >= reg.Bounds.X && cx <= reg.Bounds.X+reg.Bounds.W && cy >= reg.Bounds.Y && cy <= reg.Bounds.Y+reg.Bounds.H {
					return tt.Image().RGBAAt(int(info.Bounds.X+6), int(cy))
				}
			}
		}
		t.Fatal("Play button inside Audio player missing")
		return color.RGBA{}
	}()

	if err := tt.Click("Abyssal Tide"); err != nil {
		t.Fatalf("select Abyssal Tide from Play Queue: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if a.media.activeTrack != 1 || a.media.activeArtwork().title != "Abyssal Tide" {
		t.Fatalf("activeTrack=%d (%s), want 1 (Abyssal Tide)", a.media.activeTrack, a.media.activeArtwork().title)
	}

	var abyssBtnPx color.RGBA
	{
		reg, _ := tt.AccessByRole(ui.RoleRegion, "Audio player")
		for _, info := range tt.AccessAll() {
			if info.Role == ui.RoleToggleButton && info.Name == "Play" {
				cx, cy := centerOf(info.Bounds)
				if cx >= reg.Bounds.X && cx <= reg.Bounds.X+reg.Bounds.W && cy >= reg.Bounds.Y && cy <= reg.Bounds.Y+reg.Bounds.H {
					abyssBtnPx = tt.Image().RGBAAt(int(info.Bounds.X+6), int(cy))
					break
				}
			}
		}
	}
	if solarBtnPx == abyssBtnPx {
		t.Fatalf("AudioPlayerCard DynamicColor button did not change between Solar Flare (%+v) and Abyssal Tide (%+v)", solarBtnPx, abyssBtnPx)
	}

	// Enable "Follow cover" so the entire workbench AnimatedThemeScope adopts the extracted seed.
	amberCanvasPx := samplePixelInRegion(t, tt, "Media studio and play queue", 0.02, 0.02)
	if err := tt.Click("Follow cover"); err != nil {
		t.Fatalf("enable Follow cover: %v", err)
	}
	tt.Advance(700 * time.Millisecond)
	if !a.media.followCoverTheme {
		t.Fatal("followCoverTheme did not toggle on")
	}
	wantSeed := a.media.artworks[1].seed
	if a.activeSeed() != wantSeed {
		t.Fatalf("activeSeed()=%+v, want Abyssal Tide seed %+v", a.activeSeed(), wantSeed)
	}
	if tt.FramePending() {
		t.Fatal("Follow cover transition did not settle cleanly")
	}
	coverCanvasPx := samplePixelInRegion(t, tt, "Media studio and play queue", 0.02, 0.02)
	_ = amberCanvasPx
	_ = coverCanvasPx

	// Click an extracted PaletteFromImage swatch and confirm snackbar + custom seed apply.
	swColor := a.media.inspectedArtwork().palette[1]
	swHex := mediaColorHex(swColor)
	if err := tt.Click("Apply cover palette " + swHex); err != nil {
		t.Fatalf("click extracted palette swatch %s: %v", swHex, err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.media.customPaletteSeed != swColor {
		t.Fatalf("customPaletteSeed = %+v, want %+v", a.media.customPaletteSeed, swColor)
	}
	if !tt.HasText("Workbench theme seeded from " + swHex) {
		t.Fatal("palette swatch snackbar missing")
	}

	// Queue reorder: move Abyssal Tide up to index 0, then back down to index 1.
	if err := tt.Click("Move track up"); err != nil {
		t.Fatalf("Move track up: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if a.media.activeTrack != 0 || a.media.artworks[0].title != "Abyssal Tide" {
		t.Fatalf("Move track up failed: activeTrack=%d top=%q", a.media.activeTrack, a.media.artworks[0].title)
	}
	if err := tt.Click("Move track down"); err != nil {
		t.Fatalf("Move track down: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if a.media.activeTrack != 1 || a.media.artworks[1].title != "Abyssal Tide" {
		t.Fatalf("Move track down failed: activeTrack=%d idx1=%q", a.media.activeTrack, a.media.artworks[1].title)
	}

	// Queue filter: filter down to "Kyoto", then empty "zzz" -> Clear filter.
	qField, ok := tt.AccessByRole(ui.RoleTextField, "Filter queue")
	if !ok {
		t.Fatal("Filter queue text field missing")
	}
	tt.ClickAt(centerOf(qField.Bounds))
	tt.Type("Kyoto")
	tt.Advance(250 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleListItem, "Sakura Bloom"); !ok {
		t.Fatal("queue search 'Kyoto' missing Sakura Bloom list item")
	}
	if _, ok := tt.AccessByRole(ui.RoleListItem, "Emerald Canopy"); ok {
		t.Fatal("queue search 'Kyoto' did not filter out Emerald Canopy list item")
	}
	if err := tt.Click("Clear queue filter"); err != nil {
		t.Fatalf("click Clear queue filter: %v", err)
	}
	tt.Advance(250 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleListItem, "Emerald Canopy"); !ok {
		t.Fatal("Clear queue filter did not restore Emerald Canopy list item")
	}

	qField, _ = tt.AccessByRole(ui.RoleTextField, "Filter queue")
	tt.ClickAt(centerOf(qField.Bounds))
	tt.Type("zzz")
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("No matching tracks") {
		t.Fatal("empty queue filter did not show EmptyState")
	}
	if err := tt.Click("Clear filter"); err != nil {
		t.Fatalf("click Clear filter: %v", err)
	}
	tt.Advance(250 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleListItem, "Emerald Canopy"); !ok {
		t.Fatal("Clear filter did not restore full Play Queue")
	}

	// Audio play/pause and MiniPlayer integration.
	clickInsideRegion(t, tt, "Audio player", ui.RoleToggleButton, "Play")
	tt.Advance(600 * time.Millisecond)
	if !a.media.audioPlaying || a.media.audioPos <= 0 {
		t.Fatalf("audio playback did not advance: playing=%v pos=%v", a.media.audioPlaying, a.media.audioPos)
	}
	clickInsideRegion(t, tt, "Mini player", ui.RoleButton, "Pause")
	tt.Advance(4500 * time.Millisecond) // allow queued snackbars to finish dismissing too
	if a.media.audioPlaying {
		t.Fatal("MiniPlayer Pause did not pause audio playback")
	}
	if tt.FramePending() {
		t.Fatal("paused Audio/MiniPlayer left pending animation frames")
	}
}

func TestWorkbenchMediaGalleryShapesAndLightbox(t *testing.T) {
	a, tt := openMediaStudio(t, true)

	// Scroll down to gallery section so all gallery tiles and controls are comfortably visible.
	tt.Wheel(420, 420, 0, 900)
	tt.Advance(300 * time.Millisecond)

	// Verify responsive 1:1 square aspect ratio on md3e.Image tiles (regression guard
	// ensuring intrinsic 192x192 size does not override AspectRatio: 1 + FillWidth).
	tileInfo, ok := tt.AccessByRole(ui.RoleButton, "Open Solar Flare")
	if !ok {
		t.Fatal("Open Solar Flare gallery tile missing")
	}
	if math.Abs(float64(tileInfo.Bounds.W-tileInfo.Bounds.H)) > 2 {
		t.Fatalf("gallery tile bounds = %+v, want 1:1 square aspect", tileInfo.Bounds)
	}

	// Switch mask modes and tone filters.
	if err := tt.Click("Rounded"); err != nil {
		t.Fatalf("click Rounded mask tab: %v", err)
	}
	if err := tt.Click("Mono"); err != nil {
		t.Fatalf("click Mono mask tab: %v", err)
	}
	if err := tt.Click("Expressive"); err != nil {
		t.Fatalf("click Expressive mask tab: %v", err)
	}
	if err := tt.Click("Warm"); err != nil {
		t.Fatalf("click Warm tone filter: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("3 items") || tt.HasText("Abyssal Tide") && len(a.media.filteredGalleryIndices()) != 3 {
		t.Fatalf("Warm tone filter yielded %d items, want 3", len(a.media.filteredGalleryIndices()))
	}
	if err := tt.Click("All"); err != nil {
		t.Fatalf("click All tone filter: %v", err)
	}
	tt.Advance(300 * time.Millisecond)

	// Toggle "States" placeholder demo -> verify Retry button recovers simulated error tile.
	if err := tt.Click("States"); err != nil {
		t.Fatalf("enable States placeholder demo: %v", err)
	}
	tt.Advance(250 * time.Millisecond)
	if err := tt.Click("Retry"); err != nil {
		t.Fatalf("click Retry on simulated error gallery tile: %v", err)
	}
	tt.Advance(400 * time.Millisecond)
	if !a.media.placeholderErrFixed {
		t.Fatal("Retry button did not clear simulated error state")
	}
	if err := tt.Click("States"); err != nil {
		t.Fatalf("disable States placeholder demo: %v", err)
	}
	tt.Advance(500 * time.Millisecond)

	// Open ImageLightbox on Solar Flare, navigate next, zoom in, and close with Escape.
	if err := tt.Click("Open Solar Flare"); err != nil {
		t.Fatalf("click Open Solar Flare: %v", err)
	}
	tt.Advance(350 * time.Millisecond)
	if !a.media.lightboxOpen {
		t.Fatal("ImageLightbox did not open")
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Artwork viewer"); !ok {
		t.Fatal("Artwork viewer RoleDialog missing")
	}
	if !tt.HasText("1 / 6") || !tt.HasText("Solar Flare — Helios Synth") {
		t.Fatal("lightbox metadata missing for Solar Flare")
	}

	if err := tt.Click("Next image"); err != nil {
		t.Fatalf("lightbox Next image: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if a.media.lightboxIdx != 1 || !tt.HasText("2 / 6") {
		t.Fatalf("lightboxIdx=%d after Next image, want 1", a.media.lightboxIdx)
	}

	if err := tt.Click("Zoom in"); err != nil {
		t.Fatalf("lightbox Zoom in: %v", err)
	}
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("150%") {
		t.Fatal("lightbox zoom indicator did not update to 150%")
	}

	tt.Key(0, ui.KeyEscape)
	tt.Advance(4500 * time.Millisecond) // allow snackbars & exit animation to settle
	if a.media.lightboxOpen {
		t.Fatal("Escape did not close ImageLightbox")
	}
	if tt.FramePending() {
		t.Fatal("closing ImageLightbox left pending frames")
	}
}

func TestWorkbenchMediaVideoStageAndFullscreen(t *testing.T) {
	a, tt := openMediaStudio(t, true)

	tt.Wheel(420, 420, 0, 320)
	tt.Advance(250 * time.Millisecond)

	if _, ok := tt.AccessByRole(ui.RoleRegion, "Video stage player"); !ok {
		t.Fatal("Video stage player region missing")
	}

	// Switch clip via SegmentedTabs.
	if err := tt.Click("Prism Orbit"); err != nil {
		t.Fatalf("select Prism Orbit clip: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if a.media.activeClip != 1 {
		t.Fatalf("activeClip = %d, want 1", a.media.activeClip)
	}

	// Play video, enter Fullscreen theater overlay, exit with Escape, and pause.
	clickInsideRegion(t, tt, "Video stage player", ui.RoleToggleButton, "Play")
	tt.Advance(250 * time.Millisecond)
	if !a.media.activeVideoPlayer().Playing() {
		t.Fatal("VideoPlayer did not start playing")
	}

	clickInsideRegion(t, tt, "Video stage player", ui.RoleButton, "Fullscreen")
	tt.Advance(200 * time.Millisecond)
	if !a.media.videoFullscreen {
		t.Fatal("Fullscreen button did not open theater overlay")
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Video theater player"); !ok {
		t.Fatal("Video theater player overlay missing")
	}

	tt.Key(0, ui.KeyEscape)
	tt.Advance(200 * time.Millisecond)
	if a.media.videoFullscreen {
		t.Fatal("Escape did not exit video theater overlay")
	}

	clickInsideRegion(t, tt, "Video stage player", ui.RoleToggleButton, "Pause")
	tt.Advance(400 * time.Millisecond)
	if a.media.activeVideoPlayer().Playing() {
		t.Fatal("Pause did not stop VideoPlayer")
	}
	if tt.FramePending() {
		t.Fatal("paused VideoPlayer left pending frames")
	}
}

func TestWorkbenchMediaBreakpointsRTLAndReduceMotion(t *testing.T) {
	a, tt := openMediaStudio(t, true)

	// 1. Expanded RTL: SupportingPane mirrors Play Queue to the left of main studio.
	a.rtl = true
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	queueList, ok := tt.AccessByRole(ui.RoleList, "Play queue")
	if !ok {
		t.Fatal("Play queue list missing in Expanded layout")
	}
	audioCard, ok := tt.AccessByRole(ui.RoleRegion, "Audio player")
	if !ok {
		t.Fatal("Audio player missing in Expanded layout")
	}
	if queueList.Bounds.X >= audioCard.Bounds.X {
		t.Fatalf("RTL Play Queue (%+v) should sit left of Audio player (%+v)", queueList.Bounds, audioCard.Bounds)
	}
	a.rtl = false
	tt.Frame()
	tt.Advance(200 * time.Millisecond)

	// 2. Compact (480x760): SupportingPane collapses to studio; Queue chip switches to Play Queue and Back returns.
	tt.SetSize(480, 760)
	tt.Advance(500 * time.Millisecond)
	if !tt.HasText("Media & Gallery Studio") || tt.HasText("Play Queue") {
		t.Fatal("Compact Media Studio should start on main studio without replacing it with Play Queue")
	}
	if err := tt.Click("Queue (6)"); err != nil {
		t.Fatalf("click Queue (6) on Compact: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("Play Queue") {
		t.Fatal("Queue (6) did not switch Compact pane to Play Queue")
	}
	if err := tt.Click("Back to studio"); err != nil {
		t.Fatalf("click Back to studio on Compact: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("Media & Gallery Studio") {
		t.Fatal("Back to studio did not restore main Media Studio")
	}

	// 3. ReduceMotion: track switch + Follow cover + theme swap settle immediately with 0 pending frames.
	tt.SetSize(1160, 760)
	a.reduce = true
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if err := tt.Click("Follow cover"); err != nil {
		t.Fatalf("enable Follow cover under ReduceMotion: %v", err)
	}
	if err := tt.Click("Emerald Canopy"); err != nil {
		t.Fatalf("select Emerald Canopy under ReduceMotion: %v", err)
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion Media Studio left pending frames after track/theme change")
	}
	checkNamedControls(t, tt, "media studio reduced motion")
}
