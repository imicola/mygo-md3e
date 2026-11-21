package md3e

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

// rgbOf converts a ui.Color to the image color a tester reads.
func rgbOf(c ui.Color) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// fabContainerProbe samples a FAB's container inside its leading edge, clear of
// the centered icon.
func fabContainerProbe(t *testing.T, tt *ui.Tester, b ui.Rect) color.RGBA {
	t.Helper()
	return tt.Image().RGBAAt(int(b.X)+6, int(b.Y+b.H/2))
}

// TestFABSizesAndSchemes checks the three exact size boxes, their icon sizes,
// and the four container/content pairings pixel by pixel.
func TestFABSizesAndSchemes(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	sizes := []struct {
		name string
		size FABSize
		side float32
		icon float32
	}{
		{"Small", FABSizeSmall, 40, 24},
		{"Medium", FABSizeMedium, 56, 24},
		{"Large", FABSizeLarge, 96, 36},
	}
	schemes := []struct {
		name      string
		scheme    FABColorScheme
		container ui.Color
	}{
		{"Primary", FABPrimary, th.Colors.PrimaryContainer},
		{"Secondary", FABSecondary, th.Colors.SecondaryContainer},
		{"Tertiary", FABTertiary, th.Colors.TertiaryContainer},
		{"Surface", FABSurface, th.Colors.Surface},
	}

	for _, size := range sizes {
		for _, scheme := range schemes {
			tt := ui.NewTester(func(c *ui.Context) {
				ThemeScope(c, th, func() {
					ui.Box(c).Size(240, 200).Background(canvasRGB).Padding(40).Children(func() {
						FAB(c, "Compose", mdicon.Add.Draw, FABOptions{Size: size.size, Color: scheme.scheme})
					})
				})
			}, 240, 200)

			info := buttonInfo(t, tt, ui.RoleButton, "Compose")
			if info.Bounds.W != size.side || info.Bounds.H != size.side {
				t.Fatalf("%s/%s: box = %v×%v, want %v×%v",
					size.name, scheme.name, info.Bounds.W, info.Bounds.H, size.side, size.side)
			}
			got := fabContainerProbe(t, tt, info.Bounds)
			if absColorDiff(got, rgbOf(scheme.container)) > 4 {
				t.Fatalf("%s/%s: container = %v, want %v", size.name, scheme.name, got, rgbOf(scheme.container))
			}
			// The centered icon paints content ink distinct from the container.
			cx, cy := int(info.Bounds.X+info.Bounds.W/2), int(info.Bounds.Y+info.Bounds.H/2)
			if absColorDiff(tt.Image().RGBAAt(cx, cy), got) < 30 {
				t.Fatalf("%s/%s: no icon ink at the center over %v", size.name, scheme.name, got)
			}
		}
	}
}

// TestFABDisabledAndHidden checks the MD3E disabled palette and that hiding the
// button fades and shrinks it without leaving frames pending.
func TestFABDisabledAndHidden(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	visible := true
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 200).Background(canvasRGB).Padding(40).Children(func() {
				FAB(c, "Compose", mdicon.Add.Draw, FABOptions{Visible: &visible})
			})
		})
	}, 240, 200)

	info := buttonInfo(t, tt, ui.RoleButton, "Compose")
	if info.Bounds.W != 56 || info.Bounds.H != 56 {
		t.Fatalf("zero-value FAB = %v×%v, want the standard 56×56 medium size",
			info.Bounds.W, info.Bounds.H)
	}
	px := fabContainerProbe(t, tt, info.Bounds)
	if absColorDiff(px, rgbOf(th.Colors.PrimaryContainer)) > 4 {
		t.Fatalf("visible FAB container = %v, want the primary container", px)
	}

	disabled := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 200).Background(canvasRGB).Padding(40).Children(func() {
				FAB(c, "Compose", mdicon.Add.Draw, FABOptions{Disabled: true})
			})
		})
	}, 240, 200)
	dinfo := buttonInfo(t, disabled, ui.RoleButton, "Compose")
	if !dinfo.Disabled {
		t.Fatal("disabled FAB does not report itself disabled")
	}
	want := rgbOf(th.Colors.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)).Over(canvasRGB))
	if got := fabContainerProbe(t, disabled, dinfo.Bounds); absColorDiff(got, want) > 4 {
		t.Fatalf("disabled FAB container = %v, want the OnSurface 12%% %v", got, want)
	}

	visible = false
	tt.Frame()
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("hiding the FAB left frames pending")
	}
	if info := buttonInfo(t, tt, ui.RoleButton, "Compose"); info.Focusable && !info.Disabled {
		t.Fatalf("hidden FAB still reports itself available: %+v", info)
	}

	visible = true
	tt.Frame()
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("showing the FAB left frames pending")
	}
	if px := fabContainerProbe(t, tt, buttonInfo(t, tt, ui.RoleButton, "Compose").Bounds); absColorDiff(px, rgbOf(th.Colors.PrimaryContainer)) > 4 {
		t.Fatalf("shown FAB container = %v, want the primary container", px)
	}
}

