package md3e

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// PaginationOptions configures Pagination. The zero value produces an enabled,
// Filled-accent 1-based pagination bar with Previous/Next buttons, 1 sibling
// page on each side of the current page, and tabular numerals.
type PaginationOptions struct {
	// ShowEdges adds First page and Last page icon buttons at the ends of the
	// pagination bar.
	ShowEdges bool
	// SiblingCount sets how many page numbers render immediately before and
	// after the current page. Values <= 0 default to 1.
	SiblingCount int
	// MaxButtons caps the total number of page and ellipsis slots between
	// Previous and Next when > 0. Values <= 0 use SiblingCount (7 slots when
	// SiblingCount defaults to 1).
	MaxButtons int
	// Variant selects the MD3E look of the active page button (ButtonFilled by
	// default, or ButtonTonal for SecondaryContainer). Inactive page buttons
	// render as ButtonText (or ButtonOutlined when Variant is ButtonOutlined).
	Variant ButtonVariant
	// OnChanged runs when clicking or activating a button changes the active
	// page, after *page is updated.
	OnChanged func(page int)
	// Label names the navigation landmark for assistive technology
	// ("Pagination" when empty).
	Label string
	// Disabled disables interaction across every button in the pagination bar.
	Disabled bool
}

type (
	paginationLocalPageKey struct{}
	paginationCornerKey    struct{}
	paginationFirstKey     struct{}
	paginationPrevKey      struct{}
	paginationPageKey      struct{ page int }
	paginationEllipsisKey  struct{ end bool }
	paginationNextKey      struct{}
	paginationLastKey      struct{}
)

type paginationMetrics struct {
	btnH     float32
	padX     float32
	gap      float32
	iconSize float32
	pressedR float32
	typeRole tokens.TypeRole
}

func paginationMetricsFor(d tokens.Density) paginationMetrics {
	switch d.Normalize() {
	case tokens.DensityCompact:
		return paginationMetrics{
			btnH:     28,
			padX:     6,
			gap:      2,
			iconSize: 16,
			pressedR: 6,
			typeRole: tokens.LabelMedium,
		}
	case tokens.DensityDesktop:
		return paginationMetrics{
			btnH:     32,
			padX:     8,
			gap:      3,
			iconSize: 18,
			pressedR: 6,
			typeRole: tokens.LabelLarge,
		}
	default:
		return paginationMetrics{
			btnH:     36,
			padX:     10,
			gap:      4,
			iconSize: 20,
			pressedR: 8,
			typeRole: tokens.LabelLarge,
		}
	}
}

