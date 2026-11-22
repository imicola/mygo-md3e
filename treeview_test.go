package md3e

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/tokens"
)

func treeItemInfo(t *testing.T, tt *ui.Tester, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleTreeItem, name)
	if !ok {
		t.Fatalf("RoleTreeItem %q missing from access tree", name)
	}
	return info
}

func treeChevronCenter(bounds ui.Rect, depth int, th *Theme, rtl bool) (float32, float32) {
	m := treeMetricsFor(th)
	offset := m.padStart + float32(depth)*m.indentStep + m.arrowSize/2
	x := bounds.X + offset
	if rtl {
		x = bounds.X + bounds.W - offset
	}
	return x, bounds.Y + bounds.H/2
}

func TestTreeViewExpandSelectionLocalOpenAndCallbacks(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	srcOpen := false
	var selected any
	var events []string

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(360, 340).Background(canvasRGB).Padding(12).Children(func() {
				TreeView(c.Key("files"), []TreeNode{
					{
						Key:   "src",
						Label: "src",
						Icon:  mdicon.Edit.Draw,
						Badge: "3",
						Open:  &srcOpen,
						OnClick: func() {
							events = append(events, "click:src")
						},
						Children: []TreeNode{
							{
								Key:   "src/main.go",
								Label: "main.go",
								OnClick: func() {
									events = append(events, "click:main.go")
								},
							},
							{
								// Open == nil with Children falls back to stable ui.Local state.
								Key:   "src/cmd",
								Label: "cmd",
								Children: []TreeNode{
									{Key: "src/cmd/app.go", Label: "app.go"},
								},
							},
						},
					},
					{
						Key:   "go.mod",
						Label: "go.mod",
						OnClick: func() {
							events = append(events, "click:go.mod")
						},
					},
				}, TreeViewOptions{
					Selected: &selected,
					OnSelect: func(key any) {
						events = append(events, "select:"+key.(string))
					},
					Label: "Project files",
				})
			})
		})
	}, 360, 340)

	if tt.FramePending() {
		t.Fatal("initial settled TreeView left FramePending true")
	}
	if _, ok := tt.AccessByRole(ui.RoleTree, "Project files"); !ok {
		t.Fatal("RoleTree Project files missing")
	}
	if tt.HasText("main.go") {
		t.Fatal("closed branch unexpectedly rendered children")
	}

	// Clicking the disclosure chevron expands src without selecting it or firing OnClick.
	srcInfo := treeItemInfo(t, tt, "src")
	if srcInfo.Description != "3" {
		t.Fatalf("src badge Description = %q, want \"3\"", srcInfo.Description)
	}
	cx, cy := treeChevronCenter(srcInfo.Bounds, 0, th, false)
	tt.ClickAt(cx, cy)
	if !srcOpen || !tt.HasText("main.go") || !tt.HasText("cmd") {
		t.Fatalf("clicking chevron did not expand src: srcOpen=%v texts=%q", srcOpen, tt.Texts())
	}
	if selected != nil || len(events) != 0 {
		t.Fatalf("clicking chevron triggered selection/callbacks: selected=%v events=%v", selected, events)
	}
	if !tt.FramePending() {
		t.Fatal("normal-motion chevron expand should animate with FastSpatial, not snap")
	}
	tt.Move(2, 2)
	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("chevron expand spring did not settle")
	}

	// Expand local-state branch "cmd" (Open == nil) by clicking its chevron.
	cmdInfo := treeItemInfo(t, tt, "cmd")
	cmdX, cmdY := treeChevronCenter(cmdInfo.Bounds, 1, th, false)
	tt.ClickAt(cmdX, cmdY)
	if !tt.HasText("app.go") {
		t.Fatal("clicking chevron on Open==nil branch did not expand via ui.Local")
	}
	if selected != nil || len(events) != 0 {
		t.Fatalf("clicking local branch chevron triggered selection: selected=%v events=%v", selected, events)
	}

	// Collapse src and re-expand: cmd's local state stays closed/open? Wait:
	// while src closes, item.Group unmounts cmd; verify re-collapsing cmd itself works cleanly.
	tt.ClickAt(cmdX, cmdY)
	if tt.HasText("app.go") {
		t.Fatal("second click on local branch chevron did not collapse cmd")
	}
	tt.Advance(600 * time.Millisecond)

	// Clicking row body (away from chevron) selects item and runs OnSelect before OnClick.
	mainInfo := treeItemInfo(t, tt, "main.go")
	tt.ClickAt(mainInfo.Bounds.X+mainInfo.Bounds.W*0.6, mainInfo.Bounds.Y+mainInfo.Bounds.H/2)
	tt.Move(2, 2)
	tt.Advance(400 * time.Millisecond)
	if selected != "src/main.go" {
		t.Fatalf("row click selected=%v, want \"src/main.go\"", selected)
	}
	if !slices.Equal(events, []string{"select:src/main.go", "click:main.go"}) {
		t.Fatalf("events=%v, want [select:src/main.go click:main.go]", events)
	}
	mainInfo = treeItemInfo(t, tt, "main.go")
	if !mainInfo.Selected {
		t.Fatal("selected RoleTreeItem does not report Selected=true")
	}
	px := tt.Image().RGBAAt(int(mainInfo.Bounds.X+mainInfo.Bounds.W-12), int(mainInfo.Bounds.Y+mainInfo.Bounds.H/2))
	if !colorMatches(px, th.Colors.SecondaryContainer, 8) {
		t.Fatalf("selected node pixel=%+v, want SecondaryContainer %+v", px, rgbOf(th.Colors.SecondaryContainer))
	}
	if tt.FramePending() {
		t.Fatal("TreeView left FramePending true after selection")
	}
}

