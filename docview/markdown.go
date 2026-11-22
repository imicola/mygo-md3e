package docview

import (
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/markdown/parser"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/codeview"
	"github.com/imicola/mygo-md3e/tokens"
)

// MarkdownThemeOptions customizes how MarkdownTheme and MarkdownThemeFor map
// an *md3e.Theme to *markdown.Theme.
type MarkdownThemeOptions struct {
	// FontSize overrides the density-resolved body text size in DIP when
	// positive and finite.
	FontSize float32
	// HeadingRules draws hairline bottom rules beneath H1 and H2 headings.
	HeadingRules bool
}

// MarkdownOptions configures docview.Markdown and docview.MarkdownStream.
type MarkdownOptions struct {
	// Theme optionally overrides individual fields of the MD3E-derived
	// markdown.Theme. Non-zero fields override the MD3E defaults.
	Theme *markdown.Theme
	// FontSize overrides the body text size in DIP when positive and finite.
	FontSize float32
	// HeadingRules draws hairline bottom rules beneath H1 and H2 headings.
	HeadingRules bool
	// Extensions selects the Markdown syntax extensions (defaults to
	// parser.Default when zero).
	Extensions parser.Extensions

	// Anchor scrolls to the named heading slug when the document opens, changes,
	// or AnchorVersion increments.
	Anchor string
	// AnchorVersion triggers a repeat scroll to Anchor when changed.
	AnchorVersion uint64
	// AnchorBlock resolves custom anchor names to block indices in d.
	AnchorBlock func(d *parser.Document, name string) int

	// State optionally links this Markdown view with docview.Outline for heading
	// extraction, layout offset measurement, active scroll-spy tracking, and
	// programmatic section jumps. Anchor also accepts the unique slugs from State.
	State *State
	// Scroll optionally supplies the enclosing ui.Scroll's ScrollState so
	// State.JumpTo and Outline track and smoothly scroll the viewport.
	Scroll *ui.ScrollState

	// Link & inline interaction hooks forwarded to ui/markdown:
	OnLink   func(url string)
	LinkMenu func(m *ui.Menu, url string)
	LinkIcon func(dest string, size float32) (markdown.InlineObject, bool)
	Preview  func(c *ui.Context, dest string) func(tip ui.Element)
	OnTask   func(line int, checked bool)
	Resolve  func(in parser.Inline) (dest string, ok bool)
	CodeLink func(code string) (dest string, ok bool)

	// Image & custom block hooks:
	LoadImage  func(url string) (ui.ImageSource, error)
	CodeBlock  func(c *ui.Context, b *markdown.CodeBlockView)
	Code       map[string]func(c *ui.Context, b *markdown.CodeBlockView)
	WrapCode   bool
	Math       func(c *ui.Context, tex string)
	InlineMath func(tex string, size float32) (markdown.InlineObject, bool)
	HTML       func(c *ui.Context, html string)
	Block      func(c *ui.Context, d *parser.Document, i int) bool

	// Streaming & selection options:
	Reveal         *markdown.Reveal
	ShareSelection bool

	// Snackbar optionally displays a confirmation toast when the user clicks
	// Copy on a fenced code block inside the Markdown document.
	Snackbar *md3e.SnackbarState
	// OnCopyCode runs when the user copies a fenced code block.
	OnCopyCode func(code, lang string)
}

type (
	mdBlockKey        struct{ i int }
	mdAlertInnerKey   struct{}
	mdHeadingInnerKey struct{}
	mdViewInnerKey    struct{}
	mdLocalStateKey   struct{}
)

type mdViewLocalState struct {
	opts               markdown.Options
	revealed           bool
	outlinePaintRootY  float32
	outlineBuildQueued bool
}

// MarkdownTheme derives a *markdown.Theme from md3e.CurrentTheme(c).
func MarkdownTheme(c *ui.Context, opts ...MarkdownThemeOptions) *markdown.Theme {
	return MarkdownThemeFor(md3e.CurrentTheme(c), opts...)
}