// Pagination builds a Material Design 3 Expressive 1-based pagination bar
// (pages 1..totalPages) inside a ui.RoleNavigation horizontal focus group:
// optional First/Last edge buttons, Previous/Next icon buttons, TabularNums
// page capsules with FastSpatial press corner morphing, and automatic ellipsis
// windowing when totalPages exceeds the visible window.
//
// Out-of-bounds *page values are safely clamped into [1, totalPages] (or 0 when
// totalPages <= 0), Previous/First disable automatically on page 1, Next/Last
// disable on totalPages, and navigation icons mirror cleanly in ui.RightToLeft.
func Pagination(c *ui.Context, page *int, totalPages int, opts ...PaginationOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Pagination takes at most one PaginationOptions")
	}
	var o PaginationOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	pm := paginationMetricsFor(th.Density)
	rtl := c.Direction() == ui.RightToLeft

	label := o.Label
	if label == "" {
		label = "Pagination"
	}

	root := ui.Row(c).
		Role(ui.RoleNavigation).
		Label(label).
		FocusGroup(ui.Horizontal).
		Shrink(0).
		AlignItems(ui.Center).
		Gap(pm.gap).
		DimDisabled(false).
		Disabled(o.Disabled)

	pagePtr := page
	if pagePtr == nil {
		pagePtr = ui.Local(root, paginationLocalPageKey{}, func() int { return 1 })
	}

	if totalPages <= 0 {
		totalPages = 0
		*pagePtr = 0
	} else {
		if *pagePtr < 1 {
			*pagePtr = 1
		} else if *pagePtr > totalPages {
			*pagePtr = totalPages
		}
	}

	cur := *pagePtr
	disabledAll := o.Disabled || root.IsDisabled()
	atStart := disabledAll || totalPages <= 0 || cur <= 1
	atEnd := disabledAll || totalPages <= 0 || cur >= totalPages

	slots := computePaginationSlots(cur, totalPages, o.SiblingCount, o.MaxButtons)
	nextPage := cur
	hasUpdate := false

	prevIcon := mdicon.ChevronLeft.Draw
	nextIcon := mdicon.ChevronRight.Draw
	firstIcon := drawPaginationFirstIcon
	lastIcon := drawPaginationLastIcon
	if rtl {
		prevIcon = mdicon.ChevronRight.Draw
		nextIcon = mdicon.ChevronLeft.Draw
		firstIcon = drawPaginationLastIcon
		lastIcon = drawPaginationFirstIcon
	}

	buildNavButton := func(key any, btnLabel string, icon IconDraw, disabled bool, target int) {
		el := ui.ButtonBase(c.Key(key)).
			DimDisabled(false).
			FocusRing(false).
			Size(pm.btnH, pm.btnH).
			Shrink(0).
			AlignItems(ui.Center).
			Justify(ui.Center).
			Label(btnLabel).
			Disabled(disabled)

		ink := cs.OnSurfaceVariant
		if disabled {
			ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		}
		el.TextColor(ink)

		targetR := pm.btnH / 2
		if !disabled && el.Pressed() {
			targetR = pm.pressedR
		}
		radius := animateWithTheme(el, th, paginationCornerKey{}, targetR, motion.FastSpatial)
		el.Radius(radius)

		attachRippleWithContext(c, th, el, StateLayerOptions{
			Color:    cs.OnSurface,
			Radius:   radius,
			Disabled: disabled,
		})

		el.Children(func() {
			drawIconBox(c, icon, pm.iconSize, ink)
		})

		if !disabled && el.Clicked() {
			nextPage = target
			hasUpdate = true
		}
	}

	buildPageButton := func(p int) {
		active := p == cur
		el := ui.ButtonBase(c.Key(paginationPageKey{page: p})).
			DimDisabled(false).
			FocusRing(false).
			Height(pm.btnH).
			MinWidth(pm.btnH).
			PaddingX(pm.padX).
			Shrink(0).
			AlignItems(ui.Center).
			Justify(ui.Center).
			Label("Page " + strconv.Itoa(p)).
			Disabled(disabledAll).
			AccessSelected(active)
		if active {
			el.Current(ui.CurrentPage)
		}

		pal := paginationPagePalette(th, o.Variant, active, disabledAll)
		el.TextColor(pal.content)
		if pal.container.A > 0 {
			el.Background(pal.container)
		}
		if pal.border.A > 0 {
			el.Border(1, pal.border)
		}

		targetR := pm.btnH / 2
		if !disabledAll && el.Pressed() {
			targetR = pm.pressedR
		}
		radius := animateWithTheme(el, th, paginationCornerKey{}, targetR, motion.FastSpatial)
		el.Radius(radius)

		borderInset := float32(0)
		if pal.border.A > 0 {
			borderInset = 1
		}
		attachRippleWithContext(c, th, el, StateLayerOptions{
			Color:       pal.content,
			Radius:      radius,
			BorderInset: borderInset,
			Disabled:    disabledAll,
		})

		el.Children(func() {
			txt := ui.Text(c, strconv.Itoa(p)).NoWrap()
			applyTypeStyleWithTheme(th, txt, pm.typeRole, active)
			TabularNums(txt)
			txt.TextColor(pal.content)
		})

		if !disabledAll && el.Clicked() {
			nextPage = p
			hasUpdate = true
		}
	}

	buildEllipsis := func(end bool) {
		ink := cs.OnSurfaceVariant
		if disabledAll {
			ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		}
		box := ui.Box(c.Key(paginationEllipsisKey{end: end})).
			Size(pm.btnH, pm.btnH).
			Shrink(0).
			AlignItems(ui.Center).
			Justify(ui.Center).
			Role(ui.RoleNone)
		box.Children(func() {
			drawIconBox(c, mdicon.MoreHoriz.Draw, pm.iconSize, ink)
		})
	}

	root.Children(func() {
		if o.ShowEdges {
			buildNavButton(paginationFirstKey{}, "First page", firstIcon, atStart, 1)
		}
		buildNavButton(paginationPrevKey{}, "Previous page", prevIcon, atStart, max(1, cur-1))

		for i := range slots {
			sl := slots[i]
			switch sl.kind {
			case paginationSlotEllipsisStart:
				buildEllipsis(false)
			case paginationSlotEllipsisEnd:
				buildEllipsis(true)
			default:
				buildPageButton(sl.page)
			}
		}

		buildNavButton(paginationNextKey{}, "Next page", nextIcon, atEnd, min(totalPages, cur+1))
		if o.ShowEdges {
			buildNavButton(paginationLastKey{}, "Last page", lastIcon, atEnd, totalPages)
		}
	})

	if hasUpdate && !disabledAll && totalPages >= 1 && nextPage != cur {
		*pagePtr = nextPage
		if o.OnChanged != nil {
			o.OnChanged(nextPage)
		}
	}

	return root
}

