package main

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"

	md3e "mygo-md3e"
	"mygo-md3e/tokens"
)

// colorRole is one swatch of the live color scheme.
type colorRole struct {
	name string
	bg   ui.Color
	fg   ui.Color
}

// themePageView shows the live dynamic color roles, the type scale and the
// tonal elevation steps of the theme the header builds.
func (a *app) themePageView(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors

	page(c, "Theme, Typography & Color Roles",
		"The HCT color system behind the seed in the header: 45+ semantic roles, the full MD3E type scale and tonal elevation.",
		func() {
			section(c, "Dynamic color roles", func() {
				seed := seedPresets[a.seedIdx]
				caption(c, fmt.Sprintf("%s seed · %s variant · %s contrast · %s · Primary/OnPrimary %.2f:1",
					seed.name, variantNames[a.variantIdx], contrastName(a.contrast), appearanceName(a.darkIdx),
					tokens.ContrastRatio(cs.Primary, cs.OnPrimary)))
				roles := []colorRole{
					{"Primary", cs.Primary, cs.OnPrimary},
					{"OnPrimaryContainer", cs.PrimaryContainer, cs.OnPrimaryContainer},
					{"Secondary", cs.Secondary, cs.OnSecondary},
					{"SecondaryContainer", cs.SecondaryContainer, cs.OnSecondaryContainer},
					{"Tertiary", cs.Tertiary, cs.OnTertiary},
					{"TertiaryContainer", cs.TertiaryContainer, cs.OnTertiaryContainer},
					{"Error", cs.Error, cs.OnError},
					{"ErrorContainer", cs.ErrorContainer, cs.OnErrorContainer},
					{"Surface", cs.Surface, cs.OnSurface},
					{"SurfaceContainer", cs.SurfaceContainer, cs.OnSurface},
					{"SurfaceContainerHighest", cs.SurfaceContainerHighest, cs.OnSurface},
					{"InverseSurface", cs.InverseSurface, cs.InverseOnSurface},
					{"Outline", cs.Outline, cs.Surface},
					{"OutlineVariant", cs.OutlineVariant, cs.Surface},
					{"Scrim", cs.Scrim, cs.Surface},
					{"Shadow", cs.Shadow, cs.Surface},
				}
				ui.Row(c).Wrap().Gap(8).Children(func() {
					for _, role := range roles {
						ui.Column(c).Width(148).Height(60).Padding(8).Gap(2).
							Justify(ui.Center).Radius(12).Background(role.bg).
							Border(1, cs.OutlineVariant.Alpha(0.6)).Children(func() {
							md3e.Text(c, role.name, tokens.LabelMedium, true).TextColor(role.fg)
							md3e.Text(c, hexOf(role.bg), tokens.LabelSmall).TextColor(role.fg.Alpha(0.8))
						})
					}
				})
			})

			section(c, "Typography scale", func() {
				type sample struct {
					role tokens.TypeRole
					name string
				}
				samples := []sample{
					{tokens.DisplayLarge, "Display Large"},
					{tokens.DisplayMedium, "Display Medium"},
					{tokens.DisplaySmall, "Display Small"},
					{tokens.HeadlineLarge, "Headline Large"},
					{tokens.HeadlineMedium, "Headline Medium"},
					{tokens.HeadlineSmall, "Headline Small"},
					{tokens.TitleLarge, "Title Large"},
					{tokens.TitleMedium, "Title Medium"},
					{tokens.TitleSmall, "Title Small"},
					{tokens.BodyLarge, "Body Large"},
					{tokens.BodyMedium, "Body Medium"},
					{tokens.BodySmall, "Body Small"},
					{tokens.LabelLarge, "Label Large"},
					{tokens.LabelMedium, "Label Medium"},
					{tokens.LabelSmall, "Label Small"},
				}
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					for _, s := range samples {
						md3e.Text(c, s.name+" — The quick brown fox jumps over the lazy dog", s.role)
					}
					md3e.Text(c, "Emphasized Title Large", tokens.TitleLarge, true).
						TextColor(cs.Primary)
				})
			})

			section(c, "Tonal elevation", func() {
				caption(c, "Level 0 to 5: surface tint ratios and key/ambient shadows.")
				ui.Row(c).Wrap().Gap(12).Children(func() {
					for level := tokens.Level0; level <= tokens.Level5; level++ {
						bg := md3e.SurfaceColorForElevation(th, level)
						box := ui.Column(c).Size(120, 76).Padding(10).Gap(4).
							Justify(ui.Center).Radius(16).Background(bg)
						md3e.ApplyElevationShadows(th, box, level)
						box.Children(func() {
							md3e.Text(c, "Level "+strconv.Itoa(int(level)), tokens.TitleSmall, true)
							md3e.Text(c, hexOf(bg), tokens.LabelSmall).TextColor(cs.OnSurfaceVariant)
						})
					}
				})
			})
		})
}

// contrastName names a contrast index.
func contrastName(i int) string {
	switch i {
	case 1:
		return "Medium"
	case 2:
		return "High"
	}
	return "Standard"
}

// appearanceName names a light or dark index.
func appearanceName(i int) string {
	if i == 1 {
		return "Dark"
	}
	return "Light"
}

// hexOf formats a color as #RRGGBBAA.
func hexOf(col ui.Color) string {
	return fmt.Sprintf("#%02X%02X%02X%02X", col.R, col.G, col.B, col.A)
}
