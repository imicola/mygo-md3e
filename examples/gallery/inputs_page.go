package main

import (
	"github.com/egoist/mygo/ui"

	md3e "mygo-md3e"
	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

// inputsPage holds the state of the selection and input demos.
type inputsPage struct {
	ready    bool
	checked  bool
	mixed    ui.CheckState
	radio    int
	switchOn bool
	slider   float64
	lo, hi   float64
	filterA  bool
	filterB  bool
	tags     []string
	name     string
	secret   string
	show     bool
	bio      string
	choice   int
}

func (p *inputsPage) ensure() {
	if !p.ready {
		p.ready = true
		p.checked = true
		p.mixed = ui.CheckMixed
		p.radio = 1
		p.switchOn = true
		p.slider = 0.35
		p.lo, p.hi = 0.25, 0.7
		p.tags = []string{"Kotlin", "Swift", "Go"}
	}
}

// choices are the select demo's options.
var choices = []string{"Compact", "Comfortable", "Spacious"}

// inputsPageView shows the selection controls, sliders, chips, text fields and
// the select.
func (a *app) inputsPageView(c *ui.Context) {
	p := &a.inputs
	p.ensure()

	page(c, "Selection & Inputs",
		"Switches, tri-state check boxes, radios, sliders and range sliders, the four chip families, text fields and a select.",
		func() {
			section(c, "Selection controls", func() {
				ui.Row(c).Wrap().Gap(24).AlignItems(ui.Start).Children(func() {
					ui.Column(c).Gap(6).Children(func() {
						md3e.Checkbox(c.Key("check"), &p.checked, md3e.CheckboxOptions{Label: "Wi-Fi"})
						md3e.CheckboxState(c.Key("check-mixed"), &p.mixed, md3e.CheckboxOptions{Label: "Mixed state"})
						md3e.Checkbox(c.Key("check-error"), &p.checked, md3e.CheckboxOptions{
							Label: "Error state", Error: true, ErrorMessage: "This choice is invalid",
						})
						md3e.Checkbox(c.Key("check-disabled"), &p.checked, md3e.CheckboxOptions{
							Label: "Disabled", Disabled: true,
						})
					})
					md3e.RadioGroup(c, func() {
						md3e.Radio(c.Key("radio-compact"), &p.radio, 0, "Compact")
						md3e.Radio(c.Key("radio-comfortable"), &p.radio, 1, "Comfortable")
						md3e.Radio(c.Key("radio-spacious"), &p.radio, 2, "Spacious")
					}, md3e.RadioGroupOptions{Label: "Density"})
					ui.Column(c).Gap(6).Children(func() {
						md3e.Switch(c.Key("switch"), &p.switchOn, md3e.SwitchOptions{Label: "Notifications"})
						md3e.Switch(c.Key("switch-icons"), &p.switchOn, md3e.SwitchOptions{
							Label: "With icons", OffIcon: mdicon.Close.Draw, OnIcon: mdicon.Check.Draw,
						})
						md3e.Switch(c.Key("switch-off"), &p.switchOn, md3e.SwitchOptions{
							Label: "Disabled switch", Disabled: true,
						})
					})
				})
			})

			section(c, "Sliders", func() {
				ui.Column(c).FillWidth().Gap(16).Children(func() {
					md3e.Text(c, "Continuous", tokens.LabelMedium, true)
					md3e.Slider(c.Key("slider"), &p.slider, 0, 1, md3e.SliderOptions{Label: "Volume"})
					md3e.Text(c, "Discrete with stops", tokens.LabelMedium, true)
					md3e.Slider(c.Key("slider-steps"), &p.slider, 0, 1, md3e.SliderOptions{
						Label: "Discrete volume", Step: 0.2,
					})
					md3e.Text(c, "Centered", tokens.LabelMedium, true)
					md3e.Slider(c.Key("slider-centered"), &p.slider, 0, 1, md3e.SliderOptions{
						Label: "Balance", Centered: true,
					})
					md3e.Text(c, "Range", tokens.LabelMedium, true)
					md3e.RangeSlider(c.Key("range"), &p.lo, &p.hi, 0, 1, md3e.RangeSliderOptions{
						Label: "Price range", Step: 0.05,
					})
					md3e.Slider(c.Key("slider-disabled"), &p.slider, 0, 1, md3e.SliderOptions{
						Label: "Disabled slider", Disabled: true,
					})
				})
			})

			section(c, "Chips", func() {
				md3e.ChipGroup(c, func() {
					md3e.AssistChip(c.Key("assist"), "Directions", md3e.ChipOptions{
						LeadingIcon: mdicon.ArrowForward.Draw,
						OnClick:     func() { p.choice = 0 },
					})
					md3e.SuggestionChip(c.Key("suggestion"), "Try a theme", md3e.ChipOptions{
						LeadingIcon: mdicon.Star.Draw,
					})
					md3e.FilterChip(c.Key("filter-a"), &p.filterA, "Has photo", md3e.FilterChipOptions{
						LeadingIcon: mdicon.Visibility.Draw,
					})
					md3e.FilterChip(c.Key("filter-b"), &p.filterB, "Elevated", md3e.FilterChipOptions{
						Elevated: true,
					})
					md3e.AssistChip(c.Key("assist-disabled"), "Disabled", md3e.ChipOptions{Disabled: true})
				}, md3e.ChipGroupOptions{Label: "Chip samples"})
				ui.Row(c).Wrap().Gap(8).Children(func() {
					for _, tag := range append([]string(nil), p.tags...) {
						tag := tag
						md3e.InputChip(c.Key("tag-"+tag), tag, md3e.InputChipOptions{
							TrailingIcon: mdicon.Edit.Draw,
							OnRemove:     func() { p.removeTag(tag) },
						})
					}
					if len(p.tags) == 0 {
						caption(c, "Every chip is removed; it comes back with the reset button below.")
					}
				})
				if md3e.Button(c.Key("tags-reset"), "Reset chips", md3e.ButtonOptions{
					Variant: md3e.ButtonText, Size: tokens.SizeXS,
				}).Clicked() {
					p.tags = []string{"Kotlin", "Swift", "Go"}
				}
			})

			section(c, "Text fields", func() {
				ui.Row(c).Wrap().Gap(16).AlignItems(ui.Start).Children(func() {
					ui.Column(c).Width(280).Gap(14).Children(func() {
						md3e.TextField(c.Key("field-name"), &p.name, md3e.TextFieldOptions{
							Variant: md3e.TextFieldOutlined, Label: "Full name", SupportingText: "Outlined variant",
						})
						md3e.TextField(c.Key("field-filled"), &p.name, md3e.TextFieldOptions{
							Variant: md3e.TextFieldFilled, Label: "Filled field", Prefix: "@",
						})
						md3e.TextField(c.Key("field-error"), &p.name, md3e.TextFieldOptions{
							Variant: md3e.TextFieldOutlined, Label: "Email", Error: true,
							ErrorMessage: "Enter a valid address",
						})
					})
					ui.Column(c).Width(280).Gap(14).Children(func() {
						icon := mdicon.Visibility.Draw
						if p.show {
							icon = mdicon.VisibilityOff.Draw
						}
						md3e.TextField(c.Key("field-password"), &p.secret, md3e.TextFieldOptions{
							Variant: md3e.TextFieldOutlined, Label: "Password", Password: !p.show,
							TrailingIcon: icon, TrailingLabel: "Show password",
							OnTrailing: func() { p.show = !p.show },
						})
						md3e.TextField(c.Key("field-readonly"), &p.name, md3e.TextFieldOptions{
							Variant: md3e.TextFieldFilled, Label: "Read only", ReadOnly: true,
						})
						md3e.TextField(c.Key("field-disabled"), &p.name, md3e.TextFieldOptions{
							Variant: md3e.TextFieldOutlined, Label: "Disabled", Disabled: true,
						})
					})
					ui.Column(c).Width(320).Gap(14).Children(func() {
						md3e.TextArea(c.Key("field-bio"), &p.bio, md3e.TextFieldOptions{
							Variant: md3e.TextFieldOutlined, Label: "Bio", MaxLength: 140,
							CharacterCounter: true, Height: 120,
						})
					})
				})
			})

			section(c, "Select", func() {
				md3e.DropdownMenu(c.Key("select"), "Select: "+choices[p.choice], choiceItems(&p.choice),
					md3e.MenuOptions{Label: "Density select"})
				caption(c, "The Expressive dropdown menu as a select: checked radio rows and a chevron trigger.")
			})
		})
}

// choiceItems builds the select's menu rows.
func choiceItems(choice *int) []md3e.MenuItem {
	items := make([]md3e.MenuItem, 0, len(choices))
	for i, label := range choices {
		idx, name := i, label
		items = append(items, md3e.MenuItem{
			Label:   name,
			Radio:   true,
			Checked: i == *choice,
			OnClick: func() { *choice = idx },
		})
	}
	return items
}

// removeTag drops one input chip.
func (p *inputsPage) removeTag(tag string) {
	for i, t := range p.tags {
		if t == tag {
			p.tags = append(p.tags[:i], p.tags[i+1:]...)
			return
		}
	}
}

// unusedTokens keeps the tokens import used while the page grows.
var _ = tokens.SizeXS