// MarkdownThemeFor maps an *md3e.Theme onto a complete *markdown.Theme using
// MD3E ColorScheme roles, TypographyScale, Density, Shapes, MonoFont, and
// codeview.ThemeFromMD3E syntax colors. Passing nil uses md3e.DefaultTheme(nil).
func MarkdownThemeFor(th *md3e.Theme, opts ...MarkdownThemeOptions) *markdown.Theme {
	if len(opts) > 1 {
		panic("docview: MarkdownThemeFor takes at most one MarkdownThemeOptions")
	}
	var o MarkdownThemeOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	cs := th.Colors
	scale := th.EffectiveTextScale()

	bodyLarge := th.TypeStyle(tokens.BodyLarge)
	bodyMedium := th.TypeStyle(tokens.BodyMedium)
	refSize := sanitizePositiveF32(float32(bodyLarge.Size), 16)

	var bodySize float32
	var lineHeight float32
	switch th.Density.Normalize() {
	case tokens.DensityCompact:
		bodySize = 13 * scale
		lineHeight = 1.42
	case tokens.DensityDesktop:
		bodySize = sanitizePositiveF32(float32(bodyMedium.Size), 14) * scale
		lineHeight = 1.46
	default:
		bodySize = refSize * scale
		lh := sanitizePositiveF32(float32(bodyLarge.LineHeight), 24)
		lineHeight = lh / refSize
	}
	if explicit := sanitizePositiveF32(o.FontSize, 0); explicit > 0 {
		bodySize = explicit
	}

	codeFont := th.MonoFont
	if codeFont == "" {
		codeFont = "monospace"
	}

	textCol := pickColor(cs.OnSurface, fallbackDocInk(th.Dark))
	mutedCol := pickColor(cs.OnSurfaceVariant, textCol.Alpha(0.72))
	linkCol := pickColor(cs.Primary, md3e.DefaultSeedColor)
	hairlineCol := resolveHairlineColor(th)
	ruleCol := pickColor(hairlineCol, cs.OutlineVariant, cs.Outline, mutedCol.Alpha(0.3))
	inlineCodeBg := textCol.Alpha(0.075)
	if th.Dark {
		inlineCodeBg = textCol.Alpha(0.11)
	}
	codeBlockBg := pickColor(cs.SurfaceContainerHigh, cs.SurfaceContainerHighest, inlineCodeBg)
	tableHeader := pickColor(cs.SurfaceContainerHigh, cs.SurfaceContainer, textCol.Alpha(0.05))
	tableStripe := pickColor(cs.SurfaceContainerLow, cs.Surface)
	markCol := pickColor(cs.TertiaryContainer, cs.SecondaryContainer, ui.RGBA(250, 204, 21, 0.32))
	quoteBar := linkCol.Alpha(0.55)
	if th.Contrast >= tokens.ContrastHigh {
		quoteBar = linkCol
	}

	hRoles := [6]tokens.TypeRole{
		tokens.HeadlineLarge,
		tokens.HeadlineMedium,
		tokens.HeadlineSmall,
		tokens.TitleLarge,
		tokens.TitleMedium,
		tokens.TitleSmall,
	}
	hTops := [6]float32{1.55, 1.42, 1.28, 1.16, 1.05, 0.98}
	hBots := [6]float32{0.58, 0.52, 0.46, 0.40, 0.36, 0.32}

	var headings [6]markdown.HeadingStyle
	for idx, role := range hRoles {
		ts := th.TypeStyle(role, true)
		em := float32(ts.Size) / refSize
		if em <= 0 {
			em = 1
		}
		weight := ts.Weight
		if idx < 4 && weight < 600 {
			weight = 600
		}
		col := textCol
		if idx == 5 {
			col = mutedCol
		}
		headings[idx] = markdown.HeadingStyle{
			Size:   em,
			Weight: weight,
			Color:  col,
			Top:    hTops[idx],
			Bottom: hBots[idx],
			Rule:   o.HeadingRules && idx < 2,
		}
	}

	strongWeight := th.TypeStyle(tokens.BodyLarge, true).Weight
	if strongWeight < 600 {
		strongWeight = 600
	}

	return &markdown.Theme{
		FontSize:            bodySize,
		Font:                th.Font,
		CodeFont:            codeFont,
		LineHeight:          lineHeight,
		CodeLineHeight:      1.44,
		Text:                textCol,
		Muted:               mutedCol,
		Link:                linkCol,
		Strong:              textCol,
		StrongWeight:        strongWeight,
		LinkWeight:          500,
		Code:                textCol,
		CodeBackground:      inlineCodeBg,
		CodeSize:            0.875,
		BlockSpacing:        0.75,
		ItemSpacing:         0.25,
		Headings:            headings,
		ListIndent:          1.6,
		QuoteBar:            quoteBar,
		QuoteText:           mutedCol,
		QuoteBarWidth:       3,
		Alerts:              [5]ui.Color{linkCol, pickColor(th.Success, cs.Tertiary, linkCol), pickColor(cs.Tertiary, linkCol), pickColor(th.Warning, cs.Secondary, linkCol), pickColor(cs.Error, linkCol)},
		Rule:                ruleCol,
		CodeBlockBackground: codeBlockBg,
		CodeBlockBorder:     hairlineCol,
		CodeBlockRadius:     resolveCodeRadius(th, tokens.ShapeMedium, 0),
		CodeBlockPadding:    0.9,
		CodeColors:          codeview.ThemeFromMD3E(th),
		TableBorder:         ruleCol,
		TableHeader:         tableHeader,
		TableStripe:         tableStripe,
		TableCellPadding:    [2]float32{0.65, 0.42},
		TableMaxColumn:      24,
		Mark:                markCol,
		Checkbox:            linkCol,
	}
}

