package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// node is one managed workspace node. clip optionally gives its avatar an
// expressive polygon silhouette.
type node struct {
	id      string
	name    string
	short   string
	role    string
	kind    string
	state   string
	status  md3e.AvatarStatus
	clip    *shape.RoundedPolygon
	tone    md3e.SemanticTone
	summary string
}

// nodes are the demo's six managed nodes.
var nodes = []node{
	{
		id: "aurora", name: "Aurora Cluster", short: "Orchestration", role: "Orchestration · 6 pods", kind: "Orchestrator",
		state: "Active", status: md3e.AvatarStatusOnline,
		clip: shape.Cookie9Sided(), tone: md3e.ToneSuccess,
		summary: "Keeps the service graph warm and rolls new pods behind the stable channel.",
	},
	{
		id: "beacon", name: "Beacon Relay", short: "Edge gateway", role: "Edge gateway · eu-west", kind: "Gateway",
		state: "Active", status: md3e.AvatarStatusOnline,
		tone:    md3e.TonePrimary,
		summary: "Terminates edge sessions and forwards compressed deltas to the cluster.",
	},
	{
		id: "comet", name: "Comet Worker", short: "Batch queue", role: "Batch queue · 12 jobs", kind: "Worker",
		state: "Idle", status: md3e.AvatarStatusAway,
		tone:    md3e.ToneNeutral,
		summary: "Drains the nightly batch queue and parks between scheduled windows.",
	},
	{
		id: "delta", name: "Delta Indexer", short: "Search index", role: "Search index · 2 shards", kind: "Indexer",
		state: "Active", status: md3e.AvatarStatusBusy,
		clip: shape.Sunny(), tone: md3e.ToneTertiary,
		summary: "Merges shard segments continuously so the workspace stays searchable.",
	},
	{
		id: "ember", name: "Ember Cache", short: "Hot cache", role: "Hot cache · 94% hit", kind: "Cache",
		state: "Idle", status: md3e.AvatarStatusOffline,
		tone:    md3e.ToneNeutral,
		summary: "Warms the hot object cache before the next editing session starts.",
	},
	{
		id: "flux", name: "Flux Streamer", short: "Telemetry", role: "Telemetry fan-out", kind: "Streamer",
		state: "Active", status: md3e.AvatarStatusOnline,
		clip: shape.SoftBurst(), tone: md3e.ToneSecondary,
		summary: "Fans metrics out to dashboards, alerts and the activity log.",
	},
}

// filteredNodes applies the active status tab and search query.
func (a *appState) filteredNodes() []node {
	query := strings.ToLower(strings.TrimSpace(a.query))
	out := make([]node, 0, len(nodes))
	for _, n := range nodes {
		switch a.tab {
		case 1:
			if n.state != "Active" {
				continue
			}
		case 2:
			if n.state != "Idle" {
				continue
			}
		}
		if query != "" && !strings.Contains(strings.ToLower(n.name+" "+n.role), query) {
			continue
		}
		out = append(out, n)
	}
	return out
}

// countState counts nodes in state for the filter tabs.
func countState(state string) int {
	count := 0
	for _, n := range nodes {
		if n.state == state {
			count++
		}
	}
	return count
}

// selectFirstVisible keeps the detail pane on a node the filter still shows.
func (a *appState) selectFirstVisible() {
	items := a.filteredNodes()
	if len(items) == 0 {
		return
	}
	for _, n := range items {
		if n.id == a.selectedID {
			return
		}
	}
	a.selectedID = items[0].id
}

// currentNode resolves the selected node, falling back to the first one.
func (a *appState) currentNode() node {
	for _, n := range nodes {
		if n.id == a.selectedID {
			return n
		}
	}
	return nodes[0]
}

// filterTabs are the status tabs with live counts.
func (a *appState) filterTabs() []md3e.TabItem {
	return []md3e.TabItem{
		{Label: "All", Badge: fmt.Sprint(len(nodes))},
		{Label: "Active", Badge: fmt.Sprint(countState("Active"))},
		{Label: "Idle", Badge: fmt.Sprint(countState("Idle"))},
	}
}

