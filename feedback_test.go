package md3e

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// feedbackCanvas builds a 480x320 white canvas, its content centered.
func feedbackCanvas(th *Theme, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(480, 320).Background(canvasRGB).
				Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				build(c)
			})
		})
	}, 480, 320)
}

// badgeInfo returns an element's access info by accessible name.
func badgeInfo(t *testing.T, tt *ui.Tester, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.Access(name)
	if !ok {
		t.Fatalf("%q not in the access tree", name)
	}
	return info
}

// TestBadgeSizesCountsAndVisibility checks the small dot's 6x6 DIP box, the
// large badge's 16 DIP pill, numeric truncation, zero handling and colors.
func TestBadgeSizesCountsAndVisibility(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	tt := feedbackCanvas(th, func(c *ui.Context) {
		ui.Row(c).Gap(24).Children(func() {
			SmallBadge(c, BadgeOptions{Label: "Small"})
			Badge(c, "3", BadgeOptions{Label: "Three"})
			Badge(c, "1000", BadgeOptions{Label: "Many"})
			Badge(c, "0", BadgeOptions{Label: "Hidden zero"})
			Badge(c, "0", BadgeOptions{Label: "Shown zero", ShowZero: true})
			Badge(c, "5", BadgeOptions{Label: "Invisible", Visible: boolPtr(false)})
		})
	})
	small := badgeInfo(t, tt, "Small")
	if math.Abs(float64(small.Bounds.W)-6) > 0.5 || math.Abs(float64(small.Bounds.H)-6) > 0.5 {
		t.Fatalf("small badge box %v, want 6x6", small.Bounds)
	}
	if px := tt.Image().RGBAAt(int(small.Bounds.X+3), int(small.Bounds.Y+3)); !colorMatches(px, cs.Error, 12) {
		t.Fatalf("small badge %+v is not Error %+v", px, rgbOf(cs.Error))
	}

	three := badgeInfo(t, tt, "Three")
	if three.Bounds.H < 16 {
		t.Fatalf("large badge height %.1f, want at least 16", three.Bounds.H)
	}
	if !tt.HasText("3") {
		t.Fatal("the large badge does not show its count")
	}
	if px := tt.Image().RGBAAt(int(three.Bounds.X+2), int(three.Bounds.Y+8)); !colorMatches(px, cs.Error, 12) {
		t.Fatalf("large badge %+v is not Error %+v", px, rgbOf(cs.Error))
	}

	if !tt.HasText("999+") {
		t.Fatal("a count above MaxCount does not read 999+")
	}
	if tt.HasText("1000") {
		t.Fatal("the badge shows the untruncated count")
	}
	if _, ok := tt.Access("Hidden zero"); ok {
		t.Fatal("a zero badge shows without ShowZero")
	}
	if !tt.HasText("0") {
		t.Fatal("a zero badge with ShowZero does not show")
	}
	if _, ok := tt.Access("Invisible"); ok {
		t.Fatal("a badge with Visible=false still shows")
	}
}

// boolPtr returns p.
func boolPtr(p bool) *bool { return &p }

// TestBadgedBoxAnchorsTopEnd checks the badge sits on the content's top-end
// corner and mirrors to the left under RTL.
func TestBadgedBoxAnchorsTopEnd(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := feedbackCanvas(th, func(c *ui.Context) {
		BadgedBox(c,
			func() { Button(c, "Inbox") },
			func() ui.Element { return Badge(c, "3", BadgeOptions{Label: "3 unread"}) },
			BadgedBoxOptions{Label: "Inbox group"},
		)
	})
	anchor := badgeInfo(t, tt, "Inbox")
	badge := badgeInfo(t, tt, "3 unread")
	if _, ok := tt.Access("Inbox group"); !ok {
		t.Fatal("the badged box lost its group description")
	}
	check := func(label string, wantX float32) {
		t.Helper()
		cx := badge.Bounds.X + badge.Bounds.W/2
		cy := badge.Bounds.Y + badge.Bounds.H/2
		if math.Abs(float64(cx-wantX)) > 2 {
			t.Fatalf("%s: badge center x %.1f, want the content's edge %.1f", label, cx, wantX)
		}
		if math.Abs(float64(cy-anchor.Bounds.Y)) > 2 {
			t.Fatalf("%s: badge center y %.1f, want the content's top %.1f", label, cy, anchor.Bounds.Y)
		}
	}
	check("LTR", anchor.Bounds.X+anchor.Bounds.W)
	tt.SetDirection(ui.RightToLeft)
	anchor = badgeInfo(t, tt, "Inbox")
	badge = badgeInfo(t, tt, "3 unread")
	check("RTL", anchor.Bounds.X)
}

