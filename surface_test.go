package md3e

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

func absColorDiff(a, b color.RGBA) int {
	abs := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}
	return abs(int(a.R)-int(b.R)) + abs(int(a.G)-int(b.G)) + abs(int(a.B)-int(b.B))
}

func TestElevationSurfaceColorsAndShadows(t *testing.T) {
	light := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Dark: false})
	dark := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Dark: true})

	for _, tc := range []struct {
		name string
		th   *Theme
	}{
		{name: "light", th: light},
		{name: "dark", th: dark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			levels := []tokens.ElevationLevel{
				tokens.Level0, tokens.Level1, tokens.Level2,
				tokens.Level3, tokens.Level4, tokens.Level5,
			}
			var rendered []color.RGBA
			for _, lvl := range levels {
				img := ui.Render(func(c *ui.Context) {
					ThemeScope(c, tc.th, func() {
						ui.Box(c).Size(120, 120).Background(ui.RGB(255, 255, 255)).Padding(20).Children(func() {
							Surface(c, SurfaceOptions{
								Elevation: lvl,
								Radius:    8,
							}).Size(60, 40)
						})
					})
				}, 120, 120, 1)

				// Sample surface interior center at (50, 40).
				center := img.RGBAAt(50, 40)
				rendered = append(rendered, center)

				// Sample shadow region just below the 60x40 box (placed at (20,20)..(80,60)): (50, 62).
				below := img.RGBAAt(50, 62)
				if lvl == tokens.Level0 {
					if below != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
						t.Fatalf("Level0 rendered shadow pixel below box: %+v", below)
					}
				} else {
					if below == (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
						t.Fatalf("%s did not render shadow pixel below surface", tc.name)
					}
				}
			}

			for i := 0; i < len(rendered)-1; i++ {
				if rendered[i] == rendered[i+1] {
					t.Fatalf("%s elevation Level%d and Level%d rendered identical center pixel %+v",
						tc.name, i, i+1, rendered[i])
				}
			}

			base := ui.RGB(240, 240, 240)
			tinted := ApplyTonalElevation(tc.th, base, tokens.Level3)
			if tinted == base {
				t.Fatalf("%s ApplyTonalElevation did not tint custom base color", tc.name)
			}
		})
	}
}

// TestRippleOnButtonBaseDoesNotDoublePressLayer checks that a real button's
// ripple is the pressed feedback alone, that the layer stays at the hover
// opacity, and that a press on the top-left corner starts the wave there
// instead of jumping to the gradient's default center.
func TestRippleOnButtonBaseDoesNotDoublePressLayer(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(280, 160).Background(ui.RGB(255, 255, 255)).Padding(20).Children(func() {
				b := ui.ButtonBase(c).Size(200, 80).Background(ui.RGB(255, 255, 255)).Label("Pad")
				_ = b.Clicked()
				AttachRipple(c, b, StateLayerOptions{Color: ui.RGB(0, 0, 0), BorderInset: 1})
			})
		})
	}, 280, 160)

	b := buttonInfo(t, tt, ui.RoleButton, "Pad")
	x, y := b.Bounds.X+1, b.Bounds.Y+1
	// A point outside the label near the pressed corner, and the corner itself.
	farX := int(b.Bounds.X) + 30
	midY := int(b.Bounds.Y + b.Bounds.H/2)
	tt.Move(x, y)
	tt.Press(x, y)
	corner := tt.Image().RGBAAt(int(b.Bounds.X)+3, int(b.Bounds.Y)+3)
	far := tt.Image().RGBAAt(farX, midY)
	if corner.R >= far.R {
		t.Fatalf("wave did not start at the pressed corner: corner R=%d far R=%d", corner.R, far.R)
	}

	tt.Advance(400 * time.Millisecond)
	settled := tt.Image().RGBAAt(farX, midY)
	// Hover 8% plus the 10% ripple leaves ~211; the doubled pressed layer
	// would be ~206.
	if settled.R < 208 || settled.R > 216 {
		t.Fatalf("pressed ripple pixel R = %d, want about 211 (no doubled 10%% layer)", settled.R)
	}
	tt.Release(x, y)
	tt.Move(0, 0)
	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("the pressed ripple did not settle")
	}
}

