package md3e

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
)

// rectsOverlap reports whether two boxes share any area.
func rectsOverlap(a, b ui.Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// toolbarItems returns two toolbar actions counting their presses.
func toolbarItems(clicks *[2]int) []FloatingToolbarItem {
	return []FloatingToolbarItem{
		{Label: "Bold", Icon: mdicon.Edit.Draw, OnClick: func() { clicks[0]++ }},
		{Label: "Add", Icon: mdicon.Add.Draw, OnClick: func() { clicks[1]++ }},
	}
}

// TestFloatingToolbarLayoutsAndTones checks the horizontal and vertical pill
// dimensions, the standard and vibrant container pixels, the coupled FABs not
// overlapping, and both being clickable.
func TestFloatingToolbarLayoutsAndTones(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	orientations := []struct {
		name string
		o    ToolbarOrientation
	}{
		{"Horizontal", ToolbarHorizontal},
		{"Vertical", ToolbarVertical},
	}
	tones := []struct {
		name string
		tone FloatingToolbarTone
		bg   ui.Color
	}{
		{"Standard", ToolbarStandard, th.Colors.SurfaceContainer},
		{"Vibrant", ToolbarVibrant, th.Colors.PrimaryContainer},
	}

	for _, ori := range orientations {
		for _, tone := range tones {
			var clicks [2]int
			fabClicks := 0
			tt := ui.NewTester(func(c *ui.Context) {
				ThemeScope(c, th, func() {
					ui.Box(c).Size(520, 320).Background(canvasRGB).Padding(40).Children(func() {
						FloatingToolbar(c, toolbarItems(&clicks), FloatingToolbarOptions{
							Orientation: ori.o,
							Tone:        tone.tone,
							Label:       "Tools",
							Fab: &FloatingToolbarFab{
								Label:   "Compose",
								Icon:    mdicon.Add.Draw,
								OnClick: func() { fabClicks++ },
							},
						})
					})
				})
			}, 520, 320)

			bar := buttonInfo(t, tt, ui.RoleToolbar, "Tools")
			switch ori.o {
			case ToolbarHorizontal:
				if bar.Bounds.H != 64 {
					t.Fatalf("%s/%s: height = %v, want 64", ori.name, tone.name, bar.Bounds.H)
				}
				if want := float32(8*2 + 2*48 + 4); math.Abs(float64(bar.Bounds.W-want)) > 1 {
					t.Fatalf("%s/%s: width = %v, want %v", ori.name, tone.name, bar.Bounds.W, want)
				}
			default:
				if bar.Bounds.W != 64 {
					t.Fatalf("%s/%s: width = %v, want 64", ori.name, tone.name, bar.Bounds.W)
				}
				if want := float32(8*2 + 2*48 + 4); math.Abs(float64(bar.Bounds.H-want)) > 1 {
					t.Fatalf("%s/%s: height = %v, want %v", ori.name, tone.name, bar.Bounds.H, want)
				}
			}

			px := tt.Image().RGBAAt(int(bar.Bounds.X)+4, int(bar.Bounds.Y+bar.Bounds.H/2))
			if absColorDiff(px, rgbOf(tone.bg)) > 4 {
				t.Fatalf("%s/%s: container = %v, want %v", ori.name, tone.name, px, rgbOf(tone.bg))
			}

			fab := buttonInfo(t, tt, ui.RoleButton, "Compose")
			if fab.Bounds.W != 56 || fab.Bounds.H != 56 {
				t.Fatalf("%s/%s: coupled FAB = %v×%v, want the standard 56×56",
					ori.name, tone.name, fab.Bounds.W, fab.Bounds.H)
			}
			if rectsOverlap(bar.Bounds, fab.Bounds) {
				t.Fatalf("%s/%s: the toolbar and the FAB overlap: %v and %v",
					ori.name, tone.name, bar.Bounds, fab.Bounds)
			}
			if ori.o == ToolbarHorizontal {
				if fab.Bounds.X < bar.Bounds.X+bar.Bounds.W+8 {
					t.Fatalf("%s/%s: FAB x=%v is not past the toolbar end %v",
						ori.name, tone.name, fab.Bounds.X, bar.Bounds.X+bar.Bounds.W)
				}
			} else if fab.Bounds.Y < bar.Bounds.Y+bar.Bounds.H+8 {
				t.Fatalf("%s/%s: FAB y=%v is not below the toolbar end %v",
					ori.name, tone.name, fab.Bounds.Y, bar.Bounds.Y+bar.Bounds.H)
			}

			bold := buttonInfo(t, tt, ui.RoleButton, "Bold")
			bx, by := center(bold.Bounds)
			tt.ClickAt(bx, by)
			fx, fy := center(fab.Bounds)
			tt.ClickAt(fx, fy)
			if clicks[0] != 1 {
				t.Fatalf("%s/%s: Bold clicked %d times, want 1", ori.name, tone.name, clicks[0])
			}
			if fabClicks != 1 {
				t.Fatalf("%s/%s: FAB clicked %d times, want 1", ori.name, tone.name, fabClicks)
			}
		}
	}
}

// TestFloatingToolbarKeyboardCollapseAndHide checks the arrow-key focus order,
// the collapse and expand spring settling, and the visibility fade settling
// under ReduceMotion.
func TestFloatingToolbarKeyboardCollapseAndHide(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	expanded := true
	visible := true
	var clicks [2]int
	tt := ui.NewTesterWith(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(520, 320).Background(canvasRGB).Padding(40).Children(func() {
				FloatingToolbar(c, toolbarItems(&clicks), FloatingToolbarOptions{
					Label:    "Tools",
					Expanded: &expanded,
					Visible:  &visible,
				})
			})
		})
	}, 520, 320, ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Bold") {
		t.Fatal("Tab did not focus the first toolbar item")
	}
	tt.Key(0, ui.KeyRight)
	if !tt.Focused("Add") {
		t.Fatal("Right did not move the focus to the second toolbar item")
	}

	bar := buttonInfo(t, tt, ui.RoleToolbar, "Tools")
	fullWidth := bar.Bounds.W
	expanded = false
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("ReduceMotion collapse asked for an animation frame")
	}
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("collapsing the toolbar left frames pending")
	}
	bar = buttonInfo(t, tt, ui.RoleToolbar, "Tools")
	if math.Abs(float64(bar.Bounds.W-64)) > 1 {
		t.Fatalf("collapsed toolbar width = %v, want 64", bar.Bounds.W)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Add"); ok {
		t.Fatal("the collapsed toolbar still exposes its hidden items")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Bold") {
		t.Fatal("Tab did not reach the remaining toolbar item")
	}
	tt.Key(0, ui.KeyRight)
	if !tt.Focused("Bold") {
		t.Fatal("Right moved the focus into a collapsed, invisible toolbar item")
	}

	expanded = true
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("ReduceMotion expand asked for an animation frame")
	}
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("expanding the toolbar left frames pending")
	}
	if bar = buttonInfo(t, tt, ui.RoleToolbar, "Tools"); math.Abs(float64(bar.Bounds.W-fullWidth)) > 1 {
		t.Fatalf("re-expanded toolbar width = %v, want %v", bar.Bounds.W, fullWidth)
	}

	visible = false
	tt.Frame()
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("hiding the toolbar left frames pending")
	}
	visible = true
	tt.Frame()
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("showing the toolbar left frames pending")
	}
}

