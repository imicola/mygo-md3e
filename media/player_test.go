package media

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

func makeTestGIFBytes(t *testing.T, w, h, frames, delayCs int) []byte {
	t.Helper()
	pal := color.Palette{
		color.RGBA{R: 15, G: 20, B: 30, A: 255},
		color.RGBA{R: 220, G: 40, B: 40, A: 255},
		color.RGBA{R: 40, G: 190, B: 60, A: 255},
		color.RGBA{R: 40, G: 100, B: 230, A: 255},
	}
	g := &gif.GIF{}
	for i := range frames {
		img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
		idx := uint8(1 + (i % (len(pal) - 1)))
		for y := range h {
			for x := range w {
				img.SetColorIndex(x, y, idx)
			}
		}
		g.Image = append(g.Image, img)
		g.Delay = append(g.Delay, delayCs)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatalf("gif.EncodeAll: %v", err)
	}
	return buf.Bytes()
}

type uncomparableArtwork struct {
	image.Image
	metadata []byte
}

func TestAudioCoverMemoAcrossOptionAndArtworkChanges(t *testing.T) {
	warm := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	cool := image.NewNRGBA(warm.Bounds())
	for i := range 16 {
		warm.Set(i%4, i/4, color.NRGBA{R: 230, G: 80, B: 40, A: 255})
		cool.Set(i%4, i/4, color.NRGBA{R: 20, G: 100, B: 230, A: 255})
	}
	track := TrackInfo{ArtworkImage: warm}
	dynamic := true
	var seed ui.Color
	var artwork ui.ImageSource
	var theme *md3e.Theme
	var active bool
	tt := ui.NewTester(func(c *ui.Context) {
		artwork, theme, active = resolveTrackCoverAndTheme(c, ui.Box(c), track, dynamic, seed, tokens.Expressive, true)
	}, 100, 100)
	if !active || artwork == nil || theme == nil {
		t.Fatal("initial dynamic artwork is missing")
	}
	warmTheme := theme.Colors.Primary

	dynamic = false
	track.ArtworkImage = cool
	tt.Frame()
	if active {
		t.Fatal("dynamic theme enabled with no seed or opt-in")
	}
	coolBitmap := artwork
	track.Artwork = coolBitmap
	track.ArtworkImage = warm
	tt.Frame()
	track.Artwork = nil
	tt.Frame()
	if artwork == coolBitmap {
		t.Error("retained old bitmap after hiding and changing raw artwork")
	}
	track.ArtworkImage = cool
	tt.Frame()
	dynamic = true
	tt.Frame()
	if !active || theme.Colors.Primary == warmTheme {
		t.Error("reused old artwork seed after re-enabling dynamic color")
	}

	seed = ui.RGB(200, 30, 170)
	track.ArtworkImage = warm
	tt.Frame()
	wantOverride := md3e.ThemeFromSeed(seed, md3e.ThemeOptions{Variant: tokens.Expressive}).Colors.Primary
	if !active || theme.Colors.Primary != wantOverride {
		t.Error("explicit seed did not override the artwork seed")
	}
	bitmapWithOverride := artwork
	tt.Frame()
	if artwork != bitmapWithOverride {
		t.Error("explicit seed caused the bitmap to be rebuilt")
	}
	seed = ui.Color{}
	track.ArtworkImage = uncomparableArtwork{Image: warm, metadata: []byte{1}}
	tt.Frame()
	tt.Frame()
	if !active || artwork == nil || theme.Colors.Primary != warmTheme {
		t.Fatal("uncomparable artwork failed to produce a bitmap and matching theme")
	}
}