// TestSurfaceTonalElevationOverride checks that a lower TonalElevation can
// override a higher Elevation's tone while its shadows remain.
func TestSurfaceTonalElevationOverride(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	img := ui.Render(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(120, 120).Background(ui.RGB(255, 255, 255)).Padding(20).Children(func() {
				Surface(c, SurfaceOptions{
					Elevation:      tokens.Level3,
					TonalElevation: tokens.Level1,
					Radius:         8,
				}).Size(60, 40)
			})
		})
	}, 120, 120, 1)

	center := img.RGBAAt(50, 40)
	if want := rgbOf(SurfaceColorForElevation(th, tokens.Level1)); absColorDiff(center, want) > 4 {
		t.Fatalf("tonal override center = %v, want the Level1 tone %v", center, want)
	}
	if below := img.RGBAAt(50, 62); below == (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatal("the Level3 shadows disappeared with the lower tonal override")
	}
}

// TestSurfacePolygonClip checks that a custom polygon Surface with Clip keeps
// its children inside the surface's box.
func TestSurfacePolygonClip(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	gray := color.RGBA{R: 220, G: 220, B: 220, A: 255}
	img := ui.Render(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 200).Background(ui.RGB(220, 220, 220)).Padding(20).Children(func() {
				s := Surface(c, SurfaceOptions{
					Color: ui.RGB(255, 255, 255),
					Shape: shape.Circle(),
					Clip:  true,
				}).Size(80, 80)
				s.Children(func() {
					ui.Box(c).Size(300, 300).Background(ui.RGB(255, 0, 0))
				})
			})
		})
	}, 240, 200, 1)

	if px := img.RGBAAt(60, 60); px.R < 200 {
		t.Fatalf("the oversized child did not paint inside the surface: %v", px)
	}
	if px := img.RGBAAt(160, 60); px != gray {
		t.Fatalf("the oversized child escaped the clipped surface: %v", px)
	}
}

func TestTypographyScaleAndTextHelper(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{TextScale: 1.5})

	var gotMetrics [tokens.TypeRoleCount]TypeStyleMetrics
	var emphMetrics [tokens.TypeRoleCount]TypeStyleMetrics

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Children(func() {
				for role := tokens.TypeRole(0); role < tokens.TypeRoleCount; role++ {
					gotMetrics[role] = TypeStyleOf(c, role)
					emphMetrics[role] = TypeStyleOf(c, role, true)
				}
				Text(c, "Display", tokens.DisplayLarge)
				Text(c, "Body", tokens.BodyMedium)
				Text(c, "Label", tokens.LabelSmall, true)
			})
		})
	}, 400, 400)

	for role := tokens.TypeRole(0); role < tokens.TypeRoleCount; role++ {
		wantStd := th.Typography.Style(role, false)
		wantEmp := th.Typography.Style(role, true)
		if math.Abs(float64(gotMetrics[role].Size)-wantStd.Size*1.5) > 0.01 {
			t.Fatalf("role %d scaled size = %v, want %v", role, gotMetrics[role].Size, wantStd.Size*1.5)
		}
		if emphMetrics[role].Weight != wantEmp.Weight || emphMetrics[role].Weight <= gotMetrics[role].Weight {
			t.Fatalf("role %d emphasized weight = %d, std = %d", role, emphMetrics[role].Weight, gotMetrics[role].Weight)
		}
	}

	dispRect, ok1 := tt.Find("Display")
	bodyRect, ok2 := tt.Find("Body")
	labelRect, ok3 := tt.Find("Label")
	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("expected all styled texts in layout: %v %v %v", ok1, ok2, ok3)
	}
	if dispRect.H <= bodyRect.H || bodyRect.H <= labelRect.H {
		t.Fatalf("unexpected text heights: display=%v body=%v label=%v", dispRect.H, bodyRect.H, labelRect.H)
	}
}