func paginationPagePalette(th *Theme, variant ButtonVariant, active, disabled bool) buttonColors {
	cs := th.Colors
	if disabled {
		content := cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		if active {
			return buttonColors{
				container: cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
				content:   content,
			}
		}
		if variant == ButtonOutlined {
			return buttonColors{
				content: content,
				border:  cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
			}
		}
		return buttonColors{content: content}
	}
	if active {
		switch variant {
		case ButtonTonal:
			return buttonColors{container: cs.SecondaryContainer, content: cs.OnSecondaryContainer}
		case ButtonElevated:
			return buttonColors{container: cs.PrimaryContainer, content: cs.OnPrimaryContainer}
		default:
			return buttonColors{container: cs.Primary, content: cs.OnPrimary}
		}
	}
	if variant == ButtonOutlined {
		return buttonColors{content: cs.OnSurface, border: cs.OutlineVariant}
	}
	return buttonColors{content: cs.OnSurface}
}

func drawPaginationFirstIcon(p *ui.Painter, r ui.Rect, ink ui.Color) {
	if p == nil || r.W <= 0 || r.H <= 0 || ink.A == 0 {
		return
	}
	barW := max(float32(1.5), r.W*0.09)
	barH := r.H * 0.5
	barX := r.X + r.W*0.22
	barY := r.Y + (r.H-barH)/2
	p.Fill(ui.Rect{X: barX, Y: barY, W: barW, H: barH}, ink, barW/2)
	mdicon.ChevronLeft.Draw(p, ui.Rect{
		X: r.X + r.W*0.16,
		Y: r.Y,
		W: r.W * 0.84,
		H: r.H,
	}, ink)
}

func drawPaginationLastIcon(p *ui.Painter, r ui.Rect, ink ui.Color) {
	if p == nil || r.W <= 0 || r.H <= 0 || ink.A == 0 {
		return
	}
	barW := max(float32(1.5), r.W*0.09)
	barH := r.H * 0.5
	barX := r.X + r.W*0.78 - barW
	barY := r.Y + (r.H-barH)/2
	p.Fill(ui.Rect{X: barX, Y: barY, W: barW, H: barH}, ink, barW/2)
	mdicon.ChevronRight.Draw(p, ui.Rect{
		X: r.X,
		Y: r.Y,
		W: r.W * 0.84,
		H: r.H,
	}, ink)
}

type paginationSlotKind uint8

const (
	paginationSlotPage paginationSlotKind = iota
	paginationSlotEllipsisStart
	paginationSlotEllipsisEnd
)

type paginationSlot struct {
	kind paginationSlotKind
	page int
}