func makeRawColorImage(w, h int, c color.NRGBA) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestVideoPlayerPlaybackControlsShortcutsAndAutoHide(t *testing.T) {
	gifData := makeTestGIFBytes(t, 64, 48, 60, 10) // 6.0s duration
	dec, err := ui.NewGIFDecoder(gifData)
	if err != nil {
		t.Fatalf("NewGIFDecoder: %v", err)
	}
	player, err := ui.NewVideoPlayer(dec)
	if err != nil {
		t.Fatalf("NewVideoPlayer: %v", err)
	}
	defer player.Close()
	player.Pause()

	th := md3e.ThemeFromSeed(ui.RGB(103, 80, 164))
	fullscreen := false
	fsCalls := 0

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Size(640, 520).Padding(16).Children(func() {
				VideoPlayer(c, player, VideoPlayerOptions{
					Title:         "Sample Clip",
					Buffered:      0.5,
					AutoHideDelay: 2 * time.Second,
					Fullscreen:    &fullscreen,
					OnFullscreen:  func(fs bool) { fsCalls++ },
				})
			})
		})
	}, 640, 520)

	tt.Frame()
	if !tt.HasText("Sample Clip") || !tt.HasText("00:00 / 00:06") {
		t.Fatalf("expected title and duration timestamps, got texts=%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(ui.RoleSlider, "Video timeline"); !ok {
		t.Fatal("expected accessible Video timeline slider")
	}

	// Click Play button -> starts playing.
	if err := tt.Click("Play"); err != nil {
		t.Fatalf("Click Play: %v", err)
	}
	if !player.Playing() {
		t.Fatal("player did not start playing after Play click")
	}

	// Auto-hide: advance past AutoHideDelay (2s) + fade duration -> controls hide.
	tt.Advance(2500 * time.Millisecond)
	if !player.Playing() {
		t.Fatal("player stopped playing unexpectedly")
	}
	if _, ok := tt.AccessByRole(ui.RoleToggleButton, "Pause"); ok {
		t.Fatal("controls remained visible in access tree after AutoHideDelay")
	}

	// Pointer move over stage reveals controls immediately.
	tt.Move(320, 200)
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleToggleButton, "Pause"); !ok {
		t.Fatal("pointer move did not reveal auto-hidden controls")
	}

	// Mute button and Volume check, Playback rate menu selection, Fullscreen toggle.
	if err := tt.Click("Mute"); err != nil {
		t.Fatalf("Click Mute: %v", err)
	}
	if !player.Muted() {
		t.Fatal("Mute button did not mute player")
	}

	if err := tt.Click("1x"); err != nil {
		t.Fatalf("Click speed menu 1x: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleMenu, "Playback speed 1x"); !ok {
		t.Fatal("expected Playback speed 1x RoleMenu popup")
	}
	if err := tt.Click("1.5x"); err != nil {
		t.Fatalf("Click 1.5x speed item: %v", err)
	}
	tt.Advance(200 * time.Millisecond)
	if player.Rate() != 1.5 {
		t.Fatalf("player.Rate()=%v, want 1.5", player.Rate())
	}

	if err := tt.Click("Fullscreen"); err != nil {
		t.Fatalf("Click Fullscreen: %v", err)
	}
	if !fullscreen || fsCalls != 1 {
		t.Fatalf("fullscreen=%v fsCalls=%d, want true and 1", fullscreen, fsCalls)
	}

	// Pause stops animation & keeps controls visible permanently while paused.
	if err := tt.Click("Pause"); err != nil {
		t.Fatalf("Click Pause: %v", err)
	}
	if player.Playing() {
		t.Fatal("Pause did not pause player")
	}

	// Drag scrubber to roughly 50% (seek).
	scrubBox, ok := tt.Find("Video timeline")
	if !ok {
		t.Fatal("Video timeline slider not found")
	}
	tt.Press(scrubBox.X+scrubBox.W*0.1, scrubBox.Y+scrubBox.H*0.5)
	tt.Move(scrubBox.X+scrubBox.W*0.5, scrubBox.Y+scrubBox.H*0.5)
	tt.Release(scrubBox.X+scrubBox.W*0.5, scrubBox.Y+scrubBox.H*0.5)
	tt.Advance(200 * time.Millisecond)
	if player.Position() < 1500*time.Millisecond {
		t.Fatalf("expected seek forward near middle, pos=%v", player.Position())
	}

	// Accessibility name audit: zero unnamed interactive controls.
	for _, info := range tt.AccessAll() {
		switch info.Role {
		case ui.RoleButton, ui.RoleSlider, ui.RoleToggleButton:
			if strings.TrimSpace(info.Name) == "" {
				t.Fatalf("unnamed control in VideoPlayer access tree: %+v", info)
			}
		}
	}

	// Keyboard shortcuts: M unmutes, F toggles fullscreen, K resumes play.
	tt.Key(0, ui.KeyM)
	if player.Muted() {
		t.Fatal("KeyM did not unmute player")
	}
	tt.Key(0, ui.KeyF)
	if fullscreen {
		t.Fatal("KeyF did not toggle fullscreen off")
	}
	tt.Key(0, ui.KeyK)
	if !player.Playing() {
		t.Fatal("KeyK did not resume playback")
	}
	tt.Key(0, ui.KeyK)
	if player.Playing() {
		t.Fatal("second KeyK did not pause playback")
	}
}