func TestStateLayerHoverZeroRebuildAndOpacity(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	builds := 0

	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, th, func() {
			ui.Box(c).Size(200, 120).Background(ui.RGB(255, 255, 255)).Padding(20).Children(func() {
				s := Surface(c, SurfaceOptions{
					Color:  ui.RGB(255, 255, 255),
					Radius: 12,
				}).Size(120, 60)
				AttachStateLayer(c, s, StateLayerOptions{
					Color:  ui.RGB(0, 0, 0),
					Radius: 12,
				})
			})
		})
	}, 200, 120)

	if builds != 1 {
		t.Fatalf("initial builds = %d, want 1", builds)
	}
	idlePx := tt.Image().RGBAAt(80, 50)
	if idlePx != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("idle interior pixel = %+v, want pure white", idlePx)
	}

	// Hover inside Surface: must not spin rebuilds on frame, and DrawOver animates
	// the 8% StateLayer strictly via partial repaints (0 rebuilds across Advance).
	tt.Move(80, 50)
	if builds > 2 {
		t.Fatalf("builds after hover Move = %d, expected at most 1 Tester settle build", builds)
	}
	afterMoveBuilds := builds
	tt.Advance(48 * time.Millisecond)
	midHoverPx := tt.Image().RGBAAt(80, 50)
	tt.Advance(220 * time.Millisecond)
	if builds != afterMoveBuilds {
		t.Fatalf("builds grew during StateLayer hover animation: got %d, want %d (0 rebuilds during Advance)", builds, afterMoveBuilds)
	}
	if tt.FramePending() {
		t.Fatal("FramePending should be false once StateLayer hover settled")
	}
	hoverPx := tt.Image().RGBAAt(80, 50)
	if midHoverPx.R >= idlePx.R || midHoverPx.R <= hoverPx.R {
		t.Fatalf("expected smooth hover fade mid (%d) between idle (%d) and settled (%d)", midHoverPx.R, idlePx.R, hoverPx.R)
	}
	// Black at 0.08 alpha over white 255 -> 255 * (1 - 0.08) ≈ 235.
	if hoverPx.R < 229 || hoverPx.R > 240 {
		t.Fatalf("hover pixel R = %d, want ~235 (8%% black overlay on white)", hoverPx.R)
	}

	// Move pointer out: DrawOver fades StateLayer back to 0 without rebuilding during Advance.
	tt.Move(5, 5)
	afterLeaveBuilds := builds
	tt.Advance(250 * time.Millisecond)
	if builds != afterLeaveBuilds {
		t.Fatalf("builds grew during StateLayer hover leave fade: got %d, want %d", builds, afterLeaveBuilds)
	}
	if tt.FramePending() {
		t.Fatal("FramePending should be false after StateLayer faded out")
	}
	leftPx := tt.Image().RGBAAt(80, 50)
	if leftPx != idlePx {
		t.Fatalf("pixel after hover leave = %+v, want %+v", leftPx, idlePx)
	}
}

func TestExpressiveRipplePropagationClippingAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	asymmetricRadii := [4]float32{28, 0, 28, 0}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			// Window-sized root canvas filled neutral gray so corner clip bleed is unmistakably distinct.
			ui.Box(c).Size(240, 120).Background(ui.RGB(220, 220, 220)).Padding(20).Children(func() {
				s := Surface(c, SurfaceOptions{
					Color:    ui.RGB(255, 255, 255),
					Radii:    asymmetricRadii,
					HasRadii: true,
					Clip:     true,
				}).Size(200, 80)
				AttachRipple(c, s, StateLayerOptions{
					Color:    ui.RGB(0, 0, 0),
					Radii:    asymmetricRadii,
					HasRadii: true,
				})
			})
		})
	}, 240, 120)

	// Surface occupies [20, 20]..[220, 100].
	// Top-left corner pixel (22, 22) is (2, 2) local — outside 28dp rounded corner (dist from (28,28) is ~36.8 > 28).
	// Bottom-right corner pixel (218, 98) is (198, 78) local — outside 28dp bottom-right rounded corner.
	outsideTL := color.RGBA{R: 220, G: 220, B: 220, A: 255}

	if px := tt.Image().RGBAAt(22, 22); px != outsideTL {
		t.Fatalf("top-left outside 28dp corner = %+v, want background %+v", px, outsideTL)
	}
	if px := tt.Image().RGBAAt(218, 98); px != outsideTL {
		t.Fatalf("bottom-right outside 28dp corner = %+v, want background %+v", px, outsideTL)
	}

	// Press near left edge of surface at window (44, 60) -> local (24, 40).
	tt.Press(44, 60)
	nearPress := tt.Image().RGBAAt(46, 60)
	farSide := tt.Image().RGBAAt(205, 60) // distance ~161dp; at t=0 initial radius is ~0.20*maxR (~36dp).
	if nearPress.R >= 245 {
		t.Fatalf("expected ripple darken near press origin at t=0, got %+v", nearPress)
	}
	if farSide.R <= nearPress.R {
		t.Fatalf("expected unreached far side (%+v) to remain lighter than near press (%+v) at t=0", farSide, nearPress)
	}

	// Check strict asymmetric 4-corner clipping during active ripple press!
	if px := tt.Image().RGBAAt(22, 22); px != outsideTL {
		t.Fatalf("ripple bled outside TL 28dp corner: %+v", px)
	}

	// Advance 360ms while still held: wave reaches far side AND settles with zero pending frames.
	tt.Advance(360 * time.Millisecond)
	farHeld := tt.Image().RGBAAt(205, 60)
	if farHeld.R >= 245 {
		t.Fatalf("expected expanding ripple to reach far side after 360ms, got %+v", farHeld)
	}
	if px := tt.Image().RGBAAt(218, 98); px != outsideTL {
		t.Fatalf("fully expanded ripple bled outside BR 28dp corner: %+v", px)
	}
	if tt.FramePending() {
		t.Fatal("FramePending should settle to false even while press is held after full expansion")
	}

	// Release press and advance until fade completes and pointer moves out.
	tt.Release(44, 60)
	tt.Move(5, 5)
	tt.Advance(450 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("FramePending must be false after ripple release and fade complete")
	}
	if settled := tt.Image().RGBAAt(46, 60); settled != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("surface pixel after ripple faded out = %+v, want pure white", settled)
	}

	// Enable ReduceMotion: immediate static feedback, zero circular wave frames, FramePending always false.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Move(60, 60)
	if tt.FramePending() {
		t.Fatal("ReduceMotion hover should not request pending animation frames")
	}
	if hoverRM := tt.Image().RGBAAt(60, 60); hoverRM.R < 228 || hoverRM.R > 240 {
		t.Fatalf("ReduceMotion static hover pixel R = %d, want ~235", hoverRM.R)
	}

	tt.Press(44, 60)
	if tt.FramePending() {
		t.Fatal("ReduceMotion press should not queue ripple animation frames")
	}
	nearRM := tt.Image().RGBAAt(46, 60)
	farRM := tt.Image().RGBAAt(205, 60)
	if absColorDiff(nearRM, farRM) > 2 || nearRM.R >= 240 {
		t.Fatalf("ReduceMotion press should show immediate uniform pressed state layer: near=%+v far=%+v", nearRM, farRM)
	}

	tt.Release(44, 60)
	tt.Move(5, 5)
	if tt.FramePending() {
		t.Fatal("ReduceMotion release should settle with FramePending == false")
	}
}

func TestSurfacePolygonShapeClipKeyboardAndRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	// 1. Custom *shape.RoundedPolygon surface + ripple clipping check + keyboard Enter/Space pulse.
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(160, 160).Background(ui.RGB(200, 200, 200)).Padding(20).Children(func() {
				b := ui.ButtonBase(c).Size(100, 100)
				_ = b.Clicked()
				AttachRipple(c, b, StateLayerOptions{
					Color: ui.RGB(0, 0, 0),
					Shape: shape.Cookie4Sided(),
				})
				b.Children(func() {
					Surface(c, SurfaceOptions{
						Color: ui.RGB(255, 255, 255),
						Shape: shape.Cookie4Sided(),
					}).Size(100, 100)
				})
			})
		})
	}, 160, 160)

	// Box is [20..120] x [20..120]. Cookie4Sided pinches inward at corners, so (22, 22) is outside polygon!
	bg := color.RGBA{R: 200, G: 200, B: 200, A: 255}
	if px := tt.Image().RGBAAt(22, 22); px != bg {
		t.Fatalf("corner outside Cookie4Sided polygon = %+v, want background %+v", px, bg)
	}

	// Trigger keyboard focus + Space ripple at center (70, 70).
	tt.Key(0, ui.KeyTab)
	tt.KeyDown(ui.KeySpace)
	if center := tt.Image().RGBAAt(70, 70); center.R >= 245 {
		t.Fatalf("expected keyboard Space ripple darken at polygon center (70, 70), got %+v", center)
	}
	if px := tt.Image().RGBAAt(22, 22); px != bg {
		t.Fatalf("ripple bled outside Cookie4Sided polygon corner: %+v", px)
	}
	tt.KeyUp(ui.KeySpace)
	tt.Advance(500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("keyboard ripple should settle with FramePending == false")
	}

	// 2. RTL logical PaddingStart / PaddingEnd verification (window width 200 matches Surface width).
	rtlTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			Surface(c, SurfaceOptions{
				PaddingStart: 40,
				PaddingEnd:   4,
			}).Width(200).Children(func() {
				Text(c, "Hi", tokens.BodyMedium)
			})
		})
	}, 200, 80)

	ltrRect, ok := rtlTT.Find("Hi")
	if !ok {
		t.Fatal("LTR text not found")
	}
	rtlTT.SetDirection(ui.RightToLeft)
	rtlRect, ok := rtlTT.Find("Hi")
	if !ok {
		t.Fatal("RTL text not found")
	}
	if ltrRect.X != 40 || rtlRect.X != 4 {
		t.Fatalf("unexpected RTL PaddingStart behavior: ltr=%+v (want X=40) rtl=%+v (want X=4)", ltrRect, rtlRect)
	}
}

