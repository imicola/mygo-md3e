package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "mygo-md3e"
	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// motionPage holds the progress console's value.
type motionPage struct {
	value float64
	ready bool
}

func (p *motionPage) ensure() {
	if !p.ready {
		p.ready, p.value = true, 0.4
	}
}

// springTokens are the spring channels the page plots.
var springTokens = []motion.Token{
	motion.FastSpatial, motion.DefaultSpatial, motion.SlowSpatial,
	motion.FastEffects, motion.DefaultEffects, motion.SlowEffects,
}

// motionPageView shows the spring physics, the loading indicators and the
// four progress indicators.
func (a *app) motionPageView(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	p := &a.motionPage
	p.ensure()

	page(c, "Motion & Indicators",
		"MD3E spring physics as ζ → c conversions, the shape-morphing loading indicator, and the linear, wavy and circular progress families.",
		func() {
			section(c, "Spring tokens", func() {
				caption(c, fmt.Sprintf("%s scheme · damping computed as c = 2ζ√k, never a raw damping ratio.",
					a.scheme()))
				ui.Row(c).Wrap().Gap(10).Children(func() {
					for _, token := range springTokens {
						drawSpringCard(c, th, a.scheme(), token)
					}
				})
			})

			section(c, "Loading indicators", func() {
				caption(c, "Standard and contained styles; they follow ReduceMotion in the header.")
				ui.Row(c).Wrap().Gap(20).AlignItems(ui.Center).Children(func() {
					md3e.LoadingIndicator(c, md3e.LoadingOptions{Style: md3e.LoadingStandard, Size: 32, Label: "Loading standard small"})
					md3e.LoadingIndicator(c, md3e.LoadingOptions{Style: md3e.LoadingStandard, Size: 56, Label: "Loading standard large"})
					md3e.LoadingIndicator(c, md3e.LoadingOptions{Style: md3e.LoadingContained, Size: 40, Label: "Loading contained small"})
					md3e.LoadingIndicator(c, md3e.LoadingOptions{Style: md3e.LoadingContained, Size: 72, Label: "Loading contained large"})
				})
			})

			section(c, "Progress console", func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
					md3e.Text(c, "0%", tokens.LabelSmall).TextColor(th.Colors.OnSurfaceVariant)
					md3e.Slider(c.Key("progress-value"), &p.value, 0, 1, md3e.SliderOptions{
						Label: "Progress value", Step: 0.01,
						FormatLabel: func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) },
					}).Grow(1)
					md3e.Text(c, "100%", tokens.LabelSmall).TextColor(th.Colors.OnSurfaceVariant)
				})
				ui.Column(c).FillWidth().Gap(14).Children(func() {
					md3e.Text(c, "Linear · determinate", tokens.LabelMedium, true)
					md3e.LinearProgress(c, p.value, md3e.ProgressOptions{Label: "Linear progress"})
					md3e.Text(c, "Linear wavy · determinate", tokens.LabelMedium, true)
					md3e.LinearWavyProgress(c, p.value, md3e.ProgressOptions{Label: "Wavy linear progress"})
					md3e.Text(c, "Linear · indeterminate", tokens.LabelMedium, true)
					md3e.LinearProgress(c, -1, md3e.ProgressOptions{Label: "Indeterminate linear progress"})
					md3e.Text(c, "Linear wavy · indeterminate", tokens.LabelMedium, true)
					md3e.LinearWavyProgress(c, -1, md3e.ProgressOptions{Label: "Indeterminate wavy linear progress"})
				})
				ui.Row(c).Wrap().Gap(24).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
						md3e.CircularProgress(c, p.value, md3e.ProgressOptions{Label: "Circular progress", Size: 64})
						caption(c, "Circular")
					})
					ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
						md3e.CircularWavyProgress(c, p.value, md3e.ProgressOptions{Label: "Wavy circular progress", Size: 64})
						caption(c, "Wavy circular")
					})
					ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
						md3e.CircularProgress(c, -1, md3e.ProgressOptions{Label: "Indeterminate circular progress", Size: 64})
						caption(c, "Indeterminate")
					})
					ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
						md3e.CircularWavyProgress(c, -1, md3e.ProgressOptions{Label: "Indeterminate wavy circular progress", Size: 64})
						caption(c, "Indeterminate wavy")
					})
				})
			})
		})
}

// drawSpringCard plots one spring token's response.
func drawSpringCard(c *ui.Context, th *md3e.Theme, scheme motion.Scheme, token motion.Token) {
	cs := th.Colors
	spec := motion.SpecOf(scheme, token)
	ui.Column(c).Width(232).Height(150).Padding(12).Gap(6).Radius(16).
		Background(cs.SurfaceContainerLowest).Border(1, cs.OutlineVariant).Children(func() {
		md3e.Text(c, spec.Token.String(), tokens.TitleSmall, true)
		md3e.Text(c, fmt.Sprintf("ζ %.2f · k %.0f · c %.1f", spec.DampingRatio, spec.Stiffness, spec.Damping()),
			tokens.LabelSmall).TextColor(cs.OnSurfaceVariant)
		md3e.Text(c, fmt.Sprintf("settles in %d ms", spec.SettleTime().Milliseconds()),
			tokens.LabelSmall).TextColor(cs.Tertiary)
		ui.Box(c).FillWidth().Grow(1).MinHeight(60).Draw(func(p *ui.Painter, r ui.Rect) {
			drawSpringCurve(p, r, spec, cs)
		})
	})
}

// drawSpringCurve draws progress over a 750 ms window with reference lines at
// 0 and 1.
func drawSpringCurve(p *ui.Painter, r ui.Rect, spec motion.Spec, cs tokens.ColorScheme) {
	pad := float32(6)
	w, h := r.W-2*pad, r.H-2*pad
	if w <= 8 || h <= 8 {
		return
	}
	yOf := func(v float64) float32 {
		return r.Y + pad + h*float32(1-v/1.2)
	}
	p.Line(r.X+pad, yOf(0), r.X+pad+w, yOf(0), 1, cs.OutlineVariant)
	p.Line(r.X+pad, yOf(1), r.X+pad+w, yOf(1), 1, cs.Outline.Alpha(0.6))
	opts := spec.SpringOptions()
	var path ui.Path
	const samples = 72
	for i := 0; i <= samples; i++ {
		frac := float64(i) / samples
		v := opts.Progress(time.Duration(frac*750) * time.Millisecond)
		x := r.X + pad + w*float32(frac)
		if i == 0 {
			path.MoveTo(x, yOf(v))
		} else {
			path.LineTo(x, yOf(v))
		}
	}
	p.StrokePath(&path, 2, cs.Tertiary)
}
