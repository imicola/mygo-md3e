package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

// TopAppBarVariant selects one of the four Material Design 3 Expressive top
// app bar layouts.
type TopAppBarVariant int

const (
	// TopAppBarSmall is the 64 DIP bar with a left-aligned title.
	TopAppBarSmall TopAppBarVariant = iota
	// TopAppBarCenterAligned is the 64 DIP bar with a centered title.
	TopAppBarCenterAligned
	// TopAppBarMedium expands to 112 DIP with a large title that collapses to
	// 64 DIP with ScrollProgress.
	TopAppBarMedium
	// TopAppBarLarge expands to 152 DIP with an even larger title that
	// collapses to 64 DIP with ScrollProgress.
	TopAppBarLarge
)

// AppBarAction is one trailing action of a TopAppBar or SearchAppBar.
type AppBarAction struct {
	// Label is the accessible name of the action.
	Label string
	// Icon draws the action's icon.
	Icon IconDraw
	// OnClick runs when the action is pressed.
	OnClick func()
}

// TopAppBarOptions configures TopAppBar. The zero value is a small, expanded
// bar with no navigation icon, subtitle or actions.
type TopAppBarOptions struct {
	// Variant selects the Small, CenterAligned, Medium or Large layout.
	Variant TopAppBarVariant
	// NavigationIcon draws the leading navigation button; none when nil.
	NavigationIcon IconDraw
	// NavigationLabel names the navigation button; "Back" when empty.
	NavigationLabel string
	// OnNavigation runs when the navigation button is pressed.
	OnNavigation func()
	// Subtitle shows a supporting line under the title.
	Subtitle string
	// Actions are the trailing icon buttons.
	Actions []AppBarAction
	// Label overrides the accessible name of the bar; the title by default.
	Label string
	// ScrollProgress collapses Medium and Large bars as it goes from 0, fully
	// expanded, to 1, the small 64 DIP bar with the Surface Container tone.
	// Values outside 0..1 are clamped.
	ScrollProgress float32
}

// Top app bar metrics, in DIP.
const (
	topAppBarCollapsedHeight = 64
	topAppBarMediumHeight    = 112
	topAppBarLargeHeight     = 152
	appBarIconSize           = 48
	appBarBarPadding         = 4
)

