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

func TestDataTable100KVirtualization(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tokens.DensityDesktop})
	state := &ui.ListState{}
	cols := []DataTableColumn{
		{Title: "ID", Width: 80, Numeric: true},
		{Title: "Name"},
	}
	builtRows := map[int]bool{}

	tt := ui.NewTester(func(c *ui.Context) {
		clear(builtRows)
		ThemeScope(c, th, func() {
			ui.Column(c).Size(480, 320).Background(canvasRGB).Children(func() {
				DataTable(c, cols, 100_000, func(row, col int) {
					builtRows[row] = true
					if col == 0 {
						DataTableCellText(c, fmt.Sprintf("%d", row), DataTableCellOptions{Numeric: true})
						return
					}
					DataTableCellText(c, fmt.Sprintf("Row %d", row))
				}, DataTableOptions{
					State: state,
					Label: "Large table",
				}).Grow(1).FillWidth()
			})
		})
	}, 480, 320)

	if len(builtRows) == 0 || len(builtRows) >= 40 {
		t.Fatalf("initial 100K table built %d distinct rows, want >0 and <40", len(builtRows))
	}
	if builtRows[50_000] {
		t.Fatal("initial frame unexpectedly built off-screen row 50000")
	}
	if tt.FramePending() {
		t.Fatal("settled 100K DataTable left FramePending true")
	}

	state.ScrollTo(50_000, ui.Start)
	tt.Frame()
	if len(builtRows) == 0 || len(builtRows) >= 40 {
		t.Fatalf("scrolled 100K table built %d distinct rows, want >0 and <40", len(builtRows))
	}
	if !builtRows[50_000] || !tt.HasText("Row 50000") {
		t.Fatal("scrolling to row 50000 did not render row 50000")
	}
	if tt.FramePending() {
		t.Fatal("scrolled 100K DataTable left FramePending true")
	}
}

func TestDataTableSortAndCallback(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	sort := ui.SortOrder{Column: "Name"}
	var gotSort ui.SortOrder
	sortChanges := 0
	cols := []DataTableColumn{
		{Title: "Name", Sortable: true},
		{Title: "Kind", Width: 100},
		{Title: "Size", Width: 90, Numeric: true, Sortable: true},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(500, 260).Background(canvasRGB).Padding(8).Children(func() {
				DataTable(c, cols, 3, func(row, col int) {
					DataTableCellText(c, fmt.Sprintf("c%d-%d", row, col))
				}, DataTableOptions{
					Sort: &sort,
					OnSortChanged: func(s ui.SortOrder, by []ui.SortOrder) {
						sortChanges++
						gotSort = s
					},
					Label: "Sortable table",
				}).Grow(1).FillWidth()
			})
		})
	}, 500, 260)

	if err := tt.Click("Size"); err != nil {
		t.Fatalf("click Size header: %v", err)
	}
	if sort != (ui.SortOrder{Column: "Size"}) || gotSort != sort || sortChanges != 1 {
		t.Fatalf("first click Size: sort=%+v got=%+v changes=%d", sort, gotSort, sortChanges)
	}

	if err := tt.Click("Size"); err != nil {
		t.Fatalf("second click Size: %v", err)
	}
	if sort != (ui.SortOrder{Column: "Size", Descending: true}) || gotSort != sort || sortChanges != 2 {
		t.Fatalf("second click Size: sort=%+v got=%+v changes=%d", sort, gotSort, sortChanges)
	}

	// Non-sortable column does not trigger sort changes.
	if err := tt.Click("Kind"); err != nil {
		t.Fatalf("click Kind header: %v", err)
	}
	if sortChanges != 2 {
		t.Fatalf("clicking non-sortable Kind triggered OnSortChanged (%d)", sortChanges)
	}

	// Multi-column SortBy keeps Sort synchronized with first column.
	var sortBy []ui.SortOrder
	var primary ui.SortOrder
	var lastMulti []ui.SortOrder
	ttMulti := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(500, 260).Background(canvasRGB).Children(func() {
				DataTable(c, cols, 2, func(row, col int) {
					DataTableCellText(c, fmt.Sprintf("m%d-%d", row, col))
				}, DataTableOptions{
					Sort:   &primary,
					SortBy: &sortBy,
					OnSortChanged: func(s ui.SortOrder, by []ui.SortOrder) {
						lastMulti = by
					},
					Label: "Multi sort table",
				}).Grow(1).FillWidth()
			})
		})
	}, 500, 260)

	if err := ttMulti.Click("Name"); err != nil {
		t.Fatalf("multi click Name: %v", err)
	}
	if err := ttMulti.ClickWith(ui.Shift, "Size"); err != nil {
		t.Fatalf("multi Shift-click Size: %v", err)
	}
	wantBy := []ui.SortOrder{{Column: "Name"}, {Column: "Size"}}
	if !slices.Equal(sortBy, wantBy) || !slices.Equal(lastMulti, wantBy) || primary != wantBy[0] {
		t.Fatalf("multi-sort got sortBy=%+v lastMulti=%+v primary=%+v, want %+v", sortBy, lastMulti, primary, wantBy)
	}
}

