package md3e

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/shape"
)

func TestSkeletonPulseReduceMotionOffscreenAndSanitization(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	nan := float32(math.NaN())
	inf := float32(math.Inf(1))

	// 1. Visible animated skeleton requests frames and paints non-empty pixels.
	animTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(280, 180).Background(canvasRGB).Padding(16).Gap(10).Children(func() {
				Skeleton(c, SkeletonOptions{
					Shape:  SkeletonCircle,
					Height: 40,
					Label:  "Avatar loading",
				})
				Skeleton(c, SkeletonOptions{
					Lines:   3,
					Height:  14,
					LineGap: 6,
					Label:   "Paragraph loading",
				})
				Skeleton(c, SkeletonOptions{
					Shape:  SkeletonPill,
					Width:  nan,
					Height: inf,
				})
			})
		})
	}, 280, 180)
	if !animTT.FramePending() {
		t.Fatal("visible animated Skeleton should request animation frames")
	}
	info, ok := animTT.AccessByRole(ui.RoleProgress, "Avatar loading")
	if !ok {
		t.Fatal("Skeleton missing RoleProgress")
	}
	if got := countNonBgPixelsInRect(animTT.Image(), info.Bounds, white); got < 400 {
		t.Fatalf("Skeleton circle painted too few pixels: %d", got)
	}

	// 2. Static & ReduceMotion request zero frames.
	staticTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 80).Background(canvasRGB).Padding(16).Children(func() {
				Skeleton(c, SkeletonOptions{Static: true, Height: 20})
			})
		})
	}, 240, 80)
	if staticTT.FramePending() {
		t.Fatal("Static Skeleton must not request frames")
	}

	reduceTh := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	reduceTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, reduceTh, func() {
			ui.Box(c).Direction(ui.RightToLeft).Size(240, 100).Background(canvasRGB).Padding(16).Children(func() {
				Skeleton(c, SkeletonOptions{Lines: 2, Height: 14})
			})
		})
	}, 240, 100)
	if reduceTT.FramePending() {
		t.Fatal("ReduceMotion Skeleton must not request frames")
	}

	// 3. Off-screen Skeleton inside a scroll container must request zero frames.
	offTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Scroll(c).Size(240, 100).Background(canvasRGB).Children(func() {
				ui.Column(c).Children(func() {
					ui.Box(c).Size(200, 320)
					Skeleton(c, SkeletonOptions{Height: 20})
				})
			})
		})
	}, 240, 100)
	offTT.Frame()
	if offTT.FramePending() {
		t.Fatal("off-screen Skeleton must not request frames when outside p.Visible()")
	}

	// 4. Ghost-draw exit safety.
	show := true
	exitTr := ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}
	exitTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 80).Background(canvasRGB).Padding(16).Children(func() {
				if show {
					Skeleton(c.Key("exit-skel"), SkeletonOptions{Height: 20}).Transition(exitTr)
				}
			})
		})
	}, 240, 80)
	exitTT.Advance(80 * time.Millisecond)
	show = false
	exitTT.Frame()
	exitTT.Advance(400 * time.Millisecond)
	if exitTT.FramePending() {
		t.Fatal("exited Skeleton left frames pending")
	}
}

func TestEmptyStateActionsAndShapeSlot(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	primaryClicks := 0
	secondaryClicks := 0

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(440, 320).Background(canvasRGB).Padding(16).Children(func() {
				EmptyState(c, "No Results Found", EmptyStateOptions{
					Icon:                 mdicon.Search.Draw,
					IconShape:            shape.SoftBurst(),
					Description:          "Try adjusting your filters or creating a new entry.",
					PrimaryActionLabel:   "Create Item",
					OnPrimaryAction:      func() { primaryClicks++ },
					SecondaryActionLabel: "Reset Filters",
					OnSecondaryAction:    func() { secondaryClicks++ },
				})
			})
		})
	}, 440, 320)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "No Results Found"); !ok {
		t.Fatal("EmptyState missing RoleGroup")
	}
	pri, ok := tt.AccessByRole(ui.RoleButton, "Create Item")
	if !ok {
		t.Fatal("EmptyState missing primary action button")
	}
	sec, ok := tt.AccessByRole(ui.RoleButton, "Reset Filters")
	if !ok {
		t.Fatal("EmptyState missing secondary action button")
	}
	tt.ClickAt(center(pri.Bounds))
	tt.ClickAt(center(sec.Bounds))
	if primaryClicks != 1 || secondaryClicks != 1 {
		t.Fatalf("EmptyState clicks pri=%d sec=%d, want 1/1", primaryClicks, secondaryClicks)
	}
	tt.Move(2, 2)
	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("EmptyState animations did not settle")
	}
}

func TestBannerTonesActionsAndDismissVisibility(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := true
	retryClicks := 0
	dismissed := 0

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(520, 220).Background(canvasRGB).Padding(16).Gap(12).Children(func() {
				Banner(c.Key("err-banner"), "Connection lost to upstream gateway.", BannerOptions{
					Tone:  ToneError,
					Icon:  mdicon.ErrorOutline.Draw,
					Title: "Network Error",
					Open:  &open,
					Actions: []BannerAction{
						{Label: "Retry", OnClick: func() { retryClicks++ }},
					},
					OnDismiss: func() { dismissed++ },
				})
				Banner(c.Key("info-banner"), "All nodes are healthy.", BannerOptions{
					Tone: ToneSuccess,
				})
			})
		})
	}, 520, 220)

	if _, ok := tt.AccessByRole(ui.RoleAlert, "Network Error: Connection lost to upstream gateway."); !ok {
		t.Fatal("Error Banner missing RoleAlert")
	}
	if _, ok := tt.AccessByRole(ui.RoleStatus, "All nodes are healthy."); !ok {
		t.Fatal("Success Banner missing RoleStatus")
	}

	retryBtn, ok := tt.AccessByRole(ui.RoleButton, "Retry")
	if !ok {
		t.Fatal("Banner missing Retry action button")
	}
	tt.ClickAt(center(retryBtn.Bounds))
	if retryClicks != 1 {
		t.Fatalf("Retry clicks=%d want 1", retryClicks)
	}

	dismissBtn, ok := tt.AccessByRole(ui.RoleButton, "Dismiss")
	if !ok {
		t.Fatal("Banner missing Dismiss button")
	}
	tt.ClickAt(center(dismissBtn.Bounds))
	if open || dismissed != 1 {
		t.Fatalf("after dismiss: open=%v dismissed=%d, want false/1", open, dismissed)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlert, "Network Error: Connection lost to upstream gateway."); ok {
		t.Fatal("dismissed Banner should no longer appear in accessibility tree")
	}
	tt.Move(2, 2)
	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("Banner animations did not settle")
	}
}
