package md3e

import (
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

// splitFixture builds a split button with a counting menu, keeping its
// expansion state and click counts across rebuilds.
type splitFixture struct {
	tt           *ui.Tester
	open         *bool
	actionClicks int
	menuBuilds   int
}

func newSplitFixture(t *testing.T, o SplitButtonOptions) *splitFixture {
	t.Helper()
	th := ThemeFromSeed(DefaultSeedColor)
	f := &splitFixture{}
	content := o.Menu
	if content == nil {
		content = func(ui.Element) {}
	}
	f.tt = ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(460, 280).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				opts := o
				opts.Menu = func(panel ui.Element) {
					f.menuBuilds++
					content(panel)
				}
				_, action, _, open := SplitButton(c, "Save", opts)
				f.open = open
				if action.Clicked() {
					f.actionClicks++
				}
			})
		})
	}, 460, 280)
	return f
}

// buttonInfo returns the access info of a button by accessible name.
func buttonInfo(t *testing.T, tt *ui.Tester, role ui.Role, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(role, name)
	if !ok {
		t.Fatalf("%q not in the access tree", name)
	}
	return info
}

// center returns the center of a box.
func center(b ui.Rect) (float32, float32) {
	return b.X + b.W/2, b.Y + b.H/2
}

// TestSplitButtonDisabledClosesOpenMenu checks that disabling the control
// while the menu shows closes it and clears the expanded state.
func TestSplitButtonDisabledClosesOpenMenu(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	disabled := false
	open := false
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(460, 280).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				SplitButton(c, "Save", SplitButtonOptions{
					Size:     tokens.SizeS,
					Disabled: disabled,
					Open:     &open,
					Menu: func(panel ui.Element) {
						panel.Children(func() { ui.ButtonBase(c).Label("Archive").Children(func() { ui.Text(c, "Archive") }) })
					},
				})
			})
		})
	}, 460, 280)

	arrow := buttonInfo(t, tt, ui.RoleButton, "More options")
	rx, ry := center(arrow.Bounds)
	tt.ClickAt(rx, ry)
	if !open {
		t.Fatal("arrow click did not open the menu")
	}
	tt.Advance(600 * time.Millisecond)

	disabled = true
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	if open {
		t.Fatal("disabling the split button left the menu open")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Archive"); ok {
		t.Fatal("the disabled split button still shows its menu")
	}
	if buttonInfo(t, tt, ui.RoleButton, "More options").Expanded {
		t.Fatal("assistive technology still sees the disabled arrow expanded")
	}
}

// TestSplitButtonIconOnlyWidth checks that an empty label adds no empty text
// and its gap to a pure-icon split button.
func TestSplitButtonIconOnlyWidth(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(460, 280).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				SplitButton(c, "", SplitButtonOptions{
					Size:        tokens.SizeS,
					LeadingIcon: mdicon.Add.Draw,
					Menu:        func(ui.Element) {},
				})
			})
		})
	}, 460, 280)

	var action ui.AccessInfo
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleButton && a.Name == "" {
			action = a
		}
	}
	if action.Bounds.W == 0 {
		t.Fatal("the icon-only action segment was not laid out")
	}
	spec := tokens.SizeS.Spec()
	want := float32(2*spec.Padding) + float32(spec.IconSize)
	if action.Bounds.W != want {
		t.Fatalf("icon-only action width = %v, want %v without an empty label gap", action.Bounds.W, want)
	}
}

// TestSplitButtonPressedSeamKeepsInnerRadius checks that pressing a large split
// button shrinks its outer corners without inflating the 8 DIP seam.
func TestSplitButtonPressedSeamKeepsInnerRadius(t *testing.T) {
	span := connectedInnerRadii
	seamFill := func(tt *ui.Tester, b ui.Rect) int {
		return cornerFillAt(tt.Image(), int(b.X+b.W)-span, int(b.Y), span)
	}

	f := newSplitFixture(t, SplitButtonOptions{Size: tokens.SizeL})
	tt := f.tt
	tt.Move(5, 5)
	tt.Advance(600 * time.Millisecond)
	action := buttonInfo(t, tt, ui.RoleButton, "Save")
	idle := seamFill(tt, action.Bounds)

	cx, cy := center(action.Bounds)
	tt.Move(cx, cy)
	tt.Press(cx, cy)
	tt.Advance(800 * time.Millisecond)
	if got := seamFill(tt, action.Bounds); got > idle+span*span/8 {
		t.Fatalf("pressed seam corner fill %d grew from %d: the 8 DIP seam inflated", got, idle)
	}
	tt.Release(cx, cy)
	tt.Move(5, 5)
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("the split button did not settle after the press")
	}
}