// TopAppBar builds a Material Design 3 Expressive top app bar: a Small or
// CenterAligned 64 DIP bar, or a Medium (112 DIP) or Large (152 DIP) bar whose
// height, title size and Surface -> Surface Container tone interpolate with
// ScrollProgress as the content scrolls under it. It carries a leading
// navigation button, the title and optional subtitle, and trailing icon
// buttons; the row mirrors right to left.
//
//	md3e.TopAppBar(c, "Inbox", md3e.TopAppBarOptions{
//		Variant:        md3e.TopAppBarLarge,
//		NavigationIcon: mdicon.Menu.Draw,
//		Subtitle:       "3 unread",
//		ScrollProgress: app.scroll,
//	})
func TopAppBar(c *ui.Context, title string, opts ...TopAppBarOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: TopAppBar takes at most one TopAppBarOptions")
	}
	if title == "" {
		panic("md3e: TopAppBar requires a title")
	}
	var o TopAppBarOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	p := clampF32(o.ScrollProgress, 0, 1)
	expandedHeight := float32(topAppBarCollapsedHeight)
	switch o.Variant {
	case TopAppBarMedium:
		expandedHeight = topAppBarMediumHeight
	case TopAppBarLarge:
		expandedHeight = topAppBarLargeHeight
	}
	height := expandedHeight + (topAppBarCollapsedHeight-expandedHeight)*p

	collapsedBG := cs.SurfaceContainer
	if collapsedBG.A == 0 {
		collapsedBG = SurfaceColorForElevation(th, tokens.Level2)
	}
	background := cs.Surface.Mix(collapsedBG, p)

	bar := ui.Column(c).
		Height(height).
		Clip().
		Background(background).
		TextColor(cs.OnSurface).
		Role(ui.RoleRegion).
		Label(labelOr(o.Label, title))

	bigRole := tokens.HeadlineMedium
	switch o.Variant {
	case TopAppBarMedium:
		bigRole = tokens.HeadlineSmall
	case TopAppBarSmall, TopAppBarCenterAligned:
		bigRole = tokens.TitleLarge
	}
	smallStyle := th.TypeStyle(tokens.TitleLarge, false)
	bigStyle := th.TypeStyle(bigRole, false)
	textScale := th.EffectiveTextScale()
	titleSize := (float32(bigStyle.Size) + (float32(smallStyle.Size)-float32(bigStyle.Size))*p) * textScale
	titleLine := (float32(bigStyle.LineHeight) + (float32(smallStyle.LineHeight)-float32(bigStyle.LineHeight))*p) * textScale
	titleTracking := (float32(bigStyle.Tracking) + (float32(smallStyle.Tracking)-float32(bigStyle.Tracking))*p) * textScale

	inlineTitle := o.Variant == TopAppBarSmall || o.Variant == TopAppBarCenterAligned || p >= 0.999
	centered := o.Variant == TopAppBarCenterAligned

	bar.Children(func() {
		top := ui.Row(c).
			Height(topAppBarCollapsedHeight).
			FillWidth().
			PaddingX(appBarBarPadding).
			AlignItems(ui.Center)
		top.Children(func() {
			inlineTitleSubtitle := func() {
				applyAppBarTitle(c, title, float32(smallStyle.Size)*textScale, float32(smallStyle.LineHeight)*textScale,
					smallStyle.Weight, float32(smallStyle.Tracking)*textScale, th.Font)
				if o.Subtitle != "" {
					applyAppBarSubtitle(c, o.Subtitle, th)
				}
			}
			switch {
			case centered && inlineTitle:
				// Symmetric side slots keep the title on the bar's geometric
				// center, however many navigation or trailing icons there are.
				side := float32(0)
				if o.NavigationIcon != nil {
					side = appBarIconSize
				}
				if n := float32(len(o.Actions)) * appBarIconSize; n > side {
					side = n
				}
				lead := ui.Row(c).Width(side).Shrink(0).AlignItems(ui.Center)
				lead.Children(func() {
					if o.NavigationIcon != nil {
						appBarIconSide(c, th, labelOr(o.NavigationLabel, "Back"), o.NavigationIcon, o.OnNavigation, false)
					}
				})
				slot := ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center)
				slot.Children(inlineTitleSubtitle)
				trail := ui.Row(c).Width(side).Shrink(0).Justify(ui.End).AlignItems(ui.Center)
				trail.Children(func() {
					for i := range o.Actions {
						action := &o.Actions[i]
						appBarIconSide(c, th, action.Label, action.Icon, action.OnClick, false)
					}
				})
			case inlineTitle:
				if o.NavigationIcon != nil {
					appBarIconSide(c, th, labelOr(o.NavigationLabel, "Back"), o.NavigationIcon, o.OnNavigation, false)
				}
				slot := ui.Column(c).Grow(1).Justify(ui.Center)
				if o.NavigationIcon == nil {
					// The bar's own 4 DIP padding plus this lands the title
					// 16 DIP from the edge, as MD3E specifies.
					slot.PaddingStart(12)
				}
				slot.Children(inlineTitleSubtitle)
				for i := range o.Actions {
					action := &o.Actions[i]
					appBarIconSide(c, th, action.Label, action.Icon, action.OnClick, false)
				}
			default:
				if o.NavigationIcon != nil {
					appBarIconSide(c, th, labelOr(o.NavigationLabel, "Back"), o.NavigationIcon, o.OnNavigation, false)
				}
				ui.Box(c).Grow(1)
				for i := range o.Actions {
					action := &o.Actions[i]
					appBarIconSide(c, th, action.Label, action.Icon, action.OnClick, false)
				}
			}
		})

		if !inlineTitle {
			area := ui.Column(c).
				FillWidth().
				Grow(1).
				Padding(0, 0, 16, 0).
				PaddingStart(16).
				PaddingEnd(16).
				Justify(ui.End).
				Clip()
			area.Children(func() {
				applyAppBarTitle(c, title, titleSize, titleLine, bigStyle.Weight, titleTracking, th.Font)
				if o.Subtitle != "" {
					applyAppBarSubtitle(c, o.Subtitle, th)
				}
			})
		}
	})
	return bar
}