func TestRippleExitTransitionNoExpiredHandlePanic(t *testing.T) {
	exitTr := ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}

	t.Run("direct-button-exit-and-reappear", func(t *testing.T) {
		show := true
		clicks := 0
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, nil, func() {
				ui.Row(c).Size(240, 120).Padding(16).Children(func() {
					if show {
						b := Button(c.Key("exit-btn"), "ExitBtn").
							Transition(exitTr)
						if b.Clicked() {
							clicks++
						}
					}
				})
			})
		}, 240, 120)

		r, ok := tt.Find("ExitBtn")
		if !ok {
			t.Fatal("ExitBtn not found")
		}
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		tt.Move(cx, cy)
		tt.Press(cx, cy)
		tt.Advance(20 * time.Millisecond)
		tt.Release(cx, cy)
		tt.Advance(10 * time.Millisecond)

		// Remove during active ripple/hover exit and step until full convergence.
		show = false
		tt.Frame()
		for range 30 {
			tt.Advance(16 * time.Millisecond)
		}
		if tt.FramePending() {
			t.Fatal("direct exit transition should converge without pending frames")
		}

		// Re-show, exit again, then re-appear mid-exit without panicking.
		show = true
		tt.Frame()
		tt.Advance(250 * time.Millisecond)
		if r2, ok := tt.Find("ExitBtn"); ok {
			tt.Move(r2.X+r2.W/2, r2.Y+r2.H/2)
		}
		show = false
		tt.Frame()
		tt.Advance(40 * time.Millisecond)
		show = true
		tt.Frame()
		tt.Advance(300 * time.Millisecond)
		if tt.FramePending() {
			t.Fatal("re-appeared button should converge with no pending frames")
		}
	})

	t.Run("ancestor-transition-exit-and-reappear", func(t *testing.T) {
		show := true
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, nil, func() {
				ui.Column(c).Size(260, 140).Padding(16).Children(func() {
					if show {
						ui.Row(c.Key("wrap")).
							Transition(ui.ElementTransition{
								Duration: 180 * time.Millisecond,
								Exit:     &ui.Motion{Opacity: 0, Collapse: true},
							}).
							Children(func() {
								Button(c, "ChildBtn")
							})
					}
				})
			})
		}, 260, 140)

		r, ok := tt.Find("ChildBtn")
		if !ok {
			t.Fatal("ChildBtn not found")
		}
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		tt.Move(cx, cy)
		tt.Press(cx, cy)
		tt.Advance(15 * time.Millisecond)

		// Remove ancestor mid-press, advance halfway through exit, then re-appear.
		show = false
		tt.Frame()
		tt.Advance(32 * time.Millisecond)
		tt.Release(cx, cy)
		tt.Advance(16 * time.Millisecond)
		show = true
		tt.Frame()
		tt.Advance(300 * time.Millisecond)

		// Full exit to settled state.
		show = false
		tt.Frame()
		tt.Advance(300 * time.Millisecond)
		if tt.FramePending() {
			t.Fatal("ancestor exit transition should settle with zero pending frames")
		}
	})
}

