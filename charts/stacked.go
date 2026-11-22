package charts

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// StackedSegment describes one slice of a horizontal StackedProgress bar.
type StackedSegment struct {
	// Label names this segment in the legend, tooltip, and accessibility summary.
	Label string
	// Value is the numeric magnitude of this segment. Non-finite or <= 0 values
	// are clamped safely to 0.
	Value float64
	// Color overrides the segment's fill color; when transparent, SeriesColor is
	// assigned automatically from the active theme.
	Color ui.Color
	// ValueLabel optionally overrides the formatted value shown in the legend
	// and tooltip (for example "12.4 GB").
	ValueLabel string
}

// StackedProgressOptions configures StackedProgress.
type StackedProgressOptions struct {
	// Total sets the full-bar denominator. When <= 0 or smaller than the sum of
	// segment values, the sum of segments is used (no empty remainder).
	Total float64
	// Height sets the horizontal track height in DIP. When <= 0, scales with
	// Theme.Density (12 Comfortable / 10 Desktop / 8 Compact).
	Height float32
	// SegmentGap is the spacing between adjacent visible segments in DIP
	// (default 3).
	SegmentGap float32
	// OuterRadius is the outer end-cap corner radius in DIP (default Height/2).
	OuterRadius float32
	// InnerRadius is the interior segment corner radius in DIP (default 3).
	InnerRadius float32
	// TrackColor overrides the unfilled remainder / empty-bar fill color
	// (defaults to Theme.Colors.SurfaceContainerHighest).
	TrackColor ui.Color
	// ShowLegend displays a wrapping color-dot + label + tabular-number legend
	// below the track.
	ShowLegend bool
	// ShowPercent appends the share percentage to auto-formatted legend values
	// when StackedSegment.ValueLabel is empty.
	ShowPercent bool
	// ShowTooltip enables an instant floating MD3E readout above the hovered
	// segment. Nil defaults to true.
	ShowTooltip *bool
	// Animate springs segment proportions with MD3E DefaultSpatial when values
	// change, snapping immediately under ReduceMotion. Nil defaults to true.
	Animate *bool
	// ValueFormatter customizes legend/tooltip value text when ValueLabel is empty.
	ValueFormatter func(seg StackedSegment, pct float64) string
	// Label names the meter for assistive technology (default "Breakdown").
	Label string
	// Description overrides the auto-generated segment breakdown summary.
	Description string
}

type stackedSegAnimKey int
type stackedHoverKey struct{}

type stackedHoverState struct {
	hovered int
}

type stackedGeomSeg struct {
	segIndex int // -1 for unfilled remainder track
	x0, x1   float32
	radii    [4]float32
	color    ui.Color
}