// TestFloatingToolbarSpringRTLAndDisabledVisual checks the real spring
// collapse, the mirrored RTL row, the 38% disabled item ink, and that hiding
// does not jump to ui's 50% dimming on its first frame.
func TestFloatingToolbarSpringRTLAndDisabledVisual(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	items := []FloatingToolbarItem{
		{Label: "Bold", Icon: mdicon.Edit.Draw},
		{Label: "Add", Icon: mdicon.Add.Draw, Disabled: true},
	}
	darkest := func(tt *ui.Tester, b ui.Rect) int {
		min := 255
		for y := int(b.Y) + 8; y < int(b.Y+b.H)-8; y++ {
			for x := int(b.X) + 8; x < int(b.X+b.W)-8; x++ {
				if r := int(tt.Image().RGBAAt(x, y).R); r < min {
					min = r
				}
			}
		}
		return min
	}

	visible := true
	expanded := true
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(520, 320).Background(canvasRGB).Padding(40).Children(func() {
				FloatingToolbar(c, items, FloatingToolbarOptions{
					Label:    "Tools",
					Expanded: &expanded,
					Visible:  &visible,
				})
			})
		})
	}, 520, 320)

	bold := buttonInfo(t, tt, ui.RoleButton, "Bold")
	disabledAdd := buttonInfo(t, tt, ui.RoleButton, "Add")
	if !disabledAdd.Disabled {
		t.Fatal("the disabled item does not report itself disabled")
	}
	if got, want := darkest(tt, disabledAdd.Bounds), darkest(tt, bold.Bounds); got <= want+40 {
		t.Fatalf("disabled item ink is as dark as the enabled one (%d vs %d): no 38%% content", got, want)
	}

	// Hiding starts at full opacity: no instant 50% dim jump.
	bar := buttonInfo(t, tt, ui.RoleToolbar, "Tools")
	container := rgbOf(th.Colors.SurfaceContainer)
	visible = false
	tt.Frame()
	px := tt.Image().RGBAAt(int(bar.Bounds.X)+4, int(bar.Bounds.Y+bar.Bounds.H/2))
	if absColorDiff(px, container) > 4 {
		t.Fatalf("first hidden frame container = %v, want %v without a dim jump", px, container)
	}
	tt.Advance(1200 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("hiding the toolbar left frames pending")
	}

	// The ordinary spring path: collapsing and expanding request frames and
	// settle, and RTL mirrors the item row.
	expanded2, visible2 := true, true
	tt2 := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(520, 320).Background(canvasRGB).Padding(40).Children(func() {
				FloatingToolbar(c, items, FloatingToolbarOptions{
					Label:    "Tools",
					Expanded: &expanded2,
					Visible:  &visible2,
				})
			})
		})
	}, 520, 320)

	bar2 := buttonInfo(t, tt2, ui.RoleToolbar, "Tools")
	full := bar2.Bounds.W
	expanded2 = false
	tt2.Frame()
	if !tt2.FramePending() {
		t.Fatal("collapsing the toolbar should spring, not snap")
	}
	tt2.Advance(1200 * time.Millisecond)
	if tt2.FramePending() {
		t.Fatal("the collapse spring did not settle")
	}
	if got := buttonInfo(t, tt2, ui.RoleToolbar, "Tools").Bounds.W; math.Abs(float64(got-64)) > 1 {
		t.Fatalf("collapsed toolbar width = %v, want 64", got)
	}
	expanded2 = true
	tt2.Frame()
	if !tt2.FramePending() {
		t.Fatal("expanding the toolbar should spring, not snap")
	}
	tt2.Advance(1200 * time.Millisecond)
	if got := buttonInfo(t, tt2, ui.RoleToolbar, "Tools").Bounds.W; math.Abs(float64(got-full)) > 1 {
		t.Fatalf("re-expanded toolbar width = %v, want %v", got, full)
	}

	tt2.SetDirection(ui.RightToLeft)
	tt2.Advance(600 * time.Millisecond)
	bold = buttonInfo(t, tt2, ui.RoleButton, "Bold")
	add := buttonInfo(t, tt2, ui.RoleButton, "Add")
	if bold.Bounds.X <= add.Bounds.X {
		t.Fatalf("right to left row did not mirror: Bold x=%v Add x=%v", bold.Bounds.X, add.Bounds.X)
	}
}

