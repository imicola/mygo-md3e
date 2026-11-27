package docview

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/codeview"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

const codeBlockCopyDuration = 2 * time.Second

// CodeBlockOptions configures docview.CodeBlock.
type CodeBlockOptions struct {
	// Language identifies the syntax highlighting language (e.g. "go", "json",
	// "bash") and appears as the header language badge when non-empty.
	Language string
	// Title displays an optional filename or caption in the card header.
	Title string
	// ShowLineNumbers renders an unselectable TabularNums line-number gutter.
	ShowLineNumbers bool
	// Wrap wraps long lines instead of scrolling horizontally.
	Wrap bool
	// ShowCopy controls whether the header copy button is shown (nil defaults to true).
	ShowCopy *bool
	// CopyLabel overrides the copy button text ("Copy" when empty).
	CopyLabel string
	// CopiedLabel overrides the temporary feedback text ("Copied" when empty).
	CopiedLabel string
	// CopiedMessage overrides the Snackbar message when Snackbar is non-nil.
	CopiedMessage string
	// Snackbar optionally displays an MD3E Snackbar confirmation when code is copied.
	Snackbar *md3e.SnackbarState
	// OnCopy runs after the user activates the copy button.
	OnCopy func(code string)
	// Disabled disables the copy button (for example while a fenced block is
	// still streaming open).
	Disabled bool
	// ShapeScale selects the MD3E corner token (default tokens.ShapeMedium).
	ShapeScale tokens.ShapeScale
	// Radius overrides the corner radius in DIP when positive and finite.
	Radius float32
	// Hairline draws a subtle 1 DIP tonal highlight border (1.5 DIP in
	// HighContrast). Nil defaults to false.
	Hairline *bool
	// CodeColors overrides the syntax highlighting palette (defaults to
	// codeview.ThemeFromMD3E(th)).
	CodeColors *editor.Theme
	// Label overrides the accessible group label (defaults to "Code" or
	// "Code, <Language>").
	Label string
}

type (
	codeBlockHeaderKey    struct{}
	codeBlockCopyBtnKey   struct{}
	codeBlockCopyStateKey struct{}
	codeBlockScrollKey    struct{}
	codeBlockTokenKey     struct{}
)

type codeBlockCopyState struct {
	copiedAt time.Time
}

type tokenizedCodeCache struct {
	code   string
	lang   string
	theme  *editor.Theme
	lines  [][]ui.Span
	single bool
}

type codeCardOptions struct {
	language      string
	title         string
	codeToCopy    string
	showCopy      bool
	copyDisabled  bool
	copyLabel     string
	copiedLabel   string
	copiedMessage string
	snackbar      *md3e.SnackbarState
	onCopy        func(string)
	shapeScale    tokens.ShapeScale
	radius        float32
	hairline      bool
	label         string
}

