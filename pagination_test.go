package md3e

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

func TestPaginationBoundariesClampingAndEdges(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	// 1. totalPages = 0 disables all navigation buttons and clamps *page to 0.
	pZero := 4
	changesZero := 0
	ttZero := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(420, 100).Background(canvasRGB).Padding(12).Children(func() {
				Pagination(c, &pZero, 0, PaginationOptions{
					ShowEdges: true,
					OnChanged: func(int) { changesZero++ },
				})
			})
		})
	}, 420, 100)
	if pZero != 0 {
		t.Fatalf("totalPages=0 did not clamp *page to 0: got %d", pZero)
	}
	for _, name := range []string{"First page", "Previous page", "Next page", "Last page"} {
		info := buttonInfo(t, ttZero, ui.RoleButton, name)
		if !info.Disabled {
			t.Fatalf("totalPages=0 button %q should be disabled", name)
		}
		ttZero.ClickAt(center(info.Bounds))
	}
	if _, ok := ttZero.AccessByRole(ui.RoleButton, "Page 1"); ok {
		t.Fatal("totalPages=0 should not render Page 1")
	}
	if pZero != 0 || changesZero != 0 || ttZero.FramePending() {
		t.Fatalf("totalPages=0 mutated state: p=%d changes=%d pending=%v", pZero, changesZero, ttZero.FramePending())
	}

	// 2. totalPages = 1 clamps negative *page to 1 and disables both ends.
	pOne := -3
	ttOne := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(420, 100).Background(canvasRGB).Padding(12).Children(func() {
				Pagination(c, &pOne, 1, PaginationOptions{ShowEdges: true})
			})
		})
	}, 420, 100)
	if pOne != 1 {
		t.Fatalf("totalPages=1 did not clamp -3 to 1: got %d", pOne)
	}
	for _, name := range []string{"First page", "Previous page", "Next page", "Last page"} {
		if !buttonInfo(t, ttOne, ui.RoleButton, name).Disabled {
			t.Fatalf("totalPages=1 button %q should be disabled", name)
		}
	}
	page1 := buttonInfo(t, ttOne, ui.RoleButton, "Page 1")
	if !page1.Checked || page1.Current != ui.CurrentPage {
		t.Fatalf("Page 1 access state = %+v, want Checked & CurrentPage", page1)
	}

	// 3. totalPages = 5 clamps upper overflow, handles clicks on Prev/Next/First/Last/Page.
	pFive := 99
	var gotChanges []int
	ttFive := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(480, 120).Background(canvasRGB).Padding(12).Children(func() {
				Pagination(c, &pFive, 5, PaginationOptions{
					ShowEdges: true,
					OnChanged: func(p int) { gotChanges = append(gotChanges, p) },
					Label:     "Results pages",
				})
			})
		})
	}, 480, 120)

	if _, ok := ttFive.AccessByRole(ui.RoleNavigation, "Results pages"); !ok {
		t.Fatal("RoleNavigation Results pages missing")
	}
	if pFive != 5 {
		t.Fatalf("totalPages=5 did not clamp 99 to 5: got %d", pFive)
	}
	if !buttonInfo(t, ttFive, ui.RoleButton, "Next page").Disabled || !buttonInfo(t, ttFive, ui.RoleButton, "Last page").Disabled {
		t.Fatal("Next/Last should be disabled on last page 5")
	}
	if buttonInfo(t, ttFive, ui.RoleButton, "Previous page").Disabled || buttonInfo(t, ttFive, ui.RoleButton, "First page").Disabled {
		t.Fatal("Previous/First should be enabled on last page 5")
	}

	ttFive.ClickAt(center(buttonInfo(t, ttFive, ui.RoleButton, "Previous page").Bounds))
	ttFive.ClickAt(center(buttonInfo(t, ttFive, ui.RoleButton, "Page 2").Bounds))
	ttFive.ClickAt(center(buttonInfo(t, ttFive, ui.RoleButton, "Next page").Bounds))
	ttFive.ClickAt(center(buttonInfo(t, ttFive, ui.RoleButton, "First page").Bounds))
	if !buttonInfo(t, ttFive, ui.RoleButton, "First page").Disabled || !buttonInfo(t, ttFive, ui.RoleButton, "Previous page").Disabled {
		t.Fatal("First/Previous should disable on page 1")
	}
	ttFive.ClickAt(center(buttonInfo(t, ttFive, ui.RoleButton, "Last page").Bounds))
	ttFive.Move(2, 2)
	ttFive.Advance(600 * time.Millisecond)

	if !slices.Equal(gotChanges, []int{4, 2, 3, 1, 5}) {
		t.Fatalf("OnChanged history = %v, want [4 2 3 1 5]", gotChanges)
	}
	if ttFive.FramePending() {
		t.Fatal("Pagination left FramePending true")
	}
}