// Markdown renders static or completed Markdown src styled with
// md3e.CurrentTheme(c), MD3E TypographyScale, SurfaceContainerHigh CodeBlock
// cards, and MD3E semantic Callout blocks for GitHub Alerts.
func Markdown(c *ui.Context, src string, opts ...MarkdownOptions) ui.Element {
	if len(opts) > 1 {
		panic("docview: Markdown takes at most one MarkdownOptions")
	}
	var o MarkdownOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	return buildMarkdown(c, src, o, false)
}

// MarkdownStream renders streaming Markdown src (for example an LLM response
// arriving token-by-token) over ui/markdown.Stream with smooth reveal pacing,
// incremental syntax healing, MD3E theming, and automatic zero-frame settling
// when ReduceMotion is enabled on either Theme or OS Preferences.
func MarkdownStream(c *ui.Context, src string, opts ...MarkdownOptions) ui.Element {
	if len(opts) > 1 {
		panic("docview: MarkdownStream takes at most one MarkdownOptions")
	}
	var o MarkdownOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	return buildMarkdown(c, src, o, true)
}

// Revealed reports whether el, returned by Markdown or MarkdownStream this
// frame, has revealed all of its source and completed fading in.
func Revealed(el ui.Element) bool {
	if !el.Valid() {
		return false
	}
	st := ui.Local(el, mdLocalStateKey{}, func() mdViewLocalState { return mdViewLocalState{} })
	return st.revealed
}

