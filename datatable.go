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
	// tokens.ShapeLarge (16 DIP).
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
	dataTableThemeKey    struct{}
	dataTableStateKey    struct{}
	dataTableLeadKey     struct{}
	dataTableTrackKey    struct{}
	dataTableOverrideKey struct{}
)

type dataTableStateOverrides struct {
	target *ui.ListState

	overrodeKey bool
	baseKey     func(int) any

	overrodeSelected bool
	baseSelected     *int

	overrodeSelection bool
	baseSelection     ui.Selector

	overrodeLabel bool
	baseLabel     func(int) string

	overrodeHeader bool
	baseHeader     func(int) bool

	overrodeSort bool
	baseSort     *ui.SortOrder

	overrodeSortBy bool
	baseSortBy     *[]ui.SortOrder

	overrodeRoundRows bool
	baseRoundRows     bool

	overrodeCellSelect bool
	baseCellSelect     bool

	overrodeCellsStyle bool
	autoCellsStyle     bool
	baseCellsStyle     ui.CellsStyle
	lastAutoCellsStyle ui.CellsStyle

	overrodeHeaderMenu bool
	baseHeaderMenu     func(*ui.Menu, int)

	overrodeCellMenu bool
	baseCellMenu     func(*ui.Menu, int, int)
}

func (ov *dataTableStateOverrides) restore() {
	s := ov.target
	if s == nil {
		return
	}
	if ov.overrodeKey {
		s.Key = ov.baseKey
	}
	if ov.overrodeSelected {
		s.Selected = ov.baseSelected
	}
	if ov.overrodeSelection {
		s.Selection = ov.baseSelection
	}
	if ov.overrodeLabel {
		s.Label = ov.baseLabel
	}
	if ov.overrodeHeader {
		s.Header = ov.baseHeader
	}
	if ov.overrodeSort {
		s.Sort = ov.baseSort
	}
	if ov.overrodeSortBy {
		s.SortBy = ov.baseSortBy
	}
	if ov.overrodeRoundRows {
		s.RoundRows = ov.baseRoundRows
	}
	if ov.overrodeCellSelect {
		s.CellSelect = ov.baseCellSelect
	}
	if ov.overrodeCellsStyle || (ov.autoCellsStyle && s.CellsStyle == ov.lastAutoCellsStyle) {
		s.CellsStyle = ov.baseCellsStyle
	}
	if ov.overrodeHeaderMenu {
		s.HeaderMenu = ov.baseHeaderMenu
	}
	if ov.overrodeCellMenu {
		s.CellMenu = ov.baseCellMenu
	}
	*ov = dataTableStateOverrides{}
}

func dataTableDefaultCellsStyle(th *Theme) ui.CellsStyle {
	cs := th.Colors
	return ui.CellsStyle{
		Cell:     cs.PrimaryContainer,
		CellText: cs.OnPrimaryContainer,
		Row:      cs.SecondaryContainer.Alpha(0.45),
		Stripe:   cs.OnSurface.Alpha(0.03),
	}
}

