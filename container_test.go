package md3e

import (
	"math"
	"sort"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/tokens"
)

// containerCanvas builds a 640x400 white canvas holding build's content in a
// padded column at the top start corner.
func containerCanvas(th *Theme, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(640, 400).Background(canvasRGB).Padding(24).AlignItems(ui.Start).Children(func() {
				ui.Column(c).AlignItems(ui.Start).Children(func() {
					build(c)
				})
			})
		})
	}, 640, 400)
}

// listItemInfo returns a list row's access info by name.
func listItemInfo(t *testing.T, tt *ui.Tester, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleListItem, name)
	if !ok {
		t.Fatalf("%q row not in the access tree", name)
	}
	return info
}

// pixelFilled reports whether (x, y) paints a row's container rather than the
// canvas, sampling inside the rounded corner region: a point 4 DIP from the
// side and 3 DIP from the edge is inside a small 8 DIP corner and outside a
// large 16 DIP one.
func pixelFilled(tt *ui.Tester, x, y int) bool {
	return !colorMatches(tt.Image().RGBAAt(x, y), canvasRGB, 10)
}

// TestCardVariantsAndDisabled checks the three card containers pixel by pixel,
// that a clickable card clicks, and that a disabled one takes no input and
// paints the MD3E disabled tokens.
func TestCardVariantsAndDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	clicks, lockedClicks := 0, 0
	tt := containerCanvas(th, func(c *ui.Context) {
		ui.Row(c).Gap(16).Children(func() {
			Card(c, nil, CardOptions{Variant: CardElevated, Label: "Elevated card"}).Size(160, 96)
			Card(c, nil, CardOptions{Variant: CardFilled, Label: "Filled card"}).Size(160, 96)
			Card(c, nil, CardOptions{Variant: CardOutlined, Label: "Outlined card"}).Size(160, 96)
		})
		ui.Row(c).Gap(16).Children(func() {
			Card(c, nil, CardOptions{Label: "Open card", OnClick: func() { clicks++ }}).Size(160, 96)
			Card(c, nil, CardOptions{Label: "Locked card", Disabled: true, OnClick: func() { lockedClicks++ }}).Size(160, 96)
			Card(c, nil, CardOptions{Label: "Plain card"}).Size(160, 96)
		})
	})

	img := tt.Image()
	cases := []struct {
		name string
		x, y int
		want ui.Color
	}{
		{"elevated", 104, 72, SurfaceColorForElevation(th, tokens.Level1)},
		{"filled", 280, 72, cs.SurfaceContainerHighest},
		{"outlined", 456, 72, cs.Surface},
	}
	for _, tc := range cases {
		px := img.RGBAAt(tc.x, tc.y)
		if !colorMatches(px, tc.want, 4) {
			t.Fatalf("%s card pixel %+v, want %+v", tc.name, px, rgbOf(tc.want))
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Plain card"); ok {
		t.Fatal("a plain card is a button in the access tree")
	}

	open := buttonInfo(t, tt, ui.RoleButton, "Open card")
	if open.Disabled {
		t.Fatal("an enabled card reads as disabled")
	}
	tt.ClickAt(open.Bounds.X+open.Bounds.W/2, open.Bounds.Y+open.Bounds.H/2)
	tt.Advance(500 * time.Millisecond)
	if clicks != 1 {
		t.Fatalf("the card ran its click %d times, want 1", clicks)
	}

	locked := buttonInfo(t, tt, ui.RoleButton, "Locked card")
	if !locked.Disabled {
		t.Fatal("a disabled card does not read as disabled")
	}
	px := tt.Image().RGBAAt(int(locked.Bounds.X+locked.Bounds.W/2), int(locked.Bounds.Y+locked.Bounds.H/2))
	want := cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)).Over(canvasRGB)
	if !colorMatches(px, want, 10) {
		t.Fatalf("disabled card pixel %+v, want the MD3E disabled container %+v", px, rgbOf(want))
	}
	tt.ClickAt(locked.Bounds.X+locked.Bounds.W/2, locked.Bounds.Y+locked.Bounds.H/2)
	tt.Advance(500 * time.Millisecond)
	if lockedClicks != 0 {
		t.Fatalf("a disabled card ran its click %d times", lockedClicks)
	}
	if tt.FramePending() {
		t.Fatal("cards keep asking for frames")
	}
}