// TestPlainTooltipDelayEscapeAndDescription checks the delayed tip, its
// InverseSurface panel, Escape, and the anchor's description.
func TestPlainTooltipDelayEscapeAndDescription(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := feedbackCanvas(th, func(c *ui.Context) {
		PlainTooltip(c, Button(c, "Share"), "Share with others")
	})
	info := buttonInfo(t, tt, ui.RoleButton, "Share")
	if info.Description != "Share with others" {
		t.Fatalf("anchor description %q, want the tooltip text", info.Description)
	}
	if tt.HasText("Share with others") {
		t.Fatal("the tooltip shows before the pointer rests on the anchor")
	}
	cx, cy := center(info.Bounds)
	tt.Move(cx, cy)
	tt.Advance(700 * time.Millisecond)
	if !tt.HasText("Share with others") {
		t.Fatal("the tooltip does not show after the hover delay")
	}
	tip, ok := tt.AccessByRole(ui.RoleTooltip, "")
	if !ok {
		t.Fatal("the plain tooltip is not a tooltip in the access tree")
	}
	px := tt.Image().RGBAAt(int(tip.Bounds.X+tip.Bounds.W/2), int(tip.Bounds.Y+2))
	if !colorMatches(px, th.Colors.InverseSurface, 40) {
		t.Fatalf("tooltip background %+v is not InverseSurface %+v", px, rgbOf(th.Colors.InverseSurface))
	}

	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if tt.HasText("Share with others") {
		t.Fatal("Escape does not hide the tooltip")
	}
}

// TestRichTooltipAction checks the rich tooltip's title, body and interactive
// action, which the pointer-through tooltip base could not press.
func TestRichTooltipAction(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	clicks := 0
	tt := feedbackCanvas(th, func(c *ui.Context) {
		RichTooltip(c, Button(c, "Details"), RichTooltipOptions{
			Title: "Details",
			Body:  "Shows everything.",
			Action: &RichTooltipAction{
				Label:   "Archive",
				OnClick: func() { clicks++ },
			},
			Label: "Details help",
		})
	})
	info := buttonInfo(t, tt, ui.RoleButton, "Details")
	cx, cy := center(info.Bounds)
	tt.Move(cx, cy)
	tt.Advance(700 * time.Millisecond)
	if !tt.HasText("Details") || !tt.HasText("Shows everything.") || !tt.HasText("Archive") {
		t.Fatalf("the rich tooltip does not show its title, body and action: %q", tt.Texts())
	}
	tt.Click("Archive")
	tt.Advance(500 * time.Millisecond)
	if clicks != 1 {
		t.Fatalf("the rich tooltip action ran %d times, want 1", clicks)
	}
	if tt.HasText("Shows everything.") {
		t.Fatal("the rich tooltip stays after its action")
	}
}

