package md3e

import (
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/tokens"
)

// BreadcrumbItem describes one segment of a Breadcrumb navigation trail.
type BreadcrumbItem struct {
	// Label is the visible segment text and accessible name.
	Label string
	// Icon draws an optional leading icon before Label.
	Icon IconDraw
	// OnClick runs when an enabled ancestor segment (or its collapsed entry in
	// the overflow menu) is clicked or activated by Enter or Space.
	OnClick func()
	// Disabled disables interaction for this segment.
	Disabled bool
}

// BreadcrumbOptions configures Breadcrumb. The zero value produces an enabled,
// responsive trail that automatically collapses intermediate segments into an
// overflow menu when horizontal space runs short.
type BreadcrumbOptions struct {
	// MaxVisible limits how many visible breadcrumb segments (including the
	// overflow menu button when collapsed) appear at once. Zero enables
	// automatic responsive collapsing based on available layout width; positive
	// values force collapsing whenever len(items) > MaxVisible (and still
	// collapse further if available width is narrower).
	MaxVisible int
	// Separator draws the divider icon between visible segments. Nil uses an
	// RTL-adaptive ChevronRight / ChevronLeft icon.
	Separator IconDraw
	// Label names the navigation landmark for assistive technology
	// ("Breadcrumb" when empty).
	Label string
	// Disabled disables interaction across all breadcrumb segments and the
	// overflow menu.
	Disabled bool
}

type (
	breadcrumbStateKey    struct{}
	breadcrumbOverflowKey struct{}
	breadcrumbItemKey     struct{ i int }
	breadcrumbSepKey      struct{ i int }
)

type breadcrumbItemSig struct {
	label   string
	hasIcon bool
}

type breadcrumbState struct {
	open        bool
	hasLayout   bool
	builtAvailW float32
	lastEvalW   float32
	evalPasses  int
	measuredW   []float32
	measuredSep float32
	measuredEll float32
	sigs        []breadcrumbItemSig
}

type breadcrumbMetrics struct {
	itemH        float32
	padX         float32
	overflowPadX float32
	itemGap      float32
	rowGap       float32
	iconSize     float32
	sepSize      float32
	overflowMinW float32
	radius       float32
	typeRole     tokens.TypeRole
	fontSize     float32
}

func breadcrumbMetricsFor(th *Theme) breadcrumbMetrics {
	radius := th.Shapes.Radius(tokens.ShapeSmall)
	scale := th.EffectiveTextScale()
	switch th.Density.Normalize() {
	case tokens.DensityCompact:
		ts := th.TypeStyle(tokens.LabelMedium, false)
		return breadcrumbMetrics{
			itemH:        24,
			padX:         6,
			overflowPadX: 6,
			itemGap:      4,
			rowGap:       2,
			iconSize:     14,
			sepSize:      14,
			overflowMinW: 26,
			radius:       min(radius, 12),
			typeRole:     tokens.LabelMedium,
			fontSize:     float32(ts.Size) * scale,
		}
	case tokens.DensityDesktop:
		ts := th.TypeStyle(tokens.LabelLarge, false)
		return breadcrumbMetrics{
			itemH:        28,
			padX:         8,
			overflowPadX: 6,
			itemGap:      5,
			rowGap:       2,
			iconSize:     16,
			sepSize:      14,
			overflowMinW: 28,
			radius:       min(radius, 14),
			typeRole:     tokens.LabelLarge,
			fontSize:     float32(ts.Size) * scale,
		}
	default:
		ts := th.TypeStyle(tokens.LabelLarge, false)
		return breadcrumbMetrics{
			itemH:        32,
			padX:         10,
			overflowPadX: 8,
			itemGap:      6,
			rowGap:       4,
			iconSize:     18,
			sepSize:      16,
			overflowMinW: 32,
			radius:       min(radius, 16),
			typeRole:     tokens.LabelLarge,
			fontSize:     float32(ts.Size) * scale,
		}
	}
}