// TestExtendedFABExpandCollapse checks that the extended FAB springs from its
// square icon-only width to its full icon-label width and back, settling each
// time.
func TestExtendedFABExpandCollapse(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	expanded := false
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 220).Background(canvasRGB).Padding(24).Children(func() {
				ExtendedFAB(c, "New message", mdicon.Edit.Draw, ExtendedFABOptions{Expanded: &expanded, Label: "Compose"})
			})
		})
	}, 560, 220)

	collapsed := buttonInfo(t, tt, ui.RoleButton, "Compose")
	if collapsed.Bounds.H != 56 {
		t.Fatalf("extended FAB height = %v, want 56", collapsed.Bounds.H)
	}
	if collapsed.Bounds.W != 56 {
		t.Fatalf("collapsed width = %v, want the 56 DIP icon-only square", collapsed.Bounds.W)
	}
	// The icon stays drawn at the center of the collapsed square rather than
	// being squeezed away or centered off the clipped area.
	container := rgbOf(th.Colors.PrimaryContainer)
	ink := 0
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			px := tt.Image().RGBAAt(int(collapsed.Bounds.X+collapsed.Bounds.W/2)+dx,
				int(collapsed.Bounds.Y+collapsed.Bounds.H/2)+dy)
			if absColorDiff(px, container) > 30 {
				ink++
			}
		}
	}
	if ink == 0 {
		t.Fatal("collapsed extended FAB painted no icon at its center")
	}

	expanded = true
	tt.Frame()
	tt.Advance(1500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("expanding the extended FAB left frames pending")
	}
	open := buttonInfo(t, tt, ui.RoleButton, "Compose")
	if open.Bounds.W < collapsed.Bounds.W+60 {
		t.Fatalf("expanded width = %v, not clearly past the collapsed %v", open.Bounds.W, collapsed.Bounds.W)
	}
	if open.Bounds.H != 56 {
		t.Fatalf("expanded height = %v, want 56", open.Bounds.H)
	}
	labelRect, ok := tt.Find("New message")
	if !ok {
		t.Fatal("expanded label not laid out")
	}
	// The 12 DIP icon-label gap separates the 24 DIP icon from the text.
	wantLabelX := open.Bounds.X + 16 + 24 + 12
	if math.Abs(float64(labelRect.X-wantLabelX)) > 1.5 {
		t.Fatalf("expanded label x = %v, want %v after the icon and 12 DIP gap", labelRect.X, wantLabelX)
	}

	expanded = false
	tt.Frame()
	tt.Advance(1500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("collapsing the extended FAB left frames pending")
	}
	if back := buttonInfo(t, tt, ui.RoleButton, "Compose"); back.Bounds.W > 57 {
		t.Fatalf("re-collapsed width = %v, want 56", back.Bounds.W)
	}
}

// TestFABMenuKeyboardFocusGroup checks that Tab reaches the menu items and that
// the vertical focus group moves between them with the arrow keys.
func TestFABMenuKeyboardFocusGroup(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	opened := false
	clicks := 0
	items := fabMenuItems(3, &clicks)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(480, 360).Background(canvasRGB).Padding(24).
				Justify(ui.End).AlignItems(ui.End).Children(func() {
				FABMenu(c, items, FABMenuOptions{Open: &opened, Label: "Create"})
			})
		})
	}, 480, 360)

	trigger := buttonInfo(t, tt, ui.RoleButton, "Create")
	tt.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
	if !opened {
		t.Fatal("trigger click did not open the menu")
	}
	tt.Advance(1200 * time.Millisecond)

	// New folder sits above New file in the panel, so Down should move from it
	// to New file.
	labels := []string{"New note", "New folder", "New file"}
	focused := -1
	for i := 0; i < 6 && focused < 0; i++ {
		tt.Key(0, ui.KeyTab)
		for k, label := range labels {
			if tt.Focused(label) {
				focused = k
			}
		}
	}
	if focused < 0 {
		t.Fatal("Tab never reached the FAB menu items")
	}
	tt.Key(0, ui.KeyDown)
	next := -1
	for k, label := range labels {
		if tt.Focused(label) {
			next = k
		}
	}
	if next < 0 || next == focused {
		t.Fatalf("Down did not move the focus down the FAB menu: item %d -> %d", focused, next)
	}
}