// TestRichTooltipEscape checks Escape closes a hovered rich tooltip.
func TestRichTooltipEscape(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := feedbackCanvas(th, func(c *ui.Context) {
		RichTooltip(c, Button(c, "Details"), RichTooltipOptions{
			Title: "Details",
			Body:  "Shows everything.",
		})
	})
	info := buttonInfo(t, tt, ui.RoleButton, "Details")
	cx, cy := center(info.Bounds)
	tt.Move(cx, cy)
	tt.Advance(700 * time.Millisecond)
	if !tt.HasText("Shows everything.") {
		t.Fatal("the rich tooltip does not show on hover")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if tt.HasText("Shows everything.") {
		t.Fatal("Escape does not hide the rich tooltip")
	}
}

// TestRichTooltipTabReachesAction checks mouse click does not pop the rich
// tooltip, while keyboard Tab opens it immediately and reaches its action.
func TestRichTooltipTabReachesAction(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := feedbackCanvas(th, func(c *ui.Context) {
		RichTooltip(c, Button(c, "Details"), RichTooltipOptions{
			Title:  "Details",
			Body:   "Shows everything.",
			Action: &RichTooltipAction{Label: "Archive"},
		})
	})
	tt.Click("Details")
	tt.Frame()
	if tt.HasText("Archive") {
		t.Fatal("mouse click opened the rich tooltip instead of keyboard focus")
	}
	tt.ClickAt(5, 5)
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	if !tt.HasText("Archive") {
		t.Fatal("keyboard Tab focus does not show the rich tooltip at once")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Archive") {
		t.Fatal("Tab does not reach the rich tooltip's action")
	}
}

// TestRichTooltipHoverGapAndNoEarlyOpen checks hovering off before Delay does not
// open the tooltip, while crossing the gap from anchor to panel keeps it open.
func TestRichTooltipHoverGapAndNoEarlyOpen(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	clicked := 0
	tt := feedbackCanvas(th, func(c *ui.Context) {
		RichTooltip(c, Button(c, "Info"), RichTooltipOptions{
			Title: "More info",
			Body:  "Detailed explanation.",
			Action: &RichTooltipAction{
				Label:   "Open",
				OnClick: func() { clicked++ },
			},
		})
	})
	info, ok := tt.AccessByRole(ui.RoleButton, "Info")
	if !ok {
		t.Fatal("Info button missing")
	}
	cx, cy := center(info.Bounds)
	tt.Move(cx, cy)
	tt.Advance(200 * time.Millisecond)
	tt.Move(5, 5)
	tt.Advance(600 * time.Millisecond)
	if tt.HasText("Detailed explanation.") {
		t.Fatal("rich tooltip opened after pointer left before Delay")
	}

	tt.Move(cx, cy)
	tt.Advance(700 * time.Millisecond)
	if !tt.HasText("Detailed explanation.") {
		t.Fatal("rich tooltip did not open after Delay")
	}
	act, ok := tt.AccessByRole(ui.RoleButton, "Open")
	if !ok {
		t.Fatal("Open action missing")
	}
	ax, ay := center(act.Bounds)
	// Cross the gap quickly (< 120ms) onto the action panel and confirm it stays open.
	tt.Move(cx, cy+info.Bounds.H/2+4)
	tt.Advance(40 * time.Millisecond)
	tt.Move(ax, ay)
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("Detailed explanation.") {
		t.Fatal("rich tooltip closed even after pointer moved onto the panel")
	}
	tt.ClickAt(ax, ay)
	tt.Advance(300 * time.Millisecond)
	if clicked != 1 {
		t.Fatalf("rich tooltip action ran %d times, want 1", clicked)
	}
}

// TestSnackbarQueueAndAction checks the FIFO queue, the action callback, the
// status role, and that everything settles.
func TestSnackbarQueueAndAction(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	state := &SnackbarState{}
	undos := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(480, 320).Background(canvasRGB).Children(func() {
				// The host is built before the trigger, so showing from the
				// button asks for the frame that draws the snackbar.
				SnackbarHost(c, state)
				ui.Box(c).Size(480, 320).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
					if Button(c, "Notify").Clicked() {
						state.Show(SnackbarOptions{
							Message:     "First",
							ActionLabel: "Undo",
							OnAction:    func() { undos++ },
						})
						state.Show(SnackbarOptions{Message: "Second"})
					}
				})
			})
		})
	}, 480, 320)

	tt.Click("Notify")
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	if !tt.HasText("First") {
		t.Fatal("the first snackbar does not show")
	}
	if tt.HasText("Second") {
		t.Fatal("the second snackbar shows before the first goes")
	}
	if _, ok := tt.AccessByRole(ui.RoleStatus, "First"); !ok {
		t.Fatal("the snackbar is not a status in the access tree")
	}

	tt.Click("Undo")
	// The pointer rests where Undo was; move it off so the next snackbar's
	// countdown runs.
	tt.Move(5, 5)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	if undos != 1 {
		t.Fatalf("the snackbar action ran %d times, want 1", undos)
	}
	if tt.HasText("First") {
		t.Fatal("the first snackbar stays after its action")
	}
	if !tt.HasText("Second") {
		t.Fatal("the queued snackbar does not follow the first")
	}

	tt.Advance(5 * time.Second)
	tt.Frame()
	if tt.HasText("Second") {
		t.Fatal("the second snackbar does not time out")
	}
	if tt.FramePending() {
		t.Fatal("the snackbar host keeps asking for frames")
	}
}

