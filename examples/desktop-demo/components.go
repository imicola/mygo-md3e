package main

import (
	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// shapeSample is one row of the shape button gallery.
type shapeSample struct {
	label   string
	poly    *shape.RoundedPolygon
	variant md3e.ShapeButtonVariant
	icon    md3e.IconDraw
	size    tokens.SizeTier
}

// shapeSamples cover filled, tonal and outlined shape treatments.
var shapeSamples = []shapeSample{
	{"Shape circle", shape.Circle(), md3e.ShapeButtonFilled, symbols.PlayArrow.Draw, tokens.SizeS},
	{"Shape cookie", shape.Cookie9Sided(), md3e.ShapeButtonTonal, symbols.Star.Draw, tokens.SizeS},
	{"Shape sunny", shape.Sunny(), md3e.ShapeButtonOutlined, symbols.Bolt.Draw, tokens.SizeS},
	{"Shape burst", shape.SoftBurst(), md3e.ShapeButtonFilled, symbols.Add.Draw, tokens.SizeM},
	{"Shape heart", shape.Heart(), md3e.ShapeButtonTonal, symbols.Favorite.Draw, tokens.SizeS},
	{"Shape gem", shape.Gem(), md3e.ShapeButtonOutlined, symbols.Check.Draw, tokens.SizeS},
}

// toneSamples walk every SemanticTone.
var toneSamples = []struct {
	label string
	tone  md3e.SemanticTone
}{
	{"Neutral", md3e.ToneNeutral},
	{"Primary", md3e.TonePrimary},
	{"Secondary", md3e.ToneSecondary},
	{"Tertiary", md3e.ToneTertiary},
	{"Error", md3e.ToneError},
	{"Warning", md3e.ToneWarning},
	{"Success", md3e.ToneSuccess},
}

// symbolSamples show a slice of the zero-bloat symbols pack.
var symbolSamples = []struct {
	label string
	icon  md3e.IconDraw
}{
	{"Dashboard", symbols.Dashboard.Draw},
	{"Terminal", symbols.Terminal.Draw},
	{"Shield", symbols.Shield.Draw},
	{"Wifi", symbols.Wifi.Draw},
	{"Cloud", symbols.Cloud.Draw},
	{"Timer", symbols.Timer.Draw},
	{"Palette", symbols.Palette.Draw},
	{"Favorite", symbols.Favorite.Draw},
}

// componentsView is the M10 state laboratory.
func (a *appState) componentsView(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	pageScroll(c, "components-scroll", func() {
		pageHeading(c, "Component lab", "Expressive shapes, semantic tones, presence and type helpers in one place.")

		section(c, "Expressive shape buttons", "Any MD3E polygon, a pressed morph and ReduceMotion aware springs.", func() {
			ui.Row(c).FillWidth().Gap(10).Wrap().AlignItems(ui.Center).Children(func() {
				for _, s := range shapeSamples {
					sample := s
					md3e.ShapeButton(c.Key("shape-"+sample.label), sample.label, sample.icon, md3e.ShapeButtonOptions{
						Variant: sample.variant,
						Size:    sample.size,
						Shape:   sample.poly,
						OnClick: func() {
							a.snack.Show(md3e.SnackbarOptions{Message: sample.label + " pressed"})
						},
					})
				}
			})
			caption(c, "Each button morphs toward its Circle silhouette while pressed and back on release.")
		})

		section(c, "Semantic tags", "Seven harmonized tones, tonal or outlined, with an optional dismiss action.", func() {
			ui.Row(c).Gap(6).Wrap().Children(func() {
				for _, t := range toneSamples {
					md3e.Tag(c, t.label, md3e.TagOptions{Tone: t.tone})
				}
				md3e.Tag(c, "Outlined", md3e.TagOptions{Tone: md3e.TonePrimary, Variant: md3e.TagOutlined})
			})
			ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
				for _, name := range a.tags {
					tag := name
					md3e.Tag(c.Key("tag-"+tag), tag, md3e.TagOptions{
						Tone: md3e.ToneSecondary,
						OnClose: func() {
							a.removeTag(tag)
							a.dirty = true
						},
					})
				}
				if md3e.Button(c.Key("reset-tags"), "Restore tags", md3e.ButtonOptions{
					Variant:     md3e.ButtonText,
					Size:        tokens.SizeXS,
					LeadingIcon: symbols.Refresh.Draw,
				}).Clicked() {
					a.tags = []string{"Shell", "Density", "Symbols"}
					a.dirty = true
				}
			})
		})

		section(c, "Status & presence", "Pulsing status dots, outlined variants and polygon-clipped avatars.", func() {
			ui.Row(c).Gap(6).Wrap().Children(func() {
				md3e.StatusPill(c, "Running", md3e.StatusPillOptions{Tone: md3e.ToneSuccess, Pulsing: true})
				md3e.StatusPill(c, "Queued", md3e.StatusPillOptions{Tone: md3e.ToneNeutral})
				md3e.StatusPill(c, "Degraded", md3e.StatusPillOptions{Tone: md3e.ToneWarning, Outlined: true})
				md3e.StatusPill(c, "Offline", md3e.StatusPillOptions{Tone: md3e.ToneError, Outlined: true})
				md3e.StatusPill(c, "Synced", md3e.StatusPillOptions{Tone: md3e.TonePrimary})
			})
			ui.Row(c).Gap(14).AlignItems(ui.Center).Wrap().Children(func() {
				md3e.Avatar(c, md3e.AvatarOptions{
					Text: "Aurora Cluster", Size: 48, Shape: shape.Cookie9Sided(),
					Status: md3e.AvatarStatusOnline, Label: "Aurora Cluster avatar",
				})
				md3e.Avatar(c, md3e.AvatarOptions{
					Text: "Beacon Relay", Size: 44,
					Status: md3e.AvatarStatusOnline, Label: "Beacon Relay avatar",
				})
				md3e.Avatar(c, md3e.AvatarOptions{
					Text: "Comet Worker", Size: 40, Tone: md3e.ToneTertiary, HasTone: true,
					Status: md3e.AvatarStatusAway, Label: "Comet Worker avatar",
				})
				md3e.Avatar(c, md3e.AvatarOptions{
					Text: "Delta Indexer", Size: 48, Shape: shape.Sunny(),
					Status: md3e.AvatarStatusBusy, Label: "Delta Indexer avatar",
				})
				md3e.Avatar(c, md3e.AvatarOptions{
					Text: "Ember Cache", Size: 36, Tone: md3e.ToneNeutral, HasTone: true,
					Status: md3e.AvatarStatusOffline, Label: "Ember Cache avatar",
				})
			})
		})

		section(c, "Typography & symbols", "Tabular figures, middle truncation and a slice of the 252 symbols.", func() {
			md3e.TextWith(c, "14.2 ms · 128.4 MB/s · 99.982% uptime · 6 nodes", md3e.TextOptions{
				Role:        tokens.TitleMedium,
				Mono:        true,
				TabularNums: true,
			})
			caption(c, "TabularNums keeps live counters aligned; Mono uses the theme's monospace family.")
			ui.Box(c).FillWidth().MaxWidth(420).Margin(0, 0, 4, 0).Children(func() {
				md3e.TextWith(c, "~/PulseDesk/workspaces/nebula-cluster/telemetry/stream-2026-11-21T09-14.parquet",
					md3e.TextOptions{
						Role:     tokens.BodySmall,
						Mono:     true,
						Truncate: md3e.TruncateMiddle,
						MaxLines: 1,
						Color:    cs.OnSurfaceVariant,
					})
			})
			ui.Row(c).Gap(8).Wrap().Children(func() {
				for i, s := range symbolSamples {
					variant := md3e.IconBadgeOutlined
					if i%3 == 0 {
						variant = md3e.IconBadgeTonal
					}
					var clip *shape.RoundedPolygon
					if i%4 == 1 {
						clip = shape.Cookie4Sided()
					}
					md3e.IconBadge(c, s.icon, md3e.IconBadgeOptions{
						Variant: variant,
						Size:    tokens.SizeS,
						Shape:   clip,
						Label:   "Symbol " + s.label,
					})
				}
			})
		})

		section(c, "Motion preferences", "ReduceMotion settles springs, pulses and the skeleton shimmer to still frames.", func() {
			md3e.SwitchCard(c.Key("reduce-motion"), "Reduce motion", &a.reduce, md3e.SwitchCardOptions{
				Subtitle:    "Settle springs, pulses and skeleton shimmer",
				LeadingIcon: symbols.Speed.Draw,
				IconBadge:   md3e.IconBadgeOptions{Variant: md3e.IconBadgeErrorTonal, Shape: shape.Cookie12Sided()},
				Hairline:    true,
				OnChanged:   func(bool) { a.dirty = true },
			})
			caption(c, "Theme density is "+densityNames[a.densityIdx]+" · change it from the title bar.")
		})
		ui.Box(c).Height(16)
	})
}

// removeTag drops one dismissible tag.
func (a *appState) removeTag(name string) {
	for i, tag := range a.tags {
		if tag == name {
			a.tags = append(a.tags[:i], a.tags[i+1:]...)
			return
		}
	}
}