func buildMarkdown(c *ui.Context, src string, o MarkdownOptions, streaming bool) ui.Element {
	th := md3e.CurrentTheme(c)
	mdTh := MarkdownThemeFor(th, MarkdownThemeOptions{
		FontSize:     o.FontSize,
		HeadingRules: o.HeadingRules,
	})
	if o.Theme != nil {
		mdTh = mergeMarkdownTheme(mdTh, o.Theme)
	}

	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	reveal := o.Reveal
	if streaming && reduce {
		reveal = &markdown.Reveal{Unit: markdown.AtOnce, Fade: -1}
	}

	// Wrap in a stable root column so caller c.Key(...) attaches to root and
	// ui.Local preserves markdown.Options (including its internal image sync.Map)
	// across rebuilds without refetch churn.
	root := ui.Column(c).AlignSelf(ui.Stretch)
	local := ui.Local(root, mdLocalStateKey{}, func() mdViewLocalState { return mdViewLocalState{} })
	var trackHeading func(ui.Element, int)
	var previousRoot ui.Rect
	var activeChanged bool
	if state := o.State; state != nil {
		hadHeadings := len(state.items) > 0
		documentChanged := state.syncDocument(src, o.Extensions)
		if o.Scroll != nil {
			state.attachScroll(o.Scroll)
		}
		clear(state.headingTops)
		previousRoot = root.LayoutBounds()
		local.outlineBuildQueued = false
		srv := c.Services()
		requestLayoutBuild := func() {
			if !local.outlineBuildQueued {
				local.outlineBuildQueued = true
				srv.Invalidate()
			}
		}
		trackHeading = func(heading ui.Element, index int) {
			previous := heading.LayoutBounds()
			hasPrevious := previous.W > 0 && previous.H > 0 && previousRoot.W > 0 && previousRoot.H > 0
			if hasPrevious {
				state.recordHeadingTop(index, previous.Y)
			}
			// Compare document-relative positions so scrolling the whole view
			// does not cause a measurement rebuild.
			previousOffset := previous.Y - previousRoot.Y
			heading.DrawOver(func(_ *ui.Painter, r ui.Rect) {
				if heading.Valid() && r.W > 0 && r.H > 0 &&
					(!hasPrevious || math.Abs(float64(r.Y-local.outlinePaintRootY-previousOffset)) > 0.5) {
					requestLayoutBuild()
				}
			})
		}
		root.Draw(func(_ *ui.Painter, r ui.Rect) {
			if root.Valid() {
				local.outlinePaintRootY = r.Y
			}
		})
		hasHeadings := len(state.items) > 0
		root.DrawOver(func(_ *ui.Painter, r ui.Rect) {
			if !root.Valid() || r.W <= 0 || r.H <= 0 {
				return
			}
			outlineChanged := documentChanged && (hadHeadings || hasHeadings)
			sizeChanged := hasHeadings && (r.W != previousRoot.W || r.H != previousRoot.H)
			if outlineChanged || sizeChanged || activeChanged {
				requestLayoutBuild()
			}
		})
	}
	configureMarkdownOptions(&local.opts, th, mdTh, o, reveal, false, streaming, trackHeading)

	var inner ui.Element
	root.Children(func() {
		if streaming {
			inner = markdown.Stream(c.Key(mdViewInnerKey{}), src, &local.opts)
		} else {
			inner = markdown.View(c.Key(mdViewInnerKey{}), src, &local.opts)
		}
	})
	local.revealed = markdown.Revealed(inner)

	if o.State != nil {
		activeChanged = o.State.updateOffsets(previousRoot.Y)
	}
	return root
}