func TestVideoPlayerReduceMotionErrorAndNilSafety(t *testing.T) {
	th := md3e.ThemeFromSeed(ui.RGB(103, 80, 164))
	th.ReduceMotion = true
	retries := 0

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			VideoPlayer(c, nil, VideoPlayerOptions{
				Width:   400,
				Height:  240,
				OnRetry: func() { retries++ },
			})
		})
	}, 420, 260)

	tt.Frame()
	if !tt.HasText("Video source unavailable") {
		t.Fatalf("expected error fallback for nil player, texts=%v", tt.Texts())
	}
	if tt.FramePending() {
		t.Fatal("nil VideoPlayer in ReduceMotion should have zero pending frames")
	}
	if err := tt.Click("Retry"); err != nil {
		t.Fatalf("Click Retry: %v", err)
	}
	if retries != 1 {
		t.Fatalf("retries=%d, want 1", retries)
	}

	// Empty/invalid media cleanly closes or returns error across codecs/OS backends.
	if vp, err := OpenVideoData([]byte("not a valid media container")); err == nil && vp != nil {
		vp.Close()
	}
}

func TestAudioPlayerCardMiniPlayerDynamicColorAndReduceMotion(t *testing.T) {
	warmImg := makeRawColorImage(64, 64, color.NRGBA{R: 230, G: 70, B: 30, A: 255})
	coolImg := makeRawColorImage(64, 64, color.NRGBA{R: 30, G: 110, B: 235, A: 255})

	track := TrackInfo{
		Title:        "Sunrise Pulse",
		Artist:       "M3 Ensemble",
		Album:        "Expressive Vol. 1",
		ArtworkImage: warmImg,
		Position:     45 * time.Second,
		Duration:     3 * time.Minute,
		Buffered:     0.65,
	}

	th := md3e.ThemeFromSeed(ui.RGB(103, 80, 164))
	playing := false
	shuffle := false
	repeat := false
	nextCount, prevCount, expandCount, seekCount := 0, 0, 0, 0

	var observedPrimary ui.Color
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Size(440, 420).Padding(16).Gap(16).Children(func() {
				AudioPlayerCard(c, track, AudioPlayerOptions{
					Playing:      &playing,
					Shuffle:      &shuffle,
					Repeat:       &repeat,
					DynamicColor: true,
					ArtworkShape: shape.Cookie9Sided(),
					OnPlayPause:  func(bool) {},
					OnPrevious:   func() { prevCount++ },
					OnNext:       func() { nextCount++ },
					OnSeek:       func(pos time.Duration) { track.Position = pos; seekCount++ },
					OnQueue:      func() {},
				})
				MiniPlayer(c, track, MiniPlayerOptions{
					Playing:      &playing,
					DynamicColor: true,
					OnPrevious:   func() { prevCount++ },
					OnNext:       func() { nextCount++ },
					OnExpand:     func() { expandCount++ },
				})
				ui.Box(c).Size(1, 1).Draw(func(*ui.Painter, ui.Rect) {
					observedPrimary = md3e.CurrentTheme(c).Colors.Primary
				})
			})
		})
	}, 440, 420)

	// 1. Paused initial render: zero pending frames!
	for range 15 {
		tt.Advance(25 * time.Millisecond)
		if !tt.FramePending() {
			break
		}
	}
	if tt.FramePending() {
		t.Fatal("paused AudioPlayerCard/MiniPlayer should settle to FramePending() == false")
	}
	if !tt.HasText("Sunrise Pulse") || !tt.HasText("00:45") || !tt.HasText("03:00") {
		t.Fatalf("missing track info/timestamps, texts=%v", tt.Texts())
	}

	// Capture rendered card/primary pixel palette under warm cover.
	shot1 := tt.Image()
	warmPx := shot1.RGBAAt(120, 40)
	playRect, ok := tt.Find("Play")
	if !ok {
		t.Fatal("expected Play button in AudioPlayerCard")
	}
	warmBtnPx := shot1.RGBAAt(int(playRect.X+4), int(playRect.Y+playRect.H*0.5))
	_ = observedPrimary

	// Switch artwork to cool blue -> AnimatedThemeScope transitions palette cleanly.
	track.Title = "Midnight Tide"
	track.ArtworkImage = coolImg
	tt.Frame()
	for range 30 {
		tt.Advance(20 * time.Millisecond)
		if !tt.FramePending() {
			break
		}
	}
	if tt.FramePending() {
		t.Fatal("dynamic theme transition did not converge to FramePending() == false while paused")
	}
	shot2 := tt.Image()
	coolPx := shot2.RGBAAt(120, 40)
	coolBtnPx := shot2.RGBAAt(int(playRect.X+4), int(playRect.Y+playRect.H*0.5))
	if warmPx == coolPx && warmBtnPx == coolBtnPx {
		t.Fatalf("expected dynamic cover seed change to alter card colors (card=%+v->%+v btn=%+v->%+v)",
			warmPx, coolPx, warmBtnPx, coolBtnPx)
	}

	// Play button starts wave animation; clicking Next / Expand invokes callbacks.
	if err := tt.Click("Play"); err != nil {
		t.Fatalf("Click Play: %v", err)
	}
	if !playing {
		t.Fatal("playing flag did not toggle true")
	}
	if !tt.FramePending() {
		t.Fatal("playing MiniPlayer/AudioPlayerCard with wavy progress should animate frames")
	}
	if err := tt.Click("Next track"); err != nil {
		t.Fatalf("Click Next track: %v", err)
	}
	if err := tt.Click("Expand Midnight Tide"); err != nil {
		t.Fatalf("Click Expand Midnight Tide: %v", err)
	}
	if nextCount != 1 || expandCount != 1 {
		t.Fatalf("nextCount=%d expandCount=%d, want 1 and 1", nextCount, expandCount)
	}

	// Scrub seek updates track position via OnSeek.
	scrubBox, ok := tt.Find("Audio playback timeline")
	if !ok {
		t.Fatal("Audio playback timeline slider not found")
	}
	tt.Press(scrubBox.X+scrubBox.W*0.75, scrubBox.Y+scrubBox.H*0.5)
	tt.Release(scrubBox.X+scrubBox.W*0.75, scrubBox.Y+scrubBox.H*0.5)
	if seekCount == 0 {
		t.Fatal("scrubbing did not fire OnSeek callback")
	}

	// Accessibility check: every button and slider has a non-empty name.
	for _, info := range tt.AccessAll() {
		switch info.Role {
		case ui.RoleButton, ui.RoleSlider:
			if strings.TrimSpace(info.Name) == "" {
				t.Fatalf("unnamed interactive control in audio player: %+v", info)
			}
		}
	}

	// ReduceMotion while playing suppresses wavy progress & morph frames completely.
	th.ReduceMotion = true
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	for range 20 {
		tt.Advance(30 * time.Millisecond)
		if !tt.FramePending() {
			break
		}
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion playing player should emit zero animation frames")
	}
	// Trigger a state change under ReduceMotion and confirm immediate convergence with zero pending frames.
	if err := tt.Click("Pause"); err != nil {
		t.Fatalf("Click Pause under ReduceMotion: %v", err)
	}
	if playing {
		t.Fatal("expected playing == false after Pause under ReduceMotion")
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion toggle should converge without pending animation frames")
	}

	// Density scaling checks on MiniPlayer (Comfortable=64, Desktop=56, Compact=52).
	for _, tc := range []struct {
		density tokens.Density
		wantH   float32
	}{
		{tokens.DensityComfortable, 64},
		{tokens.DensityDesktop, 56},
		{tokens.DensityCompact, 52},
	} {
		dth := md3e.ThemeFromSeed(ui.RGB(103, 80, 164), md3e.ThemeOptions{Density: tc.density})
		dtt := ui.NewTester(func(c *ui.Context) {
			md3e.ThemeScope(c, dth, func() {
				MiniPlayer(c, track)
			})
		}, 400, 100)
		dtt.Frame()
		reg, ok := dtt.AccessByRole(ui.RoleRegion, track.Title)
		if !ok {
			t.Fatalf("MiniPlayer region missing for density %v", tc.density)
		}
		if reg.Bounds.H != tc.wantH {
			t.Fatalf("MiniPlayer height for density %v = %v, want %v", tc.density, reg.Bounds.H, tc.wantH)
		}
	}

	// DynamicColor FillWidth check inside an AlignItems(ui.Start) column.
	ftt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Size(520, 420).AlignItems(ui.Start).Gap(12).Children(func() {
				AudioPlayerCard(c, track, AudioPlayerOptions{
					DynamicColor: true,
					Label:        "Wide audio card",
				})
				MiniPlayer(c, track, MiniPlayerOptions{
					DynamicColor: true,
					Label:        "Wide mini player",
				})
			})
		})
	}, 520, 420)
	ftt.Frame()
	if reg, ok := ftt.AccessByRole(ui.RoleRegion, "Wide audio card"); !ok || reg.Bounds.W < 510 {
		t.Fatalf("AudioPlayerCard with DynamicColor did not fill parent width: ok=%v bounds=%+v", ok, reg.Bounds)
	}
	if reg, ok := ftt.AccessByRole(ui.RoleRegion, "Wide mini player"); !ok || reg.Bounds.W < 510 {
		t.Fatalf("MiniPlayer with DynamicColor did not fill parent width: ok=%v bounds=%+v", ok, reg.Bounds)
	}
}