func TestDataTableNumericTabularAndMono(t *testing.T) {
	th := ThemeFromSeed(ui.Hex("#0B57D0"), ThemeOptions{
		Density:  tokens.DensityDesktop,
		MonoFont: "JetBrains Mono",
	})
	var cellTheme *Theme
	cols := []DataTableColumn{
		{Title: "Service", Width: 180},
		{Title: "Latency", Width: 120, Numeric: true},
		{Title: "Digest", Width: 140, Mono: true},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(500, 220).Background(canvasRGB).Padding(8).Children(func() {
				DataTable(c, cols, 2, func(row, col int) {
					if row == 0 && col == 0 {
						cellTheme = CurrentTheme(c)
					}
					switch col {
					case 0:
						DataTableCellText(c, fmt.Sprintf("api-%d", row))
					case 1:
						DataTableCellText(c, "12 ms", DataTableCellOptions{Numeric: true})
					case 2:
						DataTableCellText(c, "abcdef123456", DataTableCellOptions{
							Mono:     true,
							Truncate: TruncateMiddle,
						})
					}
				}, DataTableOptions{
					Label: "Metrics table",
				}).Grow(1).FillWidth()
			})
		})
	}, 500, 220)

	if cellTheme == nil {
		t.Fatal("CurrentTheme(c) was not captured inside cell builder")
	}
	if cellTheme.Density != tokens.DensityDesktop || cellTheme.MonoFont != "JetBrains Mono" || cellTheme.Colors.Primary != th.Colors.Primary {
		t.Fatalf("CurrentTheme inside cell mismatched enclosing ThemeScope: density=%v mono=%q primary=%+v",
			cellTheme.Density, cellTheme.MonoFont, cellTheme.Colors.Primary)
	}

	latHdr, ok := tt.AccessByRole(ui.RoleColumnHeader, "Latency")
	if !ok {
		t.Fatal("Latency column header missing")
	}
	latText, ok := tt.Find("12 ms")
	if !ok {
		t.Fatal("Numeric cell text not found")
	}
	// Right-aligned numeric text sits near the trailing edge of the Latency column in LTR.
	hdrCenterX := latHdr.Bounds.X + latHdr.Bounds.W/2
	if latText.X < hdrCenterX {
		t.Fatalf("Numeric cell text X=%v not right-aligned past column center %v (hdr=%+v, text=%+v)",
			latText.X, hdrCenterX, latHdr.Bounds, latText)
	}
}