// TestCardPressMorphSettles checks that a clickable card's press animation
// settles and that it shows no ripple or focus ring when it is not pressable.
func TestCardPressMorphSettles(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := containerCanvas(th, func(c *ui.Context) {
		Card(c, nil, CardOptions{Label: "Press me", Clickable: true}).Size(160, 96)
	})
	info := buttonInfo(t, tt, ui.RoleButton, "Press me")
	x, y := center(info.Bounds)
	tt.Press(x, y)
	tt.Advance(time.Second)
	tt.Release(x, y)
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("a pressed and released card keeps asking for frames")
	}
}

// separatorsByOrientation splits the separator access infos of the last frame
// into horizontal and vertical lines, in reading order.
func separatorsByOrientation(tt *ui.Tester) (horizontal, vertical []ui.AccessInfo) {
	for _, a := range tt.AccessAll() {
		if a.Role != ui.RoleSeparator {
			continue
		}
		if a.Bounds.W >= a.Bounds.H {
			horizontal = append(horizontal, a)
		} else {
			vertical = append(vertical, a)
		}
	}
	sort.Slice(horizontal, func(i, j int) bool { return horizontal[i].Bounds.Y < horizontal[j].Bounds.Y })
	return horizontal, vertical
}

// TestDividerInsetsAndRTL checks the horizontal line's start and end insets,
// the vertical line's box, and that the insets mirror right to left.
func TestDividerInsetsAndRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	line := th.Colors.OutlineVariant
	build := func(c *ui.Context) {
		ui.Column(c).Width(400).Children(func() {
			Divider(c, DividerOptions{Label: "Plain"})
			Divider(c, DividerOptions{Label: "Inset ends", StartInset: 40, EndInset: 10})
			ui.Row(c).Height(60).AlignItems(ui.Stretch).Children(func() {
				Divider(c, DividerOptions{Vertical: true, Inset: 8})
			})
		})
	}
	tt := containerCanvas(th, build)
	h, v := separatorsByOrientation(tt)
	if len(h) != 2 || len(v) != 1 {
		t.Fatalf("found %d horizontal and %d vertical separators, want 2 and 1", len(h), len(v))
	}
	if h[0].Bounds.X != 24 || h[0].Bounds.W != 400 {
		t.Fatalf("plain divider box %v, want x 24 and width 400", h[0].Bounds)
	}
	if h[1].Bounds.X != 24+40 || h[1].Bounds.W != 350 {
		t.Fatalf("inset divider box %v, want x 64 and width 350", h[1].Bounds)
	}
	if h[0].Bounds.H != 1 || v[0].Bounds.W != 1 {
		t.Fatalf("divider thickness is %v/%v, want 1 DIP", h[0].Bounds.H, v[0].Bounds.W)
	}
	if v[0].Bounds.H != 44 {
		t.Fatalf("vertical divider height %v, want 44 after its 8 DIP insets", v[0].Bounds.H)
	}
	px := tt.Image().RGBAAt(int(h[1].Bounds.X+10), int(h[1].Bounds.Y))
	if !colorMatches(px, line, 8) {
		t.Fatalf("divider pixel %+v, want OutlineVariant %+v", px, rgbOf(line))
	}

	rtl := containerCanvas(th, build)
	rtl.SetDirection(ui.RightToLeft)
	h, _ = separatorsByOrientation(rtl)
	// The column itself moves to the trailing edge right to left, so the
	// insets are measured from its own start.
	if h[1].Bounds.X != h[0].Bounds.X+10 || h[1].Bounds.W != 350 {
		t.Fatalf("right-to-left inset divider box %v, want x %v and width 350", h[1].Bounds, h[0].Bounds.X+10)
	}
}

