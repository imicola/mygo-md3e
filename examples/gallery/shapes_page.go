package main

import (
	"math"

	"github.com/egoist/mygo/ui"

	md3e "mygo-md3e"
	myshape "mygo-md3e/shape"
	"mygo-md3e/tokens"
)

// shapePage holds the morph playground: the two ends, the progress and the
// built morph, rebuilt only when an end changes.
type shapePage struct {
	all      []myshape.Shape
	a, b     float64
	progress float64
	morph    *myshape.Morph
	from, to int
}

// ensure loads the catalog and builds the morph for the chosen ends.
func (p *shapePage) ensure() {
	if p.all == nil {
		p.all = myshape.All()
		byName := map[string]int{}
		for i, s := range p.all {
			byName[s.Name] = i
		}
		p.b = float64(byName["SoftBurst"])
		p.progress = 0.55
	}
	from := clampShapeIndex(int(math.Round(p.a)), len(p.all))
	to := clampShapeIndex(int(math.Round(p.b)), len(p.all))
	if p.morph == nil || p.from != from || p.to != to {
		p.from, p.to = from, to
		p.morph = myshape.NewMorph(p.all[from].Polygon(), p.all[to].Polygon())
	}
}

func clampShapeIndex(i, n int) int {
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

// shapesPageView shows the 35 built-in shapes, the live morph hero and the
// two-end control bar.
func (a *app) shapesPageView(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	p := &a.shapes
	p.ensure()

	page(c, "Shapes & Morph Playground",
		"All 35 Material 3 Expressive shapes, and a bidirectional cubic Bézier morph between any two of them.",
		func() {
			ui.Column(c).FillWidth().Padding(16).Gap(12).
				Radius(float32(th.Shapes.ExtraLarge)).Background(cs.SurfaceContainerLow).
				Border(1, cs.OutlineVariant).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					ui.Column(c).Grow(1).Gap(2).Children(func() {
						md3e.Text(c, "Morph studio", tokens.TitleMedium, true)
						md3e.Text(c, p.all[p.from].Name+"  ↔  "+p.all[p.to].Name, tokens.BodySmall).
							TextColor(cs.Primary)
					})
					if md3e.Button(c.Key("shape-swap"), "Reverse", md3e.ButtonOptions{
						Variant: md3e.ButtonTonal, Size: tokens.SizeXS,
					}).Clicked() {
						p.a, p.b = p.b, p.a
						p.progress = 1 - p.progress
					}
				})
				hero := ui.Box(c.Key("shape-hero")).FillWidth().Height(240).
					Radius(float32(th.Shapes.Large)).Background(cs.SurfaceContainerLowest).
					Border(1, cs.OutlineVariant)
				hero.Draw(func(paint *ui.Painter, r ui.Rect) {
					if r.W <= 0 || r.H <= 0 {
						return
					}
					side := min(r.W, r.H) * 0.68
					cx, cy := r.X+r.W/2, r.Y+r.H/2
					dst := ui.Rect{X: cx - side/2, Y: cy - side/2, W: side, H: side}
					halo := ui.Rect{X: cx - side*0.56, Y: cy - side*0.56, W: side * 1.12, H: side * 1.12}
					paint.Fill(halo, cs.PrimaryContainer.Alpha(0.4), halo.W/2)
					myshape.DrawMorph(paint, p.morph, float32(max(0, min(1, p.progress))), dst, cs.Primary)
				})
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
					md3e.Text(c, "0", tokens.LabelSmall).TextColor(cs.OnSurfaceVariant)
					md3e.Slider(c.Key("shape-progress"), &p.progress, 0, 1,
						md3e.SliderOptions{Label: "Morph progress", Step: 0.01}).Grow(1)
					md3e.Text(c, "1", tokens.LabelSmall).TextColor(cs.OnSurfaceVariant)
				})
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Wrap().Children(func() {
					md3e.Text(c, "From", tokens.LabelMedium, true)
					md3e.Slider(c.Key("shape-from"), &p.a, 0, float64(len(p.all)-1), md3e.SliderOptions{
						Label: "From shape", Step: 1, FormatLabel: p.shapeName,
					}).Width(220)
					md3e.Text(c, "To", tokens.LabelMedium, true)
					md3e.Slider(c.Key("shape-to"), &p.b, 0, float64(len(p.all)-1), md3e.SliderOptions{
						Label: "To shape", Step: 1, FormatLabel: p.shapeName,
					}).Width(220)
				})
			})

			section(c, "Shape catalog", func() {
				caption(c, "Choose any shape as the morph's end; the current ends read highlighted.")
				ui.Row(c).Wrap().Gap(8).Children(func() {
					for i, sh := range p.all {
						idx := i
						active := i == p.from || i == p.to
						bg := cs.SurfaceContainer
						border := cs.OutlineVariant
						ink := cs.Primary
						text := cs.OnSurfaceVariant
						if active {
							bg, border, ink, text = cs.PrimaryContainer, cs.Primary, cs.OnPrimaryContainer, cs.OnPrimaryContainer
						}
						tile := ui.ButtonBase(c.Key("shape-"+sh.Name)).Size(96, 82).
							Column().Center().Padding(8).Gap(6).Radius(16).
							Background(bg).Border(1, border).Label("Morph to " + sh.Name)
						if tile.Clicked() {
							p.b = float64(idx)
						}
						poly := sh.Polygon()
						tile.Children(func() {
							iconInk := ink
							ui.Box(c).Size(38, 38).Draw(func(paint *ui.Painter, r ui.Rect) {
								paint.FillPath(poly.ToPath(r), iconInk)
							})
							md3e.Text(c, sh.Name, tokens.LabelSmall).TextColor(text)
						})
					}
				})
			})
		})
}

// shapeName formats a shape index for a slider's value indicator.
func (p *shapePage) shapeName(v float64) string {
	return p.all[clampShapeIndex(int(math.Round(v)), len(p.all))].Name
}
