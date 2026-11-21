package main

import (
	"github.com/egoist/mygo/ui"

	md3e "mygo-md3e"
	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

// feedbackPage holds the state of the feedback and container demos.
type feedbackPage struct {
	ready   bool
	count   int
	segment int
	clicked string
}

func (p *feedbackPage) ensure() {
	if !p.ready {
		p.ready = true
		p.count = 3
		p.segment = 0
	}
}

// feedbackPageView shows badges, tooltips, snackbars, cards, dividers and the
// expressive segmented list.
func (a *app) feedbackPageView(c *ui.Context) {
	p := &a.feedback
	p.ensure()

	page(c, "Feedback & Containers",
		"Badges, plain and rich tooltips, the snackbar queue, the three card variants, dividers and the expressive segmented list.",
		func() {
			section(c, "Badges", func() {
				ui.Row(c).Wrap().Gap(24).AlignItems(ui.Center).Children(func() {
					md3e.Badge(c, "3", md3e.BadgeOptions{Label: "3 unread"})
					md3e.Badge(c, "1200", md3e.BadgeOptions{MaxCount: 999, Label: "Over 999"})
					md3e.SmallBadge(c, md3e.BadgeOptions{Label: "New activity"})
					md3e.BadgedBox(c, func() {
						md3e.IconButton(c, "Inbox", mdicon.MoreVert.Draw, md3e.IconButtonOptions{
							Variant: md3e.IconButtonTonal, Size: tokens.SizeS,
						})
					}, func() ui.Element {
						return md3e.Badge(c, "12", md3e.BadgeOptions{})
					}, md3e.BadgedBoxOptions{Label: "Inbox with 12 unread"})
					md3e.BadgedBox(c, func() {
						md3e.FAB(c, "Mail", mdicon.MoreHoriz.Draw, md3e.FABOptions{Size: md3e.FABSizeSmall})
					}, func() ui.Element {
						return md3e.SmallBadge(c, md3e.BadgeOptions{})
					}, md3e.BadgedBoxOptions{Label: "Mail with new activity"})
				})
			})

			section(c, "Tooltips", func() {
				ui.Row(c).Wrap().Gap(24).AlignItems(ui.Center).Children(func() {
					md3e.PlainTooltip(c,
						md3e.Button(c.Key("tip-plain"), "Hover for a plain tooltip", md3e.ButtonOptions{
							Variant: md3e.ButtonTonal, Size: tokens.SizeS,
						}), "Adds a file to this folder")
					md3e.RichTooltip(c,
						md3e.Button(c.Key("tip-rich"), "Hover for a rich tooltip", md3e.ButtonOptions{
							Variant: md3e.ButtonOutlined, Size: tokens.SizeS,
						}), md3e.RichTooltipOptions{
							Title: "Rich tooltips",
							Body:  "A heading, supporting text and an optional action, with its own reach for the pointer.",
							Action: &md3e.RichTooltipAction{
								Label:   "Learn more",
								OnClick: func() { p.clicked = "Learn more" },
							},
						})
				})
			})

			section(c, "Snackbars", func() {
				ui.Row(c).Wrap().Gap(10).AlignItems(ui.Center).Children(func() {
					if md3e.Button(c.Key("snack-plain"), "Show snackbar", md3e.ButtonOptions{
						Variant: md3e.ButtonFilled, Size: tokens.SizeS,
					}).Clicked() {
						a.snack.Show(md3e.SnackbarOptions{Message: "Note archived", ShowClose: true})
					}
					if md3e.Button(c.Key("snack-action"), "Show action snackbar", md3e.ButtonOptions{
						Variant: md3e.ButtonTonal, Size: tokens.SizeS,
					}).Clicked() {
						a.snack.Show(md3e.SnackbarOptions{
							Message:     "Conversation archived",
							ActionLabel: "Undo",
							OnAction:    func() { p.clicked = "Undo" },
						})
					}
				})
				caption(c, "Messages queue in the host and go by themselves after four seconds.")
			})

			section(c, "Cards", func() {
				ui.Row(c).Wrap().Gap(12).AlignItems(ui.Start).Children(func() {
					md3e.ElevatedCard(c, func() {
						cardBody(c, "Elevated card", "Level 1 shadows over SurfaceContainerLow.")
					}, md3e.CardOptions{Label: "Elevated card"})
					md3e.FilledCard(c, func() {
						cardBody(c, "Filled card", "SurfaceContainerHighest, no shadow.")
					}, md3e.CardOptions{Label: "Filled card"})
					md3e.OutlinedCard(c, func() {
						cardBody(c, "Outlined card", "OutlineVariant border over Surface.")
					}, md3e.CardOptions{Label: "Outlined card"})
					md3e.Card(c, func() {
						cardBody(c, "Clickable card", "Press this card for ripple and a corner morph.")
					}, md3e.CardOptions{
						Variant:   md3e.CardFilled,
						Clickable: true,
						OnClick:   func() { p.clicked = "Clickable card" },
						Label:     "Clickable card",
					})
				})
				if p.clicked != "" {
					caption(c, "Last action: "+p.clicked)
				}
			})

			section(c, "Dividers", func() {
				ui.Column(c).FillWidth().Gap(10).Children(func() {
					md3e.Divider(c)
					ui.Row(c).Height(28).AlignItems(ui.Center).Gap(12).Children(func() {
						md3e.Text(c, "Start", tokens.LabelMedium)
						md3e.Divider(c, md3e.DividerOptions{Vertical: true})
						md3e.Text(c, "End", tokens.LabelMedium)
						md3e.Divider(c, md3e.DividerOptions{Inset: 40})
						md3e.Text(c, "Inset", tokens.LabelMedium)
					})
				})
			})

			section(c, "Expressive segmented list", func() {
				items := []md3e.ListItem{
					{Key: "inbox", Overline: "Today", Headline: "Inbox", SupportingText: "3 unread messages", Icon: mdicon.MoreHoriz.Draw, Meta: "3"},
					{Key: "starred", Headline: "Starred", SupportingText: "Pinned conversations", Icon: mdicon.Star.Draw},
					{Key: "drafts", Headline: "Drafts", SupportingText: "2 unsent", Icon: mdicon.Edit.Draw, Meta: "2"},
					{Key: "archive", Headline: "Archive", SupportingText: "Everything else", Icon: mdicon.Check.Draw},
				}
				md3e.List(c.Key("segmented-list"), items, md3e.ListOptions{
					Segmented: true, Selected: &p.segment, Label: "Mail folders",
					OnSelect: func(i int) { p.clicked = items[i].Headline },
				})
			})
		})
}

// cardBody fills a card with a title and a supporting line.
func cardBody(c *ui.Context, title, text string) {
	ui.Column(c).Width(240).Gap(6).Children(func() {
		md3e.Text(c, title, tokens.TitleMedium, true)
		md3e.Text(c, text, tokens.BodySmall)
	})
}
