package md3e

import (
	"math"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// WindowSizeClass classifies the available layout width into the three
// Material Design 3 Adaptive breakpoints.
type WindowSizeClass int

const (
	// WindowCompact covers narrow viewports (< 600 DIP, e.g. phones and narrow panes).
	WindowCompact WindowSizeClass = iota
	// WindowMedium covers medium viewports (600..840 DIP, e.g. tablets and split windows).
	WindowMedium
	// WindowExpanded covers wide viewports (>= 840 DIP, e.g. desktop and landscape tablets).
	WindowExpanded
)

// Standard Material Design 3 Adaptive breakpoint thresholds in DIP.
const (
	windowMediumBreakpoint   float32 = 600
	windowExpandedBreakpoint float32 = 840
)

// String returns the canonical breakpoint name ("Compact", "Medium", or "Expanded").
func (s WindowSizeClass) String() string {
	switch s {
	case WindowMedium:
		return "Medium"
	case WindowExpanded:
		return "Expanded"
	default:
		return "Compact"
	}
}

// WindowSizeClassForWidth resolves the M3 Adaptive WindowSizeClass for width w
// in DIP. Non-positive, NaN, and -Inf widths return WindowCompact; +Inf returns
// WindowExpanded.
func WindowSizeClassForWidth(w float32) WindowSizeClass {
	f := float64(w)
	if math.IsNaN(f) || math.IsInf(f, -1) || w < windowMediumBreakpoint {
		return WindowCompact
	}
	if math.IsInf(f, 1) || w >= windowExpandedBreakpoint {
		return WindowExpanded
	}
	return WindowMedium
}

// WindowSizeClassOf returns the WindowSizeClass for c's current window width.
// Passing nil returns WindowCompact.
func WindowSizeClassOf(c *ui.Context) WindowSizeClass {
	if c == nil {
		return WindowCompact
	}
	w, _ := c.Size()
	return WindowSizeClassForWidth(w)
}

func resolveSizeClass(c *ui.Context, widthOverride float32) WindowSizeClass {
	if isPositiveFiniteF32(widthOverride) {
		return WindowSizeClassForWidth(widthOverride)
	}
	return WindowSizeClassOf(c)
}

// AdaptiveScaffoldOptions configures AdaptiveScaffold. The zero value adapts
// automatically between a bottom NavigationBar (Compact), a collapsed
// NavigationRail (Medium), and an expanded NavigationRail (Expanded).
type AdaptiveScaffoldOptions struct {
	// Selected points to the selected destination index; nil keeps local state.
	Selected *int
	// Changed runs when a navigation destination is chosen.
	Changed func(i int)
	// RailExpanded optionally owns the expanded state of NavigationRail. When
	// nil, WindowExpanded automatically expands the rail while WindowMedium
	// keeps it collapsed.
	RailExpanded *bool
	// RailExpandedWidth overrides the expanded NavigationRail width in DIP.
	RailExpandedWidth float32
	// NavigationLabels sets the label visibility mode of the Compact NavigationBar.
	NavigationLabels NavigationLabelMode
	// NavigationLabel names the navigation landmark for assistive technology;
	// "Navigation" when empty.
	NavigationLabel string
	// Header builds an optional full-width top header (such as WindowTitleBar
	// or TopAppBar) above the navigation and content region.
	Header func() ui.Element
	// RailHeader builds an optional header slot at the top of NavigationRail.
	RailHeader func() ui.Element
	// FAB builds an optional floating action button anchored at the bottom-end
	// corner of the content region (RTL-aware).
	FAB func() ui.Element
	// Background overrides the scaffold canvas fill; defaults to Theme.Colors.Surface.
	Background ui.Color
	// WidthOverride overrides c.Size()'s width when resolving WindowSizeClass.
	WidthOverride float32
	// Disabled disables navigation interactions.
	Disabled bool
}

type (
	adaptiveSelectedKey struct{}
	adaptiveHeaderKey   struct{}
	adaptiveNavRailKey  struct{}
	adaptiveNavBarKey   struct{}
	adaptiveContentKey  struct{}
	adaptiveFABKey      struct{}
)

// AdaptiveScaffold builds an M3 Adaptive application layout shell that switches
// seamlessly between bottom NavigationBar (WindowCompact), collapsed
// NavigationRail (WindowMedium), and expanded NavigationRail (WindowExpanded),
// while keeping the body content under a stable keyed container so local state
// and focus survive breakpoint transitions.
func AdaptiveScaffold(c *ui.Context, destinations []NavigationDestination, content func(), opts ...AdaptiveScaffoldOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: AdaptiveScaffold takes at most one AdaptiveScaffoldOptions")
	}
	var o AdaptiveScaffoldOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	bg := o.Background
	if bg.A == 0 {
		bg = th.Colors.Surface
	}
	class := resolveSizeClass(c, o.WidthOverride)
	navLabel := labelOr(o.NavigationLabel, "Navigation")

	root := ui.Column(c).
		Fill().
		Background(bg).
		TextColor(th.ContentColorFor(bg))

	selected := o.Selected
	if selected == nil {
		selected = ui.Local(root, adaptiveSelectedKey{}, func() int { return 0 })
	}

	root.Children(func() {
		if o.Header != nil {
			ui.Box(c.Key(adaptiveHeaderKey{})).FillWidth().Shrink(0).Children(func() {
				o.Header()
			})
		}

		ui.Row(c).Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
			if class != WindowCompact && len(destinations) > 0 {
				expanded := o.RailExpanded
				if expanded == nil {
					auto := class == WindowExpanded
					expanded = &auto
				}
				NavigationRail(c.Key(adaptiveNavRailKey{}), destinations, NavigationRailOptions{
					Selected:      selected,
					Changed:       o.Changed,
					Expanded:      expanded,
					ExpandedWidth: o.RailExpandedWidth,
					Header:        o.RailHeader,
					Label:         navLabel,
					Disabled:      o.Disabled,
				})
			}

			ui.Box(c.Key(adaptiveContentKey{})).
				Grow(1).
				FillHeight().
				MinWidth(0).
				MinHeight(0).
				Children(func() {
					if content != nil {
						content()
					}
					if o.FAB != nil {
						ui.Box(c.Key(adaptiveFABKey{})).
							Attach(ui.AnchorBottomEnd, ui.AnchorBottomEnd).
							End(16).
							Bottom(16).
							Children(func() {
								o.FAB()
							})
					}
				})
		})

		if class == WindowCompact && len(destinations) > 0 {
			NavigationBar(c.Key(adaptiveNavBarKey{}), destinations, NavigationBarOptions{
				Selected: selected,
				Changed:  o.Changed,
				Labels:   o.NavigationLabels,
				Label:    navLabel,
				Disabled: o.Disabled,
			})
		}
	})
	return root
}