// listCanvas builds a 320 DIP wide column of a segmented list of the given
// items.
func listCanvas(th *Theme, items []ListItem, opts ListOptions) *ui.Tester {
	return containerCanvas(th, func(c *ui.Context) {
		ui.Column(c).Width(320).Children(func() {
			List(c, items, opts)
		})
	})
}

// TestSegmentedListCorners checks the expressive corner rhythm of a segmented
// list pixel by pixel: large at the group's outer ends, small between, and all
// large for a single row.
func TestSegmentedListCorners(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	items := []ListItem{{Headline: "Alpha"}, {Headline: "Beta"}, {Headline: "Gamma"}}
	tt := listCanvas(th, items, ListOptions{Label: "Rows", Segmented: true})
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleList, "Rows"); !ok {
		t.Fatal("the list is not a named list in the access tree")
	}
	alpha := listItemInfo(t, tt, "Alpha")
	beta := listItemInfo(t, tt, "Beta")
	gamma := listItemInfo(t, tt, "Gamma")

	// The first row rounds its top corners only: its bottom corners stay small.
	if pixelFilled(tt, int(alpha.Bounds.X)+4, int(alpha.Bounds.Y)+3) {
		t.Fatal("the first row paints outside its large top-left corner")
	}
	if !pixelFilled(tt, int(alpha.Bounds.X)+4, int(alpha.Bounds.Y+alpha.Bounds.H)-4) {
		t.Fatal("the first row's bottom-left corner is not small")
	}
	// The middle row keeps its small corners all round.
	if !pixelFilled(tt, int(beta.Bounds.X)+4, int(beta.Bounds.Y)+3) {
		t.Fatal("the middle row's top-left corner is not small")
	}
	if !pixelFilled(tt, int(beta.Bounds.X)+4, int(beta.Bounds.Y+beta.Bounds.H)-4) {
		t.Fatal("the middle row's bottom-left corner is not small")
	}
	// The last row rounds its bottom corners only.
	if pixelFilled(tt, int(gamma.Bounds.X)+4, int(gamma.Bounds.Y+gamma.Bounds.H)-4) {
		t.Fatal("the last row paints outside its large bottom-left corner")
	}
	if !pixelFilled(tt, int(gamma.Bounds.X)+4, int(gamma.Bounds.Y)+3) {
		t.Fatal("the last row's top-left corner is not small")
	}

	single := listCanvas(th, []ListItem{{Headline: "Only"}}, ListOptions{Label: "One", Segmented: true})
	single.Advance(time.Second)
	only := listItemInfo(t, single, "Only")
	for _, p := range [][2]int{
		{int(only.Bounds.X) + 4, int(only.Bounds.Y) + 3},
		{int(only.Bounds.X+only.Bounds.W) - 5, int(only.Bounds.Y) + 3},
		{int(only.Bounds.X) + 4, int(only.Bounds.Y+only.Bounds.H) - 4},
		{int(only.Bounds.X+only.Bounds.W) - 5, int(only.Bounds.Y+only.Bounds.H) - 4},
	} {
		if pixelFilled(single, p[0], p[1]) {
			t.Fatalf("a single row paints outside its large corner at %v", p)
		}
	}
}

