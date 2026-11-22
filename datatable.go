package md3e

import (
	"maps"
	"slices"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// DataTableColumn describes a column of a DataTable.
type DataTableColumn struct {
	// Title heads the column in the sticky table header.
	Title string
	// ID identifies the column for layout persistence (ListState.Columns) and
	// sorting (SortOrder.Column). Title is used when ID is empty.
	ID string
	// Width is the initial column width in DIP, or 0 to share remaining space.
	Width float32
	// MinWidth and MaxWidth constrain user column resizing in DIP (0 = none).
	MinWidth float32
	MaxWidth float32
	// Align controls cell content alignment. When Numeric is true and Align is
	// left at its zero value (ui.Start), it resolves to ui.End (mirrored in RTL).
	Align ui.Align
	// Sortable allows clicking the column header to sort rows when Sort or
	// SortBy is configured.
	Sortable bool
	// Fixed locks column order and width so users cannot drag or resize it.
	Fixed bool
	// RowHeader marks the column as row headers (ui.RoleRowHeader) for screen
	// readers and whole-row selection in CellSelect mode.
	RowHeader bool
	// Numeric enables tabular lining figures ("tnum") on cell text and defaults
	// cell alignment to ui.End.
	Numeric bool
	// Mono applies CurrentTheme(c).MonoFont (or system "monospace") to the
	// column's cell context.
	Mono bool
}

// DataTableCellOptions configures DataTableCellText.
type DataTableCellOptions struct {
	// Role selects the MD3E typography role when HasRole is true (or when Role
	// is non-zero). Zero with HasRole=false uses Density-aware BodyMedium /
	// BodySmall.
	Role       tokens.TypeRole
	HasRole    bool
	Emphasized bool
	// Numeric enables OpenType tabular lining figures ("tnum").
	Numeric bool
	// Mono selects CurrentTheme(c).MonoFont (or system "monospace").
	Mono bool
	// Truncate selects end, middle, or start ellipsis when text overflows.
	Truncate TruncateMode
	// MaxLines limits lines (defaults to single-line truncation when <= 0).
	MaxLines int
	// Muted tints the text with OnSurfaceVariant when Color is transparent.
	Muted bool
	// Color overrides the foreground text color when non-transparent (A > 0).
	// Leaving Color zero preserves selected-row foreground inheritance.
	Color ui.Color
}

// DataTableMenuAction describes one item in a DataTable header or cell context
// menu bridged via ApplyDataTableMenuActions.
type DataTableMenuAction struct {
	Label     string
	Subtitle  string
	Trailing  string
	Separator bool
	Checkable bool
	Radio     bool
	Checked   bool
	Danger    bool
	Disabled  bool
	Shortcut  ui.Modifiers
	Key       ui.Key
	Submenu   []DataTableMenuAction
	OnClick   func()
}

// ApplyDataTableMenuActions populates a native ui.Menu from declarative
// DataTableMenuAction entries, handling separators, check/radio marks, danger
// styling, shortcuts, submenus, and click dispatch.
func ApplyDataTableMenuActions(m *ui.Menu, actions []DataTableMenuAction) {
	if m == nil {
		return
	}
	for i := range actions {
		a := &actions[i]
		if a.Separator {
			m.Separator()
			continue
		}
		if len(a.Submenu) > 0 {
			sub := a.Submenu
			it := m.Submenu(a.Label, func(child *ui.Menu) {
				ApplyDataTableMenuActions(child, sub)
			})
			applyMenuItemProps(it, a)
			continue
		}
		it := m.Item(a.Label)
		switch {
		case a.Radio:
			it.Radio(a.Checked)
		case a.Checkable || a.Checked:
			it.Checked(a.Checked)
		}
		applyMenuItemProps(it, a)
		if a.Key != 0 {
			it.Shortcut(a.Shortcut, a.Key)
		}
		if it.Chosen() && !a.Disabled && a.OnClick != nil {
			a.OnClick()
		}
	}
}

func applyMenuItemProps(it *ui.MenuItem, a *DataTableMenuAction) {
	if it == nil {
		return
	}
	if a.Subtitle != "" {
		it.Subtitle(a.Subtitle)
	}
	if a.Trailing != "" {
		it.Trailing(a.Trailing)
	}
	if a.Danger {
		it.Danger()
	}
	if a.Disabled {
		it.Disabled(true)
	}
}

// DataTableOptions configures a Material Design 3 Expressive DataTable.
type DataTableOptions struct {
	// State optionally provides a caller-owned ui.ListState for scroll control,
	// sort state, column layout, cell selection, and Move animations.
	State *ui.ListState
	// ItemKey maps a row index to a stable comparable key so selection and row
	// state survive insertions, deletions, and sorting.
	ItemKey func(row int) any
	// Selected binds single-row selection (index or -1 for none).
	Selected *int
	// Selection binds multi-row selection (*ui.Selection[K]).
	Selection ui.Selector
	// RowLabel returns the accessible and type-select label for row.
	RowLabel func(row int) string
	// IsHeader reports whether row renders as a sticky full-width section
	// header.
	IsHeader func(row int) bool
	// Sort binds single-column sort state.
	Sort *ui.SortOrder
	// SortBy binds multi-column sort state (kept synchronized with Sort if both
	// are provided).
	SortBy *[]ui.SortOrder
	// Layout binds column order and user-resized widths (bidirectional sync
	// with ListState.Columns).
	Layout *ui.TableLayout
	// RoundRows rounds hovered and selected row corners using the MD3E small
	// corner token.
	RoundRows bool
	// CellSelect enables spreadsheet-style block cell selection when Selection
	// is also set.
	CellSelect bool
	// CellsStyle customizes cell selection and striped row colors.
	CellsStyle ui.CellsStyle
	// Hairline draws a subtle 1 DIP MD3E hairline border around the table shell.
	Hairline bool
	// Elevation selects the MD3E tonal surface tier and shadows for the table
	// container. Zero defaults to tonal Level1 (SurfaceContainerLow) without
	// drop shadows.
	Elevation tokens.ElevationLevel
	// ShapeScale selects the outer container corner token. Zero defaults to
	// tokens.ShapeMedium.
	ShapeScale tokens.ShapeScale
	// RowHeight fixes row height in DIP when positive and finite. When <= 0 or
	// non-finite, automatically follows Theme.Density (Comfortable: 48,
	// Desktop: 38, Compact: 32).
	RowHeight float32
	// HeaderMenu builds a context menu when right-clicking a column header
	// (col == -1 for the header area past the last column).
	HeaderMenu func(m *ui.Menu, col int)
	// HeaderCell appends custom trailing badges or tooltips inside each column
	// header cell.
	HeaderCell func(cell ui.Element, col int)
	// CellMenu builds a context menu when right-clicking a data cell.
	CellMenu func(m *ui.Menu, row, col int)
	// OnRowClick runs when a data row is primary-clicked.
	OnRowClick func(row int)
	// OnRowSubmit runs when a data row is double-clicked or activated with Enter.
	OnRowSubmit func(row int)
	// OnSortChanged runs when Sort or SortBy changes after a header click.
	OnSortChanged func(sort ui.SortOrder, sortBy []ui.SortOrder)
	// OnLayoutChanged runs once a column drag, resize, or double-click fit
	// commits a new TableLayout.
	OnLayoutChanged func(layout ui.TableLayout)
	// Label names the table for assistive technology.
	Label string
	// Disabled disables table interaction and dims content using MD3E disabled
	// opacity tokens.
	Disabled bool
}

type (
	dataTableThemeKey struct{}
	dataTableStateKey struct{}
	dataTableLeadKey  struct{}
	dataTableTrackKey struct{}
)

type dataTableTrack struct {
	hasLayout          bool
	lastSyncedLayout   ui.TableLayout
	lastReportedLayout ui.TableLayout
}

// DataTableCellText builds a single-line (or bounded-line) table cell text
// element styled with MD3E typography while preserving row foreground color
// inheritance so selected rows automatically transition to OnSecondaryContainer.
func DataTableCellText(c *ui.Context, text string, opts ...DataTableCellOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: DataTableCellText takes at most one DataTableCellOptions")
	}
	var o DataTableCellOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	role := tokens.BodyMedium
	if th.Density.Normalize() == tokens.DensityCompact {
		role = tokens.BodySmall
	}
	if o.HasRole || o.Role != 0 {
		role = o.Role
		if role < 0 || role >= tokens.TypeRoleCount {
			role = tokens.BodyMedium
		}
	}

	m := typeStyleOfWithTheme(th, role, o.Emphasized)
	el := ui.Text(c, text).
		FontSize(m.Size).
		FontWeight(m.Weight).
		FixedLineHeight(m.LineHeight).
		LetterSpacing(m.Tracking)

	if o.Mono {
		if th.MonoFont != "" {
			el.Font(th.MonoFont)
		} else {
			el.Font("monospace")
		}
	}
	if o.Numeric {
		TabularNums(el)
	}
	if o.MaxLines > 0 {
		el.MaxLines(o.MaxLines)
	} else {
		el.SingleLine()
	}
	switch o.Truncate {
	case TruncateMiddle:
		el.TruncateMiddle()
	case TruncateStart:
		el.TruncateStart()
	}

	if o.Color.A > 0 {
		el.TextColor(o.Color)
	} else if o.Muted {
		el.TextColor(th.Colors.OnSurfaceVariant)
	}
	return el
}