// TestSplitButtonActionMenuAndAccessibility checks the group and button roles,
// the main action's click, the arrow opening and closing the menu, and the
// expanded state assistive technology reads.
func TestSplitButtonActionMenuAndAccessibility(t *testing.T) {
	f := newSplitFixture(t, SplitButtonOptions{Size: tokens.SizeS})
	tt := f.tt

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Save"); !ok {
		t.Fatal("split button group not in the access tree")
	}
	action := buttonInfo(t, tt, ui.RoleButton, "Save")
	arrow := buttonInfo(t, tt, ui.RoleButton, "More options")
	if arrow.HasPopup != ui.PopupMenu {
		t.Fatalf("arrow HasPopup = %v, want a menu", arrow.HasPopup)
	}
	if arrow.Expanded {
		t.Fatal("arrow reports itself expanded while closed")
	}
	if *f.open {
		t.Fatal("split button started open")
	}

	// The action clicks alone, without opening the menu.
	ax, ay := center(action.Bounds)
	tt.ClickAt(ax, ay)
	if f.actionClicks != 1 {
		t.Fatalf("action clicks = %d, want 1", f.actionClicks)
	}
	if *f.open || f.menuBuilds != 0 {
		t.Fatalf("action click opened the menu: open=%v builds=%d", *f.open, f.menuBuilds)
	}

	// The arrow opens the menu, and the menu is built.
	rx, ry := center(arrow.Bounds)
	tt.ClickAt(rx, ry)
	if !*f.open {
		t.Fatal("arrow click did not open the menu")
	}
	tt.Advance(1000 * time.Millisecond)
	if f.menuBuilds == 0 {
		t.Fatal("menu content was not built")
	}
	if tt.FramePending() {
		t.Fatal("opening did not settle")
	}
	if !buttonInfo(t, tt, ui.RoleButton, "More options").Expanded {
		t.Fatal("arrow does not report itself expanded while open")
	}

	// Escape closes it.
	tt.Key(0, ui.KeyEscape)
	if *f.open {
		t.Fatal("Escape did not close the menu")
	}
	tt.Advance(1000 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("closing did not settle")
	}
	if buttonInfo(t, tt, ui.RoleButton, "More options").Expanded {
		t.Fatal("arrow still reports itself expanded after Escape")
	}
}

// TestSplitButtonKeyboard checks Alt+Down on the action and the arrow's own
// Space, Enter, Up and Down keys.
func TestSplitButtonKeyboard(t *testing.T) {
	f := newSplitFixture(t, SplitButtonOptions{Size: tokens.SizeS})
	tt := f.tt

	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Save") {
		t.Fatal("Tab did not focus the action")
	}
	tt.Key(ui.Alt, ui.KeyDown)
	if !*f.open {
		t.Fatal("Alt+Down on the action did not open the menu")
	}
	tt.Key(0, ui.KeyEscape)
	if *f.open {
		t.Fatal("Escape did not close the menu")
	}

	tt.Key(0, ui.KeyRight)
	action := buttonInfo(t, tt, ui.RoleButton, "Save")
	arrow := buttonInfo(t, tt, ui.RoleButton, "More options")
	if arrow.Bounds.X <= action.Bounds.X {
		t.Fatalf("Right did not move the focus to the arrow: action x=%v arrow x=%v", action.Bounds.X, arrow.Bounds.X)
	}
	tt.Key(0, ui.KeySpace)
	if !*f.open {
		t.Fatal("Space on the arrow did not open the menu")
	}
	// Down and Up on an open menu button keep it open, as WAI-ARIA asks.
	tt.Key(0, ui.KeyDown)
	if !*f.open {
		t.Fatal("Down on the arrow closed the menu")
	}
	tt.Key(0, ui.KeyUp)
	if !*f.open {
		t.Fatal("Up on the arrow closed the menu")
	}
	tt.Key(0, ui.KeyEscape)
	if *f.open {
		t.Fatal("Escape did not close the menu")
	}
	// From closed, Down and Up open it.
	tt.Key(0, ui.KeyDown)
	if !*f.open {
		t.Fatal("Down on the closed arrow did not open the menu")
	}
	tt.Key(0, ui.KeyEscape)
	if *f.open {
		t.Fatal("Escape did not close the menu a second time")
	}
	tt.Key(0, ui.KeyUp)
	if !*f.open {
		t.Fatal("Up on the closed arrow did not open the menu")
	}
	tt.Key(0, ui.KeyEscape)
	if *f.open {
		t.Fatal("Escape did not close the menu a third time")
	}
	tt.Key(0, ui.KeyEnter)
	if !*f.open {
		t.Fatal("Enter on the arrow did not open the menu")
	}
}

