package md3e

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

var groupItems = []ButtonGroupItem{{Label: "Cut"}, {Label: "Copy"}, {Label: "Paste"}}

// cornerFillAt counts the pixels in the span×span box at (x, y) that differ
// from white.
func cornerFillAt(img *image.RGBA, x, y, span int) int {
	n := 0
	for dy := 0; dy < span; dy++ {
		for dx := 0; dx < span; dx++ {
			if absColorDiff(img.RGBAAt(x+dx, y+dy), white) > 12 {
				n++
			}
		}
	}
	return n
}

// groupAccess returns the access info of a group's button by label.
func groupAccess(t *testing.T, tt *ui.Tester, label string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("button %q not in the access tree", label)
	}
	return info
}

// TestButtonGroupLargePressedSeam checks that pressing a large Connected
// group's segment keeps its 8 DIP seam instead of inflating it to the 16 DIP
// pressed radius.
func TestButtonGroupLargePressedSeam(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	span := connectedInnerRadii
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(900, 220).Background(canvasRGB).Padding(24).AlignItems(ui.Start).Children(func() {
				ButtonGroupActions(c, groupItems, nil, ButtonGroupOptions{
					Kind: ButtonGroupConnected,
					Size: tokens.SizeL,
				})
			})
		})
	}, 900, 220)

	copyBtn := groupAccess(t, tt, "Copy")
	seam := func(b ui.Rect) int { return cornerFillAt(tt.Image(), int(b.X), int(b.Y), span) }
	idle := seam(copyBtn.Bounds)

	cx, cy := copyBtn.Bounds.X+copyBtn.Bounds.W/2, copyBtn.Bounds.Y+copyBtn.Bounds.H/2
	tt.Move(cx, cy)
	tt.Press(cx, cy)
	tt.Advance(800 * time.Millisecond)
	if got := seam(copyBtn.Bounds); got > idle+span*span/8 {
		t.Fatalf("pressed seam corner fill %d grew from %d: the seam inflated", got, idle)
	}
	tt.Release(cx, cy)
	tt.Move(5, 5)
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("the large group did not settle after the press")
	}
}

// TestButtonGroupElevatedPressLiftAndTextDisabled checks the Elevated segment's
// Level1 to Level2 lift while pressed and that a disabled Text segment paints
// no container.
func TestButtonGroupElevatedPressLiftAndTextDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 200).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ButtonGroupActions(c, groupItems, nil, ButtonGroupOptions{
					Variant: ButtonElevated,
					Size:    tokens.SizeS,
				})
			})
		})
	}, 560, 200)

	cut := groupAccess(t, tt, "Cut")
	darkness := func(b ui.Rect) int {
		dark := 0
		x := int(b.X + b.W/2)
		for dy := 1; dy <= 4; dy++ {
			dark += 255 - int(tt.Image().RGBAAt(x, int(b.Y+b.H)+dy).R)
		}
		return dark
	}
	idle := darkness(cut.Bounds)
	if idle == 0 {
		t.Fatal("the elevated group cast no Level1 shadow")
	}
	cx, cy := cut.Bounds.X+cut.Bounds.W/2, cut.Bounds.Y+cut.Bounds.H/2
	tt.Move(cx, cy)
	tt.Press(cx, cy)
	tt.Advance(700 * time.Millisecond)
	if pressed := darkness(cut.Bounds); pressed <= idle {
		t.Fatalf("pressed shadow darkness %d is not above the idle %d: no Level1 to Level2 lift", pressed, idle)
	}
	tt.Release(cx, cy)
	tt.Move(5, 5)
	tt.Advance(700 * time.Millisecond)

	text := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 160).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ButtonGroupActions(c, groupItems, nil, ButtonGroupOptions{
					Variant:  ButtonText,
					Size:     tokens.SizeS,
					Disabled: true,
				})
			})
		})
	}, 560, 160)
	disabled := groupAccess(t, text, "Cut")
	px := text.Image().RGBAAt(int(disabled.Bounds.X+disabled.Bounds.W*0.15), int(disabled.Bounds.Y+disabled.Bounds.H/2))
	if px != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("disabled Text group segment painted a container: %v", px)
	}
}