// DataTable builds a Material Design 3 Expressive data table over ui.Table,
// preserving sticky headers, column drag reordering, edge resizing,
// double-click auto-fit, single/multi-column sorting, cell block selection,
// horizontal/vertical virtualization, and context menus inside an MD3E tonal
// surface container.
func DataTable(c *ui.Context, columns []DataTableColumn, count int, cell func(row, col int), opts ...DataTableOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: DataTable takes at most one DataTableOptions")
	}
	var o DataTableOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	count = max(count, 0)

	th := CurrentTheme(c)
	origUI := c.Theme()
	if origUI == nil {
		origUI = th.ToUITheme()
	}

	shapeScale := o.ShapeScale
	if shapeScale == tokens.ShapeNone {
		shapeScale = tokens.ShapeMedium
	}
	surfOpt := SurfaceOptions{
		Hairline:   o.Hairline,
		ShapeScale: shapeScale,
		Clip:       true,
	}
	if o.Elevation > tokens.Level0 {
		surfOpt.Elevation = o.Elevation
	} else {
		surfOpt.TonalElevation = tokens.Level1
		surfOpt.DisableShadow = true
	}

	root := Surface(c, surfOpt).DimDisabled(false).MinWidth(0).MinHeight(0)
	if o.Disabled {
		root.Disabled(true)
	}
	disabled := o.Disabled || root.IsDisabled()

	containerBg := resolveSurfaceBackground(th, surfOpt)
	tableTheme := buildDataTableUITheme(root, origUI, th, containerBg, disabled)
	registerThemeAlias(tableTheme, th)

	state := resolveDataTableState(root, th, o)
	track := ui.Local(root, dataTableTrackKey{}, func() dataTableTrack { return dataTableTrack{} })
	syncDataTableLayoutIn(state, o, track)

	uiCols, colAligns := convertDataTableColumns(columns)
	monoFont := th.MonoFont
	if monoFont == "" {
		monoFont = "monospace"
	}

	var prevSort ui.SortOrder
	var prevSortBy []ui.SortOrder
	if state.Sort != nil {
		prevSort = *state.Sort
	}
	if state.SortBy != nil {
		prevSortBy = slices.Clone(*state.SortBy)
	}

	clickedRow, doubleClickedRow := -1, -1

	root.Children(func() {
		ui.Themed(c, tableTheme).
			Background(ui.Transparent).
			Grow(1).FillWidth().MinWidth(0).MinHeight(0).
			Children(func() {
				wrappedCell := func(row, col int) {
					if cell == nil {
						return
					}
					if state.Header != nil && state.Header(row) {
						cell(row, 0)
						return
					}
					var colMeta DataTableColumn
					colAlign := ui.Start
					if col >= 0 && col < len(columns) {
						colMeta = columns[col]
						colAlign = colAligns[col]
					}
					needWrap := colMeta.Numeric || colMeta.Mono || (!disabled && (o.OnRowClick != nil || o.OnRowSubmit != nil))
					if !needWrap {
						cell(row, col)
						return
					}
					cellBox := ui.Row(c).
						Grow(1).FillHeight().
						AlignItems(ui.Center).
						Justify(colAlign).
						Shrink(1).MinWidth(0)
					if colMeta.Numeric {
						TabularNums(cellBox)
					}
					if colMeta.Mono {
						cellBox.Font(monoFont)
					}
					if !disabled && (o.OnRowClick != nil || o.OnRowSubmit != nil) {
						if cellBox.RowClicked() && o.OnRowClick != nil {
							clickedRow = row
						}
						if o.OnRowSubmit != nil && cellBox.RowDoubleClicked() {
							doubleClickedRow = row
						}
					}
					cellBox.Children(func() {
						cell(row, col)
					})
				}

				tableEl := ui.Table(c, state, uiCols, count, wrappedCell).
					Grow(1).FillWidth().MinWidth(0).MinHeight(0).
					Disabled(disabled)
				if o.Label != "" {
					tableEl.Label(o.Label)
				}

				tableChanged := tableEl.Changed()
				dispatchDataTablePostBuild(
					th, state, tableEl, tableChanged, count, o, track,
					prevSort, prevSortBy, clickedRow, doubleClickedRow, disabled,
				)
			})
	})

	return root
}

