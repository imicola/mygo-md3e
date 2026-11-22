package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// VirtualListOptions configures a Material Design 3 Expressive VirtualList.
// The zero value produces a density-aware virtualized list backed by ui.List
// with MD3E row heights, hover/press/ripple state layers, and scrollbar
// clearance.
type VirtualListOptions struct {
	// State optionally provides a caller-owned ui.ListState for scroll control,
	// sticky headers, multi-selection, and Move/Follow animations. When nil, an
	// internal ui.ListState is managed automatically with ui.Local.
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
	// IsHeader reports whether row renders as a sticky section header.
	IsHeader func(row int) bool
	// Segmented draws rows as MD3E segmented card pieces: first row in a
	// section rounds top corners (large), last rounds bottom corners (large),
	// inner rows keep small corners, and selected rows round all four corners
	// with SecondaryContainer fill.
	Segmented bool
	// RowHeight fixes the height of non-header rows in DIP when positive and
	// finite. When <= 0 or non-finite, non-header rows use the active
	// Theme.Density one-line height as a minimum and grow for taller content.
	RowHeight float32
	// Gap overrides the spacing between rows in DIP when positive and finite.
	// When <= 0 or non-finite, defaults to 2 DIP for Segmented lists and 0
	// otherwise.
	Gap float32
	// Padding adds inner inset padding around the virtual list content in DIP
	// (sanitized to >= 0). Trailing padding always reserves extra clearance for
	// the vertical scrollbar in both LTR and RTL layouts.
	Padding float32
	// Container wraps the list inside a SurfaceContainerLow rounded surface.
	Container bool
	// Hairline draws a subtle 1 DIP MD3E hairline border around the list
	// container.
	Hairline bool
	// ShapeScale overrides the outer container corner token when Container or
	// Hairline is enabled. Zero defaults to tokens.ShapeLarge.
	ShapeScale tokens.ShapeScale
	// OnClick runs when a non-header row is primary-clicked, after *Selected
	// and Selection have been updated.
	OnClick func(row int)
	// OnSubmit runs when a row is submitted via double-click or Enter.
	OnSubmit func(row int)
	// Label names the list for assistive technology.
	Label string
	// Disabled disables list interaction and renders disabled MD3E content
	// tokens.
	Disabled bool
}

const (
	virtualListScrollbarClearance float32 = 10
	virtualListMaxActiveRipples           = maxActiveRipples
)

type (
	virtualListThemeKey struct{}
	virtualListStateKey struct{}
	virtualListLeadKey  struct{}
	vlRowStateKey       struct{}
)

type vlRowAnimState struct {
	ripple       rippleState
	wasPressed   bool
	lastAlpha    float32
	lastRadii    [4]float32
	lastInk      ui.Color
	lastDisabled bool
}

type virtualListMetrics struct {
	oneLineH   float32
	fixedRowH  float32
	rowPadY    float32
	padX       float32
	headerMinH float32
	headerRole tokens.TypeRole
	gap        float32
	outerPad   float32
	padStart   float32
	padEnd     float32
	largeR     float32
	smallR     float32
	mediumR    float32
}