// TestTopAppBarCenteringAndTitlePadding checks the symmetric CenterAligned
// side slots with uneven icons and the 16 DIP start padding of expanded titles.
func TestTopAppBarCenteringAndTitlePadding(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cases := []struct {
		name    string
		nav     bool
		actions []AppBarAction
	}{
		{"navigation only", true, nil},
		{"action only", false, []AppBarAction{{Label: "More", Icon: mdicon.MoreVert.Draw}}},
		{"one left, two right", true, []AppBarAction{
			{Label: "More", Icon: mdicon.MoreVert.Draw},
			{Label: "Filter", Icon: mdicon.Edit.Draw},
		}},
	}
	for _, tc := range cases {
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Box(c).Size(640, 260).Background(canvasRGB).Children(func() {
					o := TopAppBarOptions{Variant: TopAppBarCenterAligned, Label: "Mail bar", Actions: tc.actions}
					if tc.nav {
						o.NavigationIcon = mdicon.Menu.Draw
					}
					TopAppBar(c, "Inbox", o)
				})
			})
		}, 640, 260)

		bar := buttonInfo(t, tt, ui.RoleRegion, "Mail bar")
		title, ok := tt.Find("Inbox")
		if !ok {
			t.Fatalf("%s: title not laid out", tc.name)
		}
		got := title.X + title.W/2
		want := bar.Bounds.X + bar.Bounds.W/2
		if math.Abs(float64(got-want)) > 1 {
			t.Fatalf("%s: title center x = %v, want the bar center %v", tc.name, got, want)
		}
	}

	// Small without a navigation icon starts the title 16 DIP from the edge.
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(640, 120).Background(canvasRGB).Children(func() {
				TopAppBar(c, "Inbox", TopAppBarOptions{Label: "Mail bar"})
			})
		})
	}, 640, 120)
	bar := buttonInfo(t, tt, ui.RoleRegion, "Mail bar")
	title, _ := tt.Find("Inbox")
	if math.Abs(float64(title.X-(bar.Bounds.X+16))) > 1 {
		t.Fatalf("small bar title x = %v, want %v", title.X, bar.Bounds.X+16)
	}

	// Medium and Large expanded titles keep the same 16 DIP start padding.
	for _, variant := range []struct {
		name string
		v    TopAppBarVariant
	}{
		{"Medium", TopAppBarMedium},
		{"Large", TopAppBarLarge},
	} {
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Box(c).Size(640, 260).Background(canvasRGB).Children(func() {
					TopAppBar(c, "Inbox", TopAppBarOptions{Variant: variant.v, Label: "Mail bar"})
				})
			})
		}, 640, 260)
		bar := buttonInfo(t, tt, ui.RoleRegion, "Mail bar")
		title, _ := tt.Find("Inbox")
		if math.Abs(float64(title.X-(bar.Bounds.X+16))) > 1 {
			t.Fatalf("%s bar title x = %v, want %v", variant.name, title.X, bar.Bounds.X+16)
		}
	}
}