func TestStateLayerOnlyPressWithoutWavesAndPolygonClip(t *testing.T) {
	poly := shape.Pill()
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, nil, func() {
			ui.Box(c).Size(200, 120).Background(ui.RGB(255, 255, 255)).Padding(20).Children(func() {
				s := Surface(c, SurfaceOptions{
					Color: ui.RGB(255, 255, 255),
					Shape: poly,
				}).Size(120, 60)
				AttachStateLayer(c, s, StateLayerOptions{
					Color: ui.RGB(0, 0, 0),
					Shape: poly,
				})
			})
		})
	}, 200, 120)

	b0 := builds
	tt.Move(80, 50)
	if builds > b0+1 {
		t.Fatalf("StateLayer hover enter caused %d builds, want at most 1 settle build", builds-b0)
	}
	afterMove := builds
	tt.Advance(250 * time.Millisecond)
	if builds != afterMove {
		t.Fatalf("StateLayer hover Advance caused %d rebuilds, want 0", builds-afterMove)
	}
	hoverPx := tt.Image().RGBAAt(80, 50)
	if hoverPx.R >= 250 {
		t.Fatalf("StateLayer-only polygon hover did not paint overlay: %+v", hoverPx)
	}

	tt.Press(80, 50)
	afterPress := builds
	tt.Advance(250 * time.Millisecond)
	if builds != afterPress {
		t.Fatalf("StateLayer press Advance caused %d rebuilds, want 0", builds-afterPress)
	}
	pressPx := tt.Image().RGBAAt(80, 50)
	if pressPx.R >= hoverPx.R {
		t.Fatalf("StateLayer-only press did not deepen overlay: hover=%+v press=%+v", hoverPx, pressPx)
	}

	tt.Release(80, 50)
	tt.Move(2, 2)
	afterLeave := builds
	tt.Advance(300 * time.Millisecond)
	if builds != afterLeave {
		t.Fatalf("StateLayer leave Advance caused %d rebuilds, want 0", builds-afterLeave)
	}
	if tt.FramePending() {
		t.Fatal("StateLayer-only polygon did not settle cleanly")
	}
}

// TestRippleOriginWithBorderInsetAndClip checks pointer ripple waves stay
// centered on the pointer press even when BorderInset and Clip are combined.
func TestRippleOriginWithBorderInsetAndClip(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, nil, func() {
			ui.Box(c).Size(200, 140).Background(ui.RGB(255, 255, 255)).Padding(20).Children(func() {
				s := Surface(c, SurfaceOptions{
					Color:  ui.RGB(255, 255, 255),
					Radius: 12,
				}).Size(140, 100)
				AttachRipple(c, s, StateLayerOptions{
					Color: ui.RGB(0, 0, 0),
					Clip: func(r ui.Rect) (ui.Rect, [4]float32) {
						return ui.Rect{X: r.X + 12, Y: r.Y + 8, W: r.W - 24, H: r.H - 16},
							[4]float32{8, 8, 8, 8}
					},
					BorderInset: 16,
				})
			})
		})
	}, 200, 140)

	// Element sits at (20,20)..(160,120). Clip sits at (32,28)..(148,112).
	// BorderInset=16 puts inner clipRect at (48,44)..(132,96), center (90,70).
	tt.Press(90, 70)
	tt.Advance(20 * time.Millisecond)

	img := tt.Image()
	centerPx := img.RGBAAt(90, 70)
	leftPx := img.RGBAAt(82, 70)
	rightPx := img.RGBAAt(98, 70)
	topPx := img.RGBAAt(90, 62)
	bottomPx := img.RGBAAt(90, 78)
	shiftedOriginPx := img.RGBAAt(62, 46)

	if centerPx.R >= 245 {
		t.Fatalf("ripple did not paint at press point (90,70): %+v", centerPx)
	}
	if diffLR := int(leftPx.R) - int(rightPx.R); diffLR < -3 || diffLR > 3 {
		t.Fatalf("ripple is horizontally asymmetric around press point: left=%+v right=%+v", leftPx, rightPx)
	}
	if diffTB := int(topPx.R) - int(bottomPx.R); diffTB < -3 || diffTB > 3 {
		t.Fatalf("ripple is vertically asymmetric around press point: top=%+v bottom=%+v", topPx, bottomPx)
	}
	if centerPx.R >= shiftedOriginPx.R {
		t.Fatalf("ripple appears shifted toward (62,46)=%+v over press (90,70)=%+v", shiftedOriginPx, centerPx)
	}

	tt.Release(90, 70)
	tt.Move(2, 2)
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("ripple with BorderInset and Clip did not settle")
	}
}