// TestButtonGroupStandardSqueeze checks that pressing a button of a Standard
// group widens it as its neighbors give way, and that releasing springs every
// width back to rest.
func TestButtonGroupStandardSqueeze(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 160).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ButtonGroupActions(c, groupItems, nil, ButtonGroupOptions{Size: tokens.SizeS})
			})
		})
	}, 560, 160)

	widths := func() [3]float32 {
		var w [3]float32
		for i, item := range groupItems {
			w[i] = groupAccess(t, tt, item.Label).Bounds.W
		}
		return w
	}
	idle := widths()

	copy := groupAccess(t, tt, "Copy")
	cx := copy.Bounds.X + copy.Bounds.W/2
	cy := copy.Bounds.Y + copy.Bounds.H/2
	tt.Move(cx, cy)
	tt.Press(cx, cy)
	if !tt.FramePending() {
		t.Fatal("pressing should start the group's width spring")
	}
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("pressed group did not settle")
	}
	pressed := widths()
	if pressed[1] < idle[1]+8 {
		t.Fatalf("pressed button width %v did not grow from %v", pressed[1], idle[1])
	}
	if pressed[0] > idle[0]-3 || pressed[2] > idle[2]-3 {
		t.Fatalf("neighbors did not give way: %v and %v, idle %v and %v",
			pressed[0], pressed[2], idle[0], idle[2])
	}

	tt.Release(cx, cy)
	tt.Move(5, 5)
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("released group did not settle")
	}
	released := widths()
	for i := range idle {
		if math.Abs(float64(released[i]-idle[i])) > 0.6 {
			t.Fatalf("width %d did not spring back: %v, idle %v", i, released[i], idle[i])
		}
	}
}

// TestButtonGroupConnectedRadiiAndSelection checks the connected outer and seam
// corners, the chosen pill morph, and the single selection state.
func TestButtonGroupConnectedRadiiAndSelection(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	span := int(tokens.SizeXS.Spec().SquareRadius)

	action := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 160).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ButtonGroupActions(c, groupItems, nil, ButtonGroupOptions{
					Kind: ButtonGroupConnected,
					Size: tokens.SizeXS,
				})
			})
		})
	}, 560, 160)

	leftOf := func(tt *ui.Tester, b ui.Rect) int { return cornerFillAt(tt.Image(), int(b.X), int(b.Y), span) }
	rightOf := func(tt *ui.Tester, b ui.Rect) int {
		return cornerFillAt(tt.Image(), int(b.X+b.W)-span, int(b.Y), span)
	}

	cut := groupAccess(t, action, "Cut")
	copyBtn := groupAccess(t, action, "Copy")
	paste := groupAccess(t, action, "Paste")
	// The outer ends keep the full radius, the seams the small inner one.
	if l, r := leftOf(action, cut.Bounds), rightOf(action, cut.Bounds); l >= r-10 {
		t.Fatalf("first button corners: outer left fill %d not clearly below the seam right %d", l, r)
	}
	if l, r := leftOf(action, paste.Bounds), rightOf(action, paste.Bounds); r >= l-10 {
		t.Fatalf("last button corners: outer right fill %d not clearly below the seam left %d", r, l)
	}
	if l, r := leftOf(action, copyBtn.Bounds), rightOf(action, copyBtn.Bounds); l < span*span*3/4 || r < span*span*3/4 {
		t.Fatalf("middle button seams are not small radii: left %d right %d, want nearly filled corners", l, r)
	}
	seamSmall := leftOf(action, copyBtn.Bounds)

	// A chosen middle button morphs from the seam radius to a full pill.
	sel := 1
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 160).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ButtonGroup(c, &sel, groupItems, ButtonGroupOptions{
					Kind: ButtonGroupConnected,
					Size: tokens.SizeXS,
				})
			})
		})
	}, 560, 160)
	chosen := groupAccess(t, tt, "Copy")
	if !chosen.Checked {
		t.Fatal("assistive technology does not see the chosen button checked")
	}
	if got := leftOf(tt, chosen.Bounds); got >= seamSmall-10 {
		t.Fatalf("chosen middle button seam fill %d did not shrink from the %d seam toward a full pill",
			got, seamSmall)
	}

	// The choice moves, and only the chosen button reports itself checked.
	cutRect := groupAccess(t, tt, "Cut").Bounds
	tt.ClickAt(cutRect.X+cutRect.W/2, cutRect.Y+cutRect.H/2)
	if sel != 0 {
		t.Fatalf("click chose %d, want 0", sel)
	}
	if !groupAccess(t, tt, "Cut").Checked || groupAccess(t, tt, "Copy").Checked {
		t.Fatal("assistive technology did not follow the new choice")
	}
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("chosen pill morph did not settle")
	}
}