// CodeBlock builds an MD3E SurfaceContainer code card with a language badge,
// optional filename title, syntax highlighting via ui/editor's language
// tokenizers, optional TabularNums line numbers, and a one-click copy button
// with Check feedback and optional Snackbar notification.
func CodeBlock(c *ui.Context, code string, opts ...CodeBlockOptions) ui.Element {
	if len(opts) > 1 {
		panic("docview: CodeBlock takes at most one CodeBlockOptions")
	}
	var o CodeBlockOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := md3e.CurrentTheme(c)
	cs := th.Colors
	m := docCodeMetricsFor(th)

	ct := o.CodeColors
	if ct == nil {
		ct = codeview.ThemeFromMD3E(th)
	}
	monoFont := th.MonoFont
	if monoFont == "" {
		monoFont = "monospace"
	}

	cardOpts := codeCardOptions{
		language:      strings.TrimSpace(o.Language),
		title:         strings.TrimSpace(o.Title),
		codeToCopy:    code,
		showCopy:      boolDefault(o.ShowCopy, true),
		copyDisabled:  o.Disabled,
		copyLabel:     o.CopyLabel,
		copiedLabel:   o.CopiedLabel,
		copiedMessage: o.CopiedMessage,
		snackbar:      o.Snackbar,
		onCopy:        o.OnCopy,
		shapeScale:    o.ShapeScale,
		radius:        o.Radius,
		hairline:      boolDefault(o.Hairline, false),
		label:         o.Label,
	}

	return buildCodeBlockCard(c, th, cardOpts, func() {
		var outer ui.Element
		if o.Wrap {
			outer = ui.Column(c.Key(codeBlockScrollKey{}))
		} else {
			outer = ui.ScrollHorizontal(c.Key(codeBlockScrollKey{}))
		}
		// Source code always flows left-to-right even inside RTL interfaces.
		outer.Direction(ui.LeftToRight).
			AlignSelf(ui.Stretch).
			Font(monoFont).
			FontSize(m.fontSize).
			LineHeight(m.lineHeight)

		cache := ui.Local(outer, codeBlockTokenKey{}, func() tokenizedCodeCache {
			return tokenizedCodeCache{}
		})
		lines := cache.resolve(code, cardOpts.language, ct)
		lineNumWidth := lineNumberGutterWidth(c, len(lines), monoFont, m.lineNumSize)

		outer.Children(func() {
			body := ui.Column(c).
				Padding(m.bodyPad).
				Gap(0).
				Selectable()
			if !o.Wrap {
				body.Shrink(0)
			} else {
				body.AlignSelf(ui.Stretch)
			}

			if o.ShowLineNumbers {
				gutterW := m.bodyPad + lineNumWidth + m.lineNumGap*0.5
				gutterWash := cs.OnSurface.Alpha(0.03)
				if th.Dark {
					gutterWash = cs.OnSurface.Alpha(0.04)
				}
				gutterRule := resolveHairlineColor(th)
				body.Draw(func(p *ui.Painter, r ui.Rect) {
					if r.W <= gutterW || r.H <= 0 {
						return
					}
					p.Fill(ui.Rect{X: r.X, Y: r.Y, W: gutterW, H: r.H}, gutterWash, 0)
					if gutterRule.A > 0 {
						p.Fill(ui.Rect{X: r.X + gutterW - 0.5, Y: r.Y, W: 1, H: r.H}, gutterRule, 0)
					}
				})
			}

			body.Children(func() {
				if o.ShowLineNumbers {
					gutterInk := pickColor(ct.LineNumber, cs.OnSurfaceVariant.Alpha(0.65), ct.Text.Alpha(0.65))
					for i, spans := range lines {
						row := ui.Row(c).AlignItems(ui.Baseline)
						if !o.Wrap {
							row.Shrink(0)
						} else {
							row.AlignSelf(ui.Stretch)
						}
						row.Children(func() {
							num := ui.Text(c, strconv.Itoa(i+1)).
								Width(lineNumWidth).
								Shrink(0).
								Margin(0, m.lineNumGap, 0, 0).
								TextAlign(ui.End).
								TextColor(gutterInk).
								FontSize(m.lineNumSize).
								Unselectable()
							md3e.TabularNums(num)

							txt := ui.RichText(c, spans...)
							if !o.Wrap {
								txt.NoWrap()
							} else {
								txt.Grow(1).Shrink(1).MinWidth(0)
							}
						})
					}
					return
				}

				if len(lines) == 0 {
					ui.Text(c, " ").TextColor(ct.Text)
					return
				}
				flat := make([]ui.Span, 0, len(lines)*3)
				for i, spans := range lines {
					if i > 0 {
						flat = append(flat, ui.Span{Text: "\n"})
					}
					flat = append(flat, spans...)
				}
				txt := ui.RichText(c, flat...)
				if !o.Wrap {
					txt.NoWrap()
				}
			})
		})
	})
}