// TestSearchAppBarDefaultsPlaceholderDecorationAndDisabled checks the default
// "Search" placeholder, the centered decorative icon, and that a disabled bar
// keeps its container and uses 38% content instead of ui's 50% dimming.
func TestSearchAppBarDefaultsPlaceholderDecorationAndDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	query := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 200).Background(canvasRGB).Padding(24).Children(func() {
				SearchAppBar(c, &query)
			})
		})
	}, 560, 200)

	field := buttonInfo(t, tt, ui.RoleTextField, "Search")
	if field.Placeholder != "Search" {
		t.Fatalf("default placeholder = %q, want Search", field.Placeholder)
	}
	bar := buttonInfo(t, tt, ui.RoleSearch, "Search")
	slot := ui.Rect{X: bar.Bounds.X + 8, Y: bar.Bounds.Y + 4, W: 48, H: 48}
	bg := rgbOf(th.Colors.SurfaceContainerHigh)
	minX, minY, maxX, maxY := 1<<30, 1<<30, -1, -1
	// Stay clear of the pill's rounded left end, which antialiases into the
	// canvas inside the 48 DIP slot's corners.
	for y := int(slot.Y) + 8; y < int(slot.Y+slot.H)-8; y++ {
		for x := int(slot.X) + 8; x < int(slot.X+slot.W)-8; x++ {
			if absColorDiff(tt.Image().RGBAAt(x, y), bg) > 20 {
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x), max(maxY, y)
			}
		}
	}
	if maxX < 0 {
		t.Fatal("the decorative leading icon painted nothing")
	}
	gotCX := float32(minX+maxX)/2 + 0.5
	gotCY := float32(minY+maxY)/2 + 0.5
	wantCX, wantCY := slot.X+slot.W/2, slot.Y+slot.H/2
	if math.Abs(float64(gotCX-wantCX)) > 2.5 || math.Abs(float64(gotCY-wantCY)) > 2.5 {
		t.Fatalf("decorative icon center = (%v, %v), want the 48 DIP slot center (%v, %v)",
			gotCX, gotCY, wantCX, wantCY)
	}

	disabledQuery := ""
	dt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 200).Background(canvasRGB).Padding(24).Children(func() {
				SearchAppBar(c, &disabledQuery, SearchAppBarOptions{
					Disabled: true,
					Trailing: []AppBarAction{{Label: "Filter", Icon: mdicon.MoreVert.Draw}},
				})
			})
		})
	}, 560, 200)
	dbar := buttonInfo(t, dt, ui.RoleSearch, "Search")
	px := dt.Image().RGBAAt(int(dbar.Bounds.X)+4, int(dbar.Bounds.Y+dbar.Bounds.H/2))
	if absColorDiff(px, bg) > 4 {
		t.Fatalf("disabled bar container = %v, want the undimmed %v", px, bg)
	}
	filter := buttonInfo(t, dt, ui.RoleButton, "Filter")
	darkest := 255
	for y := int(filter.Bounds.Y) + 12; y < int(filter.Bounds.Y+filter.Bounds.H)-12; y++ {
		for x := int(filter.Bounds.X) + 12; x < int(filter.Bounds.X+filter.Bounds.W)-12; x++ {
			if r := int(dt.Image().RGBAAt(x, y).R); r < darkest {
				darkest = r
			}
		}
	}
	if darkest < 100 {
		t.Fatalf("disabled trailing icon R = %d, want the 38%% OnSurface tone", darkest)
	}
}