func dataTableRowHeight(d tokens.Density) float32 {
	switch d.Normalize() {
	case tokens.DensityDesktop:
		return 38
	case tokens.DensityCompact:
		return 32
	default:
		return 48
	}
}

func buildDataTableUITheme(root ui.Element, origUI *ui.Theme, th *Theme, containerBg ui.Color, disabled bool) *ui.Theme {
	cs := th.Colors
	selectedBg := cs.SecondaryContainer
	selectedInk := cs.OnSecondaryContainer
	if selectedBg.A == 0 {
		selectedBg = cs.PrimaryContainer
		selectedInk = cs.OnPrimaryContainer
	}
	hairline := resolveHairlineColor(th, ui.Color{})

	t := ui.Local(root, dataTableThemeKey{}, func() ui.Theme { return ui.Theme{} })
	*t = *origUI
	t.Background = ui.Transparent
	t.Surface = containerBg
	t.SurfaceHover = cs.OnSurface.Alpha(float32(tokens.StateHoverOpacity)).Over(containerBg)
	t.SurfacePressed = cs.OnSurface.Alpha(float32(tokens.StatePressedOpacity)).Over(containerBg)
	t.Border = hairline
	t.Radius = th.Shapes.Radius(tokens.ShapeSmall)
	t.Spacing = float32(th.Density.SpacingUnit())
	if cs.Primary.A > 0 {
		t.Focus = cs.Primary
	}

	if disabled {
		disabledInk := cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		t.Text = disabledInk
		t.TextMuted = disabledInk
		t.Accent = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		t.InactiveAccent = t.Accent
		t.AccentText = disabledInk
	} else {
		t.Text = cs.OnSurface
		t.TextMuted = cs.OnSurfaceVariant
		t.Accent = selectedBg
		t.InactiveAccent = selectedBg.Mix(containerBg, 0.35)
		t.AccentText = selectedInk
	}
	return t
}