func configureMarkdownOptions(
	dst *markdown.Options,
	th *md3e.Theme,
	mdTh *markdown.Theme,
	o MarkdownOptions,
	reveal *markdown.Reveal,
	inner bool,
	streaming bool,
	trackHeading func(ui.Element, int),
) {
	anchor := o.Anchor
	anchorVer := o.AnchorVersion
	anchorBlock := o.AnchorBlock
	if state := o.State; !inner && state != nil {
		stateJump := state.anchorVersion > 0 && anchor == ""
		if stateJump {
			anchor = state.anchor
			anchorVer = state.anchorVersion
		}
		userAnchorBlock := o.AnchorBlock
		anchorBlock = func(d *parser.Document, name string) int {
			if stateJump && strings.EqualFold(name, anchor) {
				if index := state.outlineBlock(d, name); index > 0 {
					return index
				}
			}
			if userAnchorBlock != nil {
				if index := userAnchorBlock(d, name); index > 0 && index < d.Len() {
					return index
				}
			}
			return state.outlineBlock(d, name)
		}
	}

	dst.Theme = mdTh
	dst.Extensions = o.Extensions
	dst.LinkMenu = o.LinkMenu
	dst.OnLink = o.OnLink
	dst.LinkIcon = o.LinkIcon
	dst.Anchor = anchor
	dst.AnchorVersion = anchorVer
	dst.AnchorBlock = anchorBlock
	dst.OnTask = o.OnTask
	dst.Resolve = o.Resolve
	dst.CodeLink = o.CodeLink
	dst.Preview = o.Preview
	dst.LoadImage = o.LoadImage
	dst.Code = o.Code
	dst.WrapCode = o.WrapCode
	dst.Math = o.Math
	dst.InlineMath = o.InlineMath
	dst.HTML = o.HTML
	dst.Reveal = reveal
	dst.ShareSelection = o.ShareSelection || inner

	if o.CodeBlock != nil {
		dst.CodeBlock = o.CodeBlock
	} else {
		dst.CodeBlock = func(c *ui.Context, b *markdown.CodeBlockView) {
			buildCodeBlockCard(c.Key(mdBlockKey{b.Index}), th, codeCardOptions{
				language:     b.Language,
				codeToCopy:   b.Code(),
				showCopy:     true,
				copyDisabled: b.Open,
				snackbar:     o.Snackbar,
				onCopy: func(code string) {
					if o.OnCopyCode != nil {
						o.OnCopyCode(code, b.Language)
					}
				},
				hairline: true,
			}, func() {
				b.Body(c)
			})
		}
	}

	if inner {
		return
	}

	userBlock := o.Block
	dst.Block = func(c *ui.Context, d *parser.Document, i int) bool {
		if userBlock != nil && userBlock(c, d, i) {
			return true
		}
		b := d.Block(i)
		if trackHeading != nil && b.Kind == parser.Heading && (!streaming || !b.Open) {
			if b.Start >= 0 && b.End > b.Start && b.End <= len(d.Source()) {
				hSrc := d.Source()[b.Start:b.End]
				var innerOpts markdown.Options
				configureMarkdownOptions(&innerOpts, th, mdTh, o, &markdown.Reveal{Unit: markdown.AtOnce, Fade: -1}, true, false, nil)
				innerOpts.Anchor = ""
				innerOpts.AnchorVersion = 0
				innerOpts.AnchorBlock = nil

				hBox := ui.Column(c.Key(mdBlockKey{i})).AlignSelf(ui.Stretch)
				trackHeading(hBox, i)
				hBox.Children(func() {
					markdown.View(c.Key(mdHeadingInnerKey{}), hSrc, &innerOpts)
				})
				return true
			}
		}
		if b.Kind == parser.Quote && b.Alert != parser.NoAlert {
			kind, ok := CalloutKindFromAlert(b.Alert)
			if !ok {
				return false
			}
			innerSrc := extractAlertBody(d, i)
			pal := calloutPaletteFor(th, kind)
			innerTh := *mdTh
			innerTh.Text = pal.onContainer
			innerTh.Strong = pal.onContainer
			innerTh.Muted = pal.onContainer.Alpha(0.78)
			innerTh.Link = pal.headerInk
			innerTh.Code = pal.onContainer
			innerTh.CodeBackground = pal.onContainer.Alpha(0.11)
			innerTh.Rule = pal.accent.Alpha(0.32)
			innerTh.QuoteBar = pal.accent
			innerTh.QuoteText = pal.onContainer.Alpha(0.85)
			innerTh.BlockSpacing = 0.5

			var innerOpts markdown.Options
			configureMarkdownOptions(&innerOpts, th, &innerTh, o, &markdown.Reveal{Unit: markdown.AtOnce, Fade: -1}, true, false, nil)
			innerOpts.Anchor = ""
			innerOpts.AnchorVersion = 0
			innerOpts.AnchorBlock = nil

			Callout(c.Key(mdBlockKey{i}), kind, func() {
				if innerSrc != "" {
					markdown.View(c.Key(mdAlertInnerKey{}), innerSrc, &innerOpts)
				}
			})
			return true
		}
		return false
	}
}

