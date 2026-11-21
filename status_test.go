package md3e

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/shape"
)

func makeTestImageSource(t *testing.T, col color.RGBA) ui.ImageSource {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, col)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	src, err := ui.DecodeImage(buf.Bytes())
	if err != nil {
		t.Fatalf("ui.DecodeImage failed: %v", err)
	}
	return src
}

func TestTagVariantsActionsAndCloseIsolation(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	bodyClicks := 0
	closeClicks := 0
	simpleClicks := 0

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(360, 220).Background(canvasRGB).Padding(16).Gap(8).Children(func() {
				ui.Row(c).Gap(6).Children(func() {
					Tag(c, "Neutral", TagOptions{Tone: ToneNeutral})
					Tag(c, "Primary", TagOptions{Tone: TonePrimary, LeadingIcon: mdicon.Star.Draw})
					Tag(c, "Warn", TagOptions{Tone: ToneWarning, Variant: TagOutlined})
					Tag(c, "OK", TagOptions{Tone: ToneSuccess})
					Tag(c, "Err", TagOptions{Tone: ToneError})
				})
				Tag(c.Key("clickable"), "Filter", TagOptions{
					Tone:    ToneSecondary,
					OnClick: func() { simpleClicks++ },
				})
				Tag(c.Key("closable"), "Proxy", TagOptions{
					Tone:    ToneTertiary,
					OnClick: func() { bodyClicks++ },
					OnClose: func() { closeClicks++ },
				})
				Tag(c.Key("disabled"), "Locked", TagOptions{
					Disabled: true,
					OnClick:  func() { simpleClicks += 100 },
				})
			})
		})
	}, 360, 220)

	filterBtn, ok := tt.AccessByRole(ui.RoleButton, "Filter")
	if !ok {
		t.Fatal("Filter clickable Tag missing RoleButton")
	}
	tt.ClickAt(center(filterBtn.Bounds))
	if simpleClicks != 1 {
		t.Fatalf("Filter clicks=%d want 1", simpleClicks)
	}

	proxyBody, ok := tt.AccessByRole(ui.RoleButton, "Proxy")
	if !ok {
		t.Fatal("Proxy tag body missing RoleButton")
	}
	proxyClose, ok := tt.AccessByRole(ui.RoleButton, "Remove Proxy")
	if !ok {
		t.Fatal("Proxy close button missing RoleButton")
	}
	tt.ClickAt(center(proxyClose.Bounds))
	if closeClicks != 1 || bodyClicks != 0 {
		t.Fatalf("clicking close gave closeClicks=%d bodyClicks=%d, want 1/0", closeClicks, bodyClicks)
	}
	tt.ClickAt(center(proxyBody.Bounds))
	if bodyClicks != 1 {
		t.Fatalf("clicking body gave bodyClicks=%d want 1", bodyClicks)
	}

	locked, ok := tt.AccessByRole(ui.RoleButton, "Locked")
	if !ok || !locked.Disabled {
		t.Fatalf("disabled Tag access=%+v ok=%v", locked, ok)
	}
	tt.ClickAt(center(locked.Bounds))
	if simpleClicks != 1 {
		t.Fatalf("disabled Tag fired OnClick: simpleClicks=%d", simpleClicks)
	}
	tt.Move(2, 2)
	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("Tag animations did not settle")
	}
}