// TestSnackbarHoverPausesCountdown checks the countdown pauses while the pointer
// rests on the snackbar and resumes, timing out, once it leaves.
func TestSnackbarHoverPausesCountdown(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	state := &SnackbarState{}
	state.Show(SnackbarOptions{Message: "Saved", Duration: 3 * time.Second})
	tt := feedbackCanvas(th, func(c *ui.Context) {
		SnackbarHost(c, state)
	})
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	if !tt.HasText("Saved") {
		t.Fatal("the snackbar does not show")
	}
	info, ok := tt.AccessByRole(ui.RoleStatus, "Saved")
	if !ok {
		t.Fatal("the snackbar is not a status in the access tree")
	}
	cx, cy := center(info.Bounds)
	tt.Move(cx, cy)
	tt.Frame()
	tt.Advance(4 * time.Second)
	if !tt.HasText("Saved") {
		t.Fatal("the snackbar goes while the pointer rests on it")
	}
	tt.Move(5, 5)
	tt.Frame()
	tt.Advance(4 * time.Second)
	if tt.HasText("Saved") {
		t.Fatal("the snackbar does not time out after the pointer leaves")
	}
	if tt.FramePending() {
		t.Fatal("the snackbar host keeps asking for frames")
	}
}

// TestSnackbarDragDismissAndTheme checks a horizontal swipe dismisses the
// snackbar and that it inherits a local ThemeScope's inverse colors.
func TestSnackbarDragDismissAndTheme(t *testing.T) {
	th := ThemeFromSeed(ui.Hex("#006E2C"), ThemeOptions{Dark: true})
	state := &SnackbarState{}
	state.Show(SnackbarOptions{Message: "Swiped away", ShowClose: true})
	tt := feedbackCanvas(th, func(c *ui.Context) {
		SnackbarHost(c, state)
	})
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	info, ok := tt.AccessByRole(ui.RoleStatus, "Swiped away")
	if !ok {
		t.Fatal("the snackbar is not a status in the access tree")
	}
	// The InverseSurface fill, clear of text, the action and the close button.
	px := tt.Image().RGBAAt(int(info.Bounds.X+6), int(info.Bounds.Y+6))
	if !colorMatches(px, th.Colors.InverseSurface, 40) {
		t.Fatalf("snackbar background %+v is not the scoped InverseSurface %+v", px, rgbOf(th.Colors.InverseSurface))
	}

	x, y := info.Bounds.X+8, info.Bounds.Y+info.Bounds.H/2
	tt.Press(x, y)
	tt.Move(x+120, y)
	tt.Release(x+120, y)
	tt.Frame()
	tt.Advance(time.Second)
	if tt.HasText("Swiped away") {
		t.Fatal("a swipe past the threshold does not dismiss the snackbar")
	}
	if tt.FramePending() {
		t.Fatal("the snackbar host keeps asking for frames after a swipe")
	}
}