// ListDetailPaneOptions configures ListDetailPane.
type ListDetailPaneOptions struct {
	// SplitWidth controls the list pane width in DIP when SplitPercent is nil.
	// When both SplitWidth and SplitPercent are nil, local state initialized to
	// DefaultListWidth (320 DIP) is used.
	SplitWidth *float32
	// SplitPercent optionally controls the list pane width as a percentage
	// (0..100) via ui.SplitPercent instead of fixed DIPs.
	SplitPercent *float32
	// DefaultListWidth is the initial and double-click reset size in DIP
	// (default 320) or percentage (default 35) when SplitPercent is used.
	DefaultListWidth float32
	// ShowDetail selects whether the detail pane is displayed when collapsed to
	// a single pane under WindowCompact. Nil shows the list pane.
	ShowDetail *bool
	// WidthOverride overrides c.Size()'s width when resolving WindowSizeClass.
	WidthOverride float32
	// Label optionally names the list-detail region for assistive technology.
	Label string
}

type (
	paneSplitStateKey  struct{}
	paneSplitWidgetKey struct{}
	paneFirstRectKey   struct{}
	paneListSlotKey    struct{}
	paneDetailSlotKey  struct{}
	paneMainSlotKey    struct{}
	paneSupportSlotKey struct{}
)

