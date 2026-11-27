package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// PillTabsTone selects the color pairing for the sliding active capsule of
// PillTabs and SegmentedTabs.
type PillTabsTone int

const (
	// PillTabsToneAuto selects PillTabsTonePrimary for PillTabs, and
	// PillTabsToneSurface for SegmentedTabs.
	PillTabsToneAuto PillTabsTone = iota
	// PillTabsTonePrimary paints a Primary active capsule with OnPrimary text
	// over the unified tonal trough (Dark SurfaceContainerLowest / Light
	// SurfaceContainer).
	PillTabsTonePrimary
	// PillTabsToneSecondary paints a SecondaryContainer active capsule with
	// OnSecondaryContainer text over the unified tonal trough.
	PillTabsToneSecondary
	// PillTabsToneSurface paints a neutral tonal active capsule (Dark
	// SurfaceContainerHigh / Light SurfaceContainerLowest) with OnSurface text
	// over the unified tonal trough.
	PillTabsToneSurface
)

// PillTabsOptions configures PillTabs and SegmentedTabs. The zero value is an
// enabled, content-sized pill tab strip that manages its own selection.
type PillTabsOptions struct {
	// Selected points to the active tab index. When nil, the tab bar stores its
	// selection in local state.
	Selected *int
	// Changed runs after a click or keyboard navigation changes the selection.
	Changed func(i int)
	// Tone selects the active capsule and trough color roles.
	Tone PillTabsTone
	// Scrollable lays tabs out at their natural widths inside a horizontal
	// scroller, automatically scrolling the chosen tab into view.
	Scrollable bool
	// FullWidth stretches the capsule trough across its parent's width and
	// shares width equally among tabs (ignored when Scrollable is true).
	FullWidth bool
	// Label names the tab list for assistive technology; "Tabs" when empty.
	Label string
	// Disabled disables every tab in the strip.
	Disabled bool
}

type (
	pillTabsSelectedKey  struct{}
	pillTabsLastKey      struct{}
	pillTabsIndicatorKey struct{}
	pillTabsIndXKey      struct{}
	pillTabsIndWKey      struct{}
	pillTabsScrollKey    struct{}
	pillTabsTabKey       struct{ i int }
	pillTabsTabMixKey    struct{}
)

type pillTabsIndicatorState struct {
	x, w           float32
	set            bool
	builtRelX      float32
	builtTabW      float32
	builtListW     float32
	paintedListX   float32
	paintedListW   float32
	paintedTab     ui.Rect
}

type pillTabsMetrics struct {
	outerH   float32
	trackPad float32
	tabPadX  float32
	iconSize float32
	gap      float32
	typeRole tokens.TypeRole
}

func pillTabsMetricsFor(d tokens.Density, segmented bool) pillTabsMetrics {
	if segmented {
		switch d.Normalize() {
		case tokens.DensityCompact:
			return pillTabsMetrics{outerH: 28, trackPad: 2, tabPadX: 10, iconSize: 15, gap: 5, typeRole: tokens.LabelMedium}
		case tokens.DensityDesktop:
			return pillTabsMetrics{outerH: 32, trackPad: 3, tabPadX: 12, iconSize: 16, gap: 6, typeRole: tokens.LabelMedium}
		default:
			return pillTabsMetrics{outerH: 38, trackPad: 3, tabPadX: 14, iconSize: 18, gap: 6, typeRole: tokens.LabelLarge}
		}
	}
	switch d.Normalize() {
	case tokens.DensityCompact:
		return pillTabsMetrics{outerH: 32, trackPad: 3, tabPadX: 12, iconSize: 16, gap: 6, typeRole: tokens.LabelMedium}
	case tokens.DensityDesktop:
		return pillTabsMetrics{outerH: 36, trackPad: 3, tabPadX: 14, iconSize: 18, gap: 6, typeRole: tokens.LabelLarge}
	default:
		return pillTabsMetrics{outerH: 44, trackPad: 4, tabPadX: 16, iconSize: 20, gap: 8, typeRole: tokens.LabelLarge}
	}
}

