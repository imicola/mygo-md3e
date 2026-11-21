package main

import (
	"time"

	"github.com/egoist/mygo/ui"

	md3e "mygo-md3e"
	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

// overlaysPage holds the state of the dialogs, sheets, menus and pickers.
type overlaysPage struct {
	ready       bool
	alert       bool
	full        bool
	sheet       bool
	sheetDetent int
	standard    bool
	dateModal   bool
	date        time.Time
	from, to    time.Time
	clock       time.Time
	clock24     time.Time
	clockInput  time.Time
	last        string
}

func (p *overlaysPage) ensure() {
	if !p.ready {
		p.ready = true
		p.from = time.Date(2026, time.November, 20, 0, 0, 0, 0, time.Local)
		p.to = time.Date(2026, time.November, 26, 0, 0, 0, 0, time.Local)
		p.clock = time.Date(2026, time.January, 1, 9, 30, 0, 0, time.Local)
		p.clock24 = time.Date(2026, time.January, 1, 18, 45, 0, 0, time.Local)
		p.clockInput = time.Date(2026, time.January, 1, 7, 15, 0, 0, time.Local)
	}
}

// overlayMenuItems are the rows the menus demonstrate.
func overlayMenuItems(p *overlaysPage) []md3e.MenuItem {
	return []md3e.MenuItem{
		{Label: "Rename", Icon: mdicon.Edit.Draw, Shortcut: "⌘R", OnClick: func() { p.last = "Rename" }},
		{Label: "Duplicate", Icon: mdicon.Add.Draw, Shortcut: "⌘D", OnClick: func() { p.last = "Duplicate" }},
		{Label: "Move to folder", Icon: mdicon.ArrowForward.Draw, Submenu: []md3e.MenuItem{
			{Label: "Inbox", OnClick: func() { p.last = "Move to Inbox" }},
			{Label: "Archive", OnClick: func() { p.last = "Move to Archive" }},
		}},
		{Separator: true},
		{Label: "Delete", Icon: mdicon.Close.Draw, Shortcut: "⌘⌫", OnClick: func() { p.last = "Delete" }},
	}
}

// overlaysPageView shows the dialogs, bottom sheets, menus and the M8 pickers.
func (a *app) overlaysPageView(c *ui.Context) {
	p := &a.overlays
	p.ensure()

	page(c, "Overlays & Pickers",
		"Alert and full-screen dialogs, modal and standard bottom sheets, the menu family, and the docked, modal and range date pickers with both time pickers.",
		func() {
			section(c, "Dialogs", func() {
				ui.Row(c).Wrap().Gap(10).AlignItems(ui.Center).Children(func() {
					if md3e.Button(c.Key("open-alert"), "Open alert dialog", md3e.ButtonOptions{
						Variant: md3e.ButtonFilled, Size: tokens.SizeS,
					}).Clicked() {
						p.alert = true
					}
					if md3e.Button(c.Key("open-full"), "Open full-screen dialog", md3e.ButtonOptions{
						Variant: md3e.ButtonTonal, Size: tokens.SizeS,
					}).Clicked() {
						p.full = true
					}
				})
				md3e.AlertDialog(c, &p.alert, md3e.AlertDialogOptions{
					Icon:           mdicon.Star.Draw,
					Headline:       "Save this document?",
					SupportingText: "Your changes are kept in this device until you save them elsewhere.",
					Dismiss:        &md3e.DialogAction{Label: "Not now"},
					Confirm:        &md3e.DialogAction{Label: "Save", OnClick: func() { p.last = "Save" }},
					DismissOnScrim: true,
					Label:          "Save document",
				})
				md3e.FullScreenDialog(c, &p.full, md3e.FullScreenDialogOptions{
					Title:     "New document",
					SaveLabel: "Save",
					OnSave:    func() { p.last = "Saved document" },
					Content: func() ui.Element {
						return ui.Column(c).FillWidth().Padding(24).Gap(16).Children(func() {
							md3e.Text(c, "Full-screen dialogs hold a task of their own.", tokens.BodyMedium)
							md3e.TextField(c.Key("full-title"), &p.last, md3e.TextFieldOptions{
								Variant: md3e.TextFieldOutlined, Label: "Title",
							})
						})
					},
				})
			})

			section(c, "Bottom sheets", func() {
				ui.Row(c).Wrap().Gap(10).AlignItems(ui.Center).Children(func() {
					if md3e.Button(c.Key("open-sheet"), "Open modal sheet", md3e.ButtonOptions{
						Variant: md3e.ButtonFilled, Size: tokens.SizeS,
					}).Clicked() {
						p.sheet = true
					}
					md3e.Switch(c.Key("standard-sheet"), &p.standard, md3e.SwitchOptions{Label: "Standard sheet"})
					caption(c, "The modal sheet has two detents and pulls down to close.")
				})
				md3e.BottomSheet(c, &p.sheet, md3e.BottomSheetOptions{
					Modal: true, Detents: []float32{0.4, 0.9}, Detent: &p.sheetDetent,
					Label: "Share sheet",
					Content: func() ui.Element {
						return ui.Column(c).FillWidth().Padding(24).Gap(12).Children(func() {
							md3e.Text(c, "Share", tokens.HeadlineSmall, true)
							md3e.Text(c, "Choose where this note goes.", tokens.BodyMedium)
							md3e.List(c.Key("share-list"), []md3e.ListItem{
								{Headline: "Copy link", Icon: mdicon.Check.Draw},
								{Headline: "Send to a friend", Icon: mdicon.ArrowForward.Draw},
								{Headline: "Export as Markdown", Icon: mdicon.Edit.Draw},
							}, md3e.ListOptions{Label: "Share destinations"})
						})
					},
				})
				if p.standard {
					md3e.BottomSheet(c, &p.standard, md3e.BottomSheetOptions{
						Height: 220, Label: "Inline sheet",
						Content: func() ui.Element {
							return ui.Column(c).FillWidth().Padding(16).Gap(8).Children(func() {
								md3e.Text(c, "Standard sheet", tokens.TitleMedium, true)
								md3e.Text(c, "Laid out inline at the bottom of its parent.", tokens.BodySmall)
								if md3e.Button(c.Key("inline-sheet-close"), "Close inline sheet", md3e.ButtonOptions{
									Variant: md3e.ButtonText, Size: tokens.SizeXS,
								}).Clicked() {
									p.standard = false
								}
							})
						},
					})
				}
			})

			section(c, "Menus", func() {
				ui.Row(c).Wrap().Gap(16).AlignItems(ui.Center).Children(func() {
					md3e.DropdownMenu(c.Key("dropdown"), "Dropdown menu", overlayMenuItems(p),
						md3e.MenuOptions{Label: "Dropdown menu"})
					md3e.MenuButton(c, "More actions", mdicon.MoreVert.Draw, overlayMenuItems(p),
						md3e.MenuOptions{Label: "More actions"})
					md3e.ContextMenu(c, ui.Box(c).Padding(12).Radius(12).
						Background(md3e.CurrentTheme(c).Colors.SurfaceContainerHighest).
						Children(func() {
							md3e.Text(c, "Right-click this card for a context menu", tokens.BodySmall)
						}), overlayMenuItems(p), md3e.MenuOptions{Label: "Card actions"})
				})
				statusLine(c, "Last action: ", p.last)
			})

			section(c, "Date pickers", func() {
				ui.Row(c).Wrap().Gap(16).AlignItems(ui.Start).Children(func() {
					md3e.DatePicker(c.Key("date-docked"), &p.date, md3e.DatePickerOptions{Label: "Due date"})
					md3e.DateRangePicker(c.Key("date-range"), &p.from, &p.to, md3e.DatePickerOptions{
						Label: "Trip dates",
					})
					ui.Column(c).Width(320).Gap(10).AlignItems(ui.Start).Children(func() {
						if md3e.Button(c.Key("open-date-modal"), "Open modal date picker", md3e.ButtonOptions{
							Variant: md3e.ButtonFilled, Size: tokens.SizeS,
						}).Clicked() {
							p.dateModal = true
						}
						md3e.DatePicker(c.Key("date-modal"), &p.date, md3e.DatePickerOptions{
							Mode: md3e.DatePickerModal, Open: &p.dateModal, Label: "Modal due date",
						})
						caption(c, "The modal date picker locks the focus, closes with Escape and returns it to this button.")
					})
				})
			})

			section(c, "Time pickers", func() {
				ui.Row(c).Wrap().Gap(16).AlignItems(ui.Start).Children(func() {
					md3e.TimePicker(c.Key("clock-12"), &p.clock, md3e.TimePickerOptions{
						Label: "Start time", HourCycle: 12,
					})
					md3e.TimePicker(c.Key("clock-24"), &p.clock24, md3e.TimePickerOptions{
						Label: "End time", HourCycle: 24,
					})
					md3e.TimePicker(c.Key("clock-input"), &p.clockInput, md3e.TimePickerOptions{
						Label: "Reminder", Mode: md3e.TimePickerInput, HourCycle: 12,
					})
				})
			})
		})
}