// ListDetailPane builds a Material Design 3 Adaptive two-pane list/detail
// layout over upstream ui.Split / ui.SplitPercent with SplitReset and a themed
// MD3E drag handle indicator. Under WindowCompact it collapses into a single
// pane controlled by ShowDetail; under WindowMedium and WindowExpanded both
// panes render side by side (mirroring cleanly in ui.RightToLeft).
func ListDetailPane(c *ui.Context, list func(), detail func(), opts ...ListDetailPaneOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: ListDetailPane takes at most one ListDetailPaneOptions")
	}
	var o ListDetailPaneOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	class := resolveSizeClass(c, o.WidthOverride)
	rtl := c.Direction() == ui.RightToLeft

	root := ui.Box(c).Fill().MinWidth(0).MinHeight(0)
	if o.Label != "" {
		root.Role(ui.RoleRegion).Label(o.Label)
	}

	if class == WindowCompact {
		showDetail := o.ShowDetail != nil && *o.ShowDetail
		root.Children(func() {
			if showDetail {
				ui.Box(c.Key(paneDetailSlotKey{})).Fill().MinWidth(0).MinHeight(0).Children(func() {
					if detail != nil {
						detail()
					}
				})
			} else {
				ui.Box(c.Key(paneListSlotKey{})).Fill().MinWidth(0).MinHeight(0).Children(func() {
					if list != nil {
						list()
					}
				})
			}
		})
		return root
	}

	usePercent := o.SplitPercent != nil
	defaultVal := o.DefaultListWidth
	if !isPositiveFiniteF32(defaultVal) {
		if usePercent {
			defaultVal = 35
		} else {
			defaultVal = 320
		}
	}
	if usePercent {
		defaultVal = clampF32(defaultVal, 15, 85)
	}

	var splitPtr *float32
	if usePercent {
		splitPtr = o.SplitPercent
		if !isPositiveFiniteF32(*splitPtr) {
			*splitPtr = defaultVal
		}
	} else {
		splitPtr = o.SplitWidth
		if splitPtr == nil {
			splitPtr = ui.Local(root, paneSplitStateKey{}, func() float32 { return defaultVal })
		} else if !isPositiveFiniteF32(*splitPtr) {
			*splitPtr = defaultVal
		}
	}

	firstRect := ui.Local(root, paneFirstRectKey{}, func() ui.Rect { return ui.Rect{} })
	handleCol := paneHandleColor(th)

	firstPane := func() {
		ui.Box(c.Key(paneListSlotKey{})).Fill().MinWidth(0).MinHeight(0).
			DrawOver(func(_ *ui.Painter, r ui.Rect) {
				*firstRect = r
			}).
			Children(func() {
				if list != nil {
					list()
				}
			})
	}
	secondPane := func() {
		ui.Box(c.Key(paneDetailSlotKey{})).Fill().MinWidth(0).MinHeight(0).Children(func() {
			if detail != nil {
				detail()
			}
		})
	}

	root.Children(func() {
		var sp ui.Element
		if usePercent {
			sp = ui.SplitPercent(c.Key(paneSplitWidgetKey{}), splitPtr, firstPane, secondPane)
		} else {
			sp = ui.Split(c.Key(paneSplitWidgetKey{}), splitPtr, firstPane, secondPane)
		}
		sp.Fill().SplitReset(defaultVal).DrawOver(func(p *ui.Painter, r ui.Rect) {
			drawSplitDragHandle(p, r, *firstRect, rtl, handleCol)
		})
	})
	return root
}

// SupportingPaneOptions configures SupportingPane.
type SupportingPaneOptions struct {
	// SplitPercent controls the main pane width percentage (0..100). When nil,
	// local state initialized to DefaultMainPercent (70%) is used.
	SplitPercent *float32
	// DefaultMainPercent is the initial and double-click reset percentage for
	// the main pane (default 70; clamped to 20..85).
	DefaultMainPercent float32
	// ShowSupporting controls supporting pane visibility. When nil, defaults to
	// visible in WindowExpanded and hidden in WindowCompact/WindowMedium. When
	// non-nil, *ShowSupporting explicitly toggles visibility (under WindowCompact,
	// true switches the single pane to supporting).
	ShowSupporting *bool
	// WidthOverride overrides c.Size()'s width when resolving WindowSizeClass.
	WidthOverride float32
	// Label optionally names the supporting-pane region for assistive technology.
	Label string
}

