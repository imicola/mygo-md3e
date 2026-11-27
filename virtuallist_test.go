package md3e

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

func TestVirtualList100KVirtualization(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tokens.DensityDesktop})
	state := &ui.ListState{}
	builtInFrame := 0
	builtIndices := map[int]bool{}

	tt := ui.NewTester(func(c *ui.Context) {
		builtInFrame = 0
		clear(builtIndices)
		ThemeScope(c, th, func() {
			ui.Column(c).Size(400, 320).Background(canvasRGB).Children(func() {
				VirtualList(c, 100_000, func(r ui.ListRow) {
					builtInFrame++
					builtIndices[r.Index] = true
					Text(r.Context, fmt.Sprintf("Row %d", r.Index), tokens.BodyMedium)
				}, VirtualListOptions{
					State:     state,
					RowHeight: 40,
					Label:     "Large list",
				}).Grow(1).FillWidth()
			})
		})
	}, 400, 320)

	if builtInFrame == 0 || builtInFrame >= 40 {
		t.Fatalf("initial 100K frame built %d rows, want >0 and <40 viewport rows", builtInFrame)
	}
	if builtIndices[50_000] {
		t.Fatal("initial frame unexpectedly built off-screen row 50000")
	}
	if tt.FramePending() {
		t.Fatal("settled virtual list left pending animation frames")
	}

	state.ScrollTo(50_000, ui.Start)
	tt.Frame()
	if builtInFrame == 0 || builtInFrame >= 40 {
		t.Fatalf("scrolled 100K frame built %d rows, want >0 and <40 viewport rows", builtInFrame)
	}
	if !builtIndices[50_000] {
		t.Fatal("scrolling to row 50000 did not build row 50000")
	}
	if !tt.HasText("Row 50000") {
		t.Fatal("row 50000 text not visible after ScrollTo")
	}
	if tt.FramePending() {
		t.Fatal("scrolled virtual list left pending animation frames")
	}
}

func TestVirtualListHoverNoTreeRebuildAndRippleSettles(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := -1
	builds := 0

	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, th, func() {
			ui.Box(c).Size(360, 260).Background(canvasRGB).Padding(12).Children(func() {
				VirtualList(c, 10, func(r ui.ListRow) {
					Text(r.Context, fmt.Sprintf("Item %d", r.Index), tokens.BodyMedium)
				}, VirtualListOptions{
					Selected:  &selected,
					RowHeight: 44,
					RowLabel:  func(row int) string { return fmt.Sprintf("Item %d", row) },
					Label:     "Hover list",
				}).Grow(1).FillWidth()
			})
		})
	}, 360, 260)

	tt.Frame()
	item1 := listItemInfo(t, tt, "Item 1")
	item3 := listItemInfo(t, tt, "Item 3")
	cx, cy := center(item1.Bounds)
	ux, uy := int(item3.Bounds.X+item3.Bounds.W-24), int(item3.Bounds.Y+item3.Bounds.H/2)
	hx, hy := int(item1.Bounds.X+item1.Bounds.W-24), int(item1.Bounds.Y+item1.Bounds.H/2)

	// Hover fade-in runs through Painter.After repaints without view rebuilds.
	tt.Move(cx, cy)
	afterEnter := builds
	tt.Advance(250 * time.Millisecond)
	if builds != afterEnter {
		t.Fatalf("hover fade-in caused %d unexpected view rebuild(s)", builds-afterEnter)
	}
	if tt.FramePending() {
		t.Fatal("hover fade-in did not settle")
	}
	hoveredPx := tt.Image().RGBAAt(hx, hy)
	unhoveredPx := tt.Image().RGBAAt(ux, uy)
	if colorMatches(hoveredPx, ui.RGBA(unhoveredPx.R, unhoveredPx.G, unhoveredPx.B, 1), 4) {
		t.Fatalf("hovered row pixel %+v matches unhovered %+v; state layer not painted", hoveredPx, unhoveredPx)
	}

	// Moving off fades out cleanly without view rebuilds.
	tt.Move(2, 2)
	afterLeave := builds
	tt.Advance(300 * time.Millisecond)
	if builds != afterLeave {
		t.Fatalf("hover fade-out caused %d unexpected view rebuild(s)", builds-afterLeave)
	}
	if tt.FramePending() {
		t.Fatal("hover fade-out did not settle")
	}

	// Press + release drives ripple via DrawOver and settles with 0 rebuilds during Advance.
	tt.Press(cx, cy)
	afterPress := builds
	tt.Advance(140 * time.Millisecond)
	if builds != afterPress {
		t.Fatalf("ripple press expansion caused %d unexpected view rebuild(s)", builds-afterPress)
	}
	pressedPx := tt.Image().RGBAAt(hx, hy)
	if colorMatches(pressedPx, canvasRGB, 6) {
		t.Fatalf("pressed ripple pixel %+v stayed at canvas background", pressedPx)
	}

	tt.Release(cx, cy)
	tt.Move(2, 2)
	afterRelease := builds
	tt.Advance(time.Second)
	if builds != afterRelease {
		t.Fatalf("ripple fade-out caused %d unexpected view rebuild(s)", builds-afterRelease)
	}
	if tt.FramePending() {
		t.Fatal("virtual list ripple did not settle after release")
	}
}

