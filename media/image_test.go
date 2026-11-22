package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e"
)

func makeTestPNG(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

func makeTestBitmap(w, h int, c ui.Color) *ui.Bitmap {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	nrgba := color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, nrgba)
		}
	}
	return ui.NewBitmap(img)
}

func setupIsolatedNetImage(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	ClearImageCache()
	t.Cleanup(func() {
		ClearImageCache()
		SetImageHTTPClient(nil)
	})
}

func TestNetImageLoadFadeRetryAndEmpty(t *testing.T) {
	setupIsolatedNetImage(t)

	redPNG := makeTestPNG(t, 48, 48, color.NRGBA{R: 220, G: 30, B: 30, A: 255})
	gate := make(chan struct{})
	var gateOnce sync.Once
	openGate := func() { gateOnce.Do(func() { close(gate) }) }
	var failFirst atomic.Bool
	failFirst.Store(true)
	var requests atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch r.URL.Path {
		case "/ok.png":
			select {
			case <-gate:
			case <-r.Context().Done():
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(redPNG)
		case "/flaky.png":
			if failFirst.Load() {
				http.Error(w, "temporary failure", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(redPNG)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	defer openGate()
	SetImageHTTPClient(srv.Client())

	th := md3e.ThemeFromSeed(ui.RGB(103, 80, 164))
	canvasBG := th.Colors.Background
	okURL := srv.URL + "/ok.png"
	flakyURL := srv.URL + "/flaky.png"
	retryCount := 0

	var tt *ui.Tester
	tt = ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Row(c).Size(320, 140).AlignItems(ui.Start).Background(canvasBG).Padding(16).Gap(16).Children(func() {
				NetImage(c.Key("empty"), "", NetImageOptions{
					Width:  64,
					Height: 64,
					Label:  "Empty placeholder",
				})
				NetImage(c.Key("ok"), okURL, NetImageOptions{
					Width:  64,
					Height: 64,
					Label:  "Remote photo",
				})
				NetImage(c.Key("flaky"), flakyURL, NetImageOptions{
					Width:  64,
					Height: 64,
					Label:  "Flaky photo",
					OnRetry: func() {
						retryCount++
					},
				})
			})
		})
	}, 320, 140)

	// 1. Frame 1: Empty URL made zero requests; ok.png is loading (pulsing Skeleton).
	tt.Frame()
	if requests.Load() == 0 {
		t.Fatal("expected HTTP fetch for remote URLs")
	}
	if _, ok := tt.AccessByRole(ui.RoleImage, "Remote photo"); !ok {
		t.Fatal("expected Remote photo accessibility node while loading")
	}
	if !tt.FramePending() {
		t.Fatal("expected pulsing Skeleton FramePending() == true while ok.png is in flight")
	}
	if px := tt.Image().RGBAAt(16+64+16+32, 16+32); px.R > 200 && px.G < 80 {
		t.Fatalf("expected tonal Skeleton placeholder before gate opens, got %+v", px)
	}

	// Poll briefly until flaky.png completes its initial 500 error response.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if _, ok := tt.Find("Retry"); ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, ok := tt.Find("Retry"); !ok {
		t.Fatal("expected Retry button for 500 error response")
	}

	// 2. Retry flaky.png after clearing failure mode -> Forget + re-fetch -> loads red pixel.
	failFirst.Store(false)
	if err := tt.Click("Retry"); err != nil {
		t.Fatalf("Click Retry: %v", err)
	}
	if retryCount != 1 {
		t.Fatalf("OnRetry called %d times, want 1", retryCount)
	}

	// 3. Release ok.png gate and wait for both images to finish decoding & arrive.
	openGate()
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		tt.Advance(40 * time.Millisecond)
		shot := tt.Image()
		okCenter := shot.RGBAAt(16+64+16+32, 16+32)
		flakyCenter := shot.RGBAAt(16+64+16+64+16+32, 16+32)
		if okCenter.R > 180 && okCenter.G < 80 && flakyCenter.R > 180 && flakyCenter.G < 80 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Let fade-in finish completely.
	for range 25 {
		tt.Advance(30 * time.Millisecond)
		if !tt.FramePending() {
			break
		}
	}
	shot := tt.Image()
	okPx := shot.RGBAAt(16+64+16+32, 16+32)
	flakyPx := shot.RGBAAt(16+64+16+64+16+32, 16+32)
	if okPx.R < 200 || okPx.G > 60 {
		t.Fatalf("ok.png did not fade in to red bitmap, got %+v", okPx)
	}
	if flakyPx.R < 200 || flakyPx.G > 60 {
		t.Fatalf("flaky.png did not recover to red bitmap after Retry, got %+v", flakyPx)
	}
	if tt.FramePending() {
		t.Fatal("settled NetImage still reports FramePending() == true")
	}

	// Clicking "Empty" triggers nothing and URL="" shouldn't cache errors.
	reqsBefore := requests.Load()
	tt.Frame()
	if requests.Load() != reqsBefore {
		t.Fatal("steady frame fired duplicate HTTP requests")
	}
}