// StackedProgress builds an MD3E multi-segment horizontal proportion bar with
// outer pill caps, crisp inner segment corners and gaps, subtle vertical depth
// sheen, RTL mirroring, spring transitions, hover highlight + floating readout,
// and an optional wrapping tabular-numeral legend.
func StackedProgress(c *ui.Context, segments []StackedSegment, opts ...StackedProgressOptions) ui.Element {
	if len(opts) > 1 {
		panic("charts: StackedProgress takes at most one StackedProgressOptions")
	}
	var o StackedProgressOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	animate := boolDefault(o.Animate, true)
	showTooltip := boolDefault(o.ShowTooltip, true)
	rtl := c.Direction() == ui.RightToLeft

	defaultH := float32(12)
	switch th.Density.Normalize() {
	case tokens.DensityDesktop:
		defaultH = 10
	case tokens.DensityCompact:
		defaultH = 8
	}
	height := sanitizePositiveF32(o.Height, defaultH)
	segGap := sanitizeNonNegativeF32(o.SegmentGap, 3)
	outerR := sanitizePositiveF32(o.OuterRadius, height/2)
	innerR := sanitizeNonNegativeF32(o.InnerRadius, min(float32(3), height*0.3))
	trackCol := o.TrackColor
	if trackCol.A == 0 {
		trackCol = cs.SurfaceContainerHighest
		if trackCol.A == 0 {
			trackCol = cs.OutlineVariant
		}
	}

	// Sanitize segment values and resolve colors.
	n := len(segments)
	vals := make([]float64, n)
	colors := make([]ui.Color, n)
	sum := 0.0
	for i, seg := range segments {
		v := seg.Value
		if !isFiniteF64(v) || v < 0 {
			v = 0
		}
		vals[i] = v
		sum += v
		col := seg.Color
		if col.A == 0 {
			col = SeriesColor(th, i)
		}
		colors[i] = col
	}

	denom := sum
	if isFiniteF64(o.Total) && o.Total > denom {
		denom = o.Total
	}
	hasRemainder := denom > sum+1e-9 && denom > 0

	// Build accessible summary text.
	label := o.Label
	if label == "" {
		label = "Breakdown"
	}
	summary := o.Description
	if summary == "" {
		var parts []string
		for i, seg := range segments {
			if vals[i] <= 0 || denom <= 0 {
				continue
			}
			name := seg.Label
			if name == "" {
				name = fmt.Sprintf("Segment %d", i+1)
			}
			pct := (vals[i] / denom) * 100
			parts = append(parts, fmt.Sprintf("%s: %s", name, defaultFormatPercent(pct)))
		}
		if len(parts) == 0 {
			summary = "Empty"
		} else {
			summary = strings.Join(parts, ", ")
		}
	}

	root := ui.Column(c).FillWidth().MinWidth(0).Gap(10).
		Role(ui.RoleMeter).Label(label).Value(summary)
	if o.Description != "" {
		root.Description(o.Description)
	}

	hoverSt := ui.Local(root, stackedHoverKey{}, func() stackedHoverState { return stackedHoverState{hovered: -1} })
	if hoverSt.hovered >= n {
		hoverSt.hovered = -1
	}

	// Animate normalized fractions with MD3E DefaultSpatial.
	anim := motion.Resolve(th.Motion, motion.DefaultSpatial, reduce)
	fracs := make([]float32, n)
	for i := range n {
		target := float32(0)
		if denom > 0 && vals[i] > 0 {
			target = float32(vals[i] / denom)
		}
		f := animateF32(root, stackedSegAnimKey(i), target, anim, reduce || !animate)
		fracs[i] = clampF32(f, 0, 1)
	}

	remTarget := float32(0)
	if hasRemainder {
		remTarget = clampF32(float32((denom-sum)/denom), 0, 1)
	}
	remFrac := animateF32(root, stackedSegAnimKey(-1), remTarget, anim, reduce || !animate)
	remFrac = clampF32(remFrac, 0, 1)

	root.Children(func() {
		trackWrap := ui.Box(c.Key("track-wrap")).FillWidth().Height(height)

		// Detect pointer hover over individual segments using trackWrap's own bounds.
		activeSeg := -1
		activeAnchorX := float32(0.5)
		if px, _, over := trackWrap.PointerPosition(); over {
			if w := trackWrap.Bounds().W; w > 0 {
				geoms := computeStackedLayout(w, fracs, remFrac, colors, trackCol, segGap, outerR, innerR, rtl)
				for _, g := range geoms {
					if g.segIndex >= 0 && px >= g.x0-segGap*0.5 && px <= g.x1+segGap*0.5 {
						activeSeg = g.segIndex
						activeAnchorX = clampF32(((g.x0+g.x1)*0.5)/w, 0.08, 0.92)
						break
					}
				}
			}
		}
		hoverSt.hovered = activeSeg

		// Capture immutable snapshot for ghost-draw safe painting.
		drawFracs := append([]float32(nil), fracs...)
		drawColors := append([]ui.Color(nil), colors...)
		drawActive := activeSeg
		drawDark := th.Dark
		trackWrap.Draw(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			if vis := p.Visible(); vis.W > 0 && vis.H > 0 && !rectIntersects(r, vis) {
				return
			}

			geoms := computeStackedLayout(r.W, drawFracs, remFrac, drawColors, trackCol, segGap, outerR, innerR, rtl)
			if len(geoms) == 0 {
				p.Fill(r, trackCol, min(outerR, r.H/2))
				return
			}

			for _, g := range geoms {
				w := g.x1 - g.x0
				if w <= 0.25 {
					continue
				}
				segRect := ui.Rect{X: r.X + g.x0, Y: r.Y, W: w, H: r.H}
				col := g.color
				if drawActive >= 0 && g.segIndex >= 0 && g.segIndex != drawActive {
					col = col.Alpha(0.42)
				}
				p.FillRadii(segRect, g.radii, col)

				// Subtle upper specular sheen gives segmented bars crisp MD3E volume.
				if g.segIndex >= 0 && r.H >= 6 {
					sheenAlpha := float32(0.14)
					if drawDark {
						sheenAlpha = 0.11
					}
					if drawActive == g.segIndex {
						sheenAlpha += 0.08
					}
					sheenRect := ui.Rect{X: segRect.X, Y: segRect.Y, W: segRect.W, H: segRect.H * 0.44}
					sheenRadii := [4]float32{g.radii[0], g.radii[1], 0, 0}
					p.FillRadii(sheenRect, sheenRadii, ui.RGB(255, 255, 255).Alpha(sheenAlpha))
				}
			}
		})

		// Instant floating tooltip above hovered segment.
		if showTooltip && activeSeg >= 0 && activeSeg < n && denom > 0 {
			seg := segments[activeSeg]
			pct := (vals[activeSeg] / denom) * 100
			valText := formatStackedSegmentValue(seg, vals[activeSeg], pct, true, o.ValueFormatter)
			name := seg.Label
			if name == "" {
				name = fmt.Sprintf("Segment %d", activeSeg+1)
			}
			trackWrap.Children(func() {
				hAlign := ui.Center
				if activeAnchorX < 0.22 {
					hAlign = ui.Start
				} else if activeAnchorX > 0.78 {
					hAlign = ui.End
				}
				ui.Row(c.Key("tip-anchor")).Absolute().
					LeftPercent(activeAnchorX*100).Top(-6).Size(0, 0).
					AlignItems(ui.End).PassThrough().
					Children(func() {
						ui.Column(c).Width(0).AlignItems(hAlign).Children(func() {
							renderCompactSegmentTip(c, th, name, valText, colors[activeSeg])
						})
					})
			})
		}

		if o.ShowLegend && n > 0 {
			ui.Row(c.Key("legend")).FillWidth().MinWidth(0).Wrap().Gap(12).AlignItems(ui.Center).Children(func() {
				for i, seg := range segments {
					if vals[i] <= 0 && seg.Label == "" {
						continue
					}
					pct := 0.0
					if denom > 0 {
						pct = (vals[i] / denom) * 100
					}
					valText := formatStackedSegmentValue(seg, vals[i], pct, o.ShowPercent, o.ValueFormatter)
					dimmed := activeSeg >= 0 && activeSeg != i
					alpha := float32(1)
					if dimmed {
						alpha = 0.48
					}
					item := ui.Row(c.Key(i)).AlignItems(ui.Center).Gap(6).Opacity(alpha)
					item.Children(func() {
						dot := ui.Box(c).Size(8, 8).Radius(4).Background(colors[i]).Shrink(0)
						if activeSeg == i {
							dot.Border(1.5, cs.OnSurface.Alpha(0.5))
						}
						if seg.Label != "" {
							md3e.TextWith(c, seg.Label, md3e.TextOptions{
								Role:  tokens.BodySmall,
								Color: cs.OnSurfaceVariant,
							}).NoWrap()
						}
						if valText != "" {
							md3e.TextWith(c, valText, md3e.TextOptions{
								Role:        tokens.LabelMedium,
								Emphasized:  activeSeg == i,
								TabularNums: true,
								Color:       cs.OnSurface,
							}).NoWrap()
						}
					})
				}
			})
		}
	})

	return root
}