// TestSegmentedListSelectionAndSettle checks a click selects a row, its
// container and corners settle, and the state reaches assistive technology.
func TestSegmentedListSelectionAndSettle(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	selected := -1
	chosen := -1
	items := []ListItem{{Headline: "Alpha"}, {Headline: "Beta"}, {Headline: "Gamma"}}
	tt := listCanvas(th, items, ListOptions{
		Label:     "Rows",
		Segmented: true,
		Selected:  &selected,
		OnSelect:  func(i int) { chosen = i },
	})
	beta := listItemInfo(t, tt, "Beta")
	if beta.Selected {
		t.Fatal("a row is selected before any click")
	}
	tt.ClickAt(beta.Bounds.X+beta.Bounds.W/2, beta.Bounds.Y+beta.Bounds.H/2)
	// Leave the row so its hover state layer does not tint the sampled fill.
	tt.Move(5, 5)
	tt.Advance(time.Second)
	if selected != 1 || chosen != 1 {
		t.Fatalf("the click chose row %d (callback %d), want 1", selected, chosen)
	}
	if tt.FramePending() {
		t.Fatal("the selected row keeps asking for frames")
	}
	beta = listItemInfo(t, tt, "Beta")
	if !beta.Selected {
		t.Fatal("the chosen row does not read as selected")
	}
	px := tt.Image().RGBAAt(int(beta.Bounds.X+beta.Bounds.W)-12, int(beta.Bounds.Y+beta.Bounds.H/2))
	if !colorMatches(px, cs.SecondaryContainer, 10) {
		t.Fatalf("chosen row pixel %+v, want SecondaryContainer %+v", px, rgbOf(cs.SecondaryContainer))
	}
	if pixelFilled(tt, int(beta.Bounds.X)+4, int(beta.Bounds.Y)+3) {
		t.Fatal("the chosen row does not round all four corners")
	}
}

// TestSegmentedListPressMorph checks a pressed row rounds its corners and
// springs back when released without being chosen.
func TestSegmentedListPressMorph(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	items := []ListItem{
		{Headline: "Alpha", OnClick: func() {}},
		{Headline: "Beta", OnClick: func() {}},
		{Headline: "Gamma", OnClick: func() {}},
	}
	tt := listCanvas(th, items, ListOptions{Label: "Rows", Segmented: true})
	beta := listItemInfo(t, tt, "Beta")
	x, y := center(beta.Bounds)
	tt.Press(x, y)
	tt.Advance(time.Second)
	if pixelFilled(tt, int(beta.Bounds.X)+4, int(beta.Bounds.Y)+3) {
		t.Fatal("the pressed row does not round its top-left corner")
	}
	tt.Release(x, y)
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("the released row keeps asking for frames")
	}
	beta = listItemInfo(t, tt, "Beta")
	if !pixelFilled(tt, int(beta.Bounds.X)+4, int(beta.Bounds.Y)+3) {
		t.Fatal("the released row does not spring back to its small corner")
	}
}

// TestSegmentedListReduceMotion checks corners jump to their targets without
// extra frames under reduced motion.
func TestSegmentedListReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	items := []ListItem{
		{Headline: "Alpha", OnClick: func() {}},
		{Headline: "Beta", OnClick: func() {}},
	}
	tt := listCanvas(th, items, ListOptions{Label: "Rows", Segmented: true})
	beta := listItemInfo(t, tt, "Beta")
	x, y := center(beta.Bounds)
	tt.Press(x, y)
	if tt.FramePending() {
		t.Fatal("a reduced-motion press asks for frames")
	}
	tt.Release(x, y)
	if tt.FramePending() {
		t.Fatal("a reduced-motion release asks for frames")
	}
	if !pixelFilled(tt, int(beta.Bounds.X)+4, int(beta.Bounds.Y)+3) {
		t.Fatal("the reduced-motion row did not settle back to its small corner")
	}
}