func extractAlertBody(d *parser.Document, quoteIdx int) string {
	b := d.Block(quoteIdx)
	if b.Start < 0 || b.End <= b.Start || b.End > len(d.Source()) {
		return ""
	}
	raw := d.Source()[b.Start:b.End]
	lines := strings.Split(raw, "\n")
	if len(lines) <= 1 {
		return ""
	}
	bodyLines := make([]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		line = strings.TrimSuffix(line, "\r")
		bodyLines = append(bodyLines, stripBlockquotePrefix(line))
	}
	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
		bodyLines = bodyLines[1:]
	}
	return strings.Join(bodyLines, "\n")
}

func stripBlockquotePrefix(line string) string {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i < len(line) && line[i] == '>' {
		i++
		if i < len(line) && line[i] == ' ' {
			i++
		}
		return line[i:]
	}
	return line
}

func mergeMarkdownTheme(base, override *markdown.Theme) *markdown.Theme {
	out := *base
	if override == nil {
		return &out
	}
	f := func(dst *float32, src float32) {
		if src > 0 {
			*dst = src
		}
	}
	col := func(dst *ui.Color, src ui.Color) {
		if src.A > 0 {
			*dst = src
		}
	}
	str := func(dst *string, src string) {
		if src != "" {
			*dst = src
		}
	}
	f(&out.FontSize, override.FontSize)
	str(&out.Font, override.Font)
	str(&out.CodeFont, override.CodeFont)
	f(&out.LineHeight, override.LineHeight)
	f(&out.CodeLineHeight, override.CodeLineHeight)
	col(&out.Text, override.Text)
	col(&out.Muted, override.Muted)
	col(&out.Link, override.Link)
	col(&out.Strong, override.Strong)
	if override.StrongWeight > 0 {
		out.StrongWeight = override.StrongWeight
	}
	if override.LinkWeight > 0 {
		out.LinkWeight = override.LinkWeight
	}
	col(&out.Code, override.Code)
	col(&out.CodeBackground, override.CodeBackground)
	f(&out.CodeSize, override.CodeSize)
	f(&out.BlockSpacing, override.BlockSpacing)
	f(&out.ItemSpacing, override.ItemSpacing)
	for i := range out.Headings {
		oh := override.Headings[i]
		f(&out.Headings[i].Size, oh.Size)
		if oh.Weight > 0 {
			out.Headings[i].Weight = oh.Weight
		}
		col(&out.Headings[i].Color, oh.Color)
		f(&out.Headings[i].Top, oh.Top)
		f(&out.Headings[i].Bottom, oh.Bottom)
		if oh.Rule {
			out.Headings[i].Rule = true
		}
	}
	f(&out.ListIndent, override.ListIndent)
	col(&out.QuoteBar, override.QuoteBar)
	col(&out.QuoteText, override.QuoteText)
	f(&out.QuoteBarWidth, override.QuoteBarWidth)
	for i := range out.Alerts {
		col(&out.Alerts[i], override.Alerts[i])
	}
	col(&out.Rule, override.Rule)
	col(&out.CodeBlockBackground, override.CodeBlockBackground)
	col(&out.CodeBlockBorder, override.CodeBlockBorder)
	f(&out.CodeBlockRadius, override.CodeBlockRadius)
	f(&out.CodeBlockPadding, override.CodeBlockPadding)
	if override.CodeColors != nil {
		out.CodeColors = override.CodeColors
	}
	col(&out.TableBorder, override.TableBorder)
	col(&out.TableHeader, override.TableHeader)
	col(&out.TableStripe, override.TableStripe)
	f(&out.TableCellPadding[0], override.TableCellPadding[0])
	f(&out.TableCellPadding[1], override.TableCellPadding[1])
	f(&out.TableMaxColumn, override.TableMaxColumn)
	col(&out.Mark, override.Mark)
	col(&out.Checkbox, override.Checkbox)
	return &out
}

func fallbackDocInk(dark bool) ui.Color {
	if dark {
		return ui.Hex("#e6e0e9")
	}
	return ui.Hex("#1d1b20")
}