func TestImageLightboxNavigationZoomPanAndEscape(t *testing.T) {
	red := makeTestBitmap(40, 40, ui.RGB(217, 38, 38))
	green := makeTestBitmap(40, 40, ui.RGB(38, 191, 64))
	blue := makeTestBitmap(40, 40, ui.RGB(38, 89, 230))

	items := []LightboxItem{
		{Image: red, Title: "Red Slide", Caption: "First slide"},
		{Image: green, Title: "Green Slide", Caption: "Second slide"},
		{Image: blue, Title: "Blue Slide", Caption: "Third slide"},
	}

	th := md3e.ThemeFromSeed(ui.RGB(103, 80, 164))
	open := false
	idx := 0
	closeCount := 0
	indexChanges := 0

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Row(c).Size(640, 480).Center().Children(func() {
				if md3e.Button(c, "Open viewer").Clicked() {
					open = true
				}
			})
			ImageLightbox(c, &open, items, LightboxOptions{
				Index:          &idx,
				OnIndexChanged: func(int) { indexChanges++ },
				OnClose:        func() { closeCount++ },
			})
		})
	}, 640, 480)

	// Open via focused button click.
	if err := tt.Click("Open viewer"); err != nil {
		t.Fatalf("Click Open viewer: %v", err)
	}
	tt.Advance(300 * time.Millisecond)

	if !open {
		t.Fatal("lightbox did not open")
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Image viewer"); !ok {
		t.Fatal("expected RoleDialog Image viewer in accessibility tree")
	}
	if !tt.HasText("1 / 3") || !tt.HasText("Red Slide") || !tt.HasText("First slide") {
		t.Fatalf("missing slide 1 metadata in texts: %v", tt.Texts())
	}

	// Next slide via Next button and Right arrow key.
	if err := tt.Click("Next image"); err != nil {
		t.Fatalf("Click Next image: %v", err)
	}
	if idx != 1 || indexChanges != 1 {
		t.Fatalf("idx=%d indexChanges=%d, want 1 and 1", idx, indexChanges)
	}
	tt.Key(0, ui.KeyRight)
	if idx != 2 {
		t.Fatalf("KeyRight did not advance to slide 2, got %d", idx)
	}
	tt.Key(0, ui.KeyLeft)
	if idx != 1 {
		t.Fatalf("KeyLeft did not go back to slide 1, got %d", idx)
	}

	// Zoom in via Zoom in button, wheel, pinch, and reset.
	if err := tt.Click("Zoom in"); err != nil {
		t.Fatalf("Click Zoom in: %v", err)
	}
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("150%") {
		t.Fatalf("expected 150%% zoom indicator, texts=%v", tt.Texts())
	}

	// Pinch zoom 2.0 -> 300%.
	tt.Pinch(320, 240, 2.0)
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("300%") {
		t.Fatalf("expected 300%% zoom after pinch, texts=%v", tt.Texts())
	}

	// Drag pan while zoomed (exercise HandleInput pointer down/move/up).
	tt.Press(320, 240)
	tt.Move(400, 280)
	tt.Release(400, 280)

	// Touchpad scroll pan while zoomed, then mouse wheel zoom out.
	tt.Scroll(320, 240, 20, 10)
	tt.Wheel(320, 240, 0, 40)
	tt.Advance(250 * time.Millisecond)
	if tt.HasText("300%") {
		t.Fatalf("mouse wheel down did not zoom out from 300%%, texts=%v", tt.Texts())
	}

	// Double-click (toggles 100% -> 200% -> 100%) and keyboard =/-/0 zoom.
	tt.ClickAt(320, 240)
	tt.Advance(40 * time.Millisecond)
	tt.ClickAt(320, 240)
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("200%") {
		t.Fatalf("double-click did not zoom to 200%%, texts=%v", tt.Texts())
	}
	tt.Key(0, ui.KeyEqual)
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("250%") {
		t.Fatalf("KeyEqual did not zoom to 250%%, texts=%v", tt.Texts())
	}
	tt.Key(0, ui.KeyMinus)
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("200%") {
		t.Fatalf("KeyMinus did not zoom out to 200%%, texts=%v", tt.Texts())
	}
	tt.Key(0, ui.Key0)
	tt.Advance(250 * time.Millisecond)
	if !tt.HasText("100%") {
		t.Fatalf("Key0 did not reset zoom to 100%%, texts=%v", tt.Texts())
	}

	// Check every interactive control has a non-empty accessible name.
	for _, info := range tt.AccessAll() {
		if (info.Role == ui.RoleButton) && strings.TrimSpace(info.Name) == "" {
			t.Fatalf("unnamed button in ImageLightbox access tree: %+v", info)
		}
	}

	// Escape closes the viewer cleanly, runs OnClose, converges animations, and returns focus.
	tt.Key(0, ui.KeyEscape)
	for range 30 {
		tt.Advance(20 * time.Millisecond)
		if !tt.FramePending() {
			break
		}
	}
	if open || closeCount != 1 {
		t.Fatalf("open=%v closeCount=%d, want false and 1", open, closeCount)
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Image viewer"); ok {
		t.Fatal("dialog still present in accessibility tree after Escape close")
	}
	if tt.FramePending() {
		t.Fatal("lightbox still has FramePending after close animation settled")
	}
	if !tt.Focused("Open viewer") {
		t.Fatal("focus was not restored to Open viewer on lightbox close")
	}

	// Reopen and verify scrim click outside stage/bars dismisses cleanly.
	if err := tt.Click("Open viewer"); err != nil {
		t.Fatalf("Click Open viewer to reopen: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	tt.ClickAt(12, 420)
	for range 30 {
		tt.Advance(20 * time.Millisecond)
		if !tt.FramePending() {
			break
		}
	}
	if open || closeCount != 2 {
		t.Fatalf("scrim click did not dismiss lightbox: open=%v closeCount=%d", open, closeCount)
	}
}

func TestImageLightboxRTLAndReduceMotion(t *testing.T) {
	red := makeTestBitmap(32, 32, ui.RGB(204, 51, 51))
	blue := makeTestBitmap(32, 32, ui.RGB(51, 51, 204))
	items := []LightboxItem{
		{Image: red, Title: "First"},
		{Image: blue, Title: "Second"},
	}

	th := md3e.ThemeFromSeed(ui.RGB(103, 80, 164))
	th.ReduceMotion = true
	open := true
	idx := 0

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ImageLightbox(c, &open, items, LightboxOptions{
				Index: &idx,
			})
		})
	}, 600, 400)
	tt.SetDirection(ui.RightToLeft)

	tt.Frame()
	if tt.FramePending() {
		t.Fatal("ReduceMotion lightbox should not leave FramePending == true on open")
	}

	// In RTL layout, Previous image should be physically on the right (Start > 300) and Next on left (End < 300).
	prevBox, okPrev := tt.Find("Previous image")
	nextBox, okNext := tt.Find("Next image")
	if !okPrev || !okNext {
		t.Fatal("expected Previous image and Next image buttons")
	}
	if prevBox.X <= nextBox.X {
		t.Fatalf("expected RTL Previous image (%v) to the right of Next image (%v)", prevBox, nextBox)
	}

	// In RTL, KeyLeft steps forward (Next -> slide 1) and KeyRight steps back (Previous -> slide 0).
	tt.Key(0, ui.KeyLeft)
	if idx != 1 {
		t.Fatalf("RTL KeyLeft idx=%d, want 1", idx)
	}
	tt.Key(0, ui.KeyRight)
	if idx != 0 {
		t.Fatalf("RTL KeyRight idx=%d, want 0", idx)
	}

	// Escape with ReduceMotion closes immediately with zero pending frames.
	tt.Key(0, ui.KeyEscape)
	if open {
		t.Fatal("Escape did not close RTL lightbox")
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion lightbox left FramePending == true after close")
	}
}