// iconPixels copies the pixels of a square around the center of b.
func iconPixels(img interface{ RGBAAt(x, y int) color.RGBA }, b ui.Rect, side int) []color.RGBA {
	cx, cy := int(b.X+b.W/2), int(b.Y+b.H/2)
	var px []color.RGBA
	for y := -side; y < side; y++ {
		for x := -side; x < side; x++ {
			px = append(px, img.RGBAAt(cx+x, cy+y))
		}
	}
	return px
}

// TestSplitButtonMorphAndRTL checks the seam radii, the symmetric pill the
// arrow morphs into as the menu opens, the chevron turning 180 degrees, and the
// mirrored geometry and radii right to left.
func TestSplitButtonMorphAndRTL(t *testing.T) {
	span := int(tokens.SizeS.Spec().SquareRadius)
	f := newSplitFixture(t, SplitButtonOptions{Size: tokens.SizeS})
	tt := f.tt
	tt.Move(5, 5)

	action := buttonInfo(t, tt, ui.RoleButton, "Save")
	arrow := buttonInfo(t, tt, ui.RoleButton, "More options")
	if action.Bounds.X >= arrow.Bounds.X {
		t.Fatalf("left to right: action x=%v not before the arrow x=%v", action.Bounds.X, arrow.Bounds.X)
	}
	leftOf := func(b ui.Rect) int { return cornerFillAt(tt.Image(), int(b.X), int(b.Y), span) }
	rightOf := func(b ui.Rect) int {
		return cornerFillAt(tt.Image(), int(b.X+b.W)-span, int(b.Y), span)
	}
	// The arrow's outer end is round, its seam corner small.
	outerFill := rightOf(arrow.Bounds)
	seamFill := leftOf(arrow.Bounds)
	if seamFill <= outerFill+10 {
		t.Fatalf("arrow corners: seam fill %d not clearly above the outer %d", seamFill, outerFill)
	}
	closedIcon := iconPixels(tt.Image(), arrow.Bounds, 6)

	rx, ry := center(arrow.Bounds)
	tt.ClickAt(rx, ry)
	if !*f.open {
		t.Fatal("click did not open the menu")
	}
	tt.Advance(1000 * time.Millisecond)
	tt.Move(5, 5)
	tt.Advance(500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("opening did not settle")
	}
	if got := leftOf(arrow.Bounds); got >= seamFill-10 {
		t.Fatalf("open arrow seam fill %d did not morph from %d into a symmetric pill", got, seamFill)
	}
	openIcon := iconPixels(tt.Image(), arrow.Bounds, 6)
	diff := 0
	for i := range closedIcon {
		if closedIcon[i] != openIcon[i] {
			diff++
		}
	}
	if diff < 4 {
		t.Fatalf("chevron rotated by only %d pixels; want a visible 180 degree turn", diff)
	}

	tt.ClickAt(rx, ry)
	if *f.open {
		t.Fatal("second click did not close the menu")
	}
	tt.Advance(1000 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("closing did not settle")
	}
	if got := leftOf(arrow.Bounds); got < seamFill-10 {
		t.Fatalf("closed arrow seam fill %d did not return to the small seam %d", got, seamFill)
	}

	// Right to left: the action is on the right, its outer radius on its right.
	tt.SetDirection(ui.RightToLeft)
	tt.Advance(1000 * time.Millisecond)
	action = buttonInfo(t, tt, ui.RoleButton, "Save")
	arrow = buttonInfo(t, tt, ui.RoleButton, "More options")
	if action.Bounds.X <= arrow.Bounds.X {
		t.Fatalf("right to left: action x=%v not after the arrow x=%v", action.Bounds.X, arrow.Bounds.X)
	}
	if l, r := leftOf(action.Bounds), rightOf(action.Bounds); r >= l-10 {
		t.Fatalf("right to left action corners: outer right fill %d not clearly below the seam left %d", r, l)
	}
	if l, r := leftOf(arrow.Bounds), rightOf(arrow.Bounds); l >= r-10 {
		t.Fatalf("right to left arrow corners: outer left fill %d not clearly below the seam right %d", l, r)
	}
}