// fabMenuItems returns n labeled items with counting callbacks.
func fabMenuItems(n int, clicks *int) []FABMenuItem {
	items := make([]FABMenuItem, n)
	for i := range items {
		items[i] = FABMenuItem{
			Label:   []string{"New file", "New folder", "New note", "Upload", "Share", "Print"}[i],
			Icon:    mdicon.Add.Draw,
			OnClick: func() { *clicks += i + 1 },
		}
	}
	return items
}

// TestFABMenuItemCountBounds checks that fewer than two or more than six items
// panic, and that the boundary counts build.
func TestFABMenuItemCountBounds(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	build := func(items []FABMenuItem) func() {
		return func() {
			ui.NewTester(func(c *ui.Context) {
				ThemeScope(c, th, func() {
					ui.Box(c).Size(300, 300).Children(func() {
						FABMenu(c, items, FABMenuOptions{Label: "Create"})
					})
				})
			}, 300, 300)
		}
	}
	clicks := 0
	for _, n := range []int{0, 1, 7} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%d items did not panic", n)
				}
			}()
			build(fabMenuItems(n, &clicks))()
		}()
	}
	for _, n := range []int{2, 6} {
		build(fabMenuItems(n, &clicks))()
	}
}

// TestFABMenuActionsAndDismiss checks the scrim, the item access tree, choosing
// an item, dismissing with the scrim and the in-place close button, and the
// focus returning to the trigger each time.
func TestFABMenuActionsAndDismiss(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	opened := false
	clicks := 0
	items := fabMenuItems(2, &clicks)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(480, 360).Background(canvasRGB).Padding(24).
				Justify(ui.End).AlignItems(ui.End).Children(func() {
				FABMenu(c, items, FABMenuOptions{Open: &opened, Label: "Create"})
			})
		})
	}, 480, 360)

	trigger := buttonInfo(t, tt, ui.RoleButton, "Create")
	if trigger.Bounds.W != 56 || trigger.Bounds.H != 56 {
		t.Fatalf("FAB menu trigger = %v×%v, want the standard 56×56 FAB",
			trigger.Bounds.W, trigger.Bounds.H)
	}
	if trigger.HasPopup != ui.PopupMenu || trigger.Expanded {
		t.Fatalf("closed trigger: popup=%v expanded=%v, want a collapsed menu button",
			trigger.HasPopup, trigger.Expanded)
	}
	tx, ty := center(trigger.Bounds)
	tt.ClickAt(tx, ty)
	if !opened {
		t.Fatal("trigger click did not open the menu")
	}
	tt.Advance(1500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("opening the FAB menu left frames pending")
	}

	// The scrim covers the window, and every item is in the access tree.
	if px := tt.Image().RGBAAt(4, 4); absColorDiff(px, white) < 60 {
		t.Fatalf("scrim did not darken the window corner: %v", px)
	}
	file := buttonInfo(t, tt, ui.RoleMenuItem, "New file")
	folder := buttonInfo(t, tt, ui.RoleMenuItem, "New folder")
	closeBtn := buttonInfo(t, tt, ui.RoleButton, "Close menu")
	if file.Bounds.Y+file.Bounds.H > trigger.Bounds.Y {
		t.Fatalf("items do not sit above the trigger: item bottom %v, trigger top %v",
			file.Bounds.Y+file.Bounds.H, trigger.Bounds.Y)
	}
	if folder.Bounds.Y+folder.Bounds.H > file.Bounds.Y {
		t.Fatalf("items are not stacked upward: folder bottom %v, file top %v",
			folder.Bounds.Y+folder.Bounds.H, file.Bounds.Y)
	}
	if closeBtn.Bounds.W != 56 || closeBtn.Bounds.H != 56 {
		t.Fatalf("close button = %v×%v, want 56×56", closeBtn.Bounds.W, closeBtn.Bounds.H)
	}
	// The panel is end-aligned with its trigger, so wider items do not spill
	// past the FAB toward the window edge.
	if got, want := file.Bounds.X+file.Bounds.W, trigger.Bounds.X+trigger.Bounds.W; math.Abs(float64(got-want)) > 1 {
		t.Fatalf("menu item right edge = %v, want the trigger's %v", got, want)
	}

	// Choosing an item runs its callback, closes the menu, and gives the focus
	// back to the trigger.
	fx, fy := center(file.Bounds)
	tt.ClickAt(fx, fy)
	if clicks != 1 {
		t.Fatalf("item callback ran %d times, want 1", clicks)
	}
	if opened {
		t.Fatal("choosing an item did not close the menu")
	}
	tt.Advance(1500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("closing the FAB menu left frames pending")
	}
	if _, ok := tt.AccessByRole(ui.RoleMenuItem, "New file"); ok {
		t.Fatal("menu items stayed in the access tree after closing")
	}
	if !tt.Focused("Create") {
		t.Fatal("choosing an item did not return the focus to the trigger")
	}

	// A press on the scrim closes it.
	trigger = buttonInfo(t, tt, ui.RoleButton, "Create")
	tt.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
	tt.Advance(1200 * time.Millisecond)
	if !opened {
		t.Fatal("trigger did not reopen the menu")
	}
	tt.ClickAt(10, 10)
	if opened {
		t.Fatal("a press on the scrim did not close the menu")
	}
	tt.Advance(1200 * time.Millisecond)
	if !tt.Focused("Create") {
		t.Fatal("the scrim press did not return the focus to the trigger")
	}

	// The in-place close button closes it too.
	trigger = buttonInfo(t, tt, ui.RoleButton, "Create")
	tt.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
	tt.Advance(1200 * time.Millisecond)
	if !opened {
		t.Fatal("trigger did not reopen the menu a second time")
	}
	closeBtn = buttonInfo(t, tt, ui.RoleButton, "Close menu")
	cx, cy := center(closeBtn.Bounds)
	tt.ClickAt(cx, cy)
	if opened {
		t.Fatal("the close button did not close the menu")
	}
	tt.Advance(1200 * time.Millisecond)
	if !tt.Focused("Create") {
		t.Fatal("the close button did not return the focus to the trigger")
	}
}

