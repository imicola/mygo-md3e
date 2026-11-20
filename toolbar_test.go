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
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("collapsing the toolbar left frames pending")
	}
	bar = buttonInfo(t, tt, ui.RoleToolbar, "Tools")
	if math.Abs(float64(bar.Bounds.W-64)) > 1 {
		t.Fatalf("collapsed toolbar width = %v, want 64", bar.Bounds.W)
	}

	expanded = true
	tt.Frame()
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