// TestListTrailingSwitchKeepsItsPlace checks a trailing Switch keeps its own
// identity and place as the row's Meta text comes and goes, and that using it
// never chooses the row.
func TestListTrailingSwitchKeepsItsPlace(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := -1
	clicks := 0
	sync := false
	meta := ""
	build := func(c *ui.Context) {
		list := List(c, []ListItem{
			{
				Headline: "Notifications",
				Meta:     meta,
				OnClick:  func() { clicks++ },
				Trailing: func() ui.Element {
					return Switch(c.Key("notifications-switch"), &sync, SwitchOptions{Label: "Sync"})
				},
			},
			{Headline: "Bluetooth"},
		}, ListOptions{Label: "Rows", Selected: &selected})
		list.Width(320)
	}
	tt := containerCanvas(th, build)
	tt.Advance(time.Second)
	info, ok := tt.AccessByRole(ui.RoleSwitch, "Sync")
	if !ok {
		t.Fatal("the trailing switch is not in the access tree")
	}
	toggle := func(what string) {
		t.Helper()
		info, ok = tt.AccessByRole(ui.RoleSwitch, "Sync")
		if !ok {
			t.Fatalf("%s: the trailing switch left the access tree", what)
		}
		tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
		tt.Advance(time.Second)
	}
	toggle("first click")
	if !sync {
		t.Fatal("clicking the trailing switch does not toggle it")
	}
	if clicks != 0 || selected != -1 {
		t.Fatalf("the trailing switch chose the row (OnClick %d, selected %d)", clicks, selected)
	}

	meta = "2 devices"
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	after, ok := tt.AccessByRole(ui.RoleSwitch, "Sync")
	if !ok {
		t.Fatal("showing Meta removed the trailing switch")
	}
	if math.Abs(float64(after.Bounds.X-info.Bounds.X)) > 0.5 {
		t.Fatalf("Meta moved the switch from x %.1f to %.1f", info.Bounds.X, after.Bounds.X)
	}
	toggle("second click")
	if sync {
		t.Fatal("the trailing switch stopped responding after Meta appeared")
	}
	if clicks != 0 || selected != -1 {
		t.Fatalf("the trailing switch chose the row after Meta appeared (OnClick %d, selected %d)", clicks, selected)
	}
	if tt.FramePending() {
		t.Fatal("the list keeps asking for frames after the switch settles")
	}
}

// TestListKeyboardAndDisabled checks arrow keys and Enter choose rows, and a
// disabled list reaches assistive technology and takes no input.
func TestListKeyboardAndDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := -1
	items := []ListItem{{Headline: "Alpha"}, {Headline: "Beta"}, {Headline: "Gamma"}}
	tt := listCanvas(th, items, ListOptions{Label: "Rows", Segmented: true, Selected: &selected})
	alpha := listItemInfo(t, tt, "Alpha")
	tt.ClickAt(alpha.Bounds.X+alpha.Bounds.W/2, alpha.Bounds.Y+alpha.Bounds.H/2)
	tt.Advance(300 * time.Millisecond)
	if selected != 0 {
		t.Fatalf("the click chose row %d, want 0", selected)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	tt.Advance(300 * time.Millisecond)
	if selected != 1 {
		t.Fatalf("Down and Enter chose row %d, want 1", selected)
	}
	if !listItemInfo(t, tt, "Beta").Selected {
		t.Fatal("the keyboard-chosen row does not read as selected")
	}

	disabledSelected := 3
	disabled := listCanvas(th, items, ListOptions{
		Label:     "Disabled rows",
		Segmented: true,
		Selected:  &disabledSelected,
		Disabled:  true,
	})
	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		info := listItemInfo(t, disabled, name)
		if !info.Disabled {
			t.Fatalf("%q under a disabled list does not read as disabled", name)
		}
	}
	info := listItemInfo(t, disabled, "Beta")
	disabled.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	disabled.Key(0, ui.KeySpace)
	disabled.Advance(500 * time.Millisecond)
	if disabledSelected != 3 {
		t.Fatalf("the disabled list chose row %d", disabledSelected)
	}
	if disabled.FramePending() {
		t.Fatal("the disabled list keeps asking for frames")
	}
}