// SupportingPane builds a Material Design 3 Adaptive main + supporting split
// layout over upstream ui.SplitPercent with SplitReset and a themed MD3E drag
// handle indicator. Under WindowCompact it switches between main and supporting
// in a single pane; under wider breakpoints it shows main alone or both panes
// side by side (mirroring cleanly in ui.RightToLeft).
func SupportingPane(c *ui.Context, main func(), supporting func(), opts ...SupportingPaneOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: SupportingPane takes at most one SupportingPaneOptions")
	}
	var o SupportingPaneOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	class := resolveSizeClass(c, o.WidthOverride)
	rtl := c.Direction() == ui.RightToLeft

	root := ui.Box(c).Fill().MinWidth(0).MinHeight(0)
	if o.Label != "" {
		root.Role(ui.RoleRegion).Label(o.Label)
	}

	showSupporting := class == WindowExpanded
	if o.ShowSupporting != nil {
		showSupporting = *o.ShowSupporting
	}

	if class == WindowCompact {
		root.Children(func() {
			if showSupporting && supporting != nil {
				ui.Box(c.Key(paneSupportSlotKey{})).Fill().MinWidth(0).MinHeight(0).Children(supporting)
			} else {
				ui.Box(c.Key(paneMainSlotKey{})).Fill().MinWidth(0).MinHeight(0).Children(func() {
					if main != nil {
						main()
					}
				})
			}
		})
		return root
	}

	if !showSupporting || supporting == nil {
		root.Children(func() {
			ui.Box(c.Key(paneMainSlotKey{})).Fill().MinWidth(0).MinHeight(0).Children(func() {
				if main != nil {
					main()
				}
			})
		})
		return root
	}

	defaultPct := o.DefaultMainPercent
	if !isPositiveFiniteF32(defaultPct) {
		defaultPct = 70
	}
	defaultPct = clampF32(defaultPct, 20, 85)

	pct := o.SplitPercent
	if pct == nil {
		pct = ui.Local(root, paneSplitStateKey{}, func() float32 { return defaultPct })
	} else if !isPositiveFiniteF32(*pct) {
		*pct = defaultPct
	}

	firstRect := ui.Local(root, paneFirstRectKey{}, func() ui.Rect { return ui.Rect{} })
	handleCol := paneHandleColor(th)

	firstPane := func() {
		ui.Box(c.Key(paneMainSlotKey{})).Fill().MinWidth(0).MinHeight(0).
			DrawOver(func(_ *ui.Painter, r ui.Rect) {
				*firstRect = r
			}).
			Children(func() {
				if main != nil {
					main()
				}
			})
	}
	secondPane := func() {
		ui.Box(c.Key(paneSupportSlotKey{})).Fill().MinWidth(0).MinHeight(0).Children(supporting)
	}

	root.Children(func() {
		ui.SplitPercent(c.Key(paneSplitWidgetKey{}), pct, firstPane, secondPane).
			Fill().
			SplitReset(defaultPct).
			DrawOver(func(p *ui.Painter, r ui.Rect) {
				drawSplitDragHandle(p, r, *firstRect, rtl, handleCol)
			})
	})
	return root
}

func paneHandleColor(th *Theme) ui.Color {
	cs := th.Colors
	if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
		return cs.Outline
	}
	if cs.OutlineVariant.A > 0 {
		return cs.OutlineVariant
	}
	return cs.OnSurfaceVariant.Alpha(0.45)
}

func drawSplitDragHandle(p *ui.Painter, splitR, firstR ui.Rect, rtl bool, col ui.Color) {
	if !isPositiveFiniteF32(splitR.W) || !isPositiveFiniteF32(splitR.H) ||
		!isPositiveFiniteF32(firstR.W) || col.A == 0 {
		return
	}
	seamX := firstR.X + firstR.W
	if rtl {
		seamX = firstR.X
	}
	if seamX <= splitR.X+2 || seamX >= splitR.X+splitR.W-2 {
		return
	}
	const handleW float32 = 4
	handleH := min(float32(36), max(float32(12), splitR.H*0.25))
	p.Fill(ui.Rect{
		X: seamX - handleW*0.5,
		Y: splitR.Y + (splitR.H-handleH)*0.5,
		W: handleW,
		H: handleH,
	}, col, handleW*0.5)
}