type pillTabsPalette struct {
	trough       ui.Color
	troughBorder ui.Color
	capsule      ui.Color
	capsuleEdge  ui.Color
	borderW      float32
	activeInk    ui.Color
	inactiveInk  ui.Color
	badgeBg      ui.Color
	badgeInk     ui.Color
	activeBdgBg  ui.Color
	activeBdgInk ui.Color
	dotCol       ui.Color
}

func pillTabsPaletteOf(th *Theme, tone PillTabsTone, segmented, disabled bool) pillTabsPalette {
	cs := th.Colors
	resolved := tone
	if resolved == PillTabsToneAuto || resolved < PillTabsToneAuto || resolved > PillTabsToneSurface {
		if segmented {
			resolved = PillTabsToneSurface
		} else {
			resolved = PillTabsTonePrimary
		}
	}

	// Unified borderless trough across all tones: deeper in Dark mode so the
	// active capsule always reads as lifted, soft Container in Light mode.
	trough := cs.SurfaceContainer
	if trough.A == 0 {
		trough = SurfaceColorForElevation(th, tokens.Level2)
	}
	if th.Dark {
		trough = cs.SurfaceContainerLowest
		if trough.A == 0 {
			trough = cs.SurfaceDim
		}
	}

	var pal pillTabsPalette
	switch resolved {
	case PillTabsToneSecondary:
		pal = pillTabsPalette{
			trough:       trough,
			capsule:      cs.SecondaryContainer,
			activeInk:    cs.OnSecondaryContainer,
			inactiveInk:  cs.OnSurfaceVariant,
			badgeBg:      trough,
			badgeInk:     cs.OnSurfaceVariant,
			activeBdgBg:  cs.SecondaryContainer,
			activeBdgInk: cs.OnSecondaryContainer,
			dotCol:       cs.Error,
		}
	case PillTabsToneSurface:
		capsule := cs.SurfaceContainerLowest
		if capsule.A == 0 {
			capsule = cs.Surface
		}
		if th.Dark {
			capsule = cs.SurfaceContainerHigh
			if capsule.A == 0 {
				capsule = SurfaceColorForElevation(th, tokens.Level3)
			}
		}
		pal = pillTabsPalette{
			trough:       trough,
			capsule:      capsule,
			activeInk:    cs.OnSurface,
			inactiveInk:  cs.OnSurfaceVariant,
			badgeBg:      trough,
			badgeInk:     cs.OnSurfaceVariant,
			activeBdgBg:  capsule,
			activeBdgInk: cs.OnSurface,
			dotCol:       cs.Error,
		}
	default: // PillTabsTonePrimary
		pal = pillTabsPalette{
			trough:       trough,
			capsule:      cs.Primary,
			activeInk:    cs.OnPrimary,
			inactiveInk:  cs.OnSurfaceVariant,
			badgeBg:      trough,
			badgeInk:     cs.OnSurfaceVariant,
			activeBdgBg:  cs.Primary,
			activeBdgInk: cs.OnPrimary,
			dotCol:       cs.Error,
		}
	}

	if !disabled {
		if hcW, hcCol := highContrastBorder(th); hcW > 0 {
			pal.borderW = hcW
			pal.troughBorder = hcCol
			pal.capsuleEdge = hcCol
		}
	}

	if pal.dotCol.A == 0 {
		pal.dotCol = ui.RGB(0xB3, 0x26, 0x1E)
	}
	if disabled {
		disContainer := cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		disContent := cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		pal.capsule = disContainer
		pal.troughBorder = ui.Transparent
		pal.capsuleEdge = ui.Transparent
		pal.borderW = 0
		pal.activeInk = disContent
		pal.inactiveInk = disContent
		pal.badgeBg = disContainer
		pal.badgeInk = disContent
		pal.activeBdgBg = disContainer
		pal.activeBdgInk = disContent
		pal.dotCol = disContent
	}
	return pal
}