// Breadcrumb builds a Material Design 3 Expressive responsive navigation trail
// as a ui.RoleNavigation landmark. Ancestor segments act as interactive MD3E
// buttons, the final segment renders as a static highlighted current-page pill
// (ui.CurrentPage), and intermediate segments collapse into a MoreHoriz
// overflow dropdown menu (ui.MenuBase) when MaxVisible is exceeded or available
// layout width shrinks. Separators and menu placement adapt automatically to
// ui.RightToLeft and Theme.Density.
func Breadcrumb(c *ui.Context, items []BreadcrumbItem, opts ...BreadcrumbOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Breadcrumb takes at most one BreadcrumbOptions")
	}
	var o BreadcrumbOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	m := breadcrumbMetricsFor(th)
	rtl := c.Direction() == ui.RightToLeft

	label := o.Label
	if label == "" {
		label = "Breadcrumb"
	}

	root := ui.Row(c).
		Role(ui.RoleNavigation).
		Label(label).
		FillWidth().
		MinWidth(0).
		Clip().
		AlignItems(ui.Center).
		Gap(m.rowGap).
		DimDisabled(false).
		Disabled(o.Disabled)

	disabledAll := o.Disabled || root.IsDisabled()
	st := ui.Local(root, breadcrumbStateKey{}, func() breadcrumbState { return breadcrumbState{} })
	if st == nil {
		return root
	}

	if disabledAll || inExitingOverlay(c) || len(items) <= 1 {
		st.open = false
	}

	if len(items) == 0 {
		st.sigs = st.sigs[:0]
		st.measuredW = st.measuredW[:0]
		return root
	}

	syncBreadcrumbStateItems(st, items)

	layoutW := root.LayoutBounds().W
	availW := layoutW
	if availW <= 0 {
		winW, _ := c.Size()
		availW = winW
		if !st.hasLayout && len(items) > 1 && winW > 0 {
			st.hasLayout = true
			c.Invalidate()
		}
	} else {
		st.hasLayout = true
	}
	st.builtAvailW = availW

	plan := computeBreadcrumbPlan(items, o.MaxVisible, availW, st, m)
	if !plan.collapsed {
		st.open = false
	}

	sepIcon := o.Separator
	if sepIcon == nil {
		if rtl {
			sepIcon = mdicon.ChevronLeft.Draw
		} else {
			sepIcon = mdicon.ChevronRight.Draw
		}
	}

	mutedInk := cs.OnSurfaceVariant
	sepInk := cs.OnSurfaceVariant.Alpha(0.72)
	curBg := cs.SecondaryContainer
	curInk := cs.OnSecondaryContainer
	if curBg.A == 0 {
		curBg = cs.PrimaryContainer
		curInk = cs.OnPrimaryContainer
	}
	if disabledAll {
		mutedInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		sepInk = mutedInk
		curBg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		curInk = mutedInk
	}

	n := len(items)
	var renderedIdxs []int
	var renderedEls []ui.Element
	var overflowTrigger ui.Element
	var firstSepEl ui.Element

	buildCrumb := func(idx int, isLast bool, shrinkable bool) ui.Element {
		item := &items[idx]
		itemDisabled := disabledAll || item.Disabled
		c.Key(breadcrumbItemKey{i: idx})

		if isLast {
			bg, ink := curBg, curInk
			if itemDisabled && !disabledAll {
				bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
				ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
			}
			cur := ui.Row(c).
				DimDisabled(false).
				Height(m.itemH).
				PaddingX(m.padX).
				Gap(m.itemGap).
				Radius(m.itemH / 2).
				AlignItems(ui.Center).
				Background(bg).
				TextColor(ink).
				Role(ui.RoleText).
				Current(ui.CurrentPage)
			if shrinkable {
				cur.Shrink(1).MinWidth(0)
			} else {
				cur.Shrink(0)
			}
			if item.Label != "" {
				cur.Label(item.Label)
			}
			cur.Children(func() {
				if item.Icon != nil {
					drawIconBox(c, item.Icon, m.iconSize, ink)
				}
				if item.Label != "" {
					txt := ui.Text(c, item.Label).SingleLine().Shrink(1).MinWidth(0)
					applyTypeStyleWithTheme(th, txt, m.typeRole, true)
					txt.TextColor(ink)
				}
			})
			return cur
		}

		ink := mutedInk
		if itemDisabled && !disabledAll {
			ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		}
		btn := ui.ButtonBase(c).
			DimDisabled(false).
			FocusRing(false).
			Height(m.itemH).
			PaddingX(m.padX).
			Gap(m.itemGap).
			Radius(m.radius).
			AlignItems(ui.Center).
			Justify(ui.Center).
			Disabled(item.Disabled).
			TextColor(ink)
		if shrinkable {
			btn.Shrink(1).MinWidth(0)
		} else {
			btn.Shrink(0)
		}
		if item.Label != "" {
			btn.Label(item.Label)
		}
		attachRippleWithContext(c, th, btn, StateLayerOptions{
			Color:    cs.OnSurface,
			Radius:   m.radius,
			Disabled: itemDisabled,
		})
		btn.Children(func() {
			if item.Icon != nil {
				drawIconBox(c, item.Icon, m.iconSize, ink)
			}
			if item.Label != "" {
				txt := ui.Text(c, item.Label).SingleLine().Shrink(1).MinWidth(0)
				applyTypeStyleWithTheme(th, txt, m.typeRole)
				txt.TextColor(ink)
			}
		})
		if !itemDisabled && btn.Clicked() && item.OnClick != nil {
			item.OnClick()
		}
		return btn
	}

	buildSep := func(slot int) ui.Element {
		s := ui.Box(c.Key(breadcrumbSepKey{i: slot})).
			Size(m.sepSize, m.sepSize).
			Shrink(0).
			Role(ui.RoleNone)
		s.Draw(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			sepIcon(p, r, sepInk)
		})
		return s
	}

	root.Children(func() {
		sepIdx := 0
		needSep := false

		if plan.showFirst {
			el := buildCrumb(0, n == 1, plan.allowShrink || n == 1)
			if !plan.allowShrink && n > 1 {
				renderedIdxs = append(renderedIdxs, 0)
				renderedEls = append(renderedEls, el)
			}
			needSep = true
		}

		if plan.collapsed {
			if needSep {
				s := buildSep(sepIdx)
				if !firstSepEl.Valid() {
					firstSepEl = s
				}
				sepIdx++
			}
			c.Key(breadcrumbOverflowKey{})
			menuParts := ui.MenuBase(c, &st.open)
			trigBg := cs.SurfaceContainerHigh
			trigInk := mutedInk
			if st.open && !disabledAll {
				trigBg = curBg
				trigInk = curInk
			} else if disabledAll {
				trigBg = ui.Transparent
			}
			trigRadius := m.itemH / 2
			trig := menuParts.Trigger.
				DimDisabled(false).
				FocusRing(false).
				Height(m.itemH).
				MinWidth(m.overflowMinW).
				PaddingX(m.overflowPadX).
				Radius(trigRadius).
				Shrink(0).
				AlignItems(ui.Center).
				Justify(ui.Center).
				Disabled(disabledAll).
				Label("More path items").
				TextColor(trigInk)
			if trigBg.A > 0 {
				trig.Background(trigBg)
			}
			attachRippleWithContext(c, th, trig, StateLayerOptions{
				Color:    trigInk,
				Radius:   trigRadius,
				Disabled: disabledAll,
			})
			trig.Children(func() {
				drawIconBox(c, mdicon.MoreHoriz.Draw, m.iconSize, trigInk)
			})
			overflowTrigger = trig

			collapsedItems := items[plan.collapseStart:plan.collapseEnd]
			menuParts.Popup(func(panel ui.Element) {
				styleMenuPanel(c, th, panel, MenuOptions{Label: "More path items"})
				menuList := make([]MenuItem, len(collapsedItems))
				for k := range collapsedItems {
					it := &collapsedItems[k]
					menuList[k] = MenuItem{
						Label:    it.Label,
						Icon:     it.Icon,
						Disabled: disabledAll || it.Disabled,
						OnClick:  it.OnClick,
					}
				}
				buildMenuItems(c, th, menuParts, menuList, 0)
			})
			needSep = true
		}

		for idx := plan.tailStart; idx < n; idx++ {
			if needSep {
				s := buildSep(sepIdx)
				if !firstSepEl.Valid() {
					firstSepEl = s
				}
				sepIdx++
			}
			isLast := idx == n-1
			shrinkable := plan.allowShrink && (isLast || !plan.collapsed)
			el := buildCrumb(idx, isLast, shrinkable)
			if !shrinkable {
				renderedIdxs = append(renderedIdxs, idx)
				renderedEls = append(renderedEls, el)
			}
			needSep = true
		}
	})

	// Record actual laid-out widths from the previous frame for responsive refinement.
	totalMeasured := float32(0)
	childCount := 0
	for i, el := range renderedEls {
		if el.Valid() {
			if w := el.LayoutBounds().W; w > 0 {
				st.measuredW[renderedIdxs[i]] = w
				totalMeasured += w
				childCount++
			}
		}
	}
	if overflowTrigger.Valid() {
		if w := overflowTrigger.LayoutBounds().W; w > 0 {
			st.measuredEll = w
			totalMeasured += w
			childCount++
		}
	}
	if firstSepEl.Valid() {
		if w := firstSepEl.LayoutBounds().W; w > 0 {
			st.measuredSep = w
		}
	}
	if childCount > 1 {
		sepW := st.measuredSep
		if sepW <= 0 {
			sepW = m.sepSize
		}
		totalMeasured += float32(childCount-1) * (sepW + 2*m.rowGap)
	}

	srv := c.Services()
	canCollapseMore := !plan.allowShrink && len(items) >= 2 && (plan.tailStart < len(items)-1 || !plan.collapsed)
	root.DrawOver(func(_ *ui.Painter, r ui.Rect) {
		if !root.Valid() || r.W <= 0 || r.H <= 0 {
			return
		}
		if absF32(r.W-st.lastEvalW) > 0.5 {
			st.lastEvalW = r.W
			st.evalPasses = 0
		}
		widthChanged := absF32(r.W-st.builtAvailW) > 0.5
		overflowed := canCollapseMore && totalMeasured > r.W+0.5
		if (widthChanged || overflowed) && st.evalPasses < 4 {
			st.evalPasses++
			srv.Invalidate()
		}
	})

	return root
}