// BentoGridOptions configures BentoGrid.
type BentoGridOptions struct {
	// Columns fixes the grid column count when > 0. When <= 0, columns adapt
	// automatically to WindowSizeClass (Compact: 1, Medium: 2, Expanded: 4,
	// unless overridden by CompactColumns / MediumColumns / ExpandedColumns).
	Columns int
	// CompactColumns overrides the column count for WindowCompact (default 1).
	CompactColumns int
	// MediumColumns overrides the column count for WindowMedium (default 2).
	MediumColumns int
	// ExpandedColumns overrides the column count for WindowExpanded (default 4).
	ExpandedColumns int
	// Gap sets the grid cell spacing in DIP. When <= 0 or non-finite, adapts to
	// Theme.Density (Comfortable: 16, Desktop: 12, Compact: 8).
	Gap float32
	// Tracks optionally defines explicit ui.Track column definitions instead of
	// equal Fr(1) columns.
	Tracks []ui.Track
	// WidthOverride overrides c.Size()'s width when resolving WindowSizeClass.
	WidthOverride float32
	// Label optionally names the grid region for assistive technology.
	Label string
}

// BentoGrid builds a responsive Material Design 3 Expressive Bento card grid
// over upstream ui.Grid, automatically adapting column count to WindowSizeClass
// and track gap to Theme.Density. When resolving to a single column without
// explicit Tracks (for example under WindowCompact), BentoGrid lays children
// out in a vertical ui.Column so multi-column BentoSpan hints collapse cleanly
// into a single full-width stack without creating implicit overflow columns.
func BentoGrid(c *ui.Context, content func(), opts ...BentoGridOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: BentoGrid takes at most one BentoGridOptions")
	}
	var o BentoGridOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	gap := o.Gap
	if !isPositiveFiniteF32(gap) {
		gap = bentoGapForDensity(th.Density)
	}

	var grid ui.Element
	if len(o.Tracks) > 0 {
		grid = ui.Grid(c).FillWidth().Gap(gap).ColumnTracks(o.Tracks...)
	} else {
		cols := resolveBentoColumns(resolveSizeClass(c, o.WidthOverride), o)
		if cols <= 1 {
			grid = ui.Column(c).FillWidth().AlignItems(ui.Stretch).Gap(gap)
		} else {
			grid = ui.Grid(c).FillWidth().Gap(gap).Columns(cols)
		}
	}
	if o.Label != "" {
		grid.Role(ui.RoleRegion).Label(o.Label)
	}
	if content != nil {
		grid.Children(content)
	}
	return grid
}

// BentoSpan configures el to span cols columns and rows rows inside a BentoGrid.
// Negative values span to the last track (matching ui.Element.ColumnSpan and
// ui.Element.RowSpan semantics); 0 or 1 leaves that axis at a single cell.
func BentoSpan(el ui.Element, cols, rows int) ui.Element {
	if cols > 1 || cols < 0 {
		el.ColumnSpan(cols)
	}
	if rows > 1 || rows < 0 {
		el.RowSpan(rows)
	}
	return el
}

func resolveBentoColumns(class WindowSizeClass, o BentoGridOptions) int {
	if o.Columns > 0 {
		return o.Columns
	}
	switch class {
	case WindowCompact:
		if o.CompactColumns > 0 {
			return o.CompactColumns
		}
		return 1
	case WindowMedium:
		if o.MediumColumns > 0 {
			return o.MediumColumns
		}
		return 2
	default:
		if o.ExpandedColumns > 0 {
			return o.ExpandedColumns
		}
		return 4
	}
}

func bentoGapForDensity(d tokens.Density) float32 {
	switch d.Normalize() {
	case tokens.DensityDesktop:
		return 12
	case tokens.DensityCompact:
		return 8
	default:
		return 16
	}
}