// PillTabs builds a Material Design 3 Expressive capsule tab bar on top of
// ui.TabsBase: a full-pill tonal trough with a FastSpatial spring-animated
// sliding active capsule, optional tab icons and count/dot badges, full
// RoleTabList/RoleTab semantics, Arrow/Home/End keyboard navigation (skipping
// disabled tabs and mirroring right-to-left), and Theme.Density scaling.
//
// On the first laid-out frame the active capsule snaps directly onto the
// selected tab rather than sliding in from zero.
func PillTabs(c *ui.Context, tabs []TabItem, opts ...PillTabsOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: PillTabs takes at most one PillTabsOptions")
	}
	var o PillTabsOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	return buildPillTabs(c, tabs, o, false)
}

// SegmentedTabs builds a compact segmented capsule tab control on top of
// ui.TabsBase: a tighter density-aware trough (defaulting to SurfaceContainer
// with a SurfaceContainerLowest active capsule and subtle hairline framing)
// suited for desktop toolbars, view switchers, and filter bars.
func SegmentedTabs(c *ui.Context, tabs []TabItem, opts ...PillTabsOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: SegmentedTabs takes at most one PillTabsOptions")
	}
	var o PillTabsOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	return buildPillTabs(c, tabs, o, true)
}

func buildPillTabs(c *ui.Context, items []TabItem, o PillTabsOptions, segmented bool) ui.Element {
	th := CurrentTheme(c)
	disabledInk := th.Colors.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	disabledContainer := th.Colors.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
	label := o.Label
	if label == "" {
		label = "Tabs"
	}
	m := pillTabsMetricsFor(th.Density, segmented)
	innerH := max(m.outerH-2*m.trackPad, 16)
	outerRadius := m.outerH / 2
	innerRadius := innerH / 2

	root := ui.Row(c).Height(m.outerH).DimDisabled(false).Shrink(0)
	if o.FullWidth || o.Scrollable {
		root.FillWidth()
	}
	root.Disabled(o.Disabled)

	selected := o.Selected
	if selected == nil {
		selected = ui.Local(root, pillTabsSelectedKey{}, func() int { return 0 })
	}
	lastSelected := ui.Local(root, pillTabsLastKey{}, func() int { return -1 })
	scrollState := ui.Local(root, pillTabsScrollKey{}, func() ui.ScrollState { return ui.ScrollState{} })
	if len(items) == 0 {
		*lastSelected = -1
		return root
	}
	*selected = max(0, min(*selected, len(items)-1))

	disabledAll := o.Disabled || root.IsDisabled()
	pal := pillTabsPaletteOf(th, o.Tone, segmented, disabledAll)
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	rtl := c.Direction() == ui.RightToLeft

	var tabs ui.TabsParts
	var stackTabEls [12]ui.Element
	var tabEls []ui.Element
	if len(items) <= len(stackTabEls) {
		tabEls = stackTabEls[:len(items)]
	} else {
		tabEls = make([]ui.Element, len(items))
	}

	buildList := func() {
		tabs = ui.TabsBase(c, selected, len(items))
		tabs.List.
			DimDisabled(false).
			Label(label).
			Height(m.outerH).
			Padding(m.trackPad).
			Radius(outerRadius).
			Background(pal.trough).
			AlignItems(ui.Center)
		if pal.troughBorder.A > 0 && pal.borderW > 0 {
			tabs.List.Border(pal.borderW, pal.troughBorder)
		}
		if o.FullWidth && !o.Scrollable {
			tabs.List.FillWidth()
		}
		lowGraphicContrast := !disabledAll && pal.capsule.A > 0 && pal.trough.A > 0 &&
			tokens.ContrastRatio(pal.capsule, pal.trough) < 3.0
		tabs.List.Children(func() {
			for i := range items {
				item := &items[i]
				var key any = pillTabsTabKey{i: i}
				if item.Key != nil {
					key = item.Key
				}
				c.Key(key)
				tab := tabs.Tab(i).
					DimDisabled(false).
					FocusRing(false).
					Height(innerH).
					PaddingX(m.tabPadX).
					Gap(m.gap).
					Radius(innerRadius).
					AlignItems(ui.Center).
					Justify(ui.Center)
				if o.FullWidth && !o.Scrollable {
					tab.Grow(1)
				} else {
					tab.Shrink(0)
				}
				tabEls[i] = tab

				on := *selected == i
				itemDisabled := disabledAll || item.Disabled || tab.IsDisabled()
				selMix := clampF32(animateValue(tab, pillTabsTabMixKey{}, boolFloat(on), th.Animation(motion.FastEffects), reduce), 0, 1)
				ink := pal.inactiveInk.MixOklab(pal.activeInk, selMix)
				if itemDisabled {
					ink = disabledInk
				}
				tab.Disabled(o.Disabled || item.Disabled).
					AccessSelected(on).
					TextColor(ink)
				if item.Label != "" {
					tab.Label(item.Label)
				}
				if item.Badge != "" {
					tab.Description(item.Badge)
				}

				tab.Children(func() {
					if item.Icon != nil {
						drawIconBox(c, item.Icon, m.iconSize, ink)
					} else if lowGraphicContrast && on && !itemDisabled {
						drawIconBox(c, mdicon.Check.Draw, 14, ink)
					}
					if item.Label != "" {
						txt := ui.Text(c, item.Label).NoWrap()
						applyTypeStyleWithTheme(th, txt, m.typeRole, on)
						txt.TextColor(ink)
					}
					if item.Badge != "" {
						bdgBg := pal.badgeBg.MixOklab(pal.activeBdgBg, selMix)
						bdgInk := pal.badgeInk.MixOklab(pal.activeBdgInk, selMix)
						if itemDisabled {
							bdgBg = disabledContainer
							bdgInk = disabledInk
						}
						Badge(c, item.Badge, BadgeOptions{
							ShowZero:       true,
							ContainerColor: bdgBg,
							ContentColor:   bdgInk,
						})
					} else if item.BadgeDot {
						dotCol := pal.dotCol
						if itemDisabled {
							dotCol = disabledInk
						}
						SmallBadge(c, BadgeOptions{ContainerColor: dotCol})
					}
				})

				attachRippleWithContext(c, th, tab, StateLayerOptions{
					Color:     ink,
					Radius:    innerRadius,
					FocusRing: FocusRingInset,
					Disabled:  itemDisabled,
				})
			}
		})
	}

	var scrollEl ui.Element
	if o.Scrollable {
		root.Children(func() {
			scrollEl = ui.ScrollHorizontal(c).
				TrackScroll(scrollState).
				DimDisabled(false).
				Height(m.outerH).
				FillWidth().
				Scrollbars(false)
			scrollEl.Disabled(o.Disabled)
			scrollEl.Children(func() {
				buildList()
			})
		})
	} else {
		root.Children(func() {
			buildList()
		})
	}

	if tabs.List.Changed() && o.Changed != nil {
		o.Changed(*selected)
	}

	indicator := ui.Local(tabs.List, pillTabsIndicatorKey{}, func() pillTabsIndicatorState {
		return pillTabsIndicatorState{}
	})
	indicator.paintedTab = ui.Rect{}

	listBox := tabs.List.LayoutBounds()
	selTab := tabEls[*selected]
	tabBox := selTab.LayoutBounds()
	if listBox.W > 0 && tabBox.W > 0 {
		relX := tabBox.X - listBox.X
		targetW := tabBox.W
		indicator.builtRelX, indicator.builtTabW, indicator.builtListW = relX, tabBox.W, listBox.W
		if !indicator.set {
			indicator.set = true
			indicator.x = tabs.List.Jump(pillTabsIndXKey{}, relX)
			indicator.w = tabs.List.Jump(pillTabsIndWKey{}, targetW)
		} else {
			indicator.x = animateValue(tabs.List, pillTabsIndXKey{}, relX, th.Animation(motion.FastSpatial), reduce)
			indicator.w = animateValue(tabs.List, pillTabsIndWKey{}, targetW, th.Animation(motion.FastSpatial), reduce)
		}

		if o.Scrollable && *selected != *lastSelected {
			viewW := scrollEl.LayoutBounds().W
			if viewW > 0 {
				firstAlign := *lastSelected < 0
				*lastSelected = *selected
				relStart := relX
				if rtl {
					relStart = (listBox.X + listBox.W) - (tabBox.X + tabBox.W)
				}
				relEnd := relStart + tabBox.W
				scrollTarget := scrollState.X
				needScroll := false
				if relStart < scrollState.X {
					scrollTarget = relStart
					needScroll = true
				} else if relEnd > scrollState.X+viewW {
					scrollTarget = relEnd - viewW
					needScroll = true
				}
				if needScroll {
					clamped := clampF32(scrollTarget, 0, scrollState.MaxX)
					if absF32(scrollState.X-clamped) > 0.5 {
						if firstAlign || reduce {
							scrollState.X = clamped
						} else {
							scrollState.ScrollTo(clamped, 0, th.Animation(motion.DefaultSpatial))
						}
						c.Invalidate()
					}
				}
			}
		}
	} else if !indicator.set {
		c.Invalidate()
	}

	capsuleCol := pal.capsule
	capsuleEdge := pal.capsuleEdge
	if disabledAll || selTab.IsDisabled() {
		capsuleCol = disabledContainer
		capsuleEdge = ui.Transparent
	}

	srv := c.Services()
	builtRelX, builtTabW, builtListW := indicator.builtRelX, indicator.builtTabW, indicator.builtListW
	if selTab.Valid() {
		// Measure painted tab geometry inside Draw (not DrawOver) so the selected
		// tab retains its StateLayer & Ripple, and compare against tabList's
		// painted bounds which ran just prior to child painting.
		selTab.Draw(func(p *ui.Painter, tr ui.Rect) {
			if !selTab.Valid() || tr.W <= 0 || tr.H <= 0 {
				return
			}
			indicator.paintedTab = tr
			if indicator.paintedListW > 0 {
				if absF32(tr.W-builtTabW) > 0.5 ||
					absF32((tr.X-indicator.paintedListX)-builtRelX) > 0.5 ||
					absF32(indicator.paintedListW-builtListW) > 0.5 {
					srv.Invalidate()
				}
			}
		})
	}

	indX := indicator.x
	indW := indicator.w
	indReady := indicator.set
	trackPad := m.trackPad
	capsuleBorderW := pal.borderW

	// Paint the sliding capsule in Draw (underneath the tab labels, icons, and ripples).
	tabList := tabs.List
	tabList.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		if tabList.Valid() {
			indicator.paintedListX = r.X
			indicator.paintedListW = r.W
		}
		if !indReady {
			return
		}
		capH := max(r.H-2*trackPad, 0)
		capW := min(max(indW, 0), max(r.W-indX-trackPad, 0))
		if capW <= 0 || capH <= 0 {
			return
		}
		rad := capH / 2
		capRect := ui.Rect{
			X: r.X + indX,
			Y: r.Y + trackPad,
			W: capW,
			H: capH,
		}
		if capsuleCol.A > 0 {
			p.Fill(capRect, capsuleCol, rad)
		}
		if capsuleEdge.A > 0 && capsuleBorderW > 0 {
			half := capsuleBorderW * 0.5
			inRect := insetRect(capRect, half)
			if inRect.W > 0 && inRect.H > 0 {
				p.Stroke(inRect, capsuleEdge, max(0, rad-half), capsuleBorderW)
			}
		}
	})

	return root
}
