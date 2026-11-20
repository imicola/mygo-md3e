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
	appBarIconSize = 48
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

	inlineTitle := o.Variant == TopAppBarSmall || o.Variant == TopAppBarCenterAligned || p >= 0.999
	centered := o.Variant == TopAppBarCenterAligned

	bar.Children(func() {
		top := ui.Row(c).
			Height(topAppBarCollapsedHeight).
			FillWidth().
			PaddingX(appBarBarPadding).
			AlignItems(ui.Center)
		top.Children(func() {
			if o.NavigationIcon != nil {
				appBarIconSide(c, th, labelOr(o.NavigationLabel, "Back"), o.NavigationIcon, o.OnNavigation)
			}
			if inlineTitle {
				slot := ui.Column(c).Grow(1).Justify(ui.Center)
				if centered {
					slot.AlignItems(ui.Center)
				}
				slot.Children(func() {
					applyAppBarTitle(c, title, float32(smallStyle.Size)*textScale, float32(smallStyle.LineHeight)*textScale, smallStyle.Weight)
					if o.Subtitle != "" {
						applyAppBarSubtitle(c, o.Subtitle, th)
					}
				})
			} else {
				ui.Box(c).Grow(1)
			}
			for i := range o.Actions {
				action := &o.Actions[i]
				appBarIconSide(c, th, action.Label, action.Icon, action.OnClick)
			}
		})

		if !inlineTitle {
			area := ui.Column(c).
				FillWidth().
				Grow(1).
				PaddingStart(16).
				PaddingEnd(16).
				Padding(0, 0, 16, 0).
				Justify(ui.End).
				Clip()
			area.Children(func() {
				applyAppBarTitle(c, title, titleSize, titleLine, smallStyle.Weight)
				if o.Subtitle != "" {
					applyAppBarSubtitle(c, o.Subtitle, th)
				}
			})
		}
	})
	return bar
}

// applyAppBarTitle adds the title text with the interpolated metrics.
func applyAppBarTitle(c *ui.Context, title string, size, line float32, weight int) {
	ui.Text(c, title).NoWrap().Height(line).FontSize(size).FontWeight(weight).FixedLineHeight(line)
}

// applyAppBarSubtitle adds the supporting line under a title.
func applyAppBarSubtitle(c *ui.Context, subtitle string, th *Theme) {
	style := th.TypeStyle(tokens.BodyMedium, false)
	line := float32(style.LineHeight) * th.EffectiveTextScale()
	ui.Text(c, subtitle).NoWrap().Height(line).
		FontSize(float32(style.Size) * th.EffectiveTextScale()).
		FontWeight(style.Weight).
		FixedLineHeight(line)
}

// appBarIconSide builds a 48 DIP icon button of an app bar.
func appBarIconSide(c *ui.Context, th *Theme, label string, icon IconDraw, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Size(appBarIconSize, appBarIconSize).
		Label(label).
		TextColor(th.Colors.OnSurface).
		Radius(appBarIconSize / 2)
	AttachRipple(c, btn, StateLayerOptions{Color: th.Colors.OnSurface, Radius: appBarIconSize / 2})
	clicked := btn.Clicked()
	btn.Children(func() {
		if icon != nil {
			drawIconBox(c, icon, 24, th.Colors.OnSurface)
		}
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
		TextColor(ink).
		Radius(searchBarHeight / 2).
		Role(ui.RoleSearch).
		Label(fieldLabel)
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
				Label(leadingLabel).
				Disabled(o.Disabled).
				TextColor(ink).
				Radius(appBarIconSize / 2)
			AttachRipple(c, btn, StateLayerOptions{Color: ink, Radius: appBarIconSize / 2, Disabled: o.Disabled})
			clicked := btn.Clicked()
			btn.Children(func() {
				drawIconBox(c, leadingIcon, 24, ink)
			})
			if clicked {
				o.OnLeading()
			}
		} else {
			ui.Box(c).Size(appBarIconSize, appBarIconSize).Children(func() {
				drawIconBox(c, leadingIcon, 24, ink)
			})
		}

		input := ui.TextInputBase(c.Key("search-query"), query).
			DimDisabled(false).
			Grow(1).
			Label(fieldLabel).
			Placeholder(o.Placeholder).
			TextColor(ink).
			CaretColor(cs.Primary).
			Disabled(o.Disabled)
		ApplyTypeStyle(c, input, tokens.BodyLarge, false)
		if o.AutoFocus {
			input.AutoFocus()
		}
		changed := input.Changed()
		submitted := input.Submitted()

		if *query != "" {
			clear := ui.ButtonBase(c).
				DimDisabled(false).
				FocusRing(false).
				Size(appBarIconSize, appBarIconSize).
				Label("Clear search").
				Disabled(o.Disabled).
				TextColor(ink).
				Radius(appBarIconSize / 2)
			AttachRipple(c, clear, StateLayerOptions{Color: ink, Radius: appBarIconSize / 2, Disabled: o.Disabled})
			clicked := clear.Clicked()
			clear.Children(func() {
				drawIconBox(c, mdicon.Clear.Draw, 24, ink)
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
			appBarIconSide(c, th, action.Label, action.Icon, action.OnClick)
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