func TestVirtualListSelectionKeyboardAndSubmit(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := -1
	var clickedRows []int
	var clickedSelected []int
	var submittedRows []int

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(360, 260).Background(canvasRGB).Padding(8).Children(func() {
				VirtualList(c, 10, func(r ui.ListRow) {
					Text(r.Context, fmt.Sprintf("Row %d", r.Index), tokens.BodyMedium)
				}, VirtualListOptions{
					Selected:  &selected,
					RowHeight: 40,
					RowLabel:  func(row int) string { return fmt.Sprintf("Row %d", row) },
					OnClick: func(row int) {
						clickedRows = append(clickedRows, row)
						clickedSelected = append(clickedSelected, selected)
					},
					OnSubmit: func(row int) {
						submittedRows = append(submittedRows, row)
					},
					Label: "Selectable list",
				}).Grow(1).FillWidth()
			})
		})
	}, 360, 260)

	if err := tt.Click("Row 1"); err != nil {
		t.Fatalf("click Row 1: %v", err)
	}
	if selected != 1 || len(clickedRows) != 1 || clickedRows[0] != 1 || clickedSelected[0] != 1 {
		t.Fatalf("after click: selected=%d clickedRows=%v clickedSelected=%v", selected, clickedRows, clickedSelected)
	}

	tt.Key(0, ui.KeyDown)
	if selected != 2 {
		t.Fatalf("KeyDown selected=%d, want 2", selected)
	}
	tt.Key(0, ui.KeyUp)
	if selected != 1 {
		t.Fatalf("KeyUp selected=%d, want 1", selected)
	}
	tt.Key(0, ui.KeyEnd)
	if selected != 9 {
		t.Fatalf("KeyEnd selected=%d, want 9", selected)
	}
	tt.Key(0, ui.KeyHome)
	if selected != 0 {
		t.Fatalf("KeyHome selected=%d, want 0", selected)
	}
	tt.Key(0, ui.KeyEnter)
	if len(submittedRows) != 1 || submittedRows[0] != 0 {
		t.Fatalf("KeyEnter submittedRows=%v, want [0]", submittedRows)
	}

	// Double-click triggers OnSubmit without needing Selected/Selection too.
	tt.Advance(time.Second)
	row3 := listItemInfo(t, tt, "Row 3")
	rx, ry := center(row3.Bounds)
	tt.ClickAt(rx, ry)
	tt.ClickAt(rx, ry)
	tt.Advance(time.Second)
	if selected != 3 || len(submittedRows) != 2 || submittedRows[1] != 3 {
		t.Fatalf("double-click: selected=%d submittedRows=%v, want 3 and [0 3]", selected, submittedRows)
	}

	// Verify OnSubmit works even when Selected and Selection are both nil.
	var statelessSubmitted []int
	tt2 := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(320, 200).Background(canvasRGB).Children(func() {
				VirtualList(c, 5, func(r ui.ListRow) {
					Text(r.Context, fmt.Sprintf("Action %d", r.Index), tokens.BodyMedium)
				}, VirtualListOptions{
					RowHeight: 36,
					RowLabel:  func(row int) string { return fmt.Sprintf("Action %d", row) },
					OnSubmit:  func(row int) { statelessSubmitted = append(statelessSubmitted, row) },
					Label:     "Stateless submit list",
				}).Grow(1).FillWidth()
			})
		})
	}, 320, 200)
	act2 := listItemInfo(t, tt2, "Action 2")
	ax, ay := center(act2.Bounds)
	tt2.ClickAt(ax, ay)
	tt2.ClickAt(ax, ay)
	if len(statelessSubmitted) != 1 || statelessSubmitted[0] != 2 {
		t.Fatalf("stateless double-click submitted=%v, want [2]", statelessSubmitted)
	}
}

