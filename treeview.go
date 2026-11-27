package md3e

import (
	"reflect"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// TreeNode describes one node in a TreeView hierarchy.
type TreeNode struct {
	// Key uniquely identifies this node among its siblings and serves as the
	// selection value when TreeViewOptions.Selected is used and SelectionID is
	// nil. Comparable keys keep keyboard focus and local expansion state
	// attached to the node across sibling expansions or insertions. When
	// TreeViewOptions.Selected is used without SelectionID, Key must be unique
	// across the entire tree (or include parent identity); for sibling-scoped
	// duplicate leaf keys across branches, set SelectionID.
	Key any
	// SelectionID optionally provides a tree-unique comparable identifier for
	// TreeViewOptions.Selected and OnSelect while keeping Key sibling-scoped.
	// When nil, Key is used.
	SelectionID any
	// Label is the visible node label and accessible name.
	Label string
	// Icon draws an optional leading icon between the disclosure chevron and
	// Label.
	Icon IconDraw
	// Badge displays an optional trailing count or short tag.
	Badge string
	// Open optionally controls whether a branch node is expanded. When nil and
	// Children is non-empty, TreeView manages expansion state locally via
	// ui.Local.
	Open *bool
	// Children lists nested nodes rendered beneath this branch while open.
	Children []TreeNode
	// Disabled disables interaction for this node and its subtree.
	Disabled bool
	// OnClick runs when the node row is activated via click, Enter, or Space
	// (toggling the disclosure chevron alone does not invoke OnClick).
	OnClick func()
}

// TreeViewOptions configures TreeView. The zero value produces an enabled,
// density-aware MD3E tree without indent guide lines.
type TreeViewOptions struct {
	// Selected optionally points to the currently selected node Key. Clicking a
	// row with a non-nil Key updates *Selected before invoking OnSelect and
	// TreeNode.OnClick.
	Selected *any
	// OnSelect runs when a row is activated via click, Enter, or Space, after
	// *Selected (if non-nil) has been updated.
	OnSelect func(key any)
	// ShowIndentGuides draws subtle vertical OutlineVariant connector lines
	// beneath open branches aligned with each parent's disclosure chevron.
	ShowIndentGuides bool
	// Label names the tree landmark for assistive technology ("Tree" when
	// empty).
	Label string
	// Disabled disables interaction across the entire tree.
	Disabled bool
}

type (
	treeAutoNodeKey  struct{ i int }
	treeNodeOpenKey  struct{}
	treeRowKey       struct{}
	treeRowInitKey   struct{}
	treeSelectMixKey struct{}
	treeArrowKey     struct{}
	treeArrowInitKey struct{}
	treeChevronDegKey struct{}
	treeArrowHoverKey struct{}
)

type treeArrowInit struct {
	set     bool
	lastRTL bool
}

type treeMetrics struct {
	rowH        float32
	rowGap      float32
	indentStep  float32
	padStart    float32
	padEnd      float32
	gap         float32
	arrowSize   float32
	chevronSize float32
	iconSize    float32
	radius      float32
	typeRole    tokens.TypeRole
	badgeRole   tokens.TypeRole
	badgeH      float32
}

func treeMetricsFor(th *Theme) treeMetrics {
	radius := th.Shapes.Radius(tokens.ShapeLarge)
	switch th.Density.Normalize() {
	case tokens.DensityCompact:
		return treeMetrics{
			rowH:        28,
			rowGap:      1,
			indentStep:  16,
			padStart:    6,
			padEnd:      8,
			gap:         4,
			arrowSize:   18,
			chevronSize: 14,
			iconSize:    16,
			radius:      min(radius, 14),
			typeRole:    tokens.BodySmall,
			badgeRole:   tokens.LabelSmall,
			badgeH:      16,
		}
	case tokens.DensityDesktop:
		return treeMetrics{
			rowH:        34,
			rowGap:      2,
			indentStep:  18,
			padStart:    8,
			padEnd:      10,
			gap:         6,
			arrowSize:   20,
			chevronSize: 16,
			iconSize:    18,
			radius:      min(radius, 17),
			typeRole:    tokens.BodyMedium,
			badgeRole:   tokens.LabelSmall,
			badgeH:      18,
		}
	default:
		return treeMetrics{
			rowH:        44,
			rowGap:      2,
			indentStep:  20,
			padStart:    10,
			padEnd:      12,
			gap:         8,
			arrowSize:   24,
			chevronSize: 18,
			iconSize:    20,
			radius:      min(radius, 22),
			typeRole:    tokens.BodyLarge,
			badgeRole:   tokens.LabelSmall,
			badgeH:      18,
		}
	}
}

// TreeView builds a Material Design 3 Expressive hierarchical tree over
// ui.TreeBase, preserving full RoleTree / RoleTreeItem semantics, single-Tab
// FocusGroup navigation, Up/Down/Left/Right/Home/End and type-ahead keyboard
// handling.
//
// Branch disclosure chevrons rotate smoothly between 0° and ±90° via
// motion.FastSpatial (mirrored in ui.RightToLeft), selected rows show a
// SecondaryContainer capsule with MD3E StateLayer / Ripple feedback in
// DrawOver, and optional OutlineVariant indent guide lines align vertically
// beneath expanded parents. When TreeNode.Open is nil on a branch with
// Children, expansion state falls back to a stable ui.Local binding keyed to
// the node.
func TreeView(c *ui.Context, nodes []TreeNode, opts ...TreeViewOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: TreeView takes at most one TreeViewOptions")
	}
	var o TreeViewOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	m := treeMetricsFor(th)
	label := o.Label
	if label == "" {
		label = "Tree"
	}

	// Build ui.TreeBase first so any caller-supplied c.Key(...) attaches directly
	// to the RoleTree root element.
	tree := ui.TreeBase(c)
	tree.Root.
		DimDisabled(false).
		FillWidth().
		Gap(m.rowGap).
		Label(label).
		Disabled(o.Disabled)

	disabledAll := o.Disabled || tree.Root.IsDisabled()
	rtl := c.Direction() == ui.RightToLeft

	if len(nodes) == 0 {
		return tree.Root
	}

	tree.Root.Children(func() {
		for i := range nodes {
			buildTreeNode(c, th, cs, tree, &nodes[i], i, o, m, disabledAll, rtl)
		}
	})
	return tree.Root
}