func TestPaginationWindowEllipsisAndOptions(t *testing.T) {
	// Verify computePaginationSlots across start, middle, end, SiblingCount, and MaxButtons.
	slotStrings := func(slots []paginationSlot) []string {
		out := make([]string, len(slots))
		for i, s := range slots {
			switch s.kind {
			case paginationSlotEllipsisStart:
				out[i] = "<...>"
			case paginationSlotEllipsisEnd:
				out[i] = "<...>"
			default:
				out[i] = fmt.Sprintf("%d", s.page)
			}
		}
		return out
	}

	if got := slotStrings(computePaginationSlots(1, 20, 1, 0)); !slices.Equal(got, []string{"1", "2", "3", "4", "5", "<...>", "20"}) {
		t.Fatalf("page 1/20 slots = %v", got)
	}
	if got := slotStrings(computePaginationSlots(10, 20, 1, 0)); !slices.Equal(got, []string{"1", "<...>", "9", "10", "11", "<...>", "20"}) {
		t.Fatalf("page 10/20 slots = %v", got)
	}
	if got := slotStrings(computePaginationSlots(20, 20, 1, 0)); !slices.Equal(got, []string{"1", "<...>", "16", "17", "18", "19", "20"}) {
		t.Fatalf("page 20/20 slots = %v", got)
	}
	if got := slotStrings(computePaginationSlots(10, 20, 0, 5)); !slices.Equal(got, []string{"1", "<...>", "10", "<...>", "20"}) {
		t.Fatalf("MaxButtons=5 page 10/20 slots = %v", got)
	}

	// Render totalPages=20 at page=10 and verify ellipsis, TabularNums, and Tonal variant pixels.
	th := ThemeFromSeed(DefaultSeedColor)
	page := 10
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(640, 120).Background(canvasRGB).Padding(12).Children(func() {
				Pagination(c, &page, 20, PaginationOptions{
					Variant: ButtonTonal,
				})
			})
		})
	}, 640, 120)

	for _, want := range []int{1, 9, 10, 11, 20} {
		name := fmt.Sprintf("Page %d", want)
		if _, ok := tt.AccessByRole(ui.RoleButton, name); !ok {
			t.Fatalf("%q missing on page 10/20", name)
		}
	}
	for _, hidden := range []int{2, 8, 12, 19} {
		name := fmt.Sprintf("Page %d", hidden)
		if _, ok := tt.AccessByRole(ui.RoleButton, name); ok {
			t.Fatalf("%q should be collapsed behind ellipsis on page 10/20", name)
		}
	}

	// Active page 10 uses SecondaryContainer under ButtonTonal.
	p10 := buttonInfo(t, tt, ui.RoleButton, "Page 10")
	px := tt.Image().RGBAAt(int(p10.Bounds.X+4), int(p10.Bounds.Y+p10.Bounds.H/2))
	if !colorMatches(px, th.Colors.SecondaryContainer, 8) {
		t.Fatalf("Tonal active page pixel=%+v, want SecondaryContainer %+v", px, rgbOf(th.Colors.SecondaryContainer))
	}

	// Tabular numerals: "11" and "20" share uniform digit width.
	t11, ok11 := tt.Find("11")
	t20, ok20 := tt.Find("20")
	if !ok11 || !ok20 {
		t.Fatalf("page texts 11/20 missing: %v %v", ok11, ok20)
	}
	if math.Abs(float64(t11.W-t20.W)) > 1.0 {
		t.Fatalf("TabularNums width mismatch between \"11\" (%v) and \"20\" (%v)", t11.W, t20.W)
	}
}