// TestFloatingToolbarVerticalKeyboard checks the vertical focus group's Down
// key.
func TestFloatingToolbarVerticalKeyboard(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	var clicks [2]int
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(320, 420).Background(canvasRGB).Padding(40).Children(func() {
				FloatingToolbar(c, toolbarItems(&clicks), FloatingToolbarOptions{
					Orientation: ToolbarVertical,
					Label:       "Tools",
				})
			})
		})
	}, 320, 420)

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Bold") {
		t.Fatal("Tab did not focus the first vertical item")
	}
	tt.Key(0, ui.KeyDown)
	if !tt.Focused("Add") {
		t.Fatal("Down did not move the focus down the vertical toolbar")
	}
}

// TestTopAppBarVariantsAndScroll checks the four variant heights, the centered
// title, the subtitle below the title, the ScrollProgress height and tone
// interpolation, and the right-to-left mirror.
func TestTopAppBarVariantsAndScroll(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	variants := []struct {
		name string
		v    TopAppBarVariant
		h    float32
	}{
		{"Small", TopAppBarSmall, 64},
		{"CenterAligned", TopAppBarCenterAligned, 64},
		{"Medium", TopAppBarMedium, 112},
		{"Large", TopAppBarLarge, 152},
	}
	for _, variant := range variants {
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Box(c).Size(640, 260).Background(canvasRGB).Children(func() {
					TopAppBar(c, "Inbox", TopAppBarOptions{
						Variant:         variant.v,
						Label:           "Mail bar",
						NavigationIcon:  mdicon.Menu.Draw,
						NavigationLabel: "Menu",
						Actions:         []AppBarAction{{Label: "More", Icon: mdicon.MoreVert.Draw}},
					})
				})
			})
		}, 640, 260)

		bar := buttonInfo(t, tt, ui.RoleRegion, "Mail bar")
		if bar.Bounds.H != variant.h {
			t.Fatalf("%s: height = %v, want %v", variant.name, bar.Bounds.H, variant.h)
		}
		title, ok := tt.Find("Inbox")
		if !ok {
			t.Fatalf("%s: title not laid out", variant.name)
		}
		if variant.v == TopAppBarCenterAligned {
			if got, want := title.X+title.W/2, bar.Bounds.X+bar.Bounds.W/2; math.Abs(float64(got-want)) > 1 {
				t.Fatalf("%s: title center x = %v, want %v", variant.name, got, want)
			}
		}
	}

	// A subtitle sits below the title.
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(640, 260).Background(canvasRGB).Children(func() {
				TopAppBar(c, "Inbox", TopAppBarOptions{Label: "Mail bar", Subtitle: "3 unread"})
			})
		})
	}, 640, 260)
	title, ok := tt.Find("Inbox")
	if !ok {
		t.Fatal("title not laid out")
	}
	subtitle, ok := tt.Find("3 unread")
	if !ok {
		t.Fatal("subtitle not laid out")
	}
	if subtitle.Y < title.Y+title.H-1 {
		t.Fatalf("subtitle y=%v is not below the title bottom %v", subtitle.Y, title.Y+title.H)
	}

	// ScrollProgress takes the Medium bar from 112 to 64 and mixes the
	// Surface tone into Surface Container.
	scroll := float32(0)
	tt = ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(640, 260).Background(canvasRGB).Children(func() {
				TopAppBar(c, "Inbox", TopAppBarOptions{Variant: TopAppBarMedium, Label: "Mail bar", ScrollProgress: scroll})
			})
		})
	}, 640, 260)
	bar := buttonInfo(t, tt, ui.RoleRegion, "Mail bar")
	if bar.Bounds.H != 112 {
		t.Fatalf("expanded medium height = %v, want 112", bar.Bounds.H)
	}
	expandedPx := tt.Image().RGBAAt(int(bar.Bounds.X+bar.Bounds.W/2), int(bar.Bounds.Y)+2)
	if absColorDiff(expandedPx, rgbOf(th.Colors.Surface)) > 4 {
		t.Fatalf("expanded bar = %v, want the Surface %v", expandedPx, rgbOf(th.Colors.Surface))
	}

	scroll = 1
	tt.Frame()
	bar = buttonInfo(t, tt, ui.RoleRegion, "Mail bar")
	if bar.Bounds.H != 64 {
		t.Fatalf("collapsed medium height = %v, want 64", bar.Bounds.H)
	}
	collapsedPx := tt.Image().RGBAAt(int(bar.Bounds.X+bar.Bounds.W/2), int(bar.Bounds.Y)+2)
	if absColorDiff(collapsedPx, rgbOf(th.Colors.SurfaceContainer)) > 4 {
		t.Fatalf("collapsed bar = %v, want the Surface Container %v", collapsedPx, rgbOf(th.Colors.SurfaceContainer))
	}

	// Right to left puts the navigation button after the actions.
	rtl := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(640, 260).Background(canvasRGB).Children(func() {
				TopAppBar(c, "Inbox", TopAppBarOptions{
					Label:           "Mail bar",
					NavigationIcon:  mdicon.Menu.Draw,
					NavigationLabel: "Menu",
					Actions:         []AppBarAction{{Label: "More", Icon: mdicon.MoreVert.Draw}},
				})
			})
		})
	}, 640, 260)
	rtl.SetDirection(ui.RightToLeft)
	rtl.Advance(200 * time.Millisecond)
	nav := buttonInfo(t, rtl, ui.RoleButton, "Menu")
	more := buttonInfo(t, rtl, ui.RoleButton, "More")
	if nav.Bounds.X <= more.Bounds.X {
		t.Fatalf("right to left: navigation x=%v is not after the actions x=%v", nav.Bounds.X, more.Bounds.X)
	}
}