// VirtualList builds a virtualized Material Design 3 Expressive list over
// ui.List, instantiating only rows near the viewport so hundreds of thousands
// or millions of rows scroll smoothly.
//
// Hover, press, and ripple feedback are evaluated strictly inside each row
// container's DrawOver pass via ui.ListRow.Hovered / Pressed / PointerPosition
// so pointer movement never triggers full-tree view rebuilds. Layout spacing
// and scrollbar clearance automatically adapt to Theme.Density, ReduceMotion,
// and ui.RightToLeft directionality.
func VirtualList(c *ui.Context, count int, row func(r ui.ListRow), opts ...VirtualListOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: VirtualList takes at most one VirtualListOptions")
	}
	var o VirtualListOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	count = max(count, 0)

	th := CurrentTheme(c)
	origUI := c.Theme()
	if origUI == nil {
		origUI = th.ToUITheme()
	}

	root := buildVirtualListRoot(c, th, o)
	if o.Disabled {
		root.Disabled(true)
	}
	disabled := o.Disabled || root.IsDisabled()

	listTheme := ui.Local(root, virtualListThemeKey{}, func() ui.Theme { return ui.Theme{} })
	*listTheme = *origUI
	listTheme.Background = ui.Transparent
	listTheme.Accent = ui.Transparent
	listTheme.InactiveAccent = ui.Transparent
	listTheme.SurfaceHover = ui.Transparent
	listTheme.Radius = 0

	state := resolveVirtualListState(root, o)
	metrics := computeVirtualListMetrics(th, o)
	interactive := !disabled && (state.Selected != nil || state.Selection != nil || o.OnClick != nil || o.OnSubmit != nil)

	root.Children(func() {
		ui.Themed(c, listTheme).
			Background(ui.Transparent).
			Grow(1).FillWidth().MinWidth(0).MinHeight(0).
			Children(func() {
				listEl := ui.List(c, state, count)
				if o.ItemKey != nil {
					listEl.ItemKey(o.ItemKey)
				}
				if o.Selection != nil {
					listEl.Selection(o.Selection)
				}
				listEl.Gap(metrics.gap).
					PaddingY(metrics.outerPad).
					PaddingStart(metrics.padStart).
					PaddingEnd(metrics.padEnd).
					Grow(1).FillWidth().MinWidth(0).MinHeight(0).
					Disabled(disabled)
				if o.Label != "" {
					listEl.Label(o.Label)
				}

				clickedRow, doubleClickedRow := -1, -1
				listEl.Rows(func(r ui.ListRow) {
					if r.Context == nil {
						return
					}
					buildVirtualListRow(
						r, th, origUI, state, count, o, metrics,
						disabled, interactive, row, &clickedRow, &doubleClickedRow,
					)
				})

				if !disabled && clickedRow >= 0 && o.OnClick != nil {
					o.OnClick(clickedRow)
				}
				if !disabled && o.OnSubmit != nil {
					submitted := listEl.Submitted()
					if submitted || doubleClickedRow >= 0 {
						submittedRow := doubleClickedRow
						if state.Selected != nil && *state.Selected >= 0 && *state.Selected < count {
							submittedRow = *state.Selected
						}
						o.OnSubmit(submittedRow)
					}
				}
			})
	})

	return root
}

func buildVirtualListRoot(c *ui.Context, th *Theme, o VirtualListOptions) ui.Element {
	if o.Container || o.Hairline {
		scale := o.ShapeScale
		if scale == tokens.ShapeNone {
			scale = tokens.ShapeLarge
		}
		tonal := tokens.Level0
		if o.Container {
			tonal = tokens.Level1
		}
		return Surface(c, SurfaceOptions{
			TonalElevation: tonal,
			DisableShadow:  true,
			Hairline:       o.Hairline,
			ShapeScale:     scale,
			Clip:           true,
		}).MinWidth(0).MinHeight(0)
	}
	return ui.Column(c).MinWidth(0).MinHeight(0).Clip()
}

func resolveVirtualListState(root ui.Element, o VirtualListOptions) *ui.ListState {
	if o.State != nil {
		s := o.State
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
		return s
	}

	s := ui.Local(root, virtualListStateKey{}, func() ui.ListState { return ui.ListState{} })
	lead := ui.Local(root, virtualListLeadKey{}, func() int { return -1 })
	s.Key = o.ItemKey
	s.Selection = o.Selection
	s.Label = o.RowLabel
	s.Header = o.IsHeader
	switch {
	case o.Selected != nil:
		s.Selected = o.Selected
	case o.Selection != nil:
		s.Selected = lead
	default:
		s.Selected = nil
	}
	return s
}

func computeVirtualListMetrics(th *Theme, o VirtualListOptions) virtualListMetrics {
	oneLine, _, _ := th.Density.ListHeights()
	padX := float32(listRowPadding)
	rowPadY := float32(8)
	headerMinH := float32(36)
	headerRole := tokens.LabelLarge
	switch th.Density.Normalize() {
	case tokens.DensityDesktop:
		padX = 12
		rowPadY = 6
		headerMinH = 30
		headerRole = tokens.LabelMedium
	case tokens.DensityCompact:
		padX = 10
		rowPadY = 4
		headerMinH = 26
		headerRole = tokens.LabelMedium
	}

	gap := float32(0)
	if o.Segmented {
		gap = listSegmentGap
	}
	if isPositiveFiniteF32(o.Gap) {
		gap = o.Gap
	}

	outerPad := sanitizeNonNegF32(o.Padding)
	fixedRowH := float32(0)
	if isPositiveFiniteF32(o.RowHeight) {
		fixedRowH = o.RowHeight
	}

	return virtualListMetrics{
		oneLineH:   float32(oneLine),
		fixedRowH:  fixedRowH,
		rowPadY:    rowPadY,
		padX:       padX,
		headerMinH: headerMinH,
		headerRole: headerRole,
		gap:        gap,
		outerPad:   outerPad,
		padStart:   outerPad,
		padEnd:     outerPad + virtualListScrollbarClearance,
		largeR:     th.Shapes.Radius(tokens.ShapeLarge),
		smallR:     th.Shapes.Radius(tokens.ShapeSmall),
		mediumR:    th.Shapes.Radius(tokens.ShapeMedium),
	}
}