func resolveDataTableState(root ui.Element, th *Theme, o DataTableOptions) *ui.ListState {
	var s *ui.ListState
	if o.State != nil {
		s = o.State
		if o.ItemKey != nil {
			s.Key = o.ItemKey
		}
		if o.Selected != nil {
			s.Selected = o.Selected
		}
		if o.Selection != nil {
			s.Selection = o.Selection
		}
		if o.RowLabel != nil {
			s.Label = o.RowLabel
		}
		if o.IsHeader != nil {
			s.Header = o.IsHeader
		}
		if o.Sort != nil {
			s.Sort = o.Sort
		}
		if o.SortBy != nil {
			s.SortBy = o.SortBy
		}
		if o.RoundRows {
			s.RoundRows = true
		}
		if o.CellSelect {
			s.CellSelect = true
		}
		if o.CellsStyle != (ui.CellsStyle{}) {
			s.CellsStyle = o.CellsStyle
		}
		if o.HeaderMenu != nil {
			s.HeaderMenu = o.HeaderMenu
		}
		if o.HeaderCell != nil {
			s.HeaderCell = o.HeaderCell
		}
		if o.CellMenu != nil {
			s.CellMenu = o.CellMenu
		}
	} else {
		s = ui.Local(root, dataTableStateKey{}, func() ui.ListState { return ui.ListState{} })
		lead := ui.Local(root, dataTableLeadKey{}, func() int { return -1 })
		s.Key = o.ItemKey
		s.Selection = o.Selection
		s.Label = o.RowLabel
		s.Header = o.IsHeader
		s.Sort = o.Sort
		s.SortBy = o.SortBy
		s.RoundRows = o.RoundRows
		s.CellSelect = o.CellSelect
		s.CellsStyle = o.CellsStyle
		s.HeaderMenu = o.HeaderMenu
		s.HeaderCell = o.HeaderCell
		s.CellMenu = o.CellMenu
		switch {
		case o.Selected != nil:
			s.Selected = o.Selected
		case o.Selection != nil:
			s.Selected = lead
		default:
			s.Selected = nil
		}
	}

	if isPositiveFiniteF32(o.RowHeight) {
		s.RowHeight = o.RowHeight
	} else {
		s.RowHeight = dataTableRowHeight(th.Density)
	}
	return s
}