func buildTreeNode(
	c *ui.Context,
	th *Theme,
	cs tokens.ColorScheme,
	tree ui.TreeParts,
	n *TreeNode,
	idx int,
	o TreeViewOptions,
	m treeMetrics,
	parentDisabled bool,
	rtl bool,
) {
	slotKey := treeSlotKey(n.Key, idx)
	slot := ui.Column(c.Key(slotKey)).
		Role(ui.RoleNone).
		FillWidth().
		Gap(m.rowGap).
		DimDisabled(false).
		Disabled(n.Disabled)

	nodeDisabled := parentDisabled || n.Disabled || slot.IsDisabled()

	slot.Children(func() {
		openPtr := n.Open
		if openPtr == nil && len(n.Children) > 0 {
			openPtr = ui.Local(slot, treeNodeOpenKey{}, func() bool { return false })
		}

		c.Key(treeRowKey{})
		item := tree.Item(openPtr)
		depth := item.Depth()
		branch := openPtr != nil
		expanded := branch && *openPtr
		disabled := nodeDisabled || item.Row.IsDisabled()
		selKey := treeSelectionKey(n)
		selected := o.Selected != nil && treeKeysEqual(*o.Selected, selKey)

		selTarget := float32(0)
		if selected {
			selTarget = 1
		}
		rowInit := ui.Local(item.Row, treeRowInitKey{}, func() bool { return false })
		var selMix float32
		if !*rowInit {
			*rowInit = true
			selMix = item.Row.Jump(treeSelectMixKey{}, selTarget)
		} else {
			selMix = animateWithTheme(item.Row, th, treeSelectMixKey{}, selTarget, motion.FastEffects)
		}
		selMix = clampF32(selMix, 0, 1)

		selectedBg := cs.SecondaryContainer
		selectedInk := cs.OnSecondaryContainer
		if selectedBg.A == 0 {
			selectedBg = cs.PrimaryContainer
			selectedInk = cs.OnPrimaryContainer
		}

		bg := ui.Transparent
		if selMix > 0.001 && selectedBg.A > 0 {
			bg = selectedBg.Alpha(selMix)
		}
		ink := cs.OnSurface.MixOklab(selectedInk, selMix)
		subInk := cs.OnSurfaceVariant.MixOklab(selectedInk.Alpha(0.85), selMix)
		iconInk := cs.OnSurfaceVariant.MixOklab(selectedInk, selMix)
		if disabled {
			ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
			subInk = ink
			iconInk = ink
			if selected {
				bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
			} else {
				bg = ui.Transparent
			}
		}

		item.Row.
			DimDisabled(false).
			FocusRing(false).
			FillWidth().
			Height(m.rowH).
			PaddingStart(m.padStart + float32(depth)*m.indentStep).
			PaddingEnd(m.padEnd).
			Gap(m.gap).
			AlignItems(ui.Center).
			Radius(m.radius).
			Background(bg).
			TextColor(ink)
		if o.Selected != nil {
			item.Row.AccessSelected(selected)
		}
		if n.Badge != "" {
			item.Row.Description(n.Badge)
		}

		attachRippleWithContext(c, th, item.Row, StateLayerOptions{
			Color:    ink,
			Radius:   m.radius,
			Disabled: disabled,
		})

		rowClicked := !disabled && item.Row.Clicked()

		item.Row.Children(func() {
			arrow := ui.Box(c.Key(treeArrowKey{})).
				Size(m.arrowSize, m.arrowSize).
				Shrink(0).
				Radius(m.arrowSize / 2).
				Role(ui.RoleNone)
			if branch {
				if !disabled {
					arrow.KeepFocus()
				}
				if !disabled && arrow.Clicked() {
					*openPtr = !*openPtr
					expanded = *openPtr
				}
				// Animate normalized 0..1 progress so SpringOptions' default
				// precision (0.001) settles along the exact ~420 ms FastSpatial curve.
				targetProgress := float32(0)
				if expanded {
					targetProgress = 1
				}
				init := ui.Local(arrow, treeArrowInitKey{}, func() treeArrowInit { return treeArrowInit{} })
				var progress float32
				if !init.set {
					init.set = true
					init.lastRTL = rtl
					progress = arrow.Jump(treeChevronDegKey{}, targetProgress)
				} else {
					init.lastRTL = rtl
					progress = animateWithTheme(arrow, th, treeChevronDegKey{}, targetProgress, motion.FastSpatial)
				}
				deg := progress * 90
				if rtl {
					deg = -deg
				}

				hoverHalo := ui.Local(arrow, treeArrowHoverKey{}, func() haloFadeState { return haloFadeState{} })
				chevron := mdicon.ChevronRight.Draw
				if rtl {
					chevron = mdicon.ChevronLeft.Draw
				}
				chevronSz := m.chevronSize
				arrowInk := subInk
				thReduce := th.ReduceMotion
				arrowDisabled := disabled

				arrow.Draw(func(p *ui.Painter, r ui.Rect) {
					if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
						return
					}
					if arrow.Valid() && !arrowDisabled {
						reduce := thReduce || p.Preferences().ReduceMotion
						targetA := float32(0)
						if arrow.Hovered() {
							targetA = float32(tokens.StateHoverOpacity)
						}
						alpha, needFrame := hoverHalo.step(targetA, p.Now(), reduce)
						if needFrame && rectIntersects(r, p.Visible()) {
							p.After(16 * time.Millisecond)
						}
						if alpha > 0.001 {
							p.Fill(r, arrowInk.Alpha(alpha), min(r.W, r.H)/2)
						}
					}
					iconR := ui.Rect{
						X: r.X + (r.W-chevronSz)/2,
						Y: r.Y + (r.H-chevronSz)/2,
						W: chevronSz,
						H: chevronSz,
					}
					if absF32(deg) < 0.05 {
						chevron(p, iconR, arrowInk)
						return
					}
					p.Transform(ui.Rotation(deg), r.X+r.W/2, r.Y+r.H/2, func() {
						chevron(p, iconR, arrowInk)
					})
				})
			}

			if n.Icon != nil {
				drawIconBox(c, n.Icon, m.iconSize, iconInk)
			}

			if n.Label != "" {
				txt := ui.Text(c, n.Label).SingleLine().Shrink(1).MinWidth(0)
				applyTypeStyleWithTheme(th, txt, m.typeRole, selected)
				txt.TextColor(ink)
			}

			if n.Badge != "" {
				ui.Spacer(c)
				badgeBg := cs.SurfaceContainerHighest
				badgeInk := cs.OnSurfaceVariant
				if selected {
					badgeBg = selectedInk.Alpha(0.14)
					badgeInk = selectedInk
				}
				if disabled {
					badgeBg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
					badgeInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
				}
				badgeEl := ui.Row(c).
					Height(m.badgeH).
					MinWidth(m.badgeH).
					PaddingX(6).
					Radius(m.badgeH / 2).
					Shrink(0).
					AlignItems(ui.Center).
					Justify(ui.Center).
					Background(badgeBg).
					TextColor(badgeInk).
					AccessHidden(true)
				badgeEl.Children(func() {
					bt := ui.Text(c, n.Badge).NoWrap()
					applyTypeStyleWithTheme(th, bt, m.badgeRole)
					TabularNums(bt)
					bt.TextColor(badgeInk)
				})
			}
		})

		if rowClicked {
			if o.Selected != nil && treeKeyComparable(selKey) {
				*o.Selected = selKey
			}
			if o.OnSelect != nil {
				o.OnSelect(selKey)
			}
			if n.OnClick != nil {
				n.OnClick()
			}
		}

		if branch {
			groupCol := item.Group(func() {
				for ci := range n.Children {
					buildTreeNode(c, th, cs, tree, &n.Children[ci], ci, o, m, nodeDisabled, rtl)
				}
			})
			if groupCol.Valid() {
				groupCol.FillWidth().Gap(m.rowGap)
				if !(th.ReduceMotion || c.Preferences().ReduceMotion) {
					groupCol.Transition(ui.ElementTransition{
						Duration: 110 * time.Millisecond,
						Enter:    &ui.Motion{Opacity: 0, Y: -4},
					})
				}
				if o.ShowIndentGuides && len(n.Children) > 0 {
					guideOffset := m.padStart + float32(depth)*m.indentStep + m.arrowSize/2
					guideCol := cs.OutlineVariant
					if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
						guideCol = cs.Outline
					}
					if disabled {
						guideCol = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
					}
					groupCol.DrawOver(func(p *ui.Painter, r ui.Rect) {
						if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) || guideCol.A == 0 {
							return
						}
						lineH := r.H - 4
						if lineH <= 0 {
							return
						}
						x := r.X + guideOffset
						if rtl {
							x = r.X + r.W - guideOffset
						}
						p.Fill(ui.Rect{
							X: x - 0.5,
							Y: r.Y + 2,
							W: 1,
							H: lineH,
						}, guideCol, 0)
					})
				}
			}
		}
	})
}

func treeSelectionKey(n *TreeNode) any {
	if n.SelectionID != nil {
		return n.SelectionID
	}
	return n.Key
}

func treeKeyComparable(key any) bool {
	if key == nil {
		return false
	}
	return reflect.ValueOf(key).Comparable()
}

func treeSlotKey(key any, idx int) any {
	if treeKeyComparable(key) {
		return key
	}
	return treeAutoNodeKey{i: idx}
}

func treeKeysEqual(a, b any) bool {
	if !treeKeyComparable(a) || !treeKeyComparable(b) {
		return false
	}
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}
	return a == b
}