func TestTreeViewKeyboardNavigationAndSingleTabStop(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	srcOpen, cmdOpen := false, false
	var selected any

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(360, 340).Background(canvasRGB).Padding(12).Gap(8).Children(func() {
				Button(c, "Before")
				TreeView(c, []TreeNode{
					{
						Key:   "src",
						Label: "src",
						Open:  &srcOpen,
						Children: []TreeNode{
							{Key: "main.go", Label: "main.go"},
							{
								Key:   "cmd",
								Label: "cmd",
								Open:  &cmdOpen,
								Children: []TreeNode{
									{Key: "tool.go", Label: "tool.go"},
								},
							},
							{Key: "mux.go", Label: "mux.go"},
						},
					},
					{Key: "go.mod", Label: "go.mod"},
					{Key: "go.sum", Label: "go.sum"},
				}, TreeViewOptions{
					Selected: &selected,
					Label:    "Nav tree",
				})
				Button(c, "After")
			})
		})
	}, 360, 340)

	// Single Tab stop into tree, remembers last focused item across Tab out & Shift+Tab back.
	tt.Click("Before")
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("src") {
		t.Fatal("Tab from Before did not focus first tree item src")
	}
	tt.Key(0, ui.KeyDown)
	if !tt.Focused("go.mod") {
		t.Fatal("Down did not move focus to go.mod")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("After") {
		t.Fatal("Tab from tree item did not leave tree to After button")
	}
	tt.Key(ui.Shift, ui.KeyTab)
	if !tt.Focused("go.mod") {
		t.Fatal("Shift+Tab did not return to last focused tree item go.mod")
	}

	// Opening src above focused go.mod preserves focus on go.mod via node keys.
	srcOpen = true
	tt.Frame()
	if !tt.Focused("go.mod") {
		t.Fatal("expanding sibling above go.mod lost focus on go.mod")
	}

	// Home / Up / Down / Right / Left / End navigation.
	steps := []struct {
		key   ui.Key
		focus string
	}{
		{ui.KeyHome, "src"},
		{ui.KeyDown, "main.go"},
		{ui.KeyDown, "cmd"},
		{ui.KeyRight, "cmd"},     // opens cmd
		{ui.KeyRight, "tool.go"}, // moves to first child
		{ui.KeyLeft, "cmd"},      // moves to parent
		{ui.KeyLeft, "cmd"},      // closes cmd
		{ui.KeyDown, "mux.go"},
		{ui.KeyEnd, "go.sum"},
		{ui.KeyUp, "go.mod"},
	}
	for _, st := range steps {
		tt.Key(0, st.key)
		if !tt.Focused(st.focus) {
			t.Fatalf("after %v, focused is not %q", st.key, st.focus)
		}
	}
	if cmdOpen {
		t.Fatal("Left on open cmd did not close it")
	}

	// Enter activates focused node and updates *Selected.
	tt.Key(0, ui.KeyEnter)
	if selected != "go.mod" {
		t.Fatalf("Enter on go.mod set selected=%v, want \"go.mod\"", selected)
	}

	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("TreeView keyboard test left FramePending true")
	}
}