func TestDataTableDensityRowHeights(t *testing.T) {
	seed := DefaultSeedColor
	density := tokens.DensityComfortable
	explicitHeight := float32(0)
	state := &ui.ListState{}
	cols := []DataTableColumn{{Title: "Item"}}

	tt := ui.NewTester(func(c *ui.Context) {
		th := ThemeFromSeed(seed, ThemeOptions{Density: density})
		ThemeScope(c, th, func() {
			ui.Column(c).Size(360, 260).Background(canvasRGB).Children(func() {
				DataTable(c, cols, 3, func(row, col int) {
					DataTableCellText(c, fmt.Sprintf("DensityRow-%d", row))
				}, DataTableOptions{
					State:     state,
					RowHeight: explicitHeight,
					RowLabel:  func(row int) string { return fmt.Sprintf("DensityRow-%d", row) },
					Label:     "Density table",
				}).Grow(1).FillWidth()
			})
		})
	}, 360, 260)

	checkRowH := func(want float32, name string) {
		t.Helper()
		info, ok := tt.AccessByRole(ui.RoleRow, "DensityRow-0")
		if !ok {
			t.Fatalf("%s: RoleRow DensityRow-0 missing", name)
		}
		if math.Abs(float64(info.Bounds.H-want)) > 0.5 {
			t.Fatalf("%s: row height=%v, want %v", name, info.Bounds.H, want)
		}
	}

	checkRowH(48, "DensityComfortable")

	density = tokens.DensityDesktop
	tt.Frame()
	checkRowH(38, "DensityDesktop")

	density = tokens.DensityCompact
	tt.Frame()
	checkRowH(32, "DensityCompact")

	explicitHeight = 54
	tt.Frame()
	checkRowH(54, "ExplicitRowHeight")

	explicitHeight = float32(math.NaN())
	tt.Frame()
	checkRowH(32, "NaNRowHeightFallbackToDensity")
}

func TestDataTableSelectionKeyboardAndSubmit(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := -1
	var clickedRows, submittedRows []int
	cols := []DataTableColumn{{Title: "Name"}, {Title: "Role"}}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(400, 260).Background(canvasRGB).Children(func() {
				DataTable(c, cols, 6, func(row, col int) {
					if col == 0 {
						DataTableCellText(c, fmt.Sprintf("User %d", row))
						return
					}
					DataTableCellText(c, "Member")
				}, DataTableOptions{
					Selected:    &selected,
					RoundRows:   true,
					RowLabel:    func(row int) string { return fmt.Sprintf("Row %d", row) },
					OnRowClick:  func(row int) { clickedRows = append(clickedRows, row) },
					OnRowSubmit: func(row int) { submittedRows = append(submittedRows, row) },
					Label:       "Users table",
				}).Grow(1).FillWidth()
			})
		})
	}, 400, 260)

	if err := tt.Click("User 1"); err != nil {
		t.Fatalf("click User 1: %v", err)
	}
	if selected != 1 || !slices.Equal(clickedRows, []int{1}) {
		t.Fatalf("click User 1: selected=%d clickedRows=%v", selected, clickedRows)
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
	if selected != 5 {
		t.Fatalf("KeyEnd selected=%d, want 5", selected)
	}
	tt.Key(0, ui.KeyHome)
	if selected != 0 {
		t.Fatalf("KeyHome selected=%d, want 0", selected)
	}
	tt.Key(0, ui.KeyEnter)
	if !slices.Equal(submittedRows, []int{0}) {
		t.Fatalf("KeyEnter submittedRows=%v, want [0]", submittedRows)
	}

	// Double-click submits row 3.
	tt.Advance(time.Second)
	u3, ok := tt.Find("User 3")
	if !ok {
		t.Fatal("User 3 missing")
	}
	tt.ClickAt(u3.X+4, u3.Y+4)
	tt.ClickAt(u3.X+4, u3.Y+4)
	tt.Advance(time.Second)
	if selected != 3 || !slices.Equal(submittedRows, []int{0, 3}) {
		t.Fatalf("double-click User 3: selected=%d submittedRows=%v", selected, submittedRows)
	}

	// Multi-selection with ItemKey.
	keys := []string{"k0", "k1", "k2", "k3"}
	var chosen ui.Selection[string]
	ttMulti := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(360, 220).Background(canvasRGB).Children(func() {
				DataTable(c, []DataTableColumn{{Title: "Key"}}, len(keys), func(row, col int) {
					DataTableCellText(c, keys[row])
				}, DataTableOptions{
					ItemKey:   func(row int) any { return keys[row] },
					Selection: &chosen,
					Label:     "Multi table",
				}).Grow(1).FillWidth()
			})
		})
	}, 360, 220)

	if err := ttMulti.Click("k0"); err != nil {
		t.Fatalf("multi click k0: %v", err)
	}
	if err := ttMulti.ClickWith(ui.Shift, "k2"); err != nil {
		t.Fatalf("multi Shift-click k2: %v", err)
	}
	if err := ttMulti.ClickWith(ui.Cmd, "k1"); err != nil {
		t.Fatalf("multi Cmd-click k1: %v", err)
	}
	if !chosen.Has("k0") || chosen.Has("k1") || !chosen.Has("k2") || chosen.Has("k3") {
		t.Fatalf("multi selection gave k0=%v k1=%v k2=%v k3=%v",
			chosen.Has("k0"), chosen.Has("k1"), chosen.Has("k2"), chosen.Has("k3"))
	}
}