type dataTableTrack struct {
	hasLayout          bool
	lastSyncedLayout   ui.TableLayout
	lastReportedLayout ui.TableLayout
	columns            []DataTableColumn
	uiColumns          []ui.TableColumn
	sortBeforeBuild    []ui.SortOrder
	clickedRow         int
	doubleClickedRow   int
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
		shapeScale = tokens.ShapeLarge
	}
	surfOpt := SurfaceOptions{
		Hairline:   o.Hairline,
		ShapeScale: shapeScale,
		Clip:       true,
	}
	if !o.Hairline {
		if hcW, hcCol := highContrastBorder(th); hcW > 0 {
			surfOpt.BorderWidth = hcW
			surfOpt.BorderColor = hcCol
		}
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

	uiCols := track.updateColumns(columns)
	headerBg := SurfaceColorForElevation(th, tokens.Level2)
	if o.Elevation >= tokens.Level2 {
		headerBg = SurfaceColorForElevation(th, min(o.Elevation+1, tokens.Level5))
	}
	activeSortCol := func(col int) bool {
		if col < 0 || col >= len(uiCols) {
			return false
		}
		id := uiCols[col].ID
		if id == "" {
			id = uiCols[col].Title
		}
		if id == "" {
			return false
		}
		if state.Sort != nil && state.Sort.Column == id {
			return true
		}
		if state.SortBy != nil {
			for _, so := range *state.SortBy {
				if so.Column == id {
					return true
				}
			}
		}
		return false
	}
	userHeaderCell := o.HeaderCell
	prevHeaderCell := state.HeaderCell
	if userHeaderCell == nil && o.State != nil {
		userHeaderCell = prevHeaderCell
	}
	primaryInk := th.Colors.Primary
	state.HeaderCell = func(cell ui.Element, col int) {
		cell.Background(headerBg)
		if !disabled && activeSortCol(col) && primaryInk.A > 0 {
			cell.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if r.W <= 0 || r.H <= 2 {
					return
				}
				p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 2, W: r.W, H: 2}, primaryInk, 0)
			})
		}
		if userHeaderCell != nil {
			userHeaderCell(cell, col)
		}
	}
	if o.State != nil {
		defer func() {
			state.HeaderCell = prevHeaderCell
		}()
	}
	onRowClick := !disabled && o.OnRowClick != nil
	onRowSubmit := !disabled && o.OnRowSubmit != nil
	monoFont := th.MonoFont
	if monoFont == "" {
		monoFont = "monospace"
	}

	var prevSort ui.SortOrder
	var prevSortBy []ui.SortOrder
	if !disabled && o.OnSortChanged != nil {
		if state.Sort != nil {
			prevSort = *state.Sort
		}
		if state.SortBy != nil {
			prevSortBy = *state.SortBy
		}
	}
	// Cell builders may update the caller's sort slice in place.
	if !slices.Equal(track.sortBeforeBuild, prevSortBy) {
		if len(prevSortBy) == 0 || cap(track.sortBeforeBuild)/2 > len(prevSortBy) {
			track.sortBeforeBuild = slices.Clone(prevSortBy)
		} else {
			clear(track.sortBeforeBuild)
			track.sortBeforeBuild = append(track.sortBeforeBuild[:0], prevSortBy...)
		}
	}

	track.clickedRow, track.doubleClickedRow = -1, -1

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
					numeric, mono := false, false
					colAlign := ui.Start
					if col >= 0 && col < len(uiCols) {
						colMeta := &track.columns[col]
						numeric, mono = colMeta.Numeric, colMeta.Mono
						colAlign = uiCols[col].Align
					}
					if !numeric && !mono && !onRowClick && !onRowSubmit {
						cell(row, col)
						return
					}
					cellBox := ui.Row(c).
						Grow(1).FillHeight().
						AlignItems(ui.Center).
						Justify(colAlign).
						Shrink(1).MinWidth(0)
					if numeric {
						TabularNums(cellBox)
					}
					if mono {
						cellBox.Font(monoFont)
					}
					// RowClicked marks the upstream row clickable for builds with
					// either callback, including submit-only tables.
					if (onRowClick || onRowSubmit) && cellBox.RowClicked() && onRowClick {
						track.clickedRow = row
					}
					if onRowSubmit && cellBox.RowDoubleClicked() {
						track.doubleClickedRow = row
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
					state, tableEl, tableChanged, count, o, track, prevSort, disabled,
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

	t := *origUI
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
	cached := ui.Local(root, dataTableThemeKey{}, func() ui.Theme { return ui.Theme{} })
	if *cached != t {
		*cached = t
	}
	return cached
}