func TestTreeViewIndentGuidesRTLReduceMotionAndDisabled(t *testing.T) {
	// 1. Indent guides + RTL mirroring + initial open state snaps without pending frames.
	th := ThemeFromSeed(DefaultSeedColor)
	openRoot := true
	openSub := true
	selected := any("leaf")

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(340, 260).Background(canvasRGB).Padding(8).Children(func() {
				TreeView(c, []TreeNode{
					{
						Key:   "root",
						Label: "root",
						Open:  &openRoot,
						Children: []TreeNode{
							{
								Key:   "sub",
								Label: "sub",
								Open:  &openSub,
								Children: []TreeNode{
									{Key: "leaf", Label: "leaf"},
								},
							},
						},
					},
				}, TreeViewOptions{
					Selected:         &selected,
					ShowIndentGuides: true,
					Label:            "Guided tree",
				})
			})
		})
	}, 340, 260)

	if tt.FramePending() {
		t.Fatal("initially-open TreeView should snap chevrons and selection without pending frames")
	}

	m := treeMetricsFor(th)
	rootInfo := treeItemInfo(t, tt, "root")
	subInfo := treeItemInfo(t, tt, "sub")
	leafInfo := treeItemInfo(t, tt, "leaf")

	// Verify indent guide pixel at depth 0 beneath root's chevron center.
	guideX := int(math.Round(float64(rootInfo.Bounds.X + m.padStart + m.arrowSize/2)))
	guideY := int(subInfo.Bounds.Y + subInfo.Bounds.H/2)
	guidePx := tt.Image().RGBAAt(guideX, guideY)
	if !colorMatches(guidePx, th.Colors.OutlineVariant, 8) {
		t.Fatalf("LTR indent guide pixel at (%d,%d)=%+v, want OutlineVariant %+v",
			guideX, guideY, guidePx, rgbOf(th.Colors.OutlineVariant))
	}

	// Verify RTL mirroring: text and chevron move to the right edge, guide line mirrors right,
	// and Left opens / Right closes.
	tt.SetDirection(ui.RightToLeft)
	tt.Advance(200 * time.Millisecond)

	rootRTL := treeItemInfo(t, tt, "root")
	subRTL := treeItemInfo(t, tt, "sub")
	rootTextLTR, _ := tt.Find("root")
	_ = rootTextLTR

	rtlGuideX := int(math.Round(float64(rootRTL.Bounds.X + rootRTL.Bounds.W - (m.padStart + m.arrowSize/2))))
	rtlGuideY := int(subRTL.Bounds.Y + subRTL.Bounds.H/2)
	rtlPx := tt.Image().RGBAAt(rtlGuideX, rtlGuideY)
	if !colorMatches(rtlPx, th.Colors.OutlineVariant, 8) {
		t.Fatalf("RTL indent guide pixel at (%d,%d)=%+v, want OutlineVariant %+v",
			rtlGuideX, rtlGuideY, rtlPx, rgbOf(th.Colors.OutlineVariant))
	}

	// In RTL, leaf label sits further left (smaller X) than root label? No, Wait:
	// PaddingStart indents from the RIGHT edge in RTL, so deeper items have their right content edge shifted LEFT!
	rootTxt, _ := tt.Find("root")
	subTxt, _ := tt.Find("sub")
	if !(subTxt.X+subTxt.W < rootTxt.X+rootTxt.W) {
		t.Fatalf("RTL indentation did not shift sub inward from right edge: root=%+v sub=%+v", rootTxt, subTxt)
	}
	_ = leafInfo

	// RTL keyboard: Right closes open root, Left opens closed root.
	tt.ClickAt(rootRTL.Bounds.X+rootRTL.Bounds.W/2, rootRTL.Bounds.Y+rootRTL.Bounds.H/2)
	tt.Key(0, ui.KeyRight)
	if openRoot {
		t.Fatal("RTL Right key did not close open root")
	}
	tt.Key(0, ui.KeyLeft)
	if !openRoot {
		t.Fatal("RTL Left key did not open closed root")
	}
	tt.Advance(600 * time.Millisecond)

	// 2. ReduceMotion: toggling branch settles immediately with zero pending frames.
	thRM := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	rmOpen := false
	rmSel := any(nil)
	ttRM := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, thRM, func() {
			ui.Column(c).Size(320, 220).Background(canvasRGB).Children(func() {
				TreeView(c, []TreeNode{
					{
						Key:   "dir",
						Label: "dir",
						Open:  &rmOpen,
						Children: []TreeNode{
							{Key: "file", Label: "file"},
						},
					},
				}, TreeViewOptions{Selected: &rmSel, Label: "RM tree"})
			})
		})
	}, 320, 220)

	dirInfo := treeItemInfo(t, ttRM, "dir")
	cx, cy := treeChevronCenter(dirInfo.Bounds, 0, thRM, false)
	ttRM.ClickAt(cx, cy)
	if !rmOpen || ttRM.FramePending() {
		t.Fatalf("ReduceMotion chevron toggle: open=%v FramePending=%v", rmOpen, ttRM.FramePending())
	}

	// 3. Hover causes 0 view rebuilds and Disabled blocks clicks/keys.
	builds := 0
	hoverBuilds := 0
	disOpen := false
	disSel := any("none")
	disClicks := 0
	ttHover := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, th, func() {
			ui.Column(c).Size(320, 220).Background(canvasRGB).Gap(8).Children(func() {
				TreeView(c, []TreeNode{
					{Key: "active", Label: "active"},
				}, TreeViewOptions{Label: "Hover tree"})
				TreeView(c, []TreeNode{
					{
						Key:      "locked",
						Label:    "locked",
						Open:     &disOpen,
						OnClick:  func() { disClicks++ },
						Children: []TreeNode{{Key: "child", Label: "child"}},
					},
				}, TreeViewOptions{
					Selected: &disSel,
					Disabled: true,
					Label:    "Disabled tree",
				})
			})
		})
	}, 320, 220)

	actInfo := treeItemInfo(t, ttHover, "active")
	ttHover.Move(actInfo.Bounds.X+actInfo.Bounds.W/2, actInfo.Bounds.Y+actInfo.Bounds.H/2)
	hoverBuilds = builds
	ttHover.Advance(250 * time.Millisecond)
	if builds != hoverBuilds {
		t.Fatalf("hovering TreeView row triggered %d unexpected view rebuilds", builds-hoverBuilds)
	}
	if ttHover.FramePending() {
		t.Fatal("hovered TreeView did not settle")
	}

	lockedInfo := treeItemInfo(t, ttHover, "locked")
	if !lockedInfo.Disabled {
		t.Fatal("disabled TreeView node does not report Disabled=true")
	}
	lx, ly := treeChevronCenter(lockedInfo.Bounds, 0, th, false)
	ttHover.ClickAt(lx, ly)
	ttHover.ClickAt(lockedInfo.Bounds.X+lockedInfo.Bounds.W/2, lockedInfo.Bounds.Y+lockedInfo.Bounds.H/2)
	if disOpen || disSel != "none" || disClicks != 0 {
		t.Fatalf("disabled TreeView mutated state: open=%v sel=%v clicks=%d", disOpen, disSel, disClicks)
	}
}