func TestSurfaceHairlineGradientsBackdropAndExitSafety(t *testing.T) {
	dark := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Dark: true})
	highContrast := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Contrast: tokens.ContrastHigh})

	// 1. Hairline draws 1px border in dark and high-contrast themes without breaking BorderWidth.
	if hc := resolveHairlineColor(highContrast, ui.Color{}); hc != highContrast.Colors.Outline {
		t.Fatalf("high-contrast hairline = %+v, want Outline %+v", hc, highContrast.Colors.Outline)
	}
	imgHairline := ui.Render(func(c *ui.Context) {
		ThemeScope(c, dark, func() {
			ui.Box(c).Size(120, 80).Background(dark.Colors.Surface).Padding(10).Children(func() {
				Surface(c, SurfaceOptions{
					Color:         dark.Colors.Surface,
					Hairline:      true,
					HairlineColor: ui.RGB(255, 80, 0),
					Radius:        4,
				}).Size(100, 60)
			})
		})
	}, 120, 80, 1)
	if edge := imgHairline.RGBAAt(60, 10); edge.R < 200 || edge.B > 40 {
		t.Fatalf("hairline top edge = %+v, want bright orange hairline", edge)
	}

	// 2. LinearGradient + RadialGradient + Backdrop clipped inside rounded rect and polygon.
	rectBackdropCalls := 0
	polyBackdropCalls := 0
	imgGrad := ui.Render(func(c *ui.Context) {
		ThemeScope(c, dark, func() {
			ui.Row(c).Size(260, 120).Background(ui.RGB(20, 20, 20)).Padding(10).Gap(10).Children(func() {
				Surface(c, SurfaceOptions{
					Radius: 24,
					LinearGradient: &ui.LinearGradient{
						From:  ui.RGB(255, 0, 0),
						To:    ui.RGB(0, 0, 255),
						Angle: 90,
					},
					RadialGradient: &ui.RadialGradient{
						Stops: []ui.GradientStop{
							{Color: ui.RGBA(0, 255, 0, 0.5), At: 0},
							{Color: ui.Transparent, At: 1},
						},
					},
					Backdrop: func(p *ui.Painter, r ui.Rect) {
						rectBackdropCalls++
						// Paint a giant stripe that should stay clipped inside the 24dp corners.
						p.Fill(ui.Rect{X: r.X - 10, Y: r.Y - 10, W: r.W + 20, H: r.H + 20}, ui.RGBA(255, 255, 0, 0.25), 0)
					},
				}).Size(110, 100)

				Surface(c, SurfaceOptions{
					Shape:    shape.Circle(),
					Hairline: true,
					LinearGradient: &ui.LinearGradient{
						From:  ui.RGB(255, 100, 0),
						To:    ui.RGB(0, 180, 255),
						Angle: 90,
					},
					Backdrop: func(p *ui.Painter, r ui.Rect) {
						polyBackdropCalls++
						p.Fill(r, ui.RGBA(255, 255, 255, 0.2), 0)
					},
				}).Size(100, 100)
			})
		})
	}, 260, 120, 1)

	if rectBackdropCalls == 0 || polyBackdropCalls == 0 {
		t.Fatalf("backdrop callbacks missing: rect=%d poly=%d", rectBackdropCalls, polyBackdropCalls)
	}
	// Left side of gradient rect should be redder than right side; corner (11,11) stays clipped to dark bg.
	leftPx := imgGrad.RGBAAt(25, 60)
	rightPx := imgGrad.RGBAAt(105, 60)
	if leftPx.R <= rightPx.R || rightPx.B <= leftPx.B {
		t.Fatalf("linear gradient did not blend left->right: left=%+v right=%+v", leftPx, rightPx)
	}
	if cornerPx := imgGrad.RGBAAt(11, 11); absColorDiff(cornerPx, color.RGBA{R: 20, G: 20, B: 20, A: 255}) > 6 {
		t.Fatalf("backdrop/gradient bled outside 24dp corner: %+v", cornerPx)
	}
	// Polygon circle corner at (132, 12) stays outside the inscribed circle.
	if polyCorner := imgGrad.RGBAAt(132, 12); absColorDiff(polyCorner, color.RGBA{R: 20, G: 20, B: 20, A: 255}) > 6 {
		t.Fatalf("polygon gradient/backdrop bled outside Circle corner: %+v", polyCorner)
	}

	// 3. Zero-sized surface does not invoke Backdrop, non-finite Radii/BorderWidth do not panic,
	//    and ElementTransition ghost draw settles cleanly with FramePending() == false.
	zeroCalls := 0
	show := true
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, dark, func() {
			ui.Column(c).Size(160, 120).Children(func() {
				Surface(c, SurfaceOptions{
					Radius:      float32(math.NaN()),
					BorderWidth: float32(math.Inf(1)),
					Backdrop: func(p *ui.Painter, r ui.Rect) {
						zeroCalls++
					},
				}).Size(0, 0)
				if show {
					Surface(c.Key("ghost-surface"), SurfaceOptions{
						ShapeScale: tokens.ShapeLarge,
						Hairline:   true,
						Backdrop: func(p *ui.Painter, r ui.Rect) {
							p.Fill(r, ui.RGBA(255, 255, 255, 0.08), 0)
						},
					}).Size(100, 60).Transition(ui.ElementTransition{
						Duration: 120 * time.Millisecond,
						Exit:     &ui.Motion{Opacity: 0},
					})
				}
			})
		})
	}, 160, 120)

	if zeroCalls != 0 {
		t.Fatalf("zero-sized surface invoked Backdrop %d times, want 0", zeroCalls)
	}
	show = false
	tt.Frame()
	tt.Advance(250 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("exiting Surface with Backdrop/Hairline left pending frames")
	}
}

