package charts

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

// TooltipRow represents one labeled metric line inside a chart's floating
// MD3E RichTooltip card.
type TooltipRow struct {
	// Name is the series or slice label.
	Name string
	// Value is the formatted numeric readout rendered with TabularNums.
	Value string
	// Color is the swatch dot color for this series/slice.
	Color ui.Color
}

type chartLegendItem struct {
	Name  string
	Value string
	Color ui.Color
}

// renderChartTooltipCard builds an instant, pointer-through MD3E RichTooltip-
// styled surface panel (SurfaceContainerHigh + Level2 shadows + subtle
// OutlineVariant hairline + ShapeMedium radius + TabularNums alignment).
func renderChartTooltipCard(c *ui.Context, th *md3e.Theme, title string, rows []TooltipRow, footerLabel, footerValue string) {
	if title == "" && len(rows) == 0 && footerValue == "" {
		return
	}
	cs := th.Colors
	bg := cs.SurfaceContainerHigh
	if bg.A == 0 {
		bg = md3e.SurfaceColorForElevation(th, tokens.Level3)
	}
	border := cs.OutlineVariant
	if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
		border = cs.Outline
	} else if th.Dark {
		border = cs.OutlineVariant.Mix(cs.OnSurface, 0.14).Alpha(0.55)
	} else {
		border = cs.OutlineVariant.Alpha(0.60)
	}

	panel := ui.Column(c).MinWidth(128).MaxWidth(240).
		Padding(10, 12).Gap(6).
		Radius(th.Shapes.Radius(tokens.ShapeMedium)).
		Background(bg).Border(1, border).
		TextColor(cs.OnSurface).PassThrough()
	md3e.ApplyElevationShadows(th, panel, tokens.Level2)

	panel.Children(func() {
		if title != "" {
			md3e.TextWith(c, title, md3e.TextOptions{
				Role:       tokens.TitleSmall,
				Emphasized: true,
				MaxLines:   1,
				Color:      cs.OnSurface,
			}).NoWrap()
		}
		for i, row := range rows {
			ui.Row(c.Key(i)).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					if row.Color.A > 0 {
						ui.Box(c).Size(8, 8).Radius(4).Background(row.Color).Shrink(0)
					}
					if row.Name != "" {
						md3e.TextWith(c, row.Name, md3e.TextOptions{
							Role:  tokens.BodySmall,
							Color: cs.OnSurfaceVariant,
						}).NoWrap()
					}
				})
				if row.Value != "" {
					md3e.TextWith(c, row.Value, md3e.TextOptions{
						Role:        tokens.LabelMedium,
						Emphasized:  true,
						TabularNums: true,
						Color:       cs.OnSurface,
					}).NoWrap()
				}
			})
		}
		if footerLabel != "" || footerValue != "" {
			ui.Box(c).FillWidth().Height(1).Background(cs.OutlineVariant.Alpha(0.55))
			ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Children(func() {
				md3e.TextWith(c, footerLabel, md3e.TextOptions{
					Role:  tokens.LabelSmall,
					Color: cs.OnSurfaceVariant,
				}).NoWrap()
				md3e.TextWith(c, footerValue, md3e.TextOptions{
					Role:        tokens.LabelMedium,
					Emphasized:  true,
					TabularNums: true,
					Color:       cs.OnSurface,
				}).NoWrap()
			})
		}
	})
}

// renderChartLegend builds a wrapping interactive MD3E legend row. Clicking a
// legend chip toggles series focus (activeIndex == i) so multi-series charts
// can highlight one series and gently dim siblings.
func renderChartLegend(c *ui.Context, th *md3e.Theme, items []chartLegendItem, activeIndex int, onToggle func(int)) {
	if len(items) == 0 {
		return
	}
	cs := th.Colors
	ui.Row(c.Key("legend")).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
		for i, item := range items {
			selected := activeIndex == i
			dimmed := activeIndex >= 0 && activeIndex != i
			alpha := float32(1)
			if dimmed {
				alpha = 0.45
			}
			label := item.Name
			if label == "" {
				label = "Series"
			}
			chip := ui.ButtonBase(c.Key(i)).Row().AlignItems(ui.Center).Gap(6).
				Padding(4, 8).Radius(th.Shapes.Radius(tokens.ShapeFull)).
				FocusRing(false).Opacity(alpha).Label(label).AccessSelected(selected)
			if selected {
				chip.Background(cs.SurfaceContainerHigh)
			}
			md3e.AttachRipple(c, chip, md3e.StateLayerOptions{
				Color:  cs.OnSurface,
				Radius: th.Shapes.Radius(tokens.ShapeFull),
			})
			chip.Children(func() {
				dot := ui.Box(c).Size(9, 9).Radius(4.5).Background(item.Color).Shrink(0)
				if selected {
					dot.Border(1.5, cs.OnSurface.Alpha(0.55))
				}
				if item.Name != "" {
					md3e.TextWith(c, item.Name, md3e.TextOptions{
						Role:       tokens.LabelMedium,
						Emphasized: selected,
						Color:      cs.OnSurface,
					}).NoWrap()
				}
				if item.Value != "" {
					md3e.TextWith(c, item.Value, md3e.TextOptions{
						Role:        tokens.LabelSmall,
						TabularNums: true,
						Color:       cs.OnSurfaceVariant,
					}).NoWrap()
				}
			})
			if chip.Clicked() && onToggle != nil {
				onToggle(i)
			}
		}
	})
}

func md3eAxisLabel(c *ui.Context, th *md3e.Theme) func(chart.Side, chart.Tick) {
	ink := th.Colors.OnSurfaceVariant
	return func(_ chart.Side, tick chart.Tick) {
		md3e.TextWith(c, tick.Label, md3e.TextOptions{
			Role:        tokens.LabelSmall,
			TabularNums: true,
			Color:       ink,
		}).NoWrap()
	}
}

func gridLineColor(th *md3e.Theme) ui.Color {
	if th.Contrast >= tokens.ContrastHigh && th.Colors.Outline.A > 0 {
		return th.Colors.Outline.Alpha(0.65)
	}
	if th.Dark {
		return th.Colors.OutlineVariant.Alpha(0.45)
	}
	return th.Colors.OutlineVariant.Alpha(0.65)
}
