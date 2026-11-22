package docview

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// OutlineOptions configures docview.Outline.
type OutlineOptions struct {
	// Title sets the optional outline header caption (defaults to "On this page"
	// when ShowHeader is true).
	Title string
	// ShowHeader renders a compact header caption above the outline items.
	ShowHeader bool
	// Scroll links clicks and active-section tracking to an enclosing
	// ui.Scroll container's ScrollState.
	Scroll *ui.ScrollState
	// Items optionally supplies heading entries when State is nil or when
	// callers provide a custom outline list.
	Items []OutlineItem
	// Active optionally binds the active heading Slug when using custom Items
	// or observing selection externally.
	Active *string
	// MaxDepth limits displayed headings to Level <= MaxDepth (<= 0 displays
	// all levels 1..6).
	MaxDepth int
	// EmptyText renders a muted placeholder when no headings exist.
	EmptyText string
	// OnSelect runs after the user activates an outline item.
	OnSelect func(item OutlineItem)
	// Label overrides the navigation landmark label ("Outline" when empty).
	Label string
	// Disabled disables interaction across all outline items.
	Disabled bool
}

type (
	outlineItemKey   struct{ slug string; i int }
	outlineActiveKey struct{}
	outlineRailKey   struct{}

	outlineRailState struct {
		ys     []float32
		hs     []float32
		hasBar bool
		curY   float32
		curH   float32
		lastAt time.Time
	}
)