// TestSnackbarCloseButtonAdvancesQueue checks the close button dismisses the
// showing message, the next queued one follows, and closing the last one leaves
// the host quiet.
func TestSnackbarCloseButtonAdvancesQueue(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	state := &SnackbarState{}
	state.Show(SnackbarOptions{Message: "First", ShowClose: true})
	state.Show(SnackbarOptions{Message: "Second", ShowClose: true})
	tt := feedbackCanvas(th, func(c *ui.Context) {
		SnackbarHost(c, state)
	})
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	if !tt.HasText("First") {
		t.Fatal("the first snackbar does not show")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Dismiss"); !ok {
		t.Fatal("the close button is not a named button")
	}
	tt.Click("Dismiss")
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	if tt.HasText("First") {
		t.Fatal("the close button does not dismiss the showing snackbar")
	}
	if !tt.HasText("Second") {
		t.Fatal("the queued snackbar does not follow a closed one")
	}
	tt.Click("Dismiss")
	tt.Move(5, 5)
	tt.Frame()
	tt.Advance(time.Second)
	if tt.HasText("Second") {
		t.Fatal("the close button does not dismiss the last snackbar")
	}
	if tt.FramePending() {
		t.Fatal("the snackbar host keeps asking for frames after its queue empties")
	}
}

// TestSnackbarShortSwipeSpringsBack checks a drag let go below the threshold
// neither dismisses the snackbar nor leaves it displaced or animating.
func TestSnackbarShortSwipeSpringsBack(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	state := &SnackbarState{}
	state.Show(SnackbarOptions{Message: "Keep me", Duration: 30 * time.Second})
	tt := feedbackCanvas(th, func(c *ui.Context) {
		SnackbarHost(c, state)
	})
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	before, ok := tt.AccessByRole(ui.RoleStatus, "Keep me")
	if !ok {
		t.Fatal("the snackbar is not a status in the access tree")
	}
	x, y := before.Bounds.X+8, before.Bounds.Y+before.Bounds.H/2
	tt.Press(x, y)
	tt.Move(x+snackbarSwipeThreshold/2, y)
	tt.Release(x+snackbarSwipeThreshold/2, y)
	tt.Move(5, 5)
	tt.Frame()
	tt.Advance(time.Second)
	if !tt.HasText("Keep me") {
		t.Fatal("a short swipe dismisses the snackbar")
	}
	if tt.FramePending() {
		t.Fatal("the snackbar keeps animating after springing back")
	}
	after, ok := tt.AccessByRole(ui.RoleStatus, "Keep me")
	if !ok {
		t.Fatal("the snackbar left the access tree after springing back")
	}
	if absF32(after.Bounds.X-before.Bounds.X) > 1 {
		t.Fatalf("the snackbar rests at x %.1f, want its original %.1f", after.Bounds.X, before.Bounds.X)
	}
}