func TestDataTableHeaderAndCellContextMenu(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	var chosen ui.Selection[int]
	headerCol := -99
	headerActionRan := false
	cellRow, cellCol := -1, -1
	inspected := false

	cols := []DataTableColumn{
		{Title: "Name", Width: 140},
		{Title: "Kind", Width: 100},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(460, 260).Background(canvasRGB).Children(func() {
				DataTable(c, cols, 4, func(row, col int) {
					DataTableCellText(c, fmt.Sprintf("r%dc%d", row, col))
				}, DataTableOptions{
					Selection: &chosen,
					HeaderMenu: func(m *ui.Menu, col int) {
						headerCol = col
						ApplyDataTableMenuActions(m, []DataTableMenuAction{
							{Label: "Sort Descending", Checkable: true, Checked: true, OnClick: func() { headerActionRan = true }},
							{Separator: true},
							{Label: "More", Submenu: []DataTableMenuAction{{Label: "Reset"}}},
						})
					},
					CellMenu: func(m *ui.Menu, row, col int) {
						cellRow, cellCol = row, col
						ApplyDataTableMenuActions(m, []DataTableMenuAction{
							{Label: "Inspect", OnClick: func() { inspected = true }},
							{Label: "Delete", Danger: true},
						})
					},
					Label: "Menu table",
				}).Grow(1).FillWidth()
			})
		})
	}, 460, 260)

	if err := tt.RightClick("Name"); err != nil {
		t.Fatalf("right click Name header: %v", err)
	}
	if headerCol != 0 || !slices.Equal(tt.Menu(), []string{"Sort Descending", "-", "More"}) {
		t.Fatalf("header menu col=%d items=%v", headerCol, tt.Menu())
	}
	if err := tt.ChooseMenuItem("Sort Descending"); err != nil {
		t.Fatalf("choose header menu item: %v", err)
	}
	if !headerActionRan {
		t.Fatal("HeaderMenu OnClick did not execute")
	}

	// Right-clicking past the last column passes col == -1.
	kindHdr := headerBounds(t, tt, "Kind")
	tt.RightClickAt(420, kindHdr.Y+kindHdr.H/2)
	if headerCol != -1 || len(tt.Menu()) == 0 {
		t.Fatalf("header menu past columns got col=%d items=%v", headerCol, tt.Menu())
	}
	tt.CloseMenu()

	// Right-clicking an unselected cell selects its row and opens CellMenu.
	if err := tt.RightClick("r2c1"); err != nil {
		t.Fatalf("right click cell r2c1: %v", err)
	}
	if cellRow != 2 || cellCol != 1 || !chosen.Has(2) || !slices.Equal(tt.Menu(), []string{"Inspect", "Delete"}) {
		t.Fatalf("cell menu row=%d col=%d chosenHas2=%v menu=%v", cellRow, cellCol, chosen.Has(2), tt.Menu())
	}
	if err := tt.ChooseMenuItem("Inspect"); err != nil {
		t.Fatalf("choose cell menu item: %v", err)
	}
	if !inspected {
		t.Fatal("CellMenu Inspect OnClick did not execute")
	}
}