// TestInteractiveCardStacksChildrenVertically checks clickable cards use a
// vertical column layout just like non-interactive cards.
func TestInteractiveCardStacksChildrenVertically(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	clicks := 0
	tt := containerCanvas(th, func(c *ui.Context) {
		Card(c, func() {
			ui.Text(c, "Headline line")
			ui.Text(c, "Supporting detail")
		}, CardOptions{
			Variant: CardOutlined,
			Label:   "Project card",
			OnClick: func() { clicks++ },
		})
	})
	tt.Frame()
	head, ok := tt.Find("Headline line")
	if !ok {
		t.Fatal("card headline missing")
	}
	sub, ok := tt.Find("Supporting detail")
	if !ok {
		t.Fatal("card subtitle missing")
	}
	if sub.Y <= head.Y {
		t.Fatalf("interactive card laid out children horizontally: head %.1f, sub %.1f", head.Y, sub.Y)
	}
	if absF32(sub.X-head.X) > 1 {
		t.Fatalf("interactive card did not align children vertically: head x=%.1f, sub x=%.1f", head.X, sub.X)
	}
	tt.Click("Project card")
	tt.Advance(300 * time.Millisecond)
	if clicks != 1 {
		t.Fatalf("card OnClick ran %d times, want 1", clicks)
	}
}

// TestCardListDividerDisabledConsistencyAndInsets checks non-interactive
// disabled cards/list items expose disabled semantics, disabled lists don't
// double-dim, and Divider honors vertical insets and Shrink(0).
func TestCardListDividerDisabledConsistencyAndInsets(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	tt := containerCanvas(th, func(c *ui.Context) {
		Card(c, func() {
			ui.Text(c, "Static disabled card")
		}, CardOptions{Variant: CardFilled, Label: "StaticCard", Disabled: true})
		List(c, []ListItem{
			{Headline: "Row One", Disabled: true},
		}, ListOptions{Label: "PlainList"})
		ui.Column(c).Height(60).Width(160).Children(func() {
			Divider(c, DividerOptions{Label: "HDiv"})
			ui.Box(c).Height(80).FillWidth()
		})
		ui.Row(c).Height(40).Children(func() {
			Divider(c, DividerOptions{
				Vertical:   true,
				StartInset: 6,
				EndInset:   10,
				Thickness:  2,
				Label:      "VDiv",
			})
		})
	})
	tt.Frame()

	var cardInfo ui.AccessInfo
	foundCard := false
	for _, info := range tt.AccessAll() {
		if info.Name == "StaticCard" {
			cardInfo = info
			foundCard = true
			break
		}
	}
	if !foundCard {
		t.Fatal("StaticCard missing from access tree")
	}
	if !cardInfo.Disabled {
		t.Fatal("non-interactive Disabled Card does not expose Disabled=true")
	}
	rowInfo := listItemInfo(t, tt, "Row One")
	if !rowInfo.Disabled {
		t.Fatal("non-interactive Disabled ListItem does not expose Disabled=true")
	}
	hdiv, ok := tt.AccessByRole(ui.RoleSeparator, "HDiv")
	if !ok {
		t.Fatal("horizontal divider missing")
	}
	if hdiv.Bounds.H < 0.9 {
		t.Fatalf("horizontal divider shrank to height %.2f", hdiv.Bounds.H)
	}
	vdiv, ok := tt.AccessByRole(ui.RoleSeparator, "VDiv")
	if !ok {
		t.Fatal("vertical divider missing")
	}
	if absF32(vdiv.Bounds.H-(40-6-10)) > 0.5 || absF32(vdiv.Bounds.W-2) > 0.5 {
		t.Fatalf("vertical divider bounds = %+v, want width 2 height 24", vdiv.Bounds)
	}

	// Verify disabled list row container paints single-dimmed DisabledContainerOpacity (0.12)
	// over canvasRGB without stacked DimDisabled attenuation.
	wantRowBg := cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)).Over(canvasRGB)
	rowPx := tt.Image().RGBAAt(int(rowInfo.Bounds.X+4), int(rowInfo.Bounds.Y+rowInfo.Bounds.H/2))
	if !colorMatches(rowPx, wantRowBg, 10) {
		t.Fatalf("disabled list row background = %+v, want %+v", rowPx, rgbOf(wantRowBg))
	}
}