// TestFABMenuHidesTriggerUnderCloseButton checks that, while the menu shows,
// the trigger's rounded square, icon and shadows stop painting around the
// circular close button, even during its entry pose, and that closing shows the
// trigger again at once with the keyboard focus back on it.
func TestFABMenuHidesTriggerUnderCloseButton(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	opened := false
	clicks := 0
	items := fabMenuItems(2, &clicks)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(480, 360).Background(canvasRGB).Padding(24).
				Justify(ui.End).AlignItems(ui.End).Children(func() {
				FABMenu(c, items, FABMenuOptions{Open: &opened, Label: "Create"})
			})
		})
	}, 480, 360)

	trigger := buttonInfo(t, tt, ui.RoleButton, "Create")
	tx, ty := center(trigger.Bounds)
	tt.ClickAt(tx, ty)
	if !opened {
		t.Fatal("trigger click did not open the menu")
	}
	closeBtn := buttonInfo(t, tt, ui.RoleButton, "Close menu")
	// The round close button sits in the trigger's square box; its corners lie
	// outside both the close circle and the trigger's former rounded square's
	// own roundings only in the diagonal band a corner probe samples. Nothing
	// of the trigger may show there while the menu is open.
	cornerProbe := func(img *image.RGBA) color.RGBA {
		return img.RGBAAt(int(closeBtn.Bounds.X)+6, int(closeBtn.Bounds.Y)+6)
	}
	if px := cornerProbe(tt.Image()); absColorDiff(px, rgbOf(th.Colors.PrimaryContainer)) <= 20 {
		t.Fatalf("open menu still paints the trigger's container in the close button corner: %v", px)
	}
	tt.Advance(1500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("the open menu keeps asking for frames")
	}
	if px := cornerProbe(tt.Image()); absColorDiff(px, rgbOf(th.Colors.PrimaryContainer)) <= 20 {
		t.Fatalf("the settled close button still shows the trigger's container at its corner: %v", px)
	}

	cx, cy := center(closeBtn.Bounds)
	tt.ClickAt(cx, cy)
	if opened {
		t.Fatal("the close button did not close the menu")
	}
	tt.Frame()
	if !tt.Focused("Create") {
		t.Fatal("closing the menu did not return the focus to the trigger")
	}
	// The pointer still rests on the trigger, whose hover state layer darkens
	// its container; move it away to read the plain container color.
	tt.Move(10, 10)
	tt.Advance(400 * time.Millisecond)
	trigger = buttonInfo(t, tt, ui.RoleButton, "Create")
	if px := fabContainerProbe(t, tt, trigger.Bounds); absColorDiff(px, rgbOf(th.Colors.PrimaryContainer)) > 4 {
		t.Fatalf("the trigger did not show again at once after closing: %v", px)
	}
}