func TestVirtualListMultiSelectionAndItemKey(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	items := []string{"alpha", "beta", "gamma", "delta"}
	var chosen ui.Selection[string]

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(360, 260).Background(canvasRGB).Children(func() {
				VirtualList(c, len(items), func(r ui.ListRow) {
					Text(r.Context, items[r.Index], tokens.BodyMedium)
				}, VirtualListOptions{
					ItemKey:   func(row int) any { return items[row] },
					Selection: &chosen,
					RowHeight: 40,
					RowLabel:  func(row int) string { return items[row] },
					Label:     "Multi list",
				}).Grow(1).FillWidth()
			})
		})
	}, 360, 260)

	if err := tt.Click("alpha"); err != nil {
		t.Fatalf("click alpha: %v", err)
	}
	if err := tt.ClickWith(ui.Shift, "gamma"); err != nil {
		t.Fatalf("Shift-click gamma: %v", err)
	}
	if !chosen.Has("alpha") || !chosen.Has("beta") || !chosen.Has("gamma") || chosen.Has("delta") {
		t.Fatalf("Shift-select gave alpha=%v beta=%v gamma=%v delta=%v",
			chosen.Has("alpha"), chosen.Has("beta"), chosen.Has("gamma"), chosen.Has("delta"))
	}

	if err := tt.ClickWith(ui.Cmd, "beta"); err != nil {
		t.Fatalf("Cmd-click beta: %v", err)
	}
	if !chosen.Has("alpha") || chosen.Has("beta") || !chosen.Has("gamma") {
		t.Fatalf("Cmd-toggle gave alpha=%v beta=%v gamma=%v",
			chosen.Has("alpha"), chosen.Has("beta"), chosen.Has("gamma"))
	}

	// Prepend an item and rebuild: key-backed Selection follows items across index shifts.
	items = []string{"zero", "alpha", "beta", "gamma", "delta"}
	tt.Frame()
	if !chosen.Has("alpha") || !chosen.Has("gamma") || chosen.Has("zero") {
		t.Fatalf("key-tracked selection drifted after prepend: zero=%v alpha=%v gamma=%v",
			chosen.Has("zero"), chosen.Has("alpha"), chosen.Has("gamma"))
	}
}

func TestVirtualListSegmentedAndStickyHeader(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	state := &ui.ListState{}
	selected := 2
	isHeader := func(row int) bool { return row == 0 || row == 5 }

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(400, 320).Background(canvasRGB).Padding(16).Children(func() {
				VirtualList(c, 12, func(r ui.ListRow) {
					if isHeader(r.Index) {
						Text(r.Context, fmt.Sprintf("Header %d", r.Index), tokens.LabelLarge)
						return
					}
					Text(r.Context, fmt.Sprintf("Entry %d", r.Index), tokens.BodyMedium)
				}, VirtualListOptions{
					State:     state,
					Selected:  &selected,
					IsHeader:  isHeader,
					Segmented: true,
					RowHeight: 40,
					RowLabel: func(row int) string {
						if isHeader(row) {
							return fmt.Sprintf("Header %d", row)
						}
						return fmt.Sprintf("Entry %d", row)
					},
					Label: "Segmented list",
				}).Grow(1).FillWidth()
			})
		})
	}, 400, 320)

	entry1 := listItemInfo(t, tt, "Entry 1")
	entry2 := listItemInfo(t, tt, "Entry 2")
	entry3 := listItemInfo(t, tt, "Entry 3")
	entry4 := listItemInfo(t, tt, "Entry 4")

	// A point 4 DIP from left and 3 DIP from top/bottom sits inside small (8 DIP)
	// corners and outside large (16 DIP) corners.
	if pixelFilled(tt, int(entry1.Bounds.X)+4, int(entry1.Bounds.Y)+3) {
		t.Fatal("first row in section did not round its top-left large corner")
	}
	if !pixelFilled(tt, int(entry1.Bounds.X)+4, int(entry1.Bounds.Y+entry1.Bounds.H)-3) {
		t.Fatal("first row in section did not keep its bottom-left small corner")
	}

	// Selected row 2 renders SecondaryContainer and rounds all four corners large.
	selPx := tt.Image().RGBAAt(int(entry2.Bounds.X+entry2.Bounds.W)-24, int(entry2.Bounds.Y+entry2.Bounds.H/2))
	if !colorMatches(selPx, cs.SecondaryContainer, 8) {
		t.Fatalf("selected row fill %+v, want SecondaryContainer %+v", selPx, rgbOf(cs.SecondaryContainer))
	}
	if pixelFilled(tt, int(entry2.Bounds.X)+4, int(entry2.Bounds.Y)+3) ||
		pixelFilled(tt, int(entry2.Bounds.X)+4, int(entry2.Bounds.Y+entry2.Bounds.H)-3) {
		t.Fatal("selected segmented row did not round all four large corners")
	}

	// Inner row 3 keeps small corners top and bottom.
	if !pixelFilled(tt, int(entry3.Bounds.X)+4, int(entry3.Bounds.Y)+3) ||
		!pixelFilled(tt, int(entry3.Bounds.X)+4, int(entry3.Bounds.Y+entry3.Bounds.H)-3) {
		t.Fatal("inner segmented row did not keep small corners")
	}

	// Last row before next header (row 4) rounds bottom corners large.
	if !pixelFilled(tt, int(entry4.Bounds.X)+4, int(entry4.Bounds.Y)+3) {
		t.Fatal("last row in section did not keep top-left small corner")
	}
	if pixelFilled(tt, int(entry4.Bounds.X)+4, int(entry4.Bounds.Y+entry4.Bounds.H)-3) {
		t.Fatal("last row in section did not round its bottom-left large corner")
	}

	// Scrolling past row 0 pins Header 0 at the top of the list viewport.
	listInfo, ok := tt.AccessByRole(ui.RoleList, "Segmented list")
	if !ok {
		t.Fatal("RoleList missing from access tree")
	}
	state.ScrollTo(3, ui.Start)
	tt.Frame()
	hdrRect, ok := tt.Find("Header 0")
	if !ok {
		t.Fatal("sticky Header 0 disappeared after scrolling past row 0")
	}
	if hdrRect.Y < listInfo.Bounds.Y || hdrRect.Y > listInfo.Bounds.Y+24 {
		t.Fatalf("sticky Header 0 Y=%v not pinned near list top Y=%v", hdrRect.Y, listInfo.Bounds.Y)
	}
}

