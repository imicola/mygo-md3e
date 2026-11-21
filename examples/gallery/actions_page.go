package main

import (
	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/tokens"
)

// actionsPage holds the state of the actions demo.
type actionsPage struct {
	ready       bool
	group       int
	multi       []bool
	favorite    bool
	expandedFab bool
	expandedBar bool
	query       string
	lastAction  string
}

func (p *actionsPage) ensure() {
	if !p.ready {
		p.ready = true
		p.group = -1
		p.multi = []bool{true, false, false}
		p.expandedFab = true
		p.expandedBar = true
	}
}

// actionsPageView shows the buttons, groups, split button, FAB family, the
// app bars and the floating toolbar.
func (a *app) actionsPageView(c *ui.Context) {
	p := &a.actions
	p.ensure()

	page(c, "Actions & Bars",
		"Expressive buttons at every size, button groups, the split button, the FAB family, top app bars and the floating toolbar.",
		func() {
			section(c, "Buttons", func() {
				type variant struct {
					name string
					v    md3e.ButtonVariant
				}
				variants := []variant{
					{"Filled", md3e.ButtonFilled},
					{"Tonal", md3e.ButtonTonal},
					{"Elevated", md3e.ButtonElevated},
					{"Outlined", md3e.ButtonOutlined},
					{"Text", md3e.ButtonText},
				}
				sizes := []tokens.SizeTier{tokens.SizeXS, tokens.SizeS, tokens.SizeM, tokens.SizeL, tokens.SizeXL}
				suffix := []string{"XS", "S", "M", "L", "XL"}
				ui.Column(c).FillWidth().Gap(10).Children(func() {
					for _, v := range variants {
						ui.Row(c).Wrap().AlignItems(ui.Center).Gap(10).Children(func() {
							md3e.Text(c, v.name, tokens.LabelMedium, true).Width(72)
							for i, size := range sizes {
								label := v.name + " " + suffix[i]
								if md3e.Button(c.Key("btn-"+label), label, md3e.ButtonOptions{
									Variant: v.v, Size: size,
								}).Clicked() {
									p.lastAction = label
								}
							}
						})
					}
					ui.Row(c).Wrap().Gap(10).AlignItems(ui.Center).Children(func() {
						md3e.Button(c.Key("btn-icon"), "With icon", md3e.ButtonOptions{
							Variant: md3e.ButtonFilled, Size: tokens.SizeS, LeadingIcon: mdicon.Add.Draw,
						})
						md3e.Button(c.Key("btn-disabled"), "Disabled", md3e.ButtonOptions{
							Variant: md3e.ButtonFilled, Size: tokens.SizeS, Disabled: true,
						})
						md3e.Button(c.Key("btn-square"), "Square", md3e.ButtonOptions{
							Variant: md3e.ButtonTonal, Size: tokens.SizeS, Shape: md3e.ButtonShapeSquare,
						})
					})
					statusLine(c, "Last action: ", p.lastAction)
				})
			})

			section(c, "Button groups & split button", func() {
				ui.Row(c).Wrap().Gap(16).AlignItems(ui.Center).Children(func() {
					md3e.ButtonGroup(c.Key("group-standard"), &p.group, []md3e.ButtonGroupItem{
						{Label: "Day"}, {Label: "Week"}, {Label: "Month"},
					}, md3e.ButtonGroupOptions{
						Kind: md3e.ButtonGroupStandard, Variant: md3e.ButtonTonal, Label: "Range",
					})
					md3e.ButtonGroup(c.Key("group-connected"), &p.group, []md3e.ButtonGroupItem{
						{Label: "Day"}, {Label: "Week"}, {Label: "Month"},
					}, md3e.ButtonGroupOptions{
						Kind: md3e.ButtonGroupConnected, Variant: md3e.ButtonFilled, Label: "Connected range",
					})
				})
				ui.Row(c).Wrap().Gap(16).AlignItems(ui.Center).Children(func() {
					md3e.ButtonGroupMulti(c.Key("group-multi"), &p.multi, []md3e.ButtonGroupItem{
						{Label: "Bold"}, {Label: "Italic"}, {Label: "Underline"},
					}, md3e.ButtonGroupOptions{
						Kind: md3e.ButtonGroupConnected, Variant: md3e.ButtonTonal, Label: "Text style",
					})
					md3e.ButtonGroupActions(c.Key("group-actions"), []md3e.ButtonGroupItem{
						{Label: "Cut", LeadingIcon: mdicon.Minus.Draw},
						{Label: "Copy", LeadingIcon: mdicon.Add.Draw},
						{Label: "Paste", LeadingIcon: mdicon.Check.Draw},
					}, func(i int) {
						p.lastAction = []string{"Cut", "Copy", "Paste"}[i]
					}, md3e.ButtonGroupOptions{
						Kind: md3e.ButtonGroupConnected, Variant: md3e.ButtonOutlined, Label: "Clipboard",
					})
					group, action, _, _ := md3e.SplitButton(c.Key("split"), "Save", md3e.SplitButtonOptions{
						Variant: md3e.ButtonFilled, Size: tokens.SizeS, LeadingIcon: mdicon.Check.Draw,
						Menu: func(panel ui.Element) {
							panel.Children(func() {
								for _, label := range []string{"Save as draft", "Save a copy", "Export"} {
									row := ui.ButtonBase(c).FillWidth().Height(40).PaddingX(12).
										Radius(8).Label(label)
									row.Children(func() {
										md3e.Text(c, label, tokens.LabelLarge)
									})
									if row.Clicked() {
										p.lastAction = label
									}
								}
							})
						},
					})
					_ = group
					if action.Clicked() {
						p.lastAction = "Save"
					}
				})
			})

			section(c, "Icon buttons", func() {
				ui.Row(c).Wrap().Gap(10).AlignItems(ui.Center).Children(func() {
					for _, v := range []struct {
						name string
						v    md3e.IconButtonVariant
					}{
						{"Standard", md3e.IconButtonStandard},
						{"Filled", md3e.IconButtonFilled},
						{"Tonal", md3e.IconButtonTonal},
						{"Outlined", md3e.IconButtonOutlined},
					} {
						md3e.IconButton(c, v.name+" icon", mdicon.Star.Draw, md3e.IconButtonOptions{
							Variant: v.v, Size: tokens.SizeS,
						})
					}
					md3e.ToggleIconButton(c.Key("favorite"), &p.favorite, "Toggle favorite",
						mdicon.Star.Draw, md3e.ToggleIconButtonOptions{
							Variant: md3e.IconButtonFilled, Size: tokens.SizeS, SelectedIcon: mdicon.Check.Draw,
						})
					md3e.IconButton(c, "Disabled icon", mdicon.Close.Draw, md3e.IconButtonOptions{
						Variant: md3e.IconButtonOutlined, Size: tokens.SizeS, Disabled: true,
					})
				})
			})

			section(c, "FAB family", func() {
				ui.Row(c).Wrap().Gap(20).AlignItems(ui.Center).Children(func() {
					md3e.FAB(c, "Create note", mdicon.Add.Draw, md3e.FABOptions{Size: md3e.FABSizeSmall})
					md3e.FAB(c, "Compose", mdicon.Edit.Draw, md3e.FABOptions{Size: md3e.FABSizeMedium})
					md3e.FAB(c, "New message", mdicon.Add.Draw, md3e.FABOptions{Size: md3e.FABSizeLarge, Color: md3e.FABTertiary})
					md3e.ExtendedFAB(c.Key("ext-fab"), "New document", mdicon.Edit.Draw, md3e.ExtendedFABOptions{
						Size: md3e.FABSizeMedium, Expanded: &p.expandedFab,
					})
					md3e.FABMenu(c, []md3e.FABMenuItem{
						{Label: "Add photo", Icon: mdicon.Add.Draw, OnClick: func() { p.lastAction = "Add photo" }},
						{Label: "Add file", Icon: mdicon.Edit.Draw, OnClick: func() { p.lastAction = "Add file" }},
						{Label: "Add link", Icon: mdicon.ChevronRight.Draw, OnClick: func() { p.lastAction = "Add link" }},
					}, md3e.FABMenuOptions{Label: "Create", MenuLabel: "Create actions"})
				})
				statusLine(c, "Last action: ", p.lastAction)
			})

			section(c, "Top app bars", func() {
				ui.Column(c).FillWidth().Gap(12).Children(func() {
					ui.Box(c).FillWidth().Children(func() {
						md3e.TopAppBar(c, "Small app bar", md3e.TopAppBarOptions{
							Variant:        md3e.TopAppBarSmall,
							Subtitle:       "With a subtitle and actions",
							Actions:        []md3e.AppBarAction{{Label: "Search", Icon: mdicon.Search.Draw, OnClick: func() { p.lastAction = "Search" }}},
							OnNavigation:   func() { p.lastAction = "Back" },
							NavigationIcon: mdicon.ArrowBack.Draw,
						})
					})
					ui.Box(c).FillWidth().Children(func() {
						md3e.TopAppBar(c, "Centered app bar", md3e.TopAppBarOptions{
							Variant:        md3e.TopAppBarCenterAligned,
							NavigationIcon: mdicon.Menu.Draw,
							Actions:        []md3e.AppBarAction{{Label: "More", Icon: mdicon.MoreVert.Draw, OnClick: func() { p.lastAction = "More" }}},
						})
					})
					ui.Box(c).FillWidth().Children(func() {
						md3e.TopAppBar(c, "Medium app bar", md3e.TopAppBarOptions{
							Variant: md3e.TopAppBarMedium, Subtitle: "Medium title",
						})
					})
					ui.Box(c).FillWidth().Children(func() {
						md3e.TopAppBar(c, "Large app bar", md3e.TopAppBarOptions{
							Variant: md3e.TopAppBarLarge, Subtitle: "Large title",
							Actions: []md3e.AppBarAction{{Label: "Add", Icon: mdicon.Add.Draw, OnClick: func() { p.lastAction = "Add" }}},
						})
					})
					ui.Box(c).FillWidth().Children(func() {
						md3e.SearchAppBar(c.Key("search-bar"), &p.query, md3e.SearchAppBarOptions{
							Placeholder: "Search the gallery",
							OnSubmitted: func(q string) { p.lastAction = "Search " + q },
						})
					})
				})
			})

			section(c, "Floating toolbar", func() {
				ui.Row(c).Wrap().Gap(24).AlignItems(ui.Center).Children(func() {
					md3e.FloatingToolbar(c.Key("toolbar-h"), []md3e.FloatingToolbarItem{
						{Label: "Edit", Icon: mdicon.Edit.Draw, OnClick: func() { p.lastAction = "Edit" }},
						{Label: "Copy", Icon: mdicon.Add.Draw, OnClick: func() { p.lastAction = "Copy" }},
						{Label: "Delete", Icon: mdicon.Close.Draw, OnClick: func() { p.lastAction = "Delete" }},
					}, md3e.FloatingToolbarOptions{
						Orientation: md3e.ToolbarHorizontal, Tone: md3e.ToolbarStandard, Label: "Document actions",
						Expanded: &p.expandedBar,
						Fab:      &md3e.FloatingToolbarFab{Label: "Add", Icon: mdicon.Add.Draw, OnClick: func() { p.lastAction = "Add" }},
					})
					md3e.FloatingToolbar(c.Key("toolbar-v"), []md3e.FloatingToolbarItem{
						{Label: "Undo", Icon: mdicon.ArrowBack.Draw, OnClick: func() { p.lastAction = "Undo" }},
						{Label: "Redo", Icon: mdicon.ArrowForward.Draw, OnClick: func() { p.lastAction = "Redo" }},
					}, md3e.FloatingToolbarOptions{
						Orientation: md3e.ToolbarVertical, Tone: md3e.ToolbarVibrant, Label: "History",
					})
				})
				statusLine(c, "Last action: ", p.lastAction)
			})
		})
}