// Outline builds an MD3E document table-of-contents navigation list designed
// to pair with docview.Markdown / docview.MarkdownStream inside
// md3e.SupportingPane.
//
// When bound to a *docview.State and optional ui.ScrollState, clicking a heading
// smoothly scrolls to that section (falling back to ui/markdown's Anchor
// ScrollIntoView when Scroll is nil) while scrolling the document highlights the
// active heading with an MD3E SecondaryContainer capsule and Primary Start-edge
// indicator bar (mirrored in ui.RightToLeft).
func Outline(c *ui.Context, state *State, opts ...OutlineOptions) ui.Element {
	if len(opts) > 1 {
		panic("docview: Outline takes at most one OutlineOptions")
	}
	var o OutlineOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := md3e.CurrentTheme(c)
	cs := th.Colors
	rtl := c.Direction() == ui.RightToLeft
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	effects := th.Animation(motion.FastEffects)

	if state != nil && o.Scroll != nil {
		state.attachScroll(o.Scroll)
	}

	var items []OutlineItem
	if len(o.Items) > 0 {
		items = o.Items
	} else if state != nil {
		items = state.items
	}

	visible := make([]OutlineItem, 0, len(items))
	visibleOrigIdx := make([]int, 0, len(items))
	minLevel := 6
	for origIdx, it := range items {
		if o.MaxDepth > 0 && it.Level > o.MaxDepth {
			continue
		}
		lvl := it.Level
		if lvl < 1 {
			lvl = 1
		}
		if lvl > 6 {
			lvl = 6
		}
		it.Level = lvl
		if lvl < minLevel {
			minLevel = lvl
		}
		visible = append(visible, it)
		visibleOrigIdx = append(visibleOrigIdx, origIdx)
	}

	label := o.Label
	if label == "" {
		label = "Outline"
	}

	var rowH, padY, indentStep float32
	switch th.Density.Normalize() {
	case tokens.DensityCompact:
		rowH, padY, indentStep = 28, 4, 10
	case tokens.DensityDesktop:
		rowH, padY, indentStep = 32, 5, 12
	default:
		rowH, padY, indentStep = 38, 7, 14
	}

	root := ui.Column(c).
		FillWidth().
		Gap(2).
		DimDisabled(false).
		Role(ui.RoleNavigation).
		Label(label)

	activeVisibleIdx := -1
	if len(visible) > 0 {
		if o.Active != nil && *o.Active != "" {
			for i, it := range visible {
				if it.Slug == *o.Active {
					activeVisibleIdx = i
					break
				}
			}
		}
		if activeVisibleIdx < 0 && state != nil && len(o.Items) == 0 {
			stateActive := state.ActiveIndex()
			for i, origIdx := range visibleOrigIdx {
				if origIdx == stateActive {
					activeVisibleIdx = i
					break
				}
			}
			if activeVisibleIdx < 0 && stateActive >= 0 {
				// Pick nearest preceding visible heading when deeper headings are filtered out.
				for i, origIdx := range visibleOrigIdx {
					if origIdx <= stateActive {
						activeVisibleIdx = i
					}
				}
			}
		}
		if activeVisibleIdx < 0 {
			activeVisibleIdx = 0
		}
	}

	root.Children(func() {
		if o.ShowHeader || o.Title != "" {
			title := o.Title
			if title == "" {
				title = "On this page"
			}
			hdr := ui.Row(c).FillWidth().Padding(4, 0, 6, 0).PaddingStart(12).PaddingEnd(8)
			hdr.Children(func() {
				md3e.TextWith(c, title, md3e.TextOptions{
					Role:       tokens.LabelMedium,
					Emphasized: true,
					Color:      pickColor(cs.OnSurfaceVariant, cs.OnSurface),
				}).Unselectable()
			})
		}

		if len(visible) == 0 {
			if o.EmptyText != "" {
				ui.Row(c).FillWidth().Padding(8, 0, 8, 0).PaddingStart(12).PaddingEnd(12).Children(func() {
					md3e.TextWith(c, o.EmptyText, md3e.TextOptions{
						Role:  tokens.BodySmall,
						Color: pickColor(cs.OnSurfaceVariant, cs.OnSurface.Alpha(0.7)),
					})
				})
			}
			return
		}

		listCol := ui.Column(c).FillWidth().Gap(2).FocusGroup(ui.Vertical)
		rail := ui.Local(listCol, outlineRailKey{}, func() outlineRailState { return outlineRailState{} })
		if len(rail.ys) != len(visible) {
			rail.ys = make([]float32, len(visible))
			rail.hs = make([]float32, len(visible))
		}
		guideCol := pickColor(cs.OutlineVariant, cs.Outline).Alpha(0.55)
		barCol := pickColor(cs.Primary, md3e.DefaultSeedColor)
		pillBg := pickColor(cs.SecondaryContainer, barCol.Alpha(0.14))
		pillInk := pickColor(cs.OnSecondaryContainer, barCol)
		restInk := pickColor(cs.OnSurfaceVariant, cs.OnSurface)
		if o.Disabled {
			pillInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
			restInk = pillInk
			barCol = pillInk
		}

		listCol.Draw(func(p *ui.Painter, r ui.Rect) {
			if r.W <= 0 || r.H <= 0 || guideCol.A == 0 {
				return
			}
			x := r.X + 2
			if rtl {
				x = r.X + r.W - 3
			}
			p.Fill(ui.Rect{X: x, Y: r.Y + 4, W: 1, H: max(0, r.H-8)}, guideCol, 0.5)
		})

		listCol.Children(func() {
			for i, it := range visible {
				idx := i
				isActive := i == activeVisibleIdx
				indent := float32(it.Level-minLevel) * indentStep
				radius := minF32(rowH/2, th.Shapes.Radius(tokens.ShapeSmall))
				if radius <= 0 {
					radius = 8
				}

				btn := ui.ButtonBase(c.Key(outlineItemKey{slug: it.Slug, i: i})).
					Row().
					Justify(ui.Start).
					AlignItems(ui.Center).
					DimDisabled(false).
					FocusRing(false).
					FillWidth().
					MinHeight(rowH).
					Padding(padY, 0, padY, 0).
					PaddingStart(16 + indent).
					PaddingEnd(10).
					Radius(radius).
					Disabled(o.Disabled).
					Role(ui.RoleButton).
					Label(it.Title)

				if isActive {
					btn.Current(ui.CurrentLocation)
				}

				mixTarget := float32(0)
				if isActive {
					mixTarget = 1
				}
				mixT := mixTarget
				if reduce {
					mixT = btn.Jump(outlineActiveKey{}, mixTarget)
				} else {
					mixT = btn.AnimateTo(outlineActiveKey{}, mixTarget, effects)
				}
				if mixT < 0 {
					mixT = 0
				} else if mixT > 1 {
					mixT = 1
				}

				if mixT > 0.01 && !o.Disabled {
					btn.Background(pillBg.Alpha(mixT))
				}

				ink := restInk.MixOklab(pillInk, mixT)
				if o.Disabled {
					ink = restInk
				}
				btn.TextColor(ink)

				// Record exact paint geometry into rail before listCol.DrawOver runs
				// (keeps btn.DrawOver completely intact for AttachRipple).
				btn.Draw(func(_ *ui.Painter, r ui.Rect) {
					if idx < len(rail.ys) {
						rail.ys[idx] = r.Y
						rail.hs[idx] = r.H
					}
				})

				md3e.AttachRipple(c, btn, md3e.StateLayerOptions{
					Color:    ink,
					Radius:   radius,
					Disabled: o.Disabled,
				})

				btn.Children(func() {
					md3e.TextWith(c, it.Title, md3e.TextOptions{
						Role:       tokens.BodySmall,
						Emphasized: isActive,
						MaxLines:   2,
						Color:      ink,
					}).Shrink(1).MinWidth(0)
				})

				if btn.Clicked() && !o.Disabled {
					if o.Active != nil {
						*o.Active = it.Slug
					}
					if state != nil && len(o.Items) == 0 {
						origIdx := visibleOrigIdx[i]
						state.jumpToIndex(origIdx, jumpAnimationFor(th, c), true)
					}
					if o.OnSelect != nil {
						o.OnSelect(it)
					}
					c.Invalidate()
				}
			}
		})

		// Single gliding Primary indicator bar along the Start rail (ghost-draw safe).
		activeIdx := activeVisibleIdx
		listCol.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if r.W <= 0 || r.H <= 0 || barCol.A == 0 || activeIdx < 0 || activeIdx >= len(rail.ys) {
				return
			}
			live := listCol.Valid()
			red := reduce || p.Preferences().ReduceMotion
			if live {
				targetY := rail.ys[activeIdx]
				targetH := rail.hs[activeIdx]
				if targetH > 0 {
					now := p.Now()
					if !rail.hasBar || red {
						rail.curY = targetY
						rail.curH = targetH
						rail.hasBar = true
						rail.lastAt = now
					} else {
						dt := float64(0)
						if !rail.lastAt.IsZero() && now.After(rail.lastAt) {
							dt = now.Sub(rail.lastAt).Seconds()
						}
						rail.lastAt = now
						if dt > 0 {
							k := float32(1 - math.Exp(-dt/0.055))
							rail.curY += (targetY - rail.curY) * k
							rail.curH += (targetH - rail.curH) * k
						}
						if math.Abs(float64(targetY-rail.curY)) <= 0.25 && math.Abs(float64(targetH-rail.curH)) <= 0.25 {
							rail.curY = targetY
							rail.curH = targetH
						} else {
							p.AnimationFrame()
						}
					}
				}
			}
			if !rail.hasBar || rail.curH <= 0 {
				return
			}
			barW := float32(3)
			barH := max(10, rail.curH-12)
			x := r.X + 1
			if rtl {
				x = r.X + r.W - barW - 1
			}
			y := rail.curY + (rail.curH-barH)*0.5
			p.Fill(ui.Rect{X: x, Y: y, W: barW, H: barH}, barCol, barW*0.5)
		})
	})

	return root
}

func minF32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