func TestPaginationSmallBudgetsAndMaxInt(t *testing.T) {
	for _, tc := range []struct {
		current, budget int
		want            []int
	}{
		{1, 1, []int{1}},
		{1, 2, []int{1, math.MaxInt}},
		{50, 1, []int{50}},
		{50, 2, []int{1, 50}},
		{math.MaxInt, 1, []int{math.MaxInt}},
		{math.MaxInt, 2, []int{1, math.MaxInt}},
		{math.MaxInt, 0, []int{1, 0, math.MaxInt - 4, math.MaxInt - 3, math.MaxInt - 2, math.MaxInt - 1, math.MaxInt}},
	} {
		slots := computePaginationSlots(tc.current, math.MaxInt, 1, tc.budget)
		got := make([]int, 0, len(slots))
		for _, slot := range slots {
			got = append(got, slot.page)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("current %d, max buttons %d: slots %v, want %v", tc.current, tc.budget, got, tc.want)
		}
	}
	page := math.MaxInt - 1
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			Pagination(c, &page, math.MaxInt, PaginationOptions{MaxButtons: 1})
		})
	}, 500, 90)
	tt.SetDirection(ui.RightToLeft)
	if err := tt.Click("Next page"); err != nil || page != math.MaxInt {
		t.Fatalf("next at the final boundary: page %d, click error %v", page, err)
	}
	current := buttonInfo(t, tt, ui.RoleButton, fmt.Sprintf("Page %d", math.MaxInt))
	if current.Current != ui.CurrentPage || !buttonInfo(t, tt, ui.RoleButton, "Next page").Disabled || tt.FramePending() {
		t.Error("final page must be current, with Next disabled and no motion pending")
	}
}