func TestFocusRingAutoAndInsetAndHoverElevation(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	focusColor := th.ToUITheme().Focus

	// 1. FocusRingAuto paints outer ring on keyboard Tab; FocusRingInset stays inside box.
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Row(c).Fill().Background(ui.RGB(255, 255, 255)).Padding(24).Gap(32).Children(func() {
				b1 := ui.ButtonBase(c.Key("auto")).FocusRing(false).Size(80, 40).Radius(12).
					Background(th.Colors.Primary).Label("Auto ring")
				AttachRipple(c, b1, StateLayerOptions{Color: th.Colors.OnPrimary, Radius: 12})

				b2 := ui.ButtonBase(c.Key("inset")).FocusRing(false).Size(80, 40).Radius(12).
					Background(th.Colors.SurfaceContainerHigh).Label("Inset ring")
				AttachRipple(c, b2, StateLayerOptions{Color: th.Colors.OnSurface, Radius: 12, FocusRing: FocusRingInset})
			})
		}).Fill()
	}, 260, 100)

	tt.Key(0, ui.KeyTab)
	tt.Advance(250 * time.Millisecond)
	b1 := buttonInfo(t, tt, ui.RoleButton, "Auto ring").Bounds
	img := tt.Image()
	if px := img.RGBAAt(int(b1.X+b1.W/2), int(b1.Y-3)); absColorDiff(px, rgbOf(focusColor)) > 20 {
		t.Fatalf("FocusRingAuto did not paint outer ring at top edge: got %+v want %+v", px, rgbOf(focusColor))
	}

	tt.Key(0, ui.KeyTab)
	tt.Advance(250 * time.Millisecond)
	b2 := buttonInfo(t, tt, ui.RoleButton, "Inset ring").Bounds
	img = tt.Image()
	if px := img.RGBAAt(int(b2.X+b2.W/2), int(b2.Y-3)); px != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("FocusRingInset leaked outside top edge: got %+v", px)
	}
	if px := img.RGBAAt(int(b2.X+b2.W/2), int(b2.Y+2)); absColorDiff(px, rgbOf(focusColor)) > 24 {
		t.Fatalf("FocusRingInset did not paint inside top edge: got %+v want %+v", px, rgbOf(focusColor))
	}

	// 2. Zero-rebuild attachHoverElevation lifts drop shadow on hover and settles cleanly.
	builds := 0
	ttElev := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, th, func() {
			ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Padding(24).Children(func() {
				el := ui.Box(c).Size(100, 44).Radius(12).Background(th.Colors.SurfaceContainerLow)
				ApplyElevationShadows(th, el, tokens.Level1)
				attachHoverElevation(el, th, tokens.Level1, tokens.Level2, 12, false)
			})
		}).Fill()
	}, 180, 120)

	restBelow := ttElev.Image().RGBAAt(74, 72)
	buildsBefore := builds
	ttElev.Move(74, 46)
	ttElev.Advance(300 * time.Millisecond)
	hoverBelow := ttElev.Image().RGBAAt(74, 72)
	if builds > buildsBefore+1 {
		t.Fatalf("hover elevation rebuilt view tree: before=%d after=%d", buildsBefore, builds)
	}
	if hoverBelow.R >= restBelow.R {
		t.Fatalf("hover elevation did not darken shadow below box: rest=%+v hover=%+v", restBelow, hoverBelow)
	}
	ttElev.Move(2, 2)
	ttElev.Advance(350 * time.Millisecond)
	if ttElev.FramePending() {
		t.Fatal("hover elevation left pending frames after pointer leave")
	}
}