func computePaginationSlots(cur, total, siblingCount, maxButtons int) []paginationSlot {
	if total <= 0 {
		return nil
	}
	if maxButtons > 0 && total <= maxButtons {
		out := make([]paginationSlot, total)
		for i := range total {
			out[i] = paginationSlot{kind: paginationSlotPage, page: i + 1}
		}
		return out
	}

	sib := siblingCount
	if sib <= 0 {
		sib = 1
	}
	if maxButtons > 0 {
		if maxButtons < 3 {
			maxButtons = 3
		}
		if maxButtons == 3 {
			if cur <= 1 {
				return []paginationSlot{
					{kind: paginationSlotPage, page: 1},
					{kind: paginationSlotEllipsisEnd},
					{kind: paginationSlotPage, page: total},
				}
			}
			if cur >= total {
				return []paginationSlot{
					{kind: paginationSlotPage, page: 1},
					{kind: paginationSlotEllipsisStart},
					{kind: paginationSlotPage, page: total},
				}
			}
			return []paginationSlot{
				{kind: paginationSlotPage, page: 1},
				{kind: paginationSlotPage, page: cur},
				{kind: paginationSlotPage, page: total},
			}
		}
		if maxButtons == 4 {
			if cur <= 2 {
				return []paginationSlot{
					{kind: paginationSlotPage, page: 1},
					{kind: paginationSlotPage, page: 2},
					{kind: paginationSlotEllipsisEnd},
					{kind: paginationSlotPage, page: total},
				}
			}
			if cur >= total-1 {
				return []paginationSlot{
					{kind: paginationSlotPage, page: 1},
					{kind: paginationSlotEllipsisStart},
					{kind: paginationSlotPage, page: total - 1},
					{kind: paginationSlotPage, page: total},
				}
			}
			return []paginationSlot{
				{kind: paginationSlotPage, page: 1},
				{kind: paginationSlotPage, page: cur},
				{kind: paginationSlotEllipsisEnd},
				{kind: paginationSlotPage, page: total},
			}
		}
		sib = max(0, (maxButtons-5)/2)
	}

	const boundary = 1
	totalSlots := boundary*2 + sib*2 + 3
	if total <= totalSlots {
		out := make([]paginationSlot, total)
		for i := range total {
			out[i] = paginationSlot{kind: paginationSlotPage, page: i + 1}
		}
		return out
	}

	leftSibling := max(cur-sib, 1)
	rightSibling := min(cur+sib, total)
	showLeftDots := leftSibling > boundary+2
	showRightDots := rightSibling < total-boundary-1

	var out []paginationSlot
	switch {
	case !showLeftDots && showRightDots:
		leftCount := 3 + 2*sib
		out = make([]paginationSlot, 0, totalSlots)
		for p := 1; p <= leftCount; p++ {
			out = append(out, paginationSlot{kind: paginationSlotPage, page: p})
		}
		out = append(out,
			paginationSlot{kind: paginationSlotEllipsisEnd},
			paginationSlot{kind: paginationSlotPage, page: total},
		)
	case showLeftDots && !showRightDots:
		rightCount := 3 + 2*sib
		out = make([]paginationSlot, 0, totalSlots)
		out = append(out,
			paginationSlot{kind: paginationSlotPage, page: 1},
			paginationSlot{kind: paginationSlotEllipsisStart},
		)
		for p := total - rightCount + 1; p <= total; p++ {
			out = append(out, paginationSlot{kind: paginationSlotPage, page: p})
		}
	case showLeftDots && showRightDots:
		out = make([]paginationSlot, 0, totalSlots)
		out = append(out,
			paginationSlot{kind: paginationSlotPage, page: 1},
			paginationSlot{kind: paginationSlotEllipsisStart},
		)
		for p := leftSibling; p <= rightSibling; p++ {
			out = append(out, paginationSlot{kind: paginationSlotPage, page: p})
		}
		out = append(out,
			paginationSlot{kind: paginationSlotEllipsisEnd},
			paginationSlot{kind: paginationSlotPage, page: total},
		)
	default:
		out = make([]paginationSlot, total)
		for i := range total {
			out[i] = paginationSlot{kind: paginationSlotPage, page: i + 1}
		}
	}
	return out
}