func buildCodeBlockCard(c *ui.Context, th *md3e.Theme, o codeCardOptions, bodyFn func()) ui.Element {
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	cs := th.Colors
	m := docCodeMetricsFor(th)

	bg := pickColor(cs.SurfaceContainer, md3e.SurfaceColorForElevation(th, tokens.Level2))
	headerBg := pickColor(cs.SurfaceContainerHigh, bg)
	if !th.Dark && cs.OnSurface.A > 0 {
		headerBg = bg.MixOklab(cs.OnSurface, 0.045)
	} else if headerBg == bg && cs.OnSurface.A > 0 {
		headerBg = bg.MixOklab(cs.OnSurface, 0.04)
	}
	ink := pickColor(cs.OnSurface, th.ContentColorFor(bg))
	muted := pickColor(cs.OnSurfaceVariant, ink.Alpha(0.72))
	hairlineCol := resolveHairlineColor(th)
	highContrast := th.Contrast >= tokens.ContrastHigh
	radius := resolveCodeRadius(th, o.shapeScale, o.radius)

	label := o.label
	if label == "" {
		switch {
		case o.title != "" && o.language != "":
			label = "Code, " + o.title + " (" + o.language + ")"
		case o.title != "":
			label = "Code, " + o.title
		case o.language != "":
			label = "Code, " + o.language
		default:
			label = "Code"
		}
	}

	card := ui.Column(c).
		Background(bg).
		TextColor(ink).
		Radius(radius).
		AlignSelf(ui.Stretch).
		Clip().
		DimDisabled(false).
		Role(ui.RoleGroup).
		Label(label)
	if o.hairline && hairlineCol.A > 0 {
		bw := float32(1)
		if highContrast {
			bw = 1.5
		}
		card.Border(bw, hairlineCol)
	} else if highContrast {
		hcCol := pickColor(cs.Outline, cs.OnSurface)
		if hcCol.A > 0 {
			card.Border(1.5, hcCol)
		}
	}

	showHeader := o.language != "" || o.title != "" || o.showCopy
	card.Children(func() {
		if showHeader {
			head := ui.Row(c.Key(codeBlockHeaderKey{})).
				AlignItems(ui.Center).
				AlignSelf(ui.Stretch).
				Padding(m.headerPadY, 0, m.headerPadY, 0).
				PaddingStart(m.headerPadX).
				PaddingEnd(m.headerPadX).
				Gap(8).
				Background(headerBg)
			if o.hairline && hairlineCol.A > 0 {
				bw := float32(1)
				if highContrast {
					bw = 1.5
				}
				head.BorderWidth(0, 0, bw, 0).BorderColor(hairlineCol)
			} else if highContrast {
				if hcCol := pickColor(cs.Outline, cs.OnSurface); hcCol.A > 0 {
					head.BorderWidth(0, 0, 1.5, 0).BorderColor(hcCol)
				}
			}
			head.Children(func() {
				if o.language != "" {
					badgeBg := pickColor(cs.SecondaryContainer, cs.SurfaceContainerHigh, bg)
					badgeInk := pickColor(cs.OnSecondaryContainer, muted)
					pill := ui.Row(c).
						AlignItems(ui.Center).
						Padding(2, 8).
						Radius(th.Shapes.Radius(tokens.ShapeSmall)).
						Background(badgeBg)
					pill.Children(func() {
						txt := md3e.TextWith(c, o.language, md3e.TextOptions{
							Role:  tokens.LabelSmall,
							Mono:  true,
							Color: badgeInk,
						}).Unselectable()
						_ = txt
					})
				}
				if o.title != "" {
					md3e.TextWith(c, o.title, md3e.TextOptions{
						Role:     tokens.LabelMedium,
						MaxLines: 1,
						Color:    muted,
					}).Shrink(1).MinWidth(0).Unselectable()
				}
				ui.Spacer(c)
				if o.showCopy {
					buildCodeCopyButton(c, th, m, o)
				}
			})
		}
		if bodyFn != nil {
			bodyFn()
		}
	})
	return card
}