func buildVirtualListRow(
	r ui.ListRow,
	th *Theme,
	origUI *ui.Theme,
	state *ui.ListState,
	count int,
	o VirtualListOptions,
	m virtualListMetrics,
	disabled, interactive bool,
	row func(r ui.ListRow),
	clickedRow, doubleClickedRow *int,
) {
	cs := th.Colors
	rowShell := ui.Themed(r.Context, origUI).FillWidth().MinWidth(0)

	if state.Header != nil && state.Header(r.Index) {
		headerBg := cs.Surface
		if o.Container {
			headerBg = SurfaceColorForElevation(th, tokens.Level1)
		}
		headerInk := cs.OnSurfaceVariant
		if disabled {
			headerInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		}
		rowShell.Background(headerBg).
			TextColor(headerInk).
			MinHeight(m.headerMinH).
			Justify(ui.Center).
			PaddingStart(m.padX).
			PaddingEnd(m.padX)
		applyTypeStyleWithTheme(th, rowShell, m.headerRole)
		rowShell.Children(func() {
			if row != nil {
				row(r)
			}
		})
		return
	}

	selected := r.Selected()
	radii := virtualListRowRadii(r.Index, count, state.Header, o.Segmented, selected, m)

	baseBg := ui.Transparent
	if o.Segmented {
		if o.Container {
			baseBg = cs.Surface
		} else {
			baseBg = SurfaceColorForElevation(th, tokens.Level1)
		}
	}
	selectedBg := cs.SecondaryContainer
	selectedInk := cs.OnSecondaryContainer
	if selectedBg.A == 0 {
		selectedBg = cs.PrimaryContainer
		selectedInk = cs.OnPrimaryContainer
	}

	bg := baseBg
	ink := cs.OnSurface
	if selected {
		bg = selectedBg
		ink = selectedInk
	}
	if disabled {
		if o.Segmented || selected {
			bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		} else {
			bg = ui.Transparent
		}
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}

	rowShell.Radius(radii[0], radii[1], radii[2], radii[3]).
		Background(bg).
		TextColor(ink).
		Clip().
		Justify(ui.Center).
		PaddingStart(m.padX).
		PaddingEnd(m.padX)
	if m.fixedRowH > 0 {
		rowShell.Height(m.fixedRowH)
	} else {
		rowShell.MinHeight(m.oneLineH).PaddingY(m.rowPadY)
	}

	if !disabled && (o.OnClick != nil || o.OnSubmit != nil) {
		if rowShell.RowClicked() && o.OnClick != nil {
			*clickedRow = r.Index
		}
		if o.OnSubmit != nil && rowShell.RowDoubleClicked() {
			*doubleClickedRow = r.Index
		}
	}

	rowShell.Children(func() {
		if row != nil {
			row(r)
		}
	})

	attachVirtualListRowFeedback(rowShell, r, th, radii, ink, interactive, disabled)
}

func virtualListRowRadii(
	idx, count int,
	isHeader func(int) bool,
	segmented, selected bool,
	m virtualListMetrics,
) [4]float32 {
	if segmented {
		if selected {
			return [4]float32{m.largeR, m.largeR, m.largeR, m.largeR}
		}
		isFirst := idx == 0 || (isHeader != nil && isHeader(idx-1))
		isLast := idx == count-1 || (isHeader != nil && isHeader(idx+1))
		switch {
		case isFirst && isLast:
			return [4]float32{m.largeR, m.largeR, m.largeR, m.largeR}
		case isFirst:
			return [4]float32{m.largeR, m.largeR, m.smallR, m.smallR}
		case isLast:
			return [4]float32{m.smallR, m.smallR, m.largeR, m.largeR}
		default:
			return [4]float32{m.smallR, m.smallR, m.smallR, m.smallR}
		}
	}
	return [4]float32{m.mediumR, m.mediumR, m.mediumR, m.mediumR}
}