func TestTreeViewDensityLevelsAndAccessStates(t *testing.T) {
	densities := []struct {
		d    tokens.Density
		want float32
	}{
		{tokens.DensityComfortable, 44},
		{tokens.DensityDesktop, 34},
		{tokens.DensityCompact, 28},
	}
	for _, tc := range densities {
		th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tc.d})
		open := true
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Column(c).Size(320, 200).Background(canvasRGB).Children(func() {
					TreeView(c, []TreeNode{
						{
							Key:   "docs",
							Label: "docs",
							Open:  &open,
							Children: []TreeNode{
								{Key: "intro", Label: "intro.md"},
							},
						},
					}, TreeViewOptions{Label: "Docs tree"})
				})
			})
		}, 320, 200)

		docsInfo := treeItemInfo(t, tt, "docs")
		if math.Abs(float64(docsInfo.Bounds.H-tc.want)) > 0.5 {
			t.Fatalf("density %v row height=%v, want %v", tc.d, docsInfo.Bounds.H, tc.want)
		}
		if docsInfo.Level != 1 || !docsInfo.Expandable || !docsInfo.Expanded {
			t.Fatalf("docs access state = %+v, want Level=1 Expandable Expanded", docsInfo)
		}
		introInfo := treeItemInfo(t, tt, "intro.md")
		if introInfo.Level != 2 || introInfo.Expandable {
			t.Fatalf("intro access state = %+v, want Level=2 non-expandable", introInfo)
		}
	}
}