func buildCodeCopyButton(c *ui.Context, th *md3e.Theme, m docCodeMetrics, o codeCardOptions) {
	cs := th.Colors
	copyText := o.copyLabel
	if copyText == "" {
		copyText = "Copy"
	}
	copiedText := o.copiedLabel
	if copiedText == "" {
		copiedText = "Copied"
	}

	btn := ui.ButtonBase(c.Key(codeBlockCopyBtnKey{})).
		DimDisabled(false).
		FocusRing(false).
		Height(m.copyHeight).
		Padding(0, 10).
		Radius(m.copyHeight / 2).
		Disabled(o.copyDisabled).
		Role(ui.RoleButton).
		Unselectable()

	st := ui.Local(btn, codeBlockCopyStateKey{}, func() codeBlockCopyState {
		return codeBlockCopyState{}
	})
	now := c.Now()
	copied := !st.copiedAt.IsZero() && now.Sub(st.copiedAt) < codeBlockCopyDuration
	if !st.copiedAt.IsZero() && !copied {
		st.copiedAt = time.Time{}
	}
	ink := pickColor(cs.OnSurfaceVariant, cs.OnSurface)
	if copied {
		succBg := pickColor(th.SuccessContainer, th.Success.Alpha(0.16), cs.PrimaryContainer)
		ink = pickColor(th.OnSuccessContainer, th.Success, cs.Primary, ink)
		if succBg.A > 0 && !o.copyDisabled {
			btn.Background(succBg)
		}
	}
	if o.copyDisabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}

	accessName := "Copy code"
	if o.language != "" {
		accessName = "Copy " + o.language + " code"
	}
	if copied {
		accessName = copiedText
	}
	btn.Label(accessName)

	md3e.AttachRipple(c, btn, md3e.StateLayerOptions{
		Color:    ink,
		Radius:   m.copyHeight / 2,
		Disabled: o.copyDisabled,
	})

	srv := c.Services()
	btn.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !btn.Valid() || st.copiedAt.IsZero() {
			return
		}
		rem := codeBlockCopyDuration - p.Now().Sub(st.copiedAt)
		if rem <= 0 {
			st.copiedAt = time.Time{}
			srv.Invalidate()
			return
		}
		vis := p.Visible()
		if min(r.X+r.W, vis.X+vis.W) > max(r.X, vis.X) &&
			min(r.Y+r.H, vis.Y+vis.H) > max(r.Y, vis.Y) {
			p.After(rem)
		}
	})

	btn.Row().
		AlignItems(ui.Center).
		Gap(5).
		Children(func() {
			icon := symbols.ContentCopy.Draw
			txt := copyText
			if copied {
				icon = symbols.Check.Draw
				txt = copiedText
			}
			docIconBox(c, icon, 14, ink)
			md3e.TextWith(c, txt, md3e.TextOptions{
				Role:  tokens.LabelSmall,
				Color: ink,
			}).Unselectable()
		})

	if btn.Clicked() && !o.copyDisabled {
		c.WriteClipboard(o.codeToCopy)
		st.copiedAt = now
		c.Announce(copiedText)
		if o.snackbar != nil {
			msg := o.copiedMessage
			if msg == "" {
				if o.language != "" {
					msg = "Copied " + o.language + " code"
				} else {
					msg = "Copied to clipboard"
				}
			}
			o.snackbar.Show(md3e.SnackbarOptions{Message: msg})
		}
		if o.onCopy != nil {
			o.onCopy(o.codeToCopy)
		}
		c.Invalidate()
	}
}

func (cache *tokenizedCodeCache) resolve(code, lang string, ct *editor.Theme) [][]ui.Span {
	if cache.lines != nil && cache.code == code && cache.lang == lang && sameCodeTheme(cache.theme, ct) {
		return cache.lines
	}
	cache.code = code
	cache.lang = lang
	cache.theme = ct
	cache.lines = tokenizeCodeLines(code, lang, ct)
	return cache.lines
}

func sameCodeTheme(a, b *editor.Theme) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func tokenizeCodeLines(code, lang string, ct *editor.Theme) [][]ui.Span {
	trimmed := strings.TrimSuffix(code, "\n")
	trimmed = strings.TrimSuffix(trimmed, "\r")
	if trimmed == "" {
		return nil
	}
	rawLines := strings.Split(trimmed, "\n")

	var tok editor.Tokenizer
	if lang != "" {
		if l := editor.LanguageNamed(lang); l != nil {
			tok = l.Tokenizer
		}
	}

	out := make([][]ui.Span, len(rawLines))
	var state uint64
	var toks []editor.Token
	for i, line := range rawLines {
		line = strings.TrimSuffix(line, "\r")
		if tok == nil || line == "" {
			if line == "" {
				out[i] = []ui.Span{{Text: " ", Color: ct.Text}}
			} else {
				out[i] = []ui.Span{{Text: line, Color: ct.Text}}
			}
			continue
		}
		toks, state = tok.Tokenize(line, state, toks[:0])
		spans := make([]ui.Span, 0, len(toks)+1)
		start := 0
		for _, t := range toks {
			end := min(t.End, len(line))
			if end > start {
				sp := ui.Span{Text: line[start:end], Color: ct.TokenColor(t.Kind)}
				if style := ct.Tokens[t.Kind]; style.Bold {
					sp.Weight = 600
				}
				if style := ct.Tokens[t.Kind]; style.Italic {
					sp.Italic = true
				}
				spans = append(spans, sp)
			}
			start = max(start, end)
		}
		if start < len(line) {
			spans = append(spans, ui.Span{Text: line[start:], Color: ct.Text})
		}
		if len(spans) == 0 {
			spans = append(spans, ui.Span{Text: " ", Color: ct.Text})
		}
		out[i] = spans
	}
	return out
}

func lineNumberGutterWidth(c *ui.Context, lineCount int, font string, size float32) float32 {
	digits := max(2, len(strconv.Itoa(max(1, lineCount))))
	sample := strings.Repeat("8", digits)
	w, _ := c.MeasureText(0, ui.Span{Text: sample, Font: font, Size: size, Features: "tnum"})
	if w <= 0 {
		return float32(digits) * size * 0.65
	}
	return w + 2
}
