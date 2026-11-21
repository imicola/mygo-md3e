package md3e

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestPageTransitionFadeThroughAndSameKeyNoRedraw(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	pageKey := "alpha"
	siblingW := float32(60)
	clicksA, clicksB := 0, 0
	colA := ui.Hex("#1E88E5")
	colB := ui.Hex("#2E7D32")

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Row(c).Fill().Children(func() {
				ui.Box(c).Width(siblingW).FillHeight()
				PageTransition(c.Key("main-stage"), pageKey, func() {
					if pageKey == "alpha" {
						ui.Column(c).Fill().Background(colA).Padding(16).Children(func() {
							if Button(c.Key("btnA"), "Action A").Clicked() {
								clicksA++
							}
						})
					} else {
						ui.Column(c).Fill().Background(colB).Padding(16).Justify(ui.End).Children(func() {
							if Button(c.Key("btnB"), "Action B").Clicked() {
								clicksB++
							}
						})
					}
				}).Grow(1).FillHeight()
			})
		}).Fill()
	}, 360, 220)

	if tt.FramePending() {
		t.Fatal("initial PageTransition mount should settle immediately without animation frames")
	}
	btnA, ok := tt.AccessByRole(ui.RoleButton, "Action A")
	if !ok {
		t.Fatal("initial page A button missing from access tree")
	}

	// Repeated builds with the same key (even when sibling width resizes the
	// stage) must produce zero animation frames.
	for i := range 5 {
		if i == 2 {
			siblingW = 100
		}
		tt.Frame()
		if tt.FramePending() {
			t.Fatalf("same-key frame %d unexpectedly scheduled animation frames", i)
		}
	}

	// Switch alpha -> beta: triggers FadeThrough transition, outgoing page A
	// becomes inert immediately, and transition settles cleanly.
	pageKey = "beta"
	tt.Frame()
	if !tt.FramePending() {
		t.Fatal("changing PageTransition key did not schedule transition frames")
	}
	tt.Advance(32 * time.Millisecond)

	if _, ok := tt.AccessByRole(ui.RoleButton, "Action A"); ok {
		t.Fatal("exiting page A button should not remain in access tree during transition")
	}
	// Clicking where Action A used to sit must not trigger page A's handler.
	tt.ClickAt(btnA.Bounds.X+btnA.Bounds.W/2, btnA.Bounds.Y+btnA.Bounds.H/2)
	if clicksA != 0 {
		t.Fatalf("exiting ghost page A accepted click (%d)", clicksA)
	}

	// Mid-transition pixel in page area should be transitioning toward colB
	// rather than fully stuck at colA.
	midPx := tt.Image().RGBAAt(220, 140)
	if colorMatches(midPx, colA, 4) && colorMatches(midPx, colB, 4) {
		t.Fatalf("unexpected indistinguishable mid-transition pixel %+v", midPx)
	}

	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("PageTransition FadeThrough kept requesting frames after settle")
	}
	settledPx := tt.Image().RGBAAt(220, 140)
	if !colorMatches(settledPx, colB, 8) {
		t.Fatalf("settled page B pixel = %+v, want %+v", settledPx, rgbOf(colB))
	}
	if err := tt.Click("Action B"); err != nil {
		t.Fatalf("clicking settled page B button: %v", err)
	}
	if clicksB != 1 {
		t.Fatalf("page B clicks = %d, want 1", clicksB)
	}
}