// workspaceView is the adaptive list/detail page.
func (a *appState) workspaceView(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	ui.Column(c).Fill().Padding(14, 18, 14, 18).Gap(12).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Wrap().Children(func() {
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				md3e.Text(c, "Node workspace", tokens.HeadlineSmall, true)
				md3e.Text(c, "Drag the seam to resize · double-click it to reset", tokens.BodySmall).
					TextColor(cs.OnSurfaceVariant)
			})
			md3e.PillTabs(c.Key("node-filter"), a.filterTabs(), md3e.PillTabsOptions{
				Selected: &a.tab,
				Changed: func(i int) {
					a.tab = i
					a.selectFirstVisible()
				},
				Label: "Node filters",
			})
		})
		md3e.ListDetailPane(c, func() { a.nodeList(c) }, func() { a.nodeDetail(c) },
			md3e.ListDetailPaneOptions{
				SplitWidth: &a.split,
				ShowDetail: &a.showDetail,
				Label:      "Nodes and details",
			}).Grow(1).MinHeight(0)
	})
}

// nodeList is the searchable list pane.
func (a *appState) nodeList(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	items := a.filteredNodes()

	ui.Column(c).Fill().MinWidth(0).Padding(12).Gap(10).Background(cs.SurfaceContainerLow).Children(func() {
		md3e.Text(c, "Nodes", tokens.TitleSmall, true)

		md3e.TextField(c.Key("node-search"), &a.query, md3e.TextFieldOptions{
			Variant:       md3e.TextFieldFilled,
			Label:         "Search nodes",
			Placeholder:   "Name or role",
			LeadingIcon:   symbols.Search.Draw,
			TrailingIcon:  symbols.Close.Draw,
			TrailingLabel: "Clear search",
			OnTrailing:    func() { a.query = "" },
		})

		md3e.SwitchCard(c.Key("skeleton-preview"), "Loading preview", &a.skeleton, md3e.SwitchCardOptions{
			Subtitle:    "Show skeleton placeholders",
			LeadingIcon: symbols.HourglassEmpty.Draw,
			IconBadge:   md3e.IconBadgeOptions{Variant: md3e.IconBadgeOutlined, Shape: shape.SoftBurst()},
			Hairline:    true,
		})

		md3e.Divider(c)

		ui.Scroll(c.Key("node-scroll")).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().PaddingEnd(14).MinWidth(0).Gap(6).Children(func() {
				switch {
				case a.skeleton:
					a.skeletonRows(c)
				case len(items) == 0:
					md3e.EmptyState(c, "No nodes match", md3e.EmptyStateOptions{
						Icon:               symbols.Search.Draw,
						IconShape:          shape.Cookie7Sided(),
						Description:        fmt.Sprintf("Nothing matches %q. Try another name, or clear the filter.", a.query),
						PrimaryActionLabel: "Reset filter",
						OnPrimaryAction:    func() { a.query = "" },
						Label:              "No nodes match",
					})
				default:
					for _, n := range items {
						a.nodeTile(c, n)
					}
				}
			})
		})
	})
}

// skeletonRows are the loading placeholders of the list pane.
func (a *appState) skeletonRows(c *ui.Context) {
	for range 3 {
		ui.Row(c).FillWidth().Padding(10).Gap(12).AlignItems(ui.Center).Children(func() {
			md3e.Skeleton(c, md3e.SkeletonOptions{
				Shape:  md3e.SkeletonCircle,
				Height: 36,
				Label:  "Loading node avatar",
			})
			ui.Column(c).Grow(1).Gap(6).Children(func() {
				md3e.Skeleton(c, md3e.SkeletonOptions{Height: 12, Width: 148, Label: "Loading node title"})
				md3e.Skeleton(c, md3e.SkeletonOptions{Height: 10, Lines: 2, LineGap: 6, Label: "Loading node details"})
			})
		})
	}
}