// applyAppBarTitle adds the title text with the interpolated metrics.
func applyAppBarTitle(c *ui.Context, title string, size, line float32, weight int, tracking float32, font string) {
	t := ui.Text(c, title).NoWrap().Height(line).FontSize(size).FontWeight(weight).
		FixedLineHeight(line).LetterSpacing(tracking)
	if font != "" {
		t.Font(font)
	}
}

// applyAppBarSubtitle adds the supporting line under a title in the
// OnSurfaceVariant tone.
func applyAppBarSubtitle(c *ui.Context, subtitle string, th *Theme) {
	style := th.TypeStyle(tokens.BodyMedium, false)
	scale := th.EffectiveTextScale()
	line := float32(style.LineHeight) * scale
	t := ui.Text(c, subtitle).NoWrap().Height(line).
		FontSize(float32(style.Size) * scale).
		FontWeight(style.Weight).
		FixedLineHeight(line).
		LetterSpacing(float32(style.Tracking) * scale).
		TextColor(th.Colors.OnSurfaceVariant)
	if th.Font != "" {
		t.Font(th.Font)
	}
}

// appBarIconSide builds a 48 DIP icon button of an app bar.
func appBarIconSide(c *ui.Context, th *Theme, label string, icon IconDraw, onClick func(), disabled bool) ui.Element {
	btn := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Size(appBarIconSize, appBarIconSize).
		Label(label)
	disabled = disabled || btn.IsDisabled()
	ink := th.Colors.OnSurface
	if disabled {
		ink = ink.Alpha(float32(tokens.DisabledContentOpacity))
	}
	btn.Disabled(disabled).TextColor(ink).Radius(appBarIconSize / 2)
	AttachRipple(c, btn, StateLayerOptions{Color: ink, Radius: appBarIconSize / 2, Disabled: disabled})
	clicked := btn.Clicked()
	btn.Children(func() {
		drawIconBox(c, icon, 24, ink)
	})
	if clicked && onClick != nil {
		onClick()
	}
	return btn
}

// SearchAppBarOptions configures SearchAppBar. The zero value is an enabled bar
// with a leading search icon and a "Search" placeholder.
type SearchAppBarOptions struct {
	// Placeholder shows while the query is empty and names the field when no
	// Label overrides it.
	Placeholder string
	// Label overrides the accessible name of the text field.
	Label string
	// LeadingIcon draws the leading button's icon; mdicon.Search when nil.
	LeadingIcon IconDraw
	// LeadingLabel names the leading button; "Search" when empty.
	LeadingLabel string
	// OnLeading runs when the leading button is pressed. Without it the
	// leading icon is decorative.
	OnLeading func()
	// Trailing are extra trailing action buttons.
	Trailing []AppBarAction
	// AutoFocus focuses the field as it appears.
	AutoFocus bool
	// Disabled disables the bar and its field.
	Disabled bool
	// OnChanged runs with the query whenever the user edits it.
	OnChanged func(query string)
	// OnSubmitted runs with the query when Enter submits it.
	OnSubmitted func(query string)
}

type (
	searchClearKey    struct{}
	searchTrailingKey struct{ index int }
)

// searchBarHeight is the height of a SearchAppBar, in DIP.
const searchBarHeight = 56