func TestVirtualListRTLReduceMotionDisabledAndAccess(t *testing.T) {
	// 1. RTL scrollbar clearance and padding mirroring + Container/Hairline + non-finite guards.
	th := ThemeFromSeed(DefaultSeedColor)
	rtlClicks := 0
	ttRTL := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(360, 220).Background(canvasRGB).Padding(12).Children(func() {
				VirtualList(c, 30, func(r ui.ListRow) {
					Text(r.Context, fmt.Sprintf("Line %d", r.Index), tokens.BodyMedium)
				}, VirtualListOptions{
					Container: true,
					Hairline:  true,
					Padding:   -8,
					Gap:       float32(math.Inf(1)),
					RowHeight: float32(math.NaN()),
					RowLabel:  func(row int) string { return fmt.Sprintf("Line %d", row) },
					OnClick:   func(row int) { rtlClicks++ },
					Label:     "RTL list",
				}).Grow(1).FillWidth()
			})
		})
	}, 360, 220)

	listLTR, ok := ttRTL.AccessByRole(ui.RoleList, "RTL list")
	if !ok {
		t.Fatal("RTL list missing RoleList")
	}
	rowLTR := listItemInfo(t, ttRTL, "Line 0")
	ltrEndClearance := (listLTR.Bounds.X + listLTR.Bounds.W) - (rowLTR.Bounds.X + rowLTR.Bounds.W)
	if ltrEndClearance < virtualListScrollbarClearance-1 {
		t.Fatalf("LTR trailing clearance=%v, want >= %v", ltrEndClearance, virtualListScrollbarClearance)
	}

	ttRTL.SetDirection(ui.RightToLeft)
	listRTL, _ := ttRTL.AccessByRole(ui.RoleList, "RTL list")
	rowRTL := listItemInfo(t, ttRTL, "Line 0")
	rtlLeftClearance := rowRTL.Bounds.X - listRTL.Bounds.X
	if rtlLeftClearance < virtualListScrollbarClearance-1 {
		t.Fatalf("RTL trailing (left) scrollbar clearance=%v, want >= %v", rtlLeftClearance, virtualListScrollbarClearance)
	}

	// 2. ReduceMotion skips hover/press animations with zero pending frames.
	thRM := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	rmSel := -1
	ttRM := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, thRM, func() {
			ui.Column(c).Size(320, 200).Background(canvasRGB).Children(func() {
				VirtualList(c, 5, func(r ui.ListRow) {
					Text(r.Context, fmt.Sprintf("RM %d", r.Index), tokens.BodyMedium)
				}, VirtualListOptions{
					Selected:  &rmSel,
					RowHeight: 40,
					RowLabel:  func(row int) string { return fmt.Sprintf("RM %d", row) },
					Label:     "RM list",
				}).Grow(1).FillWidth()
			})
		})
	}, 320, 200)
	rm0 := listItemInfo(t, ttRM, "RM 0")
	rx, ry := center(rm0.Bounds)
	ttRM.Move(rx, ry)
	if ttRM.FramePending() {
		t.Fatal("ReduceMotion hover left FramePending true")
	}
	ttRM.Press(rx, ry)
	if ttRM.FramePending() {
		t.Fatal("ReduceMotion press left FramePending true")
	}
	ttRM.Release(rx, ry)
	if ttRM.FramePending() {
		t.Fatal("ReduceMotion release left FramePending true")
	}

	// 3. Disabled blocks clicks, keyboard navigation, OnClick, OnSubmit, and reports Disabled in access tree.
	disSel := -1
	disClicks, disSubmits := 0, 0
	ttDis := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(320, 200).Background(canvasRGB).Children(func() {
				VirtualList(c, 5, func(r ui.ListRow) {
					Text(r.Context, fmt.Sprintf("Locked %d", r.Index), tokens.BodyMedium)
				}, VirtualListOptions{
					Selected:  &disSel,
					RowHeight: 40,
					RowLabel:  func(row int) string { return fmt.Sprintf("Locked %d", row) },
					OnClick:   func(row int) { disClicks++ },
					OnSubmit:  func(row int) { disSubmits++ },
					Label:     "Disabled list",
					Disabled:  true,
				}).Grow(1).FillWidth()
			})
		})
	}, 320, 200)

	disList, ok := ttDis.AccessByRole(ui.RoleList, "Disabled list")
	if !ok || !disList.Disabled {
		t.Fatalf("disabled VirtualList access=%+v ok=%v", disList, ok)
	}
	locked0 := listItemInfo(t, ttDis, "Locked 0")
	lx, ly := center(locked0.Bounds)
	ttDis.ClickAt(lx, ly)
	ttDis.ClickAt(lx, ly)
	ttDis.Key(0, ui.KeyDown)
	ttDis.Key(0, ui.KeyEnter)
	if disSel != -1 || disClicks != 0 || disSubmits != 0 {
		t.Fatalf("disabled list mutated state: sel=%d clicks=%d submits=%d", disSel, disClicks, disSubmits)
	}
}