// TestButtonGroupMultiAndAction checks the multi-select toggles and the action
// group's callbacks.
func TestButtonGroupMultiAndAction(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	multi := make([]bool, 0, 3)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 160).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ButtonGroupMulti(c, &multi, groupItems, ButtonGroupOptions{Size: tokens.SizeS})
			})
		})
	}, 560, 160)
	if len(multi) != len(groupItems) {
		t.Fatalf("multi selection grew to %d, want %d", len(multi), len(groupItems))
	}
	click := func(label string) {
		info := groupAccess(t, tt, label)
		tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	}
	click("Cut")
	click("Paste")
	if !multi[0] || multi[1] || !multi[2] {
		t.Fatalf("multi selection = %v, want [true false true]", multi)
	}
	if !groupAccess(t, tt, "Cut").Checked {
		t.Fatal("checked Cut not reported")
	}
	click("Cut")
	if multi[0] {
		t.Fatal("clicking a chosen button did not clear it")
	}

	actions := make([]int, 0, 3)
	tt2 := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 160).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ButtonGroupActions(c, groupItems, func(i int) { actions = append(actions, i) },
					ButtonGroupOptions{Size: tokens.SizeS})
			})
		})
	}, 560, 160)
	for _, label := range []string{"Cut", "Paste"} {
		info := groupAccess(t, tt2, label)
		tt2.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	}
	if len(actions) != 2 || actions[0] != 0 || actions[1] != 2 {
		t.Fatalf("action callbacks = %v, want [0 2]", actions)
	}
}

// TestButtonGroupKeyboardAndRTL checks that the arrow keys move the focus among
// the buttons, that Enter or Space chooses, and that the row mirrors right to
// left.
func TestButtonGroupKeyboardAndRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	build := func(sel *int) *ui.Tester {
		return ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Box(c).Size(560, 160).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
					ButtonGroup(c, sel, groupItems, ButtonGroupOptions{
						Kind: ButtonGroupConnected,
						Size: tokens.SizeS,
					})
				})
			})
		}, 560, 160)
	}

	sel := 0
	tt := build(&sel)
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Cut") {
		t.Fatal("Tab did not focus the first button")
	}
	tt.Key(0, ui.KeyRight)
	if !tt.Focused("Copy") {
		t.Fatal("Right did not move the focus to the second button")
	}
	tt.Key(0, ui.KeySpace)
	if sel != 1 {
		t.Fatalf("Space chose %d, want 1", sel)
	}

	none := -1
	rtl := build(&none)
	rtl.SetDirection(ui.RightToLeft)
	cut := groupAccess(t, rtl, "Cut")
	copyBtn := groupAccess(t, rtl, "Copy")
	if cut.Bounds.X <= copyBtn.Bounds.X {
		t.Fatalf("right to left row did not mirror: Cut x=%v Copy x=%v", cut.Bounds.X, copyBtn.Bounds.X)
	}
	rtl.Key(0, ui.KeyTab)
	if !rtl.Focused("Cut") {
		t.Fatal("RTL Tab did not focus the first button")
	}
	// Right to left, Left moves to the next button, as the reading direction
	// does, and Right to the previous one.
	rtl.Key(0, ui.KeyLeft)
	if !rtl.Focused("Copy") {
		t.Fatal("RTL Left did not move the focus along the row")
	}
	// The connected corners mirror: in an action group laid out right to left,
	// Cut is at the right, so its outer end is its right side and the seam its
	// left.
	rtlAction := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 160).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ButtonGroupActions(c, groupItems, nil, ButtonGroupOptions{
					Kind: ButtonGroupConnected,
					Size: tokens.SizeS,
				})
			})
		})
	}, 560, 160)
	rtlAction.SetDirection(ui.RightToLeft)
	rtlAction.Advance(700 * time.Millisecond)
	span := int(tokens.SizeS.Spec().SquareRadius)
	cutRTL := groupAccess(t, rtlAction, "Cut")
	l := cornerFillAt(rtlAction.Image(), int(cutRTL.Bounds.X), int(cutRTL.Bounds.Y), span)
	r := cornerFillAt(rtlAction.Image(), int(cutRTL.Bounds.X+cutRTL.Bounds.W)-span, int(cutRTL.Bounds.Y), span)
	if r >= l-10 {
		t.Fatalf("RTL first button: outer right fill %d not clearly below the seam left %d", r, l)
	}
}

// TestButtonGroupDisabled checks that a disabled group takes no clicks and
// reports every button disabled.
func TestButtonGroupDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	clicks := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(560, 160).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ButtonGroupActions(c, groupItems, func(int) { clicks++ }, ButtonGroupOptions{
					Size:     tokens.SizeS,
					Disabled: true,
				})
			})
		})
	}, 560, 160)

	info, ok := tt.AccessByRole(ui.RoleButton, "Cut")
	if !ok {
		t.Fatal("disabled group button not in the access tree")
	}
	if !info.Disabled {
		t.Fatal("disabled group button does not report itself disabled")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	tt.Advance(600 * time.Millisecond)
	if clicks != 0 {
		t.Fatalf("disabled group handled %d clicks", clicks)
	}
	if tt.FramePending() {
		t.Fatal("disabled group left frames pending")
	}
}
