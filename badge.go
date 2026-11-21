package md3e

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// BadgeOptions configures a Badge or SmallBadge. The zero value is a visible
// Error-colored badge that shows every count, truncating above 999.
type BadgeOptions struct {
	// MaxCount caps a numeric badge: a larger count reads "999+" for the default
	// 999. A value of zero or less keeps 999.
	MaxCount int
	// Visible hides the badge when it points to false; nil shows it.
	Visible *bool
	// ShowZero shows a numeric zero count; by default a zero badge is hidden.
	ShowZero bool
	// Label overrides the accessible name; the badge's text names it otherwise.
	Label string
	// ContainerColor overrides the Error container color.
	ContainerColor ui.Color
	// ContentColor overrides the OnError content color.
	ContentColor ui.Color
}

// badgeMetrics are the Material Design 3 badge sizes in DIP.
const (
	smallBadgeSide = 6
	largeBadgeHigh = 16
	badgeMaxCount  = 999
)

// Badge builds a large Material Design 3 badge: a pill at least 16 DIP high
// and wide in the Error color, holding a count or a short text. A numeric count
// above MaxCount reads "999+" by default; zero hides it unless ShowZero is set,
// and Visible can hide any badge. It carries its text as its accessible name:
//
//	md3e.Badge(c, "3", md3e.BadgeOptions{Label: "3 unread messages"})
func Badge(c *ui.Context, text string, opts ...BadgeOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Badge takes at most one BadgeOptions")
	}
	var o BadgeOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	label, zero, visible := formatBadgeText(text, o)
	if !visible || label == "" || (zero && !o.ShowZero) {
		return ui.Box(c).Size(0, 0)
	}

	th := CurrentTheme(c)
	cs := th.Colors
	container := o.ContainerColor
	if container.A == 0 {
		container = cs.Error
	}
	content := o.ContentColor
	if content.A == 0 {
		content = cs.OnError
	}
	if container.A == 0 {
		container = ui.RGB(0xB3, 0x26, 0x1E)
	}
	if content.A == 0 {
		content = ui.RGB(0xFF, 0xFF, 0xFF)
	}

	el := ui.Row(c).Height(largeBadgeHigh).MinWidth(largeBadgeHigh).Shrink(0).
		PaddingX(4).Radius(largeBadgeHigh / 2).
		Background(container).TextColor(content).
		Justify(ui.Center).AlignItems(ui.Center)
	applyTypeStyleWithTheme(th, el, tokens.LabelSmall)
	el.Children(func() {
		ui.Text(c, label).NoWrap()
	})
	if o.Label != "" {
		el.Label(o.Label)
	} else {
		el.Label(label)
	}
	return el
}

// SmallBadge builds the 6x6 DIP Error dot of Material Design 3, without text,
// for an unread marker. Give it a Label so assistive technology reads what it
// means:
//
//	md3e.SmallBadge(c, md3e.BadgeOptions{Label: "New message"})
func SmallBadge(c *ui.Context, opts ...BadgeOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: SmallBadge takes at most one BadgeOptions")
	}
	var o BadgeOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Visible != nil && !*o.Visible {
		return ui.Box(c).Size(0, 0)
	}
	th := CurrentTheme(c)
	container := o.ContainerColor
	if container.A == 0 {
		container = th.Colors.Error
	}
	if container.A == 0 {
		container = ui.RGB(0xB3, 0x26, 0x1E)
	}
	el := ui.Box(c).Size(smallBadgeSide, smallBadgeSide).Shrink(0).
		Radius(smallBadgeSide / 2).Background(container)
	if o.Label != "" {
		el.Label(o.Label)
	}
	return el
}

// formatBadgeText formats a badge's text, reporting whether a numeric text is
// zero and whether the badge shows at all.
func formatBadgeText(text string, o BadgeOptions) (label string, zero, visible bool) {
	visible = o.Visible == nil || *o.Visible
	label = strings.TrimSpace(text)
	limit := o.MaxCount
	if limit <= 0 {
		limit = badgeMaxCount
	}
	n, err := strconv.ParseInt(label, 10, 64)
	if err != nil {
		return label, false, visible
	}
	if n > int64(limit) {
		label = fmt.Sprintf("%d+", limit)
	}
	return label, n == 0, visible
}

// BadgedBoxOptions configures BadgedBox.
type BadgedBoxOptions struct {
	// Label describes the whole group for assistive technology; a clickable
	// content element already names it, and the badge names its own count.
	Label string
}

// BadgedBox anchors badge on the top-end corner of content: the top right in a
// left-to-right layout and the top left right to left, as Material Design 3
// places a badge on an icon. The badge keeps its own accessible name, and an
// optional Label describes the pair:
//
//	md3e.BadgedBox(c, func() { md3e.IconButton(c, mdicon.Mail.Draw) }, func() ui.Element {
//		return md3e.Badge(c, "3", md3e.BadgeOptions{Label: "3 unread"})
//	}, md3e.BadgedBoxOptions{Label: "Inbox"})
func BadgedBox(c *ui.Context, content func(), badge func() ui.Element, opts ...BadgedBoxOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: BadgedBox takes at most one BadgedBoxOptions")
	}
	var o BadgedBoxOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	row := ui.Row(c).AlignItems(ui.Center)
	if o.Label != "" {
		row.Role(ui.RoleGroup).Label(o.Label)
	}
	row.Children(func() {
		if content != nil {
			content()
		}
		if badge == nil {
			return
		}
		if el := badge(); el.Valid() {
			el.Attach(ui.AnchorTopEnd, ui.AnchorCenter)
		}
	})
	return row
}