func formatStackedSegmentValue(seg StackedSegment, val, pct float64, showPercent bool, fn func(StackedSegment, float64) string) string {
	if seg.ValueLabel != "" {
		return seg.ValueLabel
	}
	if fn != nil {
		return fn(seg, pct)
	}
	if showPercent {
		return fmt.Sprintf("%s (%s)", defaultFormatNumber(val), defaultFormatPercent(pct))
	}
	return defaultFormatNumber(val)
}

func renderCompactSegmentTip(c *ui.Context, th *md3e.Theme, name, val string, swatch ui.Color) {
	cs := th.Colors
	bg := cs.SurfaceContainerHigh
	if bg.A == 0 {
		bg = md3e.SurfaceColorForElevation(th, tokens.Level3)
	}
	border := cs.OutlineVariant
	if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
		border = cs.Outline
	}
	box := ui.Row(c).AlignItems(ui.Center).Gap(6).
		Padding(5, 10).Radius(th.Shapes.Radius(tokens.ShapeSmall)).
		Background(bg).Border(1, border).PassThrough()
	md3e.ApplyElevationShadows(th, box, tokens.Level2)
	box.Children(func() {
		ui.Box(c).Size(8, 8).Radius(4).Background(swatch).Shrink(0)
		md3e.TextWith(c, name, md3e.TextOptions{
			Role:  tokens.LabelSmall,
			Color: cs.OnSurfaceVariant,
		}).NoWrap()
		md3e.TextWith(c, val, md3e.TextOptions{
			Role:        tokens.LabelMedium,
			Emphasized:  true,
			TabularNums: true,
			Color:       cs.OnSurface,
		}).NoWrap()
	})
}