// TestFABMenuEscapeAndReduceMotion checks Escape closing with the focus back on
// the trigger and the whole menu settling without extra frames under
// ReduceMotion.
func TestFABMenuEscapeAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	opened := false
	clicks := 0
	items := fabMenuItems(3, &clicks)
	tt := ui.NewTesterWith(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(480, 360).Background(canvasRGB).Padding(24).
				Justify(ui.End).AlignItems(ui.End).Children(func() {
				FABMenu(c, items, FABMenuOptions{Open: &opened, Label: "Create"})
			})
		})
	}, 480, 360, ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})

	trigger := buttonInfo(t, tt, ui.RoleButton, "Create")
	tt.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
	if !opened {
		t.Fatal("trigger click did not open the menu under ReduceMotion")
	}
	// ReduceMotion must not pose a first invisible frame and ask for another.
	if tt.FramePending() {
		t.Fatal("ReduceMotion opening asked for an extra frame")
	}
	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("ReduceMotion opening left frames pending")
	}
	buttonInfo(t, tt, ui.RoleMenuItem, "New file")

	tt.Key(0, ui.KeyEscape)
	if opened {
		t.Fatal("Escape did not close the menu")
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion closing asked for an extra frame")
	}
	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("ReduceMotion closing left frames pending")
	}
	if !tt.Focused("Create") {
		t.Fatal("Escape did not return the focus to the trigger")
	}
}

func TestFABMenuAncestorDisabledClosesAndOmitsOverlay(t *testing.T) {
	open := true
	parentDisabled := true
	itemClicks := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, nil, func() {
			ui.Row(c).Size(320, 320).Padding(24).Disabled(parentDisabled).Children(func() {
				FABMenu(c, []FABMenuItem{
					{Label: "Upload", Icon: mdicon.Edit.Draw, OnClick: func() { itemClicks++ }},
					{Label: "Share", Icon: mdicon.Add.Draw, OnClick: func() { itemClicks++ }},
				}, FABMenuOptions{Label: "Actions", Icon: mdicon.Add.Draw, Open: &open})
			})
		})
	}, 320, 320)

	if open {
		t.Fatal("expected FABMenu Open state to close when ancestor is disabled")
	}
	if _, ok := tt.Find("Upload"); ok {
		t.Fatal("disabled ancestor should prevent FABMenu overlay items from rendering")
	}
	trigger := buttonInfo(t, tt, ui.RoleButton, "Actions")
	if !trigger.Disabled || trigger.Expanded {
		t.Fatalf("trigger access disabled=%v expanded=%v, want true false", trigger.Disabled, trigger.Expanded)
	}
	tt.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
	if open || itemClicks != 0 {
		t.Fatalf("clicking disabled trigger opened menu: open=%v clicks=%d", open, itemClicks)
	}
}

func TestFABRapidToggleAndMidAnimationReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	expanded := true
	visible := true
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(320, 240).Padding(20).Gap(16).Children(func() {
				FAB(c, "Add", mdicon.Add.Draw, FABOptions{Visible: &visible})
				ExtendedFAB(c, "Compose", mdicon.Edit.Draw, ExtendedFABOptions{Expanded: &expanded})
			})
		})
	}, 320, 240)

	fullW := buttonInfo(t, tt, ui.RoleButton, "Compose").Bounds.W

	// Rapidly reverse directions mid-animation without settling.
	visible = false
	expanded = false
	tt.Frame()
	tt.Advance(32 * time.Millisecond)
	visible = true
	expanded = true
	tt.Frame()
	tt.Advance(24 * time.Millisecond)
	expanded = false
	tt.Frame()
	tt.Advance(16 * time.Millisecond)

	// Flip ReduceMotion ON mid-flight: spatial dimensions must snap immediately to target.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	tt.Advance(200 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("mid-flight ReduceMotion switch left FAB animations pending")
	}
	collapsedW := buttonInfo(t, tt, ui.RoleButton, "Compose").Bounds.W
	if collapsedW >= fullW {
		t.Fatalf("ExtendedFAB did not snap collapsed after mid-flight ReduceMotion: collapsed=%v full=%v", collapsedW, fullW)
	}
}