func convertDataTableColumns(columns []DataTableColumn) ([]ui.TableColumn, []ui.Align) {
	uiCols := make([]ui.TableColumn, len(columns))
	aligns := make([]ui.Align, len(columns))
	for i := range columns {
		c := &columns[i]
		align := c.Align
		if c.Numeric && align == ui.Start {
			align = ui.End
		}
		aligns[i] = align
		uiCols[i] = ui.TableColumn{
			Title:     c.Title,
			ID:        c.ID,
			Width:     sanitizeNonNegF32(c.Width),
			MinWidth:  sanitizeNonNegF32(c.MinWidth),
			MaxWidth:  sanitizeNonNegF32(c.MaxWidth),
			Align:     align,
			Sortable:  c.Sortable,
			Fixed:     c.Fixed,
			RowHeader: c.RowHeader,
		}
	}
	return uiCols, aligns
}

func syncDataTableLayoutIn(state *ui.ListState, o DataTableOptions, track *dataTableTrack) {
	if o.Layout == nil {
		return
	}
	if !track.hasLayout {
		if len(o.Layout.Order) > 0 || len(o.Layout.Widths) > 0 {
			state.Columns = cloneTableLayout(*o.Layout)
			track.lastSyncedLayout = cloneTableLayout(*o.Layout)
		}
		return
	}
	if !tableLayoutEqual(*o.Layout, track.lastSyncedLayout) {
		state.Columns = cloneTableLayout(*o.Layout)
		track.lastSyncedLayout = cloneTableLayout(*o.Layout)
	}
}

func dispatchDataTablePostBuild(
	th *Theme,
	state *ui.ListState,
	tableEl ui.Element,
	tableChanged bool,
	count int,
	o DataTableOptions,
	track *dataTableTrack,
	prevSort ui.SortOrder,
	prevSortBy []ui.SortOrder,
	clickedRow, doubleClickedRow int,
	disabled bool,
) {
	_ = th
	var curSort ui.SortOrder
	var curSortBy []ui.SortOrder
	if state.Sort != nil {
		curSort = *state.Sort
	}
	if state.SortBy != nil {
		curSortBy = *state.SortBy
	}
	if !disabled && o.OnSortChanged != nil && (curSort != prevSort || !slices.Equal(curSortBy, prevSortBy)) {
		o.OnSortChanged(curSort, slices.Clone(curSortBy))
	}

	if o.Layout != nil && !tableLayoutEqual(state.Columns, track.lastSyncedLayout) {
		*o.Layout = cloneTableLayout(state.Columns)
		track.lastSyncedLayout = cloneTableLayout(state.Columns)
	}
	if !track.hasLayout {
		track.hasLayout = true
		track.lastSyncedLayout = cloneTableLayout(state.Columns)
		track.lastReportedLayout = cloneTableLayout(state.Columns)
	} else if !disabled && tableChanged && !tableLayoutEqual(state.Columns, track.lastReportedLayout) {
		track.lastReportedLayout = cloneTableLayout(state.Columns)
		if o.OnLayoutChanged != nil {
			o.OnLayoutChanged(cloneTableLayout(state.Columns))
		}
	}

	if !disabled && clickedRow >= 0 && o.OnRowClick != nil {
		o.OnRowClick(clickedRow)
	}
	if !disabled && o.OnRowSubmit != nil {
		submitted := tableEl.Submitted()
		if submitted || doubleClickedRow >= 0 {
			submittedRow := doubleClickedRow
			if state.Selected != nil && *state.Selected >= 0 && *state.Selected < count {
				submittedRow = *state.Selected
			}
			o.OnRowSubmit(submittedRow)
		}
	}
}

func cloneTableLayout(l ui.TableLayout) ui.TableLayout {
	return ui.TableLayout{
		Order:  slices.Clone(l.Order),
		Widths: maps.Clone(l.Widths),
	}
}

func tableLayoutEqual(a, b ui.TableLayout) bool {
	return slices.Equal(a.Order, b.Order) && maps.Equal(a.Widths, b.Widths)
}