func TestVirtualListExternalStateOverrideRetract(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	state := &ui.ListState{}
	small := []string{"first", "second"}
	large := make([]string, 20)
	for i := range large {
		large[i] = fmt.Sprintf("large-%d", i)
	}

	useLarge := false
	oldSelected := -1
	var multi ui.Selection[int]

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(340, 260).Background(canvasRGB).Children(func() {
				if !useLarge {
					snap := small
					VirtualList(c.Key("ext-vl"), len(snap), func(r ui.ListRow) {
						Text(r.Context, snap[r.Index], tokens.BodyMedium)
					}, VirtualListOptions{
						State:     state,
						ItemKey:   func(row int) any { return snap[row] },
						Selected:  &oldSelected,
						RowHeight: 36,
						RowLabel:  func(row int) string { return snap[row] },
						Label:     "Dynamic list",
					}).Grow(1).FillWidth()
				} else {
					VirtualList(c.Key("ext-vl"), len(large), func(r ui.ListRow) {
						Text(r.Context, large[r.Index], tokens.BodyMedium)
					}, VirtualListOptions{
						State:     state,
						Selection: &multi,
						RowHeight: 36,
						RowLabel:  func(row int) string { return large[row] },
						Label:     "Dynamic list",
					}).Grow(1).FillWidth()
				}
			})
		})
	}, 340, 260)

	if err := tt.Click("second"); err != nil {
		t.Fatalf("click second: %v", err)
	}
	if oldSelected != 1 {
		t.Fatalf("oldSelected = %d, want 1", oldSelected)
	}

	// Switch to 20 items omitting ItemKey and Selected, using multi Selection instead.
	// Must not panic by calling stale 2-item ItemKey closure and must not mutate oldSelected.
	useLarge = true
	tt.Frame()
	if state.Key != nil {
		t.Fatal("stale ItemKey closure remained on external ListState after omission")
	}
	if state.Selected == &oldSelected {
		t.Fatal("stale Selected pointer remained on external ListState after omission")
	}
	if err := tt.Click("large-3"); err != nil {
		t.Fatalf("click large-3: %v", err)
	}
	if !multi.Has(3) {
		t.Fatal("multi Selection did not select row 3")
	}
	if oldSelected != 1 {
		t.Fatalf("clicking in new mode mutated oldSelected to %d, want 1", oldSelected)
	}
}