type breadcrumbPlan struct {
	showFirst     bool
	collapsed     bool
	collapseStart int
	collapseEnd   int
	tailStart     int
	allowShrink   bool
}

func syncBreadcrumbStateItems(st *breadcrumbState, items []BreadcrumbItem) {
	same := len(st.sigs) == len(items)
	if same {
		for i := range items {
			if st.sigs[i].label != items[i].Label || st.sigs[i].hasIcon != (items[i].Icon != nil) {
				same = false
				break
			}
		}
	}
	if same {
		return
	}
	st.evalPasses = 0
	st.sigs = make([]breadcrumbItemSig, len(items))
	st.measuredW = make([]float32, len(items))
	for i := range items {
		st.sigs[i] = breadcrumbItemSig{
			label:   items[i].Label,
			hasIcon: items[i].Icon != nil,
		}
	}
}

func estimateBreadcrumbItemW(m breadcrumbMetrics, it *BreadcrumbItem) float32 {
	w := m.padX * 2
	if it.Icon != nil {
		w += m.iconSize
		if it.Label != "" {
			w += m.itemGap
		}
	}
	if it.Label != "" {
		var units float32
		for _, r := range it.Label {
			switch {
			case r > 0x2E7F: // CJK / wide glyphs
				units += 1.0
			case unicode.IsUpper(r):
				units += 0.68
			case r == 'i' || r == 'l' || r == 'I' || r == '.' || r == '/' || r == '-' || r == ' ':
				units += 0.34
			default:
				units += 0.56
			}
		}
		if units == 0 && utf8.RuneCountInString(it.Label) > 0 {
			units = float32(utf8.RuneCountInString(it.Label)) * 0.56
		}
		w += units*m.fontSize + 2
	}
	return max(w, m.itemH)
}