// nodeTile is one selectable node row with an avatar and presence dot. The
// selected row fills with the high surface instead of carrying a badge, so its
// title never truncates.
func (a *appState) nodeTile(c *ui.Context, n node) {
	cs := md3e.CurrentTheme(c).Colors
	selected := n.id == a.selectedID
	variant := md3e.CardOutlined
	ringCol := cs.Surface
	if selected {
		variant = md3e.CardFilled
		ringCol = cs.SurfaceContainerHighest
	}
	md3e.ActionTile(c.Key("node-"+n.id), n.name, md3e.ActionTileOptions{
		Subtitle: n.short,
		Leading: func() {
			md3e.Avatar(c, md3e.AvatarOptions{
				Text:            n.name,
				Size:            36,
				Shape:           n.clip,
				Status:          n.status,
				StatusRingColor: ringCol,
				Label:           n.name + " avatar",
			})
		},
		Meta:       n.state,
		Variant:    variant,
		HasVariant: true,
		Hairline:   selected,
		ShapeScale: tokens.ShapeLarge,
		OnClick: func() {
			a.selectedID = n.id
			a.showDetail = true
			// Under WindowCompact ListDetailPane reads ShowDetail before it
			// builds the list, so the switch needs its own frame.
			a.dirty = true
		},
		Label: n.name,
	})
}

// nodeDetail is the main pane of the detail view plus its optional properties
// supporting pane.
func (a *appState) nodeDetail(c *ui.Context) {
	n := a.currentNode()
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	class := md3e.WindowSizeClassOf(c)

	main := func() {
		ui.Scroll(c.Key("detail-scroll")).Fill().Children(func() {
			ui.Column(c).FillWidth().Padding(20).Gap(14).Children(func() {
				if class == md3e.WindowCompact {
					if md3e.Button(c.Key("back-to-list"), "Back to list", md3e.ButtonOptions{
						Variant:     md3e.ButtonText,
						Size:        tokens.SizeXS,
						LeadingIcon: symbols.ArrowBack.Draw,
					}).Clicked() {
						a.showDetail = false
						a.dirty = true
					}
				}

				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(14).Children(func() {
					md3e.Avatar(c, md3e.AvatarOptions{
						Text:   n.name,
						Size:   56,
						Shape:  n.clip,
						Status: n.status,
						Label:  n.name + " avatar",
					})
					ui.Column(c).Grow(1).Gap(2).Children(func() {
						md3e.Text(c, n.name, tokens.TitleLarge, true)
						md3e.Text(c, n.role, tokens.BodySmall).TextColor(cs.OnSurfaceVariant)
					})
				})

				ui.Row(c).Gap(6).Wrap().Children(func() {
					md3e.StatusPill(c, n.state, md3e.StatusPillOptions{Tone: n.tone})
					md3e.Tag(c, "Managed", md3e.TagOptions{Tone: md3e.ToneTertiary})
					md3e.Tag(c, "Region eu-west", md3e.TagOptions{Tone: md3e.ToneSecondary, Variant: md3e.TagOutlined})
					md3e.Tag(c, "Density "+densityNames[a.densityIdx], md3e.TagOptions{
						Tone:        md3e.TonePrimary,
						Variant:     md3e.TagOutlined,
						LeadingIcon: symbols.ViewList.Draw,
					})
				})

				md3e.Divider(c)
				md3e.Text(c, n.summary, tokens.BodyMedium).TextColor(cs.OnSurfaceVariant)

				ui.Column(c).Gap(4).Children(func() {
					ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
						md3e.Text(c, "Profile intensity", tokens.TitleSmall, true)
						ui.Box(c).Grow(1)
						md3e.TextWith(c, fmt.Sprintf("%.0f%%", a.intensity), md3e.TextOptions{
							Role:        tokens.TitleSmall,
							Mono:        true,
							TabularNums: true,
							Color:       cs.Primary,
						})
					})
					md3e.Slider(c.Key("intensity"), &a.intensity, 0, 100, md3e.SliderOptions{
						Label: "Profile intensity",
						Step:  5,
						FormatLabel: func(v float64) string {
							return fmt.Sprintf("%.0f%%", v)
						},
					})
				})

				ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
					md3e.ShapeButton(c.Key("apply-profile"), "Apply profile", symbols.Check.Draw, md3e.ShapeButtonOptions{
						Variant:      md3e.ShapeButtonFilled,
						Size:         tokens.SizeS,
						Shape:        shape.Cookie9Sided(),
						PressedShape: shape.Circle(),
						OnClick: func() {
							a.snack.Show(md3e.SnackbarOptions{
								Message:     "Profile applied to " + n.name,
								ActionLabel: "Undo",
								OnAction:    func() { a.intensity = 62 },
							})
						},
					})
					ui.Column(c).Grow(1).Gap(2).Children(func() {
						md3e.Text(c, "Apply profile", tokens.LabelLarge)
						caption(c, "Writes the tuned intensity")
					})
					if md3e.Button(c.Key("archive-node"), "Archive", md3e.ButtonOptions{
						Variant:     md3e.ButtonOutlined,
						Size:        tokens.SizeS,
						LeadingIcon: symbols.Archive.Draw,
					}).Clicked() {
						a.snack.Show(md3e.SnackbarOptions{Message: n.name + " archived", ActionLabel: "Undo"})
					}
				})

				md3e.Divider(c)
				md3e.SwitchCard(c.Key("props-panel"), "Property panel", &a.showSupporting, md3e.SwitchCardOptions{
					Subtitle:    "Supporting pane on expanded windows",
					LeadingIcon: symbols.ViewSidebar.Draw,
					IconBadge:   md3e.IconBadgeOptions{Variant: md3e.IconBadgeTertiaryTonal, Shape: shape.Sunny()},
					Hairline:    true,
				})
			})
		})
	}

	supporting := func() { a.propertyPane(c, n) }
	// The supporting pane only makes sense beside the detail; under Compact it
	// would replace the detail it describes.
	showSupporting := a.showSupporting && class == md3e.WindowExpanded
	md3e.SupportingPane(c, main, supporting, md3e.SupportingPaneOptions{
		ShowSupporting:     &showSupporting,
		DefaultMainPercent: 58,
		Label:              "Node properties",
	})
}