// TestSnackbarSwipePromotionAndExitActionGuard checks swiping one snackbar off
// keeps the next promoted snackbar centered and ignores clicks while exiting.
func TestSnackbarSwipePromotionAndExitActionGuard(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	state := &SnackbarState{}
	actions := 0
	state.Show(SnackbarOptions{
		Message:     "First note",
		ActionLabel: "Retry",
		OnAction:    func() { actions++ },
		Duration:    30 * time.Second,
	})
	state.Show(SnackbarOptions{
		Message:  "Second note",
		Duration: 30 * time.Second,
	})
	tt := feedbackCanvas(th, func(c *ui.Context) {
		SnackbarHost(c, state)
	})
	tt.Frame()
	tt.Advance(600 * time.Millisecond)

	first, ok := tt.AccessByRole(ui.RoleStatus, "First note")
	if !ok {
		t.Fatal("first snackbar missing")
	}
	retryBtn, ok := tt.AccessByRole(ui.RoleButton, "Retry")
	if !ok {
		t.Fatal("Retry button missing")
	}
	x, y := first.Bounds.X+8, first.Bounds.Y+first.Bounds.H/2
	tt.Press(x, y)
	tt.Move(x+snackbarSwipeThreshold+20, y)
	tt.Release(x+snackbarSwipeThreshold+20, y)
	// Try clicking the Retry button while the first snackbar is exiting.
	tt.Advance(16 * time.Millisecond)
	tt.ClickAt(retryBtn.Bounds.X+retryBtn.Bounds.W/2, retryBtn.Bounds.Y+retryBtn.Bounds.H/2)
	tt.Advance(700 * time.Millisecond)

	if actions != 0 {
		t.Fatalf("exiting snackbar fired OnAction %d times, want 0", actions)
	}
	second, ok := tt.AccessByRole(ui.RoleStatus, "Second note")
	if !ok {
		t.Fatal("second snackbar was not promoted")
	}
	center1 := first.Bounds.X + first.Bounds.W/2
	center2 := second.Bounds.X + second.Bounds.W/2
	if absF32(center2-center1) > 2 {
		t.Fatalf("promoted snackbar center %.1f drifted from centered %.1f", center2, center1)
	}
}

// TestEmptyBadgePlainTooltipAndReduceMotionSnackbar checks blank Badges/
// PlainTooltips render no empty chrome and ReduceMotion snackbars leave zero
// lingering frames after showing and closing.
func TestEmptyBadgePlainTooltipAndReduceMotionSnackbar(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	cs := th.Colors
	state := &SnackbarState{}
	tt := feedbackCanvas(th, func(c *ui.Context) {
		Badge(c, "", BadgeOptions{Label: "Blank"})
		Badge(c, "   ", BadgeOptions{Label: "Spaces"})
		PlainTooltip(c, Button(c, "Quiet"), "   ")
		if Button(c, "Pop").Clicked() {
			state.Show(SnackbarOptions{Message: "RM Saved", Duration: 2 * time.Second})
		}
		SnackbarHost(c, state)
	})
	tt.Frame()
	if _, ok := tt.Access("Blank"); ok {
		t.Fatal("empty Badge rendered an access node")
	}
	if _, ok := tt.Access("Spaces"); ok {
		t.Fatal("whitespace Badge rendered an access node")
	}
	img := tt.Image()
	for y := 0; y < 320; y += 8 {
		for x := 0; x < 480; x += 8 {
			if colorMatches(img.RGBAAt(x, y), cs.Error, 10) {
				t.Fatalf("empty Badge painted Error pixel at (%d,%d)", x, y)
			}
		}
	}

	quiet, _ := tt.AccessByRole(ui.RoleButton, "Quiet")
	tt.Move(center(quiet.Bounds))
	tt.Advance(800 * time.Millisecond)

	tt.Click("Pop")
	tt.Frame()
	if !tt.HasText("RM Saved") {
		t.Fatal("ReduceMotion snackbar did not appear")
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion snackbar keeps asking for build frames")
	}
	tt.Advance(2500 * time.Millisecond)
	if tt.HasText("RM Saved") {
		t.Fatal("ReduceMotion snackbar did not time out")
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion snackbar kept requesting frames after exit")
	}
}