func computeBreadcrumbPlan(items []BreadcrumbItem, maxVisible int, availW float32, st *breadcrumbState, m breadcrumbMetrics) breadcrumbPlan {
	n := len(items)
	if n <= 1 {
		return breadcrumbPlan{showFirst: true, tailStart: 1, allowShrink: true}
	}

	itemW := func(i int) float32 {
		if i < len(st.measuredW) && st.measuredW[i] > 0 {
			return st.measuredW[i]
		}
		return estimateBreadcrumbItemW(m, &items[i])
	}
	sepW := st.measuredSep
	if sepW <= 0 {
		sepW = m.sepSize
	}
	ellW := st.measuredEll
	if ellW <= 0 {
		ellW = m.overflowMinW
	}

	planWidth := func(showFirst bool, collapsed bool, tailStart int) float32 {
		var sum float32
		slots := 0
		if showFirst {
			sum += itemW(0)
			slots++
		}
		if collapsed {
			sum += ellW
			slots++
		}
		for i := tailStart; i < n; i++ {
			sum += itemW(i)
			slots++
		}
		if slots > 1 {
			sum += float32(slots-1) * (sepW + 2*m.rowGap)
		}
		return sum
	}

	showFirst := true
	collapsed := false
	tailStart := 1

	if maxVisible > 0 && n > maxVisible {
		collapsed = true
		if maxVisible == 1 {
			showFirst = false
			tailStart = n - 1
		} else {
			tailCount := max(1, maxVisible-2)
			tailStart = max(1, n-tailCount)
		}
	}

	if availW > 0 {
		if !collapsed && n >= 3 && planWidth(true, false, 1) > availW {
			collapsed = true
			tailStart = 2
		}
		if collapsed {
			for tailStart < n-1 && planWidth(showFirst, true, tailStart) > availW {
				tailStart++
			}
		}
	}

	allowShrink := availW > 0 && ((collapsed && tailStart == n-1 && planWidth(showFirst, true, tailStart) > availW) ||
		(!collapsed && planWidth(true, false, 1) > availW))
	collapseStart := 0
	if showFirst {
		collapseStart = 1
	}
	collapseEnd := tailStart
	if !collapsed || collapseStart >= collapseEnd {
		collapsed = false
		collapseStart = 0
		collapseEnd = 0
		tailStart = 1
	}

	return breadcrumbPlan{
		showFirst:     showFirst,
		collapsed:     collapsed,
		collapseStart: collapseStart,
		collapseEnd:   collapseEnd,
		tailStart:     tailStart,
		allowShrink:   allowShrink,
	}
}