func TestStatusPillPulseReduceMotionOffscreenAndGhostExit(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	// 1. Static StatusPill requests zero frames.
	staticTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(200, 80).Background(canvasRGB).Padding(16).Children(func() {
				StatusPill(c, "Offline", StatusPillOptions{Tone: ToneNeutral})
			})
		})
	}, 200, 80)
	if _, ok := staticTT.AccessByRole(ui.RoleStatus, "Offline"); !ok {
		t.Fatal("StatusPill missing RoleStatus")
	}
	if staticTT.FramePending() {
		t.Fatal("static StatusPill should not request frames")
	}

	// 2. Pulsing StatusPill animates when visible, stops under ReduceMotion.
	pulseTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(200, 80).Background(canvasRGB).Padding(16).Children(func() {
				StatusPill(c, "Running", StatusPillOptions{Tone: ToneSuccess, Pulsing: true})
			})
		})
	}, 200, 80)
	if !pulseTT.FramePending() {
		t.Fatal("visible pulsing StatusPill should request animation frames")
	}
	pulseTT.Advance(300 * time.Millisecond)
	if !pulseTT.FramePending() {
		t.Fatal("visible pulsing StatusPill should keep pulsing")
	}

	reduceTh := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	reduceTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, reduceTh, func() {
			ui.Box(c).Size(200, 80).Background(canvasRGB).Padding(16).Children(func() {
				StatusPill(c, "Running", StatusPillOptions{Tone: ToneSuccess, Pulsing: true})
			})
		})
	}, 200, 80)
	if reduceTT.FramePending() {
		t.Fatal("ReduceMotion StatusPill must not request frames")
	}

	// 3. Off-screen pulsing StatusPill inside a scroll container must not request frames.
	offscreenTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Scroll(c).Size(200, 100).Background(canvasRGB).Children(func() {
				ui.Column(c).Children(func() {
					ui.Box(c).Size(180, 320) // push pill completely below the 100 DIP viewport
					StatusPill(c, "HiddenPulse", StatusPillOptions{Tone: ToneWarning, Pulsing: true})
				})
			})
		})
	}, 200, 100)
	offscreenTT.Frame()
	if offscreenTT.FramePending() {
		t.Fatal("off-screen pulsing StatusPill must not request frames when outside p.Visible()")
	}

	// 4. Ghost-draw exit transition must settle with zero frames pending.
	show := true
	exitTr := ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}
	exitTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(200, 80).Background(canvasRGB).Padding(16).Children(func() {
				if show {
					StatusPill(c.Key("exit-pill"), "Live", StatusPillOptions{
						Tone:    ToneSuccess,
						Pulsing: true,
					}).Transition(exitTr)
				}
			})
		})
	}, 200, 80)
	exitTT.Advance(100 * time.Millisecond)
	show = false
	exitTT.Frame()
	exitTT.Advance(400 * time.Millisecond)
	if exitTT.FramePending() {
		t.Fatal("exited pulsing StatusPill left frames pending")
	}
}

func TestAvatarInitialsImagePolygonAndRTLStatusDot(t *testing.T) {
	if got := avatarInitials("Alice Bob"); got != "AB" {
		t.Fatalf("avatarInitials(Alice Bob) = %q, want AB", got)
	}
	if got := avatarInitials("go"); got != "GO" {
		t.Fatalf("avatarInitials(go) = %q, want GO", got)
	}
	if got := avatarInitials("李明"); got != "李" {
		t.Fatalf("avatarInitials(李明) = %q, want 李", got)
	}

	th := ThemeFromSeed(DefaultSeedColor)
	blueImg := makeTestImageSource(t, color.RGBA{R: 30, G: 100, B: 220, A: 255})

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Row(c).Size(320, 120).Background(canvasRGB).Padding(20).Gap(20).Children(func() {
				Avatar(c, AvatarOptions{
					Text:   "Ada Lovelace",
					Size:   48,
					Status: AvatarStatusOnline,
				})
				Avatar(c, AvatarOptions{
					Image:  blueImg,
					Size:   48,
					Shape:  shape.Cookie9Sided(),
					Status: AvatarStatusBusy,
					Label:  "Blue Photo",
				})
			})
		})
	}, 320, 120)

	ada, ok := tt.AccessByRole(ui.RoleImage, "Ada Lovelace")
	if !ok {
		t.Fatal("Ada Avatar missing RoleImage")
	}
	photo, ok := tt.AccessByRole(ui.RoleImage, "Blue Photo")
	if !ok {
		t.Fatal("Photo Avatar missing RoleImage")
	}
	img := tt.Image()
	centerPx := img.RGBAAt(int(photo.Bounds.X+photo.Bounds.W/2), int(photo.Bounds.Y+photo.Bounds.H/2))
	if centerPx.B < 180 || centerPx.R > 60 {
		t.Fatalf("Photo Avatar center pixel = %+v, expected blue image", centerPx)
	}
	// Online dot at bottom-right in LTR.
	brPx := img.RGBAAt(int(ada.Bounds.X+ada.Bounds.W-6), int(ada.Bounds.Y+ada.Bounds.H-6))
	if brPx == white {
		t.Fatal("LTR Avatar bottom-right status dot did not paint")
	}

	// RTL places status dot at bottom-left.
	rtlTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Direction(ui.RightToLeft).Size(120, 120).Background(canvasRGB).Padding(24).Children(func() {
				Avatar(c, AvatarOptions{
					Text:   "RTL User",
					Size:   48,
					Status: AvatarStatusBusy,
				})
			})
		})
	}, 120, 120)
	rtlInfo, _ := rtlTT.AccessByRole(ui.RoleImage, "RTL User")
	rtlImg := rtlTT.Image()
	blPx := rtlImg.RGBAAt(int(rtlInfo.Bounds.X+6), int(rtlInfo.Bounds.Y+rtlInfo.Bounds.H-6))
	busyCol := avatarStatusColor(th, AvatarStatusBusy)
	if !colorMatches(blPx, busyCol, 45) {
		t.Fatalf("RTL Avatar bottom-left pixel = %+v, want busy dot %+v", blPx, rgbOf(busyCol))
	}
}