func TestPageTransitionSharedAxisDirectionRTLAndAnimatedContent(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	markerA := ui.Hex("#D32F2F")
	markerB := ui.Hex("#1976D2")

	// 1. SharedAxisY with AnimatedContent[int]: auto-infers forward (+1) when
	// key increases (0 -> 2) and reverses (-1) when key decreases (2 -> 0),
	// updating outgoing ghost direction via ChildExit.
	tab := 0
	ttY := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			AnimatedContent(c.Key("axis-y"), tab, func() {
				col := markerA
				if tab == 2 {
					col = markerB
				}
				ui.Column(c).Fill().Center().Children(func() {
					ui.Box(c).Size(60, 20).Background(col).Role(ui.RoleRegion).Label("LivePage")
				})
			}, TransitionOptions{
				Mode:     TransitionSharedAxisY,
				Distance: 40,
			}).Fill()
		}).Fill()
	}, 240, 180)

	if ttY.FramePending() {
		t.Fatal("initial AnimatedContent mount should start settled")
	}
	restPage, ok := ttY.AccessByRole(ui.RoleRegion, "LivePage")
	if !ok {
		t.Fatal("initial LivePage missing from access tree")
	}
	restY := restPage.Bounds.Y
	restMinY, restMaxY, restCnt := verticalColorSpan(ttY.Image(), 90, 150, 0, 180, markerA, 16)
	if restCnt == 0 {
		t.Fatal("initial markerA did not paint")
	}
	restMarkerCenterY := float32(restMinY+restMaxY) / 2

	// Helper to locate semi-transparent ghost pixels by dominant channel.
	dominantSpan := func(wantRed bool) (minY, maxY, cnt int) {
		minY, maxY = -1, -1
		img := ttY.Image()
		for y := 0; y < 180; y++ {
			for x := 90; x < 150; x++ {
				px := img.RGBAAt(x, y)
				match := false
				if wantRed {
					match = int(px.R)-int(px.B) > 16 && int(px.R)-int(px.G) > 16
				} else {
					match = int(px.B)-int(px.R) > 16 && int(px.B)-int(px.G) > 8
				}
				if match {
					if minY < 0 {
						minY = y
					}
					maxY = y
					cnt++
				}
			}
		}
		return minY, maxY, cnt
	}

	// Forward 0 -> 2: incoming starts below (+Y), outgoing ghost exits above (-Y).
	tab = 2
	ttY.Frame()
	if !ttY.FramePending() {
		t.Fatal("SharedAxisY forward switch did not start animation")
	}
	ttY.Advance(32 * time.Millisecond)
	liveFwd, ok := ttY.AccessByRole(ui.RoleRegion, "LivePage")
	if !ok {
		t.Fatal("entering LivePage missing mid-transition")
	}
	if !(liveFwd.Bounds.Y > restY+2) {
		t.Fatalf("SharedAxisY forward incoming Y = %.1f, want > rest %.1f", liveFwd.Bounds.Y, restY)
	}
	// Verify exiting ghost of tab 0 (markerA) actually moved above restMarkerCenterY.
	ghostFwdMin, ghostFwdMax, ghostFwdCnt := dominantSpan(true)
	if ghostFwdCnt == 0 || !(float32(ghostFwdMin+ghostFwdMax)/2 < restMarkerCenterY-2) {
		t.Fatalf("SharedAxisY forward exiting ghost markerA span=[%d,%d] (cnt=%d), want center < %.1f",
			ghostFwdMin, ghostFwdMax, ghostFwdCnt, restMarkerCenterY-2)
	}
	ttY.Advance(time.Second)
	if ttY.FramePending() {
		t.Fatal("SharedAxisY forward transition did not settle")
	}
	settledFwd, _ := ttY.AccessByRole(ui.RoleRegion, "LivePage")
	if absF32(settledFwd.Bounds.Y-restY) > 0.5 {
		t.Fatalf("settled SharedAxisY Y = %.1f, want %.1f", settledFwd.Bounds.Y, restY)
	}

	// Reverse 2 -> 0: incoming starts above (-Y), outgoing ghost exits below (+Y).
	tab = 0
	ttY.Frame()
	ttY.Advance(32 * time.Millisecond)
	liveRev, ok := ttY.AccessByRole(ui.RoleRegion, "LivePage")
	if !ok {
		t.Fatal("reversing LivePage missing mid-transition")
	}
	if !(liveRev.Bounds.Y < restY-2) {
		t.Fatalf("SharedAxisY reverse incoming Y = %.1f, want < rest %.1f", liveRev.Bounds.Y, restY)
	}
	ghostRevMin, ghostRevMax, ghostRevCnt := dominantSpan(false)
	if ghostRevCnt == 0 || !(float32(ghostRevMin+ghostRevMax)/2 > restMarkerCenterY+2) {
		t.Fatalf("SharedAxisY reverse exiting ghost markerB span=[%d,%d] (cnt=%d), want center > %.1f",
			ghostRevMin, ghostRevMax, ghostRevCnt, restMarkerCenterY+2)
	}
	ttY.Advance(time.Second)
	if ttY.FramePending() {
		t.Fatal("SharedAxisY reverse transition did not settle")
	}

	// 2. SharedAxisX mirrors direction automatically in RTL.
	step := 0
	ttX := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			PageTransition(c.Key("axis-x"), step, func() {
				ui.Column(c).Fill().Center().Children(func() {
					ui.Box(c).Size(40, 40).Role(ui.RoleRegion).Label("LivePageX")
				})
			}, TransitionOptions{
				Mode:     TransitionSharedAxisX,
				Distance: 40,
			}).Fill()
		}).Fill()
	}, 240, 180)
	restXPage, _ := ttX.AccessByRole(ui.RoleRegion, "LivePageX")
	restX := restXPage.Bounds.X

	// LTR forward 0 -> 1 enters from +X.
	step = 1
	ttX.Frame()
	ttX.Advance(32 * time.Millisecond)
	liveLTR, _ := ttX.AccessByRole(ui.RoleRegion, "LivePageX")
	if !(liveLTR.Bounds.X > restX+2) {
		t.Fatalf("SharedAxisX LTR forward incoming X = %.1f, want > rest %.1f", liveLTR.Bounds.X, restX)
	}
	ttX.Advance(time.Second)
	if ttX.FramePending() {
		t.Fatal("SharedAxisX LTR transition did not settle")
	}

	// RTL forward 1 -> 2 mirrors and enters from -X.
	ttX.SetDirection(ui.RightToLeft)
	step = 2
	ttX.Frame()
	ttX.Advance(32 * time.Millisecond)
	liveRTL, _ := ttX.AccessByRole(ui.RoleRegion, "LivePageX")
	if !(liveRTL.Bounds.X < restX-2) {
		t.Fatalf("SharedAxisX RTL forward incoming X = %.1f, want < rest %.1f", liveRTL.Bounds.X, restX)
	}
	ttX.Advance(time.Second)
	if ttX.FramePending() {
		t.Fatal("SharedAxisX RTL transition did not settle")
	}
}