func computeStackedLayout(
	totalW float32,
	fracs []float32,
	remFrac float32,
	colors []ui.Color,
	trackCol ui.Color,
	gap, outerR, innerR float32,
	rtl bool,
) []stackedGeomSeg {
	type entry struct {
		idx  int
		frac float32
		col  ui.Color
	}
	entries := make([]entry, 0, len(fracs)+1)
	fracSum := float32(0)
	for i, f := range fracs {
		if f > 0.0005 {
			entries = append(entries, entry{idx: i, frac: f, col: colors[i]})
			fracSum += f
		}
	}
	if remFrac > 0.0005 {
		entries = append(entries, entry{idx: -1, frac: remFrac, col: trackCol})
		fracSum += remFrac
	}
	if len(entries) == 0 || fracSum <= 0 {
		return nil
	}

	effGap := gap
	if len(entries) > 1 {
		maxGap := (totalW * 0.4) / float32(len(entries)-1)
		if effGap > maxGap {
			effGap = max(0, maxGap)
		}
	}
	availW := max(0, totalW-effGap*float32(len(entries)-1))
	out := make([]stackedGeomSeg, len(entries))

	cursor := float32(0)
	if rtl {
		cursor = totalW
	}
	for i, e := range entries {
		w := availW * (e.frac / fracSum)
		isFirst := i == 0
		isLast := i == len(entries)-1

		var x0, x1 float32
		var radii [4]float32
		if !rtl {
			x0 = cursor
			x1 = cursor + w
			cursor = x1 + effGap
			leftR, rightR := innerR, innerR
			if isFirst {
				leftR = outerR
			}
			if isLast {
				rightR = outerR
			}
			radii = [4]float32{leftR, rightR, rightR, leftR}
		} else {
			x1 = cursor
			x0 = cursor - w
			cursor = x0 - effGap
			rightR, leftR := innerR, innerR
			if isFirst {
				rightR = outerR
			}
			if isLast {
				leftR = outerR
			}
			radii = [4]float32{leftR, rightR, rightR, leftR}
		}
		out[i] = stackedGeomSeg{
			segIndex: e.idx,
			x0:       x0,
			x1:       x1,
			radii:    radii,
			color:    e.col,
		}
	}
	return out
}