func attachVirtualListRowFeedback(
	rowShell ui.Element,
	r ui.ListRow,
	th *Theme,
	radii [4]float32,
	ink ui.Color,
	interactive, disabled bool,
) {
	st := ui.Local(rowShell, vlRowStateKey{}, func() vlRowAnimState { return vlRowAnimState{} })
	if st == nil {
		return
	}
	st.lastRadii = radii
	st.lastInk = ink
	st.lastDisabled = disabled

	if !interactive || disabled {
		st.ripple.clearTransient()
		st.ripple.halo.reset()
		st.wasPressed = false
		st.lastAlpha = 0
		return
	}

	rowShell.DrawOver(func(p *ui.Painter, rect ui.Rect) {
		if rect.W <= 0 || rect.H <= 0 {
			return
		}
		if !rowShell.Valid() || r.Context == nil || !r.Context.Valid() {
			if st.lastAlpha > 0.001 && !st.lastDisabled {
				paintVirtualListOverlay(p, rect, st.lastRadii, st.lastInk, st.lastAlpha, nil)
			}
			return
		}
		if rowShell.IsDisabled() {
			st.ripple.clearTransient()
			st.ripple.halo.reset()
			st.wasPressed = false
			st.lastAlpha = 0
			return
		}

		reduceMotion := th.ReduceMotion || p.Preferences().ReduceMotion
		now := p.Now()
		hovered := r.Hovered()
		pressed := r.Pressed()

		if pressed && !st.wasPressed {
			if !reduceMotion {
				ox, oy, _ := r.PointerPosition()
				st.ripple.startWave(ox, oy, false, now)
			}
		} else if !pressed && st.wasPressed {
			st.ripple.releaseWaves(false, now)
		}
		st.wasPressed = pressed
		if reduceMotion {
			st.ripple.clearTransient()
		}

		targetAlpha := resolveStateLayerTarget(false, false, pressed, reduceMotion, false, hovered)
		layerAlpha, needFrame := st.ripple.halo.step(targetAlpha, now, reduceMotion)
		st.lastAlpha = layerAlpha

		var stackWaves [virtualListMaxActiveRipples]evaluatedWave
		wavesToDraw := stackWaves[:0]
		if !reduceMotion && len(st.ripple.waves) > 0 {
			kept := st.ripple.waves[:0]
			for i := range st.ripple.waves {
				w := st.ripple.waves[i]
				ox := clampF32(w.ox, 0, rect.W)
				oy := clampF32(w.oy, 0, rect.H)
				curR, enterAlpha, fadeAlpha, animating, alive := evalRippleWave(&w, ox, oy, rect, now)
				if alive && fadeAlpha > 0.001 {
					kept = append(kept, w)
					if animating {
						needFrame = true
					}
					if waveAlpha := float32(tokens.StatePressedOpacity) * enterAlpha * fadeAlpha; waveAlpha > 0.001 && curR > 0 {
						wavesToDraw = append(wavesToDraw, evaluatedWave{
							ox:        ox,
							oy:        oy,
							curR:      curR,
							waveAlpha: waveAlpha,
						})
					}
				}
			}
			st.ripple.waves = kept
		}

		if needFrame && rectIntersects(rect, p.Visible()) {
			p.After(16 * time.Millisecond)
		}
		if layerAlpha <= 0.001 && len(wavesToDraw) == 0 {
			return
		}
		paintVirtualListOverlay(p, rect, radii, ink, layerAlpha, wavesToDraw)
	})
}

func paintVirtualListOverlay(
	p *ui.Painter,
	rect ui.Rect,
	radii [4]float32,
	ink ui.Color,
	layerAlpha float32,
	wavesToDraw []evaluatedWave,
) {
	paintInClip(p, rect, radii, nil, func() {
		if layerAlpha > 0.001 {
			p.FillRadii(rect, radii, ink.Alpha(layerAlpha))
		}
		for i := range wavesToDraw {
			w := &wavesToDraw[i]
			waveInk := ink.Alpha(w.waveAlpha)
			cx, cy := w.ox/rect.W, w.oy/rect.H
			if cx == 0 && cy == 0 {
				cx = 1e-4
			}
			p.FillRadialGradient(rect, ui.RadialGradient{
				Center: ui.Point{X: cx, Y: cy},
				Radius: w.curR,
				Stops: []ui.GradientStop{
					{Color: waveInk, At: 0},
					{Color: waveInk, At: 0.80},
					{Color: ui.Transparent, At: 1},
				},
			}, 0)
		}
	})
}