func TestDataTableLayoutResizeAndPersistence(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	layout := ui.TableLayout{}
	var reported []ui.TableLayout
	cols := []DataTableColumn{
		{Title: "Name"},
		{Title: "Kind", Width: 100, MinWidth: 60},
		{Title: "Size", Width: 80},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(500, 240).Background(canvasRGB).Children(func() {
				DataTable(c, cols, 2, func(row, col int) {
					DataTableCellText(c, fmt.Sprintf("v-%d-%d", row, col))
				}, DataTableOptions{
					Layout: &layout,
					OnLayoutChanged: func(l ui.TableLayout) {
						reported = append(reported, l)
					},
					Label: "Resizable table",
				}).Grow(1).FillWidth()
			})
		})
	}, 500, 240)

	kind := headerBounds(t, tt, "Kind")
	edge := kind.X + kind.W - 2
	y := kind.Y + kind.H/2

	tt.Press(edge, y)
	tt.Move(edge+15, y)
	tt.Move(edge+30, y)
	if len(reported) != 0 {
		t.Fatalf("OnLayoutChanged fired prematurely during drag: %d times", len(reported))
	}
	tt.Release(edge+30, y)

	if len(reported) != 1 {
		t.Fatalf("OnLayoutChanged fired %d times on release, want 1", len(reported))
	}
	if layout.Widths["Kind"] != 130 || reported[0].Widths["Kind"] != 130 {
		t.Fatalf("Kind width synced=%v reported=%v, want 130", layout.Widths["Kind"], reported[0].Widths["Kind"])
	}
}