func TestPageTransitionReduceMotionNoneAndGhostSafety(t *testing.T) {
	// 1. ReduceMotion via Preferences snaps immediately across all modes.
	for _, mode := range []TransitionMode{TransitionFadeThrough, TransitionSharedAxisY, TransitionSharedAxisX} {
		cur := 0
		tt := ui.NewTesterWith(func(c *ui.Context) {
			PageTransition(c.Key("rm-pref"), cur, func() {
				ui.Text(c, "Page")
			}, TransitionOptions{Mode: mode}).Fill()
		}, 200, 120, ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})
		if tt.FramePending() {
			t.Fatalf("mode %v initial ReduceMotion mount left frames pending", mode)
		}
		cur = 1
		tt.Frame()
		if tt.FramePending() {
			t.Fatalf("mode %v ReduceMotion key switch left frames pending", mode)
		}
	}

	// 2. ReduceMotion via Theme and TransitionNone swap immediately with zero frames.
	rmTheme := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	kTheme := "one"
	ttRM := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, rmTheme, func() {
			PageTransition(c.Key("rm-theme"), kTheme, func() {
				ui.Text(c, kTheme)
			}, TransitionOptions{Mode: TransitionSharedAxisY}).Fill()
		}).Fill()
	}, 200, 120)
	kTheme = "two"
	ttRM.Frame()
	if ttRM.FramePending() {
		t.Fatal("Theme.ReduceMotion PageTransition switch left frames pending")
	}
	if !ttRM.HasText("two") || ttRM.HasText("one") {
		t.Fatal("Theme.ReduceMotion PageTransition did not swap content immediately")
	}

	kNone := "x"
	ttNone := ui.NewTester(func(c *ui.Context) {
		PageTransition(c.Key("none"), kNone, func() {
			ui.Text(c, kNone)
		}, TransitionOptions{Mode: TransitionNone}).Fill()
	}, 200, 120)
	kNone = "y"
	ttNone.Frame()
	if ttNone.FramePending() {
		t.Fatal("TransitionNone left frames pending")
	}

	// 3. Dynamic Preferences.ReduceMotion enabled right on key switch leaves zero frames.
	kDyn := 0
	ttDyn := ui.NewTester(func(c *ui.Context) {
		PageTransition(c.Key("dyn-rm"), kDyn, func() {
			ui.Box(c).Fill()
		}, TransitionOptions{Mode: TransitionFadeThrough}).Fill()
	}, 200, 120)
	kDyn = 1
	ttDyn.SetPreferences(ui.Preferences{ReduceMotion: true})
	if ttDyn.FramePending() {
		t.Fatal("enabling Preferences.ReduceMotion on key switch left frames pending")
	}

	// 4. Exit ghost safety: animated custom-drawn children inside outgoing page
	// paint cleanly with !el.Valid(), never schedule stray frames, and settle.
	th := ThemeFromSeed(DefaultSeedColor)
	ghostKey := 0
	sawInvalidDraw := false
	ttGhost := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			PageTransition(c.Key("ghost-stage"), ghostKey, func() {
				if ghostKey == 0 {
					ui.Column(c).Fill().Padding(12).Gap(8).Children(func() {
						LoadingIndicator(c)
						LinearWavyProgress(c, 0.5)
						probe := ui.Box(c).Size(40, 20)
						probe.Draw(func(_ *ui.Painter, _ ui.Rect) {
							if !probe.Valid() {
								sawInvalidDraw = true
							}
						})
					})
				} else {
					ui.Column(c).Fill().Padding(12).Children(func() {
						ui.Text(c, "Settled Page")
					})
				}
			}, TransitionOptions{
				Mode:       TransitionFadeThrough,
				Distance:   float32(math.NaN()),
				Scale:      float32(math.Inf(1)),
				AfterExits: true,
			}).Fill()
		}).Fill()
	}, 280, 180)

	ghostKey = 1
	ttGhost.Frame()
	for range 8 {
		ttGhost.Advance(24 * time.Millisecond)
	}
	if !sawInvalidDraw {
		t.Fatal("expected exiting page ghost to run Draw with !el.Valid()")
	}
	ttGhost.Advance(800 * time.Millisecond)
	if ttGhost.FramePending() {
		t.Fatal("PageTransition with animated exiting children did not settle cleanly")
	}

	// 5. Nil context, nil build, and out-of-range mode degrade safely; >1 options panics.
	if el := PageTransition(nil, "k", nil); el.Valid() {
		t.Fatal("PageTransition(nil, ...) should return invalid Element")
	}
	ttSafe := ui.NewTester(func(c *ui.Context) {
		PageTransition(c, "empty", nil, TransitionOptions{Mode: TransitionMode(99)})
	}, 100, 80)
	if ttSafe.FramePending() {
		t.Fatal("nil-build PageTransition left frames pending")
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected PageTransition with 2 options to panic")
		}
	}()
	ui.NewTester(func(c *ui.Context) {
		PageTransition(c, "a", func() {}, TransitionOptions{}, TransitionOptions{})
	}, 100, 80)
}