func resolveDataTableState(root ui.Element, th *Theme, o DataTableOptions) *ui.ListState {
	ov := ui.Local(root, dataTableOverrideKey{}, func() dataTableStateOverrides { return dataTableStateOverrides{} })
	var s *ui.ListState
	if o.State != nil {
		s = o.State
		if ov.target != nil && ov.target != s {
			ov.restore()
		}
		ov.target = s

		if o.ItemKey != nil {
			if !ov.overrodeKey {
				ov.baseKey = s.Key
				ov.overrodeKey = true
			}
			s.Key = o.ItemKey
		} else if ov.overrodeKey {
			s.Key = ov.baseKey
			ov.baseKey = nil
			ov.overrodeKey = false
		}

		if o.Selected != nil {
			if !ov.overrodeSelected {
				ov.baseSelected = s.Selected
				ov.overrodeSelected = true
			}
			s.Selected = o.Selected
		} else if ov.overrodeSelected {
			s.Selected = ov.baseSelected
			ov.baseSelected = nil
			ov.overrodeSelected = false
		}

		if o.Selection != nil {
			if !ov.overrodeSelection {
				ov.baseSelection = s.Selection
				ov.overrodeSelection = true
			}
			s.Selection = o.Selection
		} else if ov.overrodeSelection {
			s.Selection = ov.baseSelection
			ov.baseSelection = nil
			ov.overrodeSelection = false
		}

		if o.RowLabel != nil {
			if !ov.overrodeLabel {
				ov.baseLabel = s.Label
				ov.overrodeLabel = true
			}
			s.Label = o.RowLabel
		} else if ov.overrodeLabel {
			s.Label = ov.baseLabel
			ov.baseLabel = nil
			ov.overrodeLabel = false
		}

		if o.IsHeader != nil {
			if !ov.overrodeHeader {
				ov.baseHeader = s.Header
				ov.overrodeHeader = true
			}
			s.Header = o.IsHeader
		} else if ov.overrodeHeader {
			s.Header = ov.baseHeader
			ov.baseHeader = nil
			ov.overrodeHeader = false
		}

		if o.Sort != nil {
			if !ov.overrodeSort {
				ov.baseSort = s.Sort
				ov.overrodeSort = true
			}
			s.Sort = o.Sort
		} else if ov.overrodeSort {
			s.Sort = ov.baseSort
			ov.baseSort = nil
			ov.overrodeSort = false
		}

		if o.SortBy != nil {
			if !ov.overrodeSortBy {
				ov.baseSortBy = s.SortBy
				ov.overrodeSortBy = true
			}
			s.SortBy = o.SortBy
		} else if ov.overrodeSortBy {
			s.SortBy = ov.baseSortBy
			ov.baseSortBy = nil
			ov.overrodeSortBy = false
		}

		if o.RoundRows {
			if !ov.overrodeRoundRows {
				ov.baseRoundRows = s.RoundRows
				ov.overrodeRoundRows = true
			}
			s.RoundRows = true
		} else if ov.overrodeRoundRows {
			s.RoundRows = ov.baseRoundRows
			ov.overrodeRoundRows = false
		}

		if o.CellSelect {
			if !ov.overrodeCellSelect {
				ov.baseCellSelect = s.CellSelect
				ov.overrodeCellSelect = true
			}
			s.CellSelect = true
		} else if ov.overrodeCellSelect {
			s.CellSelect = ov.baseCellSelect
			ov.overrodeCellSelect = false
		}

		if o.CellsStyle != (ui.CellsStyle{}) {
			if !ov.overrodeCellsStyle && !ov.autoCellsStyle {
				ov.baseCellsStyle = s.CellsStyle
			}
			s.CellsStyle = o.CellsStyle
			ov.overrodeCellsStyle = true
			ov.autoCellsStyle = false
		} else {
			if ov.overrodeCellsStyle {
				s.CellsStyle = ov.baseCellsStyle
				ov.overrodeCellsStyle = false
			}
			if ov.autoCellsStyle && s.CellsStyle != ov.lastAutoCellsStyle {
				// Caller explicitly updated s.CellsStyle directly; adopt as caller-owned.
				ov.autoCellsStyle = false
			}
			if s.CellSelect {
				if s.CellsStyle == (ui.CellsStyle{}) || ov.autoCellsStyle {
					if !ov.autoCellsStyle {
						ov.baseCellsStyle = ui.CellsStyle{}
					}
					auto := dataTableDefaultCellsStyle(th)
					s.CellsStyle = auto
					ov.autoCellsStyle = true
					ov.lastAutoCellsStyle = auto
				}
			} else if ov.autoCellsStyle {
				s.CellsStyle = ov.baseCellsStyle
				ov.autoCellsStyle = false
			}
		}

		if o.HeaderMenu != nil {
			if !ov.overrodeHeaderMenu {
				ov.baseHeaderMenu = s.HeaderMenu
				ov.overrodeHeaderMenu = true
			}
			s.HeaderMenu = o.HeaderMenu
		} else if ov.overrodeHeaderMenu {
			s.HeaderMenu = ov.baseHeaderMenu
			ov.baseHeaderMenu = nil
			ov.overrodeHeaderMenu = false
		}

		if o.CellMenu != nil {
			if !ov.overrodeCellMenu {
				ov.baseCellMenu = s.CellMenu
				ov.overrodeCellMenu = true
			}
			s.CellMenu = o.CellMenu
		} else if ov.overrodeCellMenu {
			s.CellMenu = ov.baseCellMenu
			ov.baseCellMenu = nil
			ov.overrodeCellMenu = false
		}
	} else {
		if ov.target != nil {
			ov.restore()
		}
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
		if s.CellSelect && s.CellsStyle == (ui.CellsStyle{}) {
			s.CellsStyle = dataTableDefaultCellsStyle(th)
		}
	}

	if isPositiveFiniteF32(o.RowHeight) {
		s.RowHeight = o.RowHeight
	} else {
		s.RowHeight = dataTableRowHeight(th.Density)
	}
	return s
}

func (t *dataTableTrack) updateColumns(columns []DataTableColumn) []ui.TableColumn {
	if slices.Equal(t.columns, columns) {
		return t.uiColumns
	}
	if cap(t.columns)/2 > len(columns) {
		t.columns, t.uiColumns = nil, nil
	}
	clear(t.columns)
	clear(t.uiColumns)
	t.columns = append(t.columns[:0], columns...)
	t.uiColumns = slices.Grow(t.uiColumns[:0], len(columns))[:len(columns)]
	for i := range columns {
		c := &columns[i]
		align := c.Align
		if c.Numeric && align == ui.Start {
			align = ui.End
		}
		t.uiColumns[i] = ui.TableColumn{
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
	return t.uiColumns
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
	state *ui.ListState,
	tableEl ui.Element,
	tableChanged bool,
	count int,
	o DataTableOptions,
	track *dataTableTrack,
	prevSort ui.SortOrder,
	disabled bool,
) {
	clickedRow, doubleClickedRow := track.clickedRow, track.doubleClickedRow
	var curSort ui.SortOrder
	var curSortBy []ui.SortOrder
	if state.Sort != nil {
		curSort = *state.Sort
	}
	if state.SortBy != nil {
		curSortBy = *state.SortBy
	}
	if !disabled && o.OnSortChanged != nil && (curSort != prevSort || !slices.Equal(curSortBy, track.sortBeforeBuild)) {
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