// TestRichTooltipGhostExitAndLeaveGrace checks unmounting an anchor during an
// exit transition does not panic or leave lingering frames, and hover leave
// grace closes cleanly with zero pending frames.
func TestRichTooltipGhostExitAndLeaveGrace(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	exitTr := ui.ElementTransition{
		Duration: 200 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}
	showAnchor := true
	tt := feedbackCanvas(th, func(c *ui.Context) {
		ui.Column(c).Gap(12).Children(func() {
			if Button(c, "Toggle").Clicked() {
				showAnchor = !showAnchor
			}
			if showAnchor {
				// Use a plain clickable Box without Ripple so RichTooltip owns anchor.DrawOver.
				plain := ui.Box(c).Key("plain").Padding(8, 12).Radius(8).
					Role(ui.RoleButton).Label("PlainAnchor").
					Transition(exitTr)
				plain.Children(func() {
					ui.Text(c, "PlainAnchor")
				})
				RichTooltip(c, plain, RichTooltipOptions{
					Title: "Plain tip",
					Body:  "Plain body",
				})
				// Also test Ripple-backed Button inside an exiting block.
				wrap := ui.Box(c).Key("wrap").Transition(exitTr)
				wrap.Children(func() {
					RichTooltip(c, Button(c.Key("btn-anchor"), "ButtonAnchor"), RichTooltipOptions{
						Title: "Btn tip",
						Body:  "Button body",
					})
				})
			}
		})
	})

	plainInfo := buttonInfo(t, tt, ui.RoleButton, "PlainAnchor")
	px, py := center(plainInfo.Bounds)
	tt.Move(px, py)
	tt.Advance(700 * time.Millisecond)
	if !tt.HasText("Plain body") {
		t.Fatal("rich tooltip did not open before leave/unmount")
	}

	// Move pointer away and verify leave grace closes cleanly with no pending frames.
	tt.Move(5, 5)
	tt.Advance(300 * time.Millisecond)
	if tt.HasText("Plain body") {
		t.Fatal("rich tooltip stayed open after leave grace elapsed")
	}
	if tt.FramePending() {
		t.Fatal("rich tooltip leave timer left pending frames")
	}

	// Re-open on PlainAnchor, then unmount during exit transition so ghost DrawOver runs.
	tt.Move(px, py)
	tt.Advance(700 * time.Millisecond)
	if !tt.HasText("Plain body") {
		t.Fatal("rich tooltip did not reopen")
	}
	tt.Click("Toggle")
	tt.Advance(400 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("exiting rich tooltip anchor left pending frames")
	}
}

// TestRichTooltipClampsToViewport checks the rich tooltip clamps its width
// and height within a compact viewport while keeping Title and Action visible.
func TestRichTooltipClampsToViewport(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	var longBody strings.Builder
	for range 20 {
		longBody.WriteString("Long tooltip line of supporting explanation text.\n")
	}
	clicked := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 180).Background(canvasRGB).
				Padding(12).AlignItems(ui.Start).Children(func() {
				RichTooltip(c, Button(c, "More"), RichTooltipOptions{
					Title: "Compact Tip",
					Body:  longBody.String(),
					Action: &RichTooltipAction{
						Label:   "Got it",
						OnClick: func() { clicked++ },
					},
				})
			})
		})
	}, 240, 180)

	info := buttonInfo(t, tt, ui.RoleButton, "More")
	tt.Move(center(info.Bounds))
	tt.Advance(700 * time.Millisecond)

	panel, ok := tt.AccessByRole(ui.RoleGroup, "Compact Tip")
	if !ok {
		t.Fatal("rich tooltip group missing in compact viewport")
	}
	if panel.Bounds.X < 0 || panel.Bounds.X+panel.Bounds.W > 240.5 {
		t.Fatalf("rich tooltip escaped viewport horizontally: %+v", panel.Bounds)
	}
	if panel.Bounds.Y < 0 || panel.Bounds.Y+panel.Bounds.H > 180.5 {
		t.Fatalf("rich tooltip escaped viewport vertically: %+v", panel.Bounds)
	}
	act := buttonInfo(t, tt, ui.RoleButton, "Got it")
	if act.Bounds.Y+act.Bounds.H > panel.Bounds.Y+panel.Bounds.H+0.5 {
		t.Fatalf("rich tooltip action clipped below panel: action=%+v panel=%+v", act.Bounds, panel.Bounds)
	}
	tt.ClickAt(center(act.Bounds))
	tt.Frame()
	if clicked != 1 {
		t.Fatalf("action clicked %d times, want 1", clicked)
	}
}