// TestSplitButtonVariantsAndSizes checks that every look and size tier lays the
// two segments out at the tier height and keeps the group accessible.
func TestSplitButtonVariantsAndSizes(t *testing.T) {
	for _, variant := range buttonVariants {
		for _, size := range buttonSizes {
			f := newSplitFixture(t, SplitButtonOptions{Variant: variant.v, Size: size})
			group, ok := f.tt.AccessByRole(ui.RoleGroup, "Save")
			if !ok {
				t.Fatalf("%s/%d: group not in the access tree", variant.name, size)
			}
			action := buttonInfo(t, f.tt, ui.RoleButton, "Save")
			arrow := buttonInfo(t, f.tt, ui.RoleButton, "More options")
			spec := size.Spec()
			for _, b := range []ui.Rect{group.Bounds, action.Bounds, arrow.Bounds} {
				if b.H != float32(spec.Height) {
					t.Fatalf("%s/%d: segment height = %v, want %v", variant.name, size, b.H, spec.Height)
				}
			}
			if action.Bounds.X >= arrow.Bounds.X {
				t.Fatalf("%s/%d: action x=%v not before the arrow x=%v",
					variant.name, size, action.Bounds.X, arrow.Bounds.X)
			}
		}
	}
}

// TestSplitButtonDisabled checks the whole control disabled and the action
// alone disabled, the menu still opening.
func TestSplitButtonDisabled(t *testing.T) {
	f := newSplitFixture(t, SplitButtonOptions{Size: tokens.SizeS, Disabled: true})
	tt := f.tt
	action := buttonInfo(t, tt, ui.RoleButton, "Save")
	arrow := buttonInfo(t, tt, ui.RoleButton, "More options")
	if !action.Disabled || !arrow.Disabled {
		t.Fatalf("disabled split button reports action %v, arrow %v disabled", action.Disabled, arrow.Disabled)
	}
	ax, ay := center(action.Bounds)
	rx, ry := center(arrow.Bounds)
	tt.ClickAt(ax, ay)
	tt.ClickAt(rx, ry)
	tt.Advance(600 * time.Millisecond)
	if f.actionClicks != 0 {
		t.Fatalf("disabled action clicked %d times", f.actionClicks)
	}
	if *f.open {
		t.Fatal("disabled arrow opened the menu")
	}
	if tt.FramePending() {
		t.Fatal("disabled split button left frames pending")
	}

	g := newSplitFixture(t, SplitButtonOptions{Size: tokens.SizeS, ActionDisabled: true})
	action = buttonInfo(t, g.tt, ui.RoleButton, "Save")
	arrow = buttonInfo(t, g.tt, ui.RoleButton, "More options")
	if !action.Disabled || arrow.Disabled {
		t.Fatalf("action-disabled split button reports action %v, arrow %v disabled", action.Disabled, arrow.Disabled)
	}

	// The palettes are strict: the action alone wears MD3E's disabled OnSurface
	// container while the arrow keeps the variant's enabled container.
	th := ThemeFromSeed(DefaultSeedColor)
	rgb := func(c ui.Color) color.RGBA { return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A} }
	wantAction := rgb(th.Colors.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)).Over(canvasRGB))
	wantArrow := rgb(th.Colors.Primary)
	if got := g.tt.Image().RGBAAt(int(action.Bounds.X)+6, int(action.Bounds.Y+action.Bounds.H/2)); absColorDiff(got, wantAction) > 4 {
		t.Fatalf("action-disabled container = %v, want the disabled OnSurface 12%% %v", got, wantAction)
	}
	if got := g.tt.Image().RGBAAt(int(arrow.Bounds.X)+6, int(arrow.Bounds.Y+arrow.Bounds.H/2)); absColorDiff(got, wantArrow) > 4 {
		t.Fatalf("arrow container = %v, want the enabled Primary %v", got, wantArrow)
	}

	ax, ay = center(action.Bounds)
	rx, ry = center(arrow.Bounds)
	g.tt.ClickAt(ax, ay)
	if g.actionClicks != 0 {
		t.Fatalf("action-disabled action clicked %d times", g.actionClicks)
	}
	g.tt.ClickAt(rx, ry)
	if !*g.open {
		t.Fatal("arrow did not open with the action disabled")
	}
}