// propertyPane is the supporting pane content.
func (a *appState) propertyPane(c *ui.Context, n node) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	ui.Box(c).Fill().Background(cs.SurfaceContainerLow).Children(func() {
		ui.Scroll(c.Key("props-scroll")).Fill().Children(func() {
			ui.Column(c).FillWidth().Padding(16).Gap(10).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					md3e.IconBadge(c, symbols.Tune.Draw, md3e.IconBadgeOptions{
						Variant: md3e.IconBadgeTonal,
						Size:    tokens.SizeXS,
						Shape:   shape.Cookie4Sided(),
						Label:   "Properties",
					})
					md3e.Text(c, "Properties", tokens.TitleSmall, true)
				})
				md3e.Divider(c)
				a.propertyRow(c, "Kind", n.kind)
				a.propertyRow(c, "Region", "eu-west")
				a.propertyRow(c, "Uptime", "99.982%")
				a.propertyRow(c, "Owner", "Platform")
				a.propertyRow(c, "Revision", "2.4.118")
				md3e.Divider(c)
				caption(c, "Drag the seam to resize; double-click to reset.")
			})
		})
	})
}

// propertyRow is one label/value pair in the supporting pane.
func (a *appState) propertyRow(c *ui.Context, label, value string) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
		md3e.Text(c, label, tokens.BodySmall).TextColor(cs.OnSurfaceVariant)
		ui.Box(c).Grow(1)
		md3e.TextWith(c, value, md3e.TextOptions{
			Role:        tokens.BodySmall,
			Mono:        true,
			TabularNums: true,
			Truncate:    md3e.TruncateMiddle,
			MaxLines:    1,
		}).MaxWidthPercent(62)
	})
}