// TestSearchAppBarInputClearAndSubmit checks focusing and typing in the field,
// the clear button resetting the query, Enter submitting it, and the trailing
// actions.
func TestSearchAppBarInputClearAndSubmit(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	query := ""
	var changes []string
	submitted := ""
	filterClicks := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 200).Background(canvasRGB).Padding(24).Children(func() {
				SearchAppBar(c, &query, SearchAppBarOptions{
					Placeholder: "Search mail",
					OnChanged:   func(q string) { changes = append(changes, q) },
					OnSubmitted: func(q string) { submitted = q },
					Trailing: []AppBarAction{{
						Label:   "Filter",
						Icon:    mdicon.MoreVert.Draw,
						OnClick: func() { filterClicks++ },
					}},
				})
			})
		})
	}, 560, 200)

	bar := buttonInfo(t, tt, ui.RoleSearch, "Search mail")
	if bar.Bounds.H != 56 {
		t.Fatalf("search bar height = %v, want 56", bar.Bounds.H)
	}
	field := buttonInfo(t, tt, ui.RoleTextField, "Search mail")
	fx, fy := center(field.Bounds)
	tt.ClickAt(fx, fy)
	if !tt.Focused("Search mail") {
		t.Fatal("clicking the field did not focus it")
	}
	tt.Type("hello")
	if query != "hello" {
		t.Fatalf("typed query = %q, want hello", query)
	}
	if len(changes) == 0 || changes[len(changes)-1] != "hello" {
		t.Fatalf("change reports = %v, want the last to be hello", changes)
	}

	clear := buttonInfo(t, tt, ui.RoleButton, "Clear search")
	cx, cy := center(clear.Bounds)
	tt.ClickAt(cx, cy)
	if query != "" {
		t.Fatalf("clear left the query at %q", query)
	}
	if len(changes) == 0 || changes[len(changes)-1] != "" {
		t.Fatalf("clear did not report an empty query: %v", changes)
	}

	field = buttonInfo(t, tt, ui.RoleTextField, "Search mail")
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tt.Type("ok")
	tt.Key(0, ui.KeyEnter)
	if submitted != "ok" {
		t.Fatalf("submitted %q, want ok", submitted)
	}

	filter := buttonInfo(t, tt, ui.RoleButton, "Filter")
	tt.ClickAt(filter.Bounds.X+filter.Bounds.W/2, filter.Bounds.Y+filter.Bounds.H/2)
	if filterClicks != 1 {
		t.Fatalf("trailing action clicked %d times, want 1", filterClicks)
	}
	if tt.FramePending() {
		t.Fatal("search app bar left frames pending")
	}
}

func TestSearchAppBarClearDisappearPreservesTrailingFocus(t *testing.T) {
	query := "abc"
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, nil, func() {
			SearchAppBar(c, &query, SearchAppBarOptions{
				Trailing: []AppBarAction{
					{Label: "Voice", Icon: mdicon.Search.Draw},
				},
			})
		})
	}, 400, 96)

	// Tab to "Clear search", then Tab once more to "Voice".
	tt.Key(0, ui.KeyTab)
	for !tt.Focused("Voice") {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("Voice") {
		t.Fatal("failed to focus trailing Voice action")
	}

	// Clearing the query removes the Clear button immediately before Voice;
	// Voice must keep its focus instead of losing identity on sibling shift.
	query = ""
	tt.Frame()
	if !tt.Focused("Voice") {
		t.Fatal("SearchAppBar trailing button lost focus when Clear button disappeared")
	}
}