// SearchAppBar builds a Material Design 3 Expressive search app bar: an
// elevated, full-cornered Surface Container pill around a ui.TextInputBase
// editing *query, with an optional leading search/back button, a trailing clear
// button (mdicon.Clear) while the query is non-empty, and other trailing
// actions. Edits and Enter report through OnChanged and OnSubmitted.
//
// The input is keyed by the stable "search-query" key inside the bar, so a
// caller's c.Key on the bar itself stays with the outer element:
//
//	md3e.SearchAppBar(c, &app.query, md3e.SearchAppBarOptions{
//		Placeholder: "Search mail",
//		OnSubmitted: app.runSearch,
//	})
func SearchAppBar(c *ui.Context, query *string, opts ...SearchAppBarOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: SearchAppBar takes at most one SearchAppBarOptions")
	}
	if query == nil {
		panic("md3e: SearchAppBar requires a non-nil *string")
	}
	var o SearchAppBarOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	placeholder := labelOr(o.Placeholder, "Search")
	fieldLabel := labelOr(o.Label, placeholder)
	leadingLabel := labelOr(o.LeadingLabel, "Search")
	leadingIcon := o.LeadingIcon
	if leadingIcon == nil {
		leadingIcon = mdicon.Search.Draw
	}

	background := cs.SurfaceContainerHigh
	if background.A == 0 {
		background = SurfaceColorForElevation(th, tokens.Level2)
	}
	ink := cs.OnSurface

	bar := ui.Row(c).
		Height(searchBarHeight).
		PaddingX(8).
		Gap(4).
		AlignItems(ui.Center).
		Clip().
		Background(background).
		Radius(searchBarHeight / 2).
		DimDisabled(false).
		Role(ui.RoleSearch).
		Label(fieldLabel)
	disabled := o.Disabled || bar.IsDisabled()
	fieldInk := ink
	if disabled {
		fieldInk = ink.Alpha(float32(tokens.DisabledContentOpacity))
	}
	bar.TextColor(fieldInk)
	ApplyElevationShadows(th, bar, tokens.Level2)
	if o.Disabled {
		bar.Disabled(true)
	}

	bar.Children(func() {
		if o.OnLeading != nil {
			btn := ui.ButtonBase(c).
				DimDisabled(false).
				FocusRing(false).
				Size(appBarIconSize, appBarIconSize).
				Label(leadingLabel)
			leadingDisabled := disabled || btn.IsDisabled()
			btn.Disabled(o.Disabled).TextColor(fieldInk).Radius(appBarIconSize / 2)
			AttachRipple(c, btn, StateLayerOptions{Color: fieldInk, Radius: appBarIconSize / 2, Disabled: leadingDisabled})
			clicked := btn.Clicked()
			btn.Children(func() {
				drawIconBox(c, leadingIcon, 24, fieldInk)
			})
			if clicked {
				o.OnLeading()
			}
		} else {
			ui.Box(c).Size(appBarIconSize, appBarIconSize).Center().Children(func() {
				drawIconBox(c, leadingIcon, 24, fieldInk)
			})
		}

		input := ui.TextInputBase(c.Key("search-query"), query).
			DimDisabled(false).
			Grow(1).
			Label(fieldLabel).
			Placeholder(placeholder).
			TextColor(fieldInk).
			CaretColor(cs.Primary).
			Disabled(o.Disabled)
		ApplyTypeStyle(c, input, tokens.BodyLarge, false)
		if o.AutoFocus {
			input.AutoFocus()
		}
		changed := input.Changed()
		submitted := input.Submitted()

		if *query != "" {
			clear := ui.ButtonBase(c.Key(searchClearKey{})).
				DimDisabled(false).
				FocusRing(false).
				Size(appBarIconSize, appBarIconSize).
				Label("Clear search")
			clear.Disabled(o.Disabled).TextColor(fieldInk).Radius(appBarIconSize / 2)
			AttachRipple(c, clear, StateLayerOptions{Color: fieldInk, Radius: appBarIconSize / 2, Disabled: o.Disabled})
			clicked := clear.Clicked()
			clear.Children(func() {
				drawIconBox(c, mdicon.Clear.Draw, 24, fieldInk)
			})
			if clicked {
				*query = ""
				if o.OnChanged != nil {
					o.OnChanged("")
				}
			}
		}
		for i := range o.Trailing {
			action := &o.Trailing[i]
			appBarIconSide(c.Key(searchTrailingKey{index: i}), th, action.Label, action.Icon, action.OnClick, o.Disabled)
		}

		if changed && o.OnChanged != nil {
			o.OnChanged(*query)
		}
		if submitted && o.OnSubmitted != nil {
			o.OnSubmitted(*query)
		}
	})
	return bar
}