func TestPaginationKeyboardRTLReduceMotionDensityAndDisabled(t *testing.T) {
	// 1. Keyboard FocusGroup navigation + focus preservation when window shifts + RTL mirror.
	th := ThemeFromSeed(DefaultSeedColor)
	page := 3
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(640, 140).Background(canvasRGB).Padding(12).Gap(8).Children(func() {
				Button(c, "Before")
				Pagination(c, &page, 20)
				Button(c, "After")
			})
		})
	}, 640, 140)

	tt.Click("Before")
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Previous page") {
		t.Fatal("Tab from Before did not focus Previous page")
	}
	tt.Key(0, ui.KeyEnd)
	if !tt.Focused("Next page") {
		t.Fatal("End did not focus Next page")
	}
	// Activating Next page twice crosses page 5 (shifting ellipsis slots) while Next page keeps focus.
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeySpace)
	if page != 5 {
		t.Fatalf("Enter+Space on Next page gave page=%d, want 5", page)
	}
	if !tt.Focused("Next page") {
		t.Fatal("Next page lost focus when pagination window shifted")
	}
	tt.Key(0, ui.KeyLeft)
	if !tt.Focused("Page 20") {
		t.Fatal("Left from Next page did not focus Page 20")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("After") {
		t.Fatal("Tab did not leave Pagination FocusGroup to After")
	}

	// RTL layout mirroring: Previous page sits on the RIGHT of Next page, Page 1 on the RIGHT of Page 20,
	// and Left key moves forward toward Next page.
	tt.SetDirection(ui.RightToLeft)
	tt.Advance(400 * time.Millisecond)

	prevRTL := buttonInfo(t, tt, ui.RoleButton, "Previous page")
	nextRTL := buttonInfo(t, tt, ui.RoleButton, "Next page")
	p1RTL := buttonInfo(t, tt, ui.RoleButton, "Page 1")
	p20RTL := buttonInfo(t, tt, ui.RoleButton, "Page 20")
	if !(prevRTL.Bounds.X > p1RTL.Bounds.X && p1RTL.Bounds.X > p20RTL.Bounds.X && p20RTL.Bounds.X > nextRTL.Bounds.X) {
		t.Fatalf("RTL order invalid: prev=%v p1=%v p20=%v next=%v",
			prevRTL.Bounds, p1RTL.Bounds, p20RTL.Bounds, nextRTL.Bounds)
	}
	// Verify Previous icon (ChevronRight in RTL) tip points right vs Next icon (ChevronLeft in RTL) tip points left.
	midlineInkOffsetX := func(b ui.Rect) float32 {
		img := tt.Image()
		y := int(b.Y + b.H/2)
		var sumX, count float32
		for x := int(b.X) + 8; x < int(b.X+b.W)-8; x++ {
			if !colorMatches(img.RGBAAt(x, y), canvasRGB, 12) {
				sumX += float32(x) + 0.5
				count++
			}
		}
		if count == 0 {
			return 0
		}
		return sumX/count - (b.X + b.W/2)
	}
	prevRel := midlineInkOffsetX(prevRTL.Bounds)
	nextRel := midlineInkOffsetX(nextRTL.Bounds)
	if !(prevRel > 0.5 && nextRel < -0.5) {
		t.Fatalf("RTL nav icons not mirrored at midline: prevRel=%v nextRel=%v", prevRel, nextRel)
	}

	tt.ClickAt(center(p1RTL.Bounds))
	if page != 1 {
		t.Fatalf("clicking RTL Page 1 gave page=%d, want 1", page)
	}
	tt.Key(0, ui.KeyLeft) // In RTL, Left moves toward Page 2.
	if !tt.Focused("Page 2") {
		t.Fatalf("RTL Left from Page 1 did not focus Page 2")
	}

	// 2. ReduceMotion + Hover no-rebuild + Density heights + Disabled.
	thRM := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	rmPage := 1
	builds := 0
	ttRM := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, thRM, func() {
			ui.Column(c).Size(480, 120).Background(canvasRGB).Padding(8).Children(func() {
				Pagination(c, &rmPage, 5)
			})
		})
	}, 480, 120)

	p3 := buttonInfo(t, ttRM, ui.RoleButton, "Page 3")
	ttRM.Move(center(p3.Bounds))
	hoverBuilds := builds
	ttRM.Advance(100 * time.Millisecond)
	if builds != hoverBuilds || ttRM.FramePending() {
		t.Fatalf("ReduceMotion hover: rebuilds=%d FramePending=%v", builds-hoverBuilds, ttRM.FramePending())
	}
	ttRM.Press(center(p3.Bounds))
	ttRM.Release(center(p3.Bounds))
	if rmPage != 3 || ttRM.FramePending() {
		t.Fatalf("ReduceMotion page click: page=%d FramePending=%v", rmPage, ttRM.FramePending())
	}

	// Density heights & Disabled blocking.
	for _, tc := range []struct {
		d     tokens.Density
		wantH float32
	}{
		{tokens.DensityComfortable, 36},
		{tokens.DensityDesktop, 32},
		{tokens.DensityCompact, 28},
	} {
		dth := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tc.d})
		dp := 2
		dchanges := 0
		dtt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, dth, func() {
				ui.Column(c).Size(420, 100).Background(canvasRGB).Children(func() {
					Pagination(c, &dp, 4, PaginationOptions{
						Disabled:  true,
						OnChanged: func(int) { dchanges++ },
					})
				})
			})
		}, 420, 100)
		b := buttonInfo(t, dtt, ui.RoleButton, "Page 3")
		if math.Abs(float64(b.Bounds.H-tc.wantH)) > 0.5 || !b.Disabled {
			t.Fatalf("density %v page button height=%v (want %v) disabled=%v", tc.d, b.Bounds.H, tc.wantH, b.Disabled)
		}
		dtt.ClickAt(center(b.Bounds))
		if dp != 2 || dchanges != 0 {
			t.Fatalf("disabled Pagination mutated state: page=%d changes=%d", dp, dchanges)
		}
	}
}