func TestDataTableRTLDisabledReduceMotionAndAccess(t *testing.T) {
	// 1. RTL mirrors column order and numeric alignment, and verifies Access roles.
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	cols := []DataTableColumn{
		{Title: "Idx", Width: 60, RowHeader: true},
		{Title: "Item", Width: 160, Sortable: true},
		{Title: "Amount", Width: 120, Numeric: true},
	}
	selected := 0
	var sort ui.SortOrder

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(440, 220).Background(canvasRGB).Children(func() {
				DataTable(c, cols, 3, func(row, col int) {
					switch col {
					case 0:
						DataTableCellText(c, fmt.Sprintf("#%d", row))
					case 1:
						DataTableCellText(c, fmt.Sprintf("Item %d", row))
					case 2:
						DataTableCellText(c, fmt.Sprintf("$%d", (row+1)*10), DataTableCellOptions{Numeric: true})
					}
				}, DataTableOptions{
					Selected: &selected,
					Sort:     &sort,
					Hairline: true,
					RowLabel: func(row int) string { return fmt.Sprintf("Item %d", row) },
					Label:    "Invoice table",
				}).Grow(1).FillWidth()
			})
		})
	}, 440, 220)

	if tt.FramePending() {
		t.Fatal("ReduceMotion DataTable left FramePending true")
	}

	// Verify accessibility roles exist.
	if tbl, ok := tt.AccessByRole(ui.RoleTable, "Invoice table"); !ok || !tbl.Selectable {
		t.Fatalf("RoleTable access=%+v ok=%v", tbl, ok)
	}
	if _, ok := tt.AccessByRole(ui.RoleColumnHeader, "Item"); !ok {
		t.Fatal("RoleColumnHeader Item missing")
	}
	if _, ok := tt.AccessByRole(ui.RoleRow, "Item 0"); !ok {
		t.Fatal("RoleRow Item 0 missing")
	}
	if _, ok := tt.AccessByRole(ui.RoleRowHeader, "#0"); !ok {
		t.Fatal("RoleRowHeader #0 missing")
	}
	if _, ok := tt.AccessByRole(ui.RoleCell, ""); !ok {
		t.Fatal("RoleCell missing from access tree")
	}
	if _, ok := tt.Access("$10"); !ok {
		t.Fatal("cell text $10 missing from access tree")
	}

	itemLTR := headerBounds(t, tt, "Item")
	amtLTR := headerBounds(t, tt, "Amount")
	if !(itemLTR.X < amtLTR.X) {
		t.Fatalf("LTR header order broken: Item.X=%v Amount.X=%v", itemLTR.X, amtLTR.X)
	}

	tt.SetDirection(ui.RightToLeft)
	itemRTL := headerBounds(t, tt, "Item")
	amtRTL := headerBounds(t, tt, "Amount")
	if !(amtRTL.X < itemRTL.X) {
		t.Fatalf("RTL header order not mirrored: Amount.X=%v Item.X=%v", amtRTL.X, itemRTL.X)
	}
	amtTextRTL, ok := tt.Find("$10")
	if !ok {
		t.Fatal("RTL numeric cell text $10 missing")
	}
	// In RTL, ui.End maps to the left edge of the Amount column.
	if amtTextRTL.X > amtRTL.X+amtRTL.W/2 {
		t.Fatalf("RTL numeric cell text X=%v not aligned toward left (End) in Amount header %+v", amtTextRTL.X, amtRTL)
	}

	// 2. Disabled blocks sort, row click, keyboard nav, and submit.
	disSort := ui.SortOrder{}
	disSel := -1
	disClicks, disSubmits, disSortChanges := 0, 0, 0
	ttDis := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(380, 220).Background(canvasRGB).Children(func() {
				DataTable(c, []DataTableColumn{{Title: "Name", Sortable: true}}, 3, func(row, col int) {
					DataTableCellText(c, fmt.Sprintf("Locked %d", row))
				}, DataTableOptions{
					Selected:      &disSel,
					Sort:          &disSort,
					RowLabel:      func(row int) string { return fmt.Sprintf("Locked %d", row) },
					OnRowClick:    func(row int) { disClicks++ },
					OnRowSubmit:   func(row int) { disSubmits++ },
					OnSortChanged: func(s ui.SortOrder, by []ui.SortOrder) { disSortChanges++ },
					Label:         "Disabled table",
					Disabled:      true,
				}).Grow(1).FillWidth()
			})
		})
	}, 380, 220)

	disTbl, ok := ttDis.AccessByRole(ui.RoleTable, "Disabled table")
	if !ok || !disTbl.Disabled {
		t.Fatalf("disabled RoleTable access=%+v ok=%v", disTbl, ok)
	}
	nameHdr := headerBounds(t, ttDis, "Name")
	ttDis.ClickAt(center(nameHdr))
	row0, _ := ttDis.Find("Locked 0")
	ttDis.ClickAt(center(row0))
	ttDis.ClickAt(center(row0))
	ttDis.Key(0, ui.KeyDown)
	ttDis.Key(0, ui.KeyEnter)
	if disSort != (ui.SortOrder{}) || disSel != -1 || disClicks != 0 || disSubmits != 0 || disSortChanges != 0 {
		t.Fatalf("disabled table mutated state: sort=%+v sel=%d clicks=%d submits=%d sortChanges=%d",
			disSort, disSel, disClicks, disSubmits, disSortChanges)
	}
}

func headerBounds(t *testing.T, tt *ui.Tester, title string) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleColumnHeader, title)
	if !ok {
		t.Fatalf("column header %q not in access tree", title)
	}
	return info.Bounds
}

func TestDataTableColumnCacheUpdatesInPlaceChanges(t *testing.T) {
	columns := []DataTableColumn{{Title: "Count", Width: 100}, {Title: "Name"}}
	var track dataTableTrack
	cached := track.updateColumns(columns)
	if same := track.updateColumns(columns); &same[0] != &cached[0] {
		t.Fatal("unchanged columns should reuse the native column slice")
	}

	columns[0].Numeric = true
	columns[0].Width = 144
	updated := track.updateColumns(columns)
	if updated[0].Align != ui.End || updated[0].Width != 144 || !track.columns[0].Numeric {
		t.Fatalf("in-place change did not refresh column metadata: %+v", updated[0])
	}
	if got := track.updateColumns(nil); len(got) != 0 {
		t.Fatalf("removing all columns left stale columns: %+v", got)
	}
}
