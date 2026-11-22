package codeview

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

// ThemeFromMD3E maps an MD3E Theme onto a complete *editor.Theme for
// github.com/egoist/mygo/ui/editor (both View and DiffView) and for
// ui/markdown's CodeColors.
//
// Structural surfaces, selection highlights, cursor, guides, diagnostics, diff
// indicators, and syntax token colors adapt dynamically to th's seed color
// scheme, Dark mode, Monochrome variant, and Contrast tier while enforcing
// WCAG contrast against both SurfaceContainerLowest and SurfaceContainerHigh.
// Passing nil falls back to md3e.DefaultTheme(nil).
func ThemeFromMD3E(th *md3e.Theme) *editor.Theme {
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	cs := th.Colors
	highContrast := th.Contrast >= tokens.ContrastHigh
	mediumContrast := th.Contrast >= tokens.ContrastMedium

	bg := pickColor(cs.SurfaceContainerLowest, cs.Surface, defaultCanvas(th.Dark))
	codeSurface := pickColor(cs.SurfaceContainerHigh, bg)
	text := pickColor(cs.OnSurface, fallbackInk(th.Dark))
	muted := pickColor(cs.OnSurfaceVariant, text.Alpha(0.72))
	primary := pickColor(cs.Primary, md3e.DefaultSeedColor)
	secondary := pickColor(cs.Secondary, primary)
	tertiary := pickColor(cs.Tertiary, primary)
	errCol := pickColor(cs.Error, defaultError(th.Dark))
	warnCol := pickColor(th.Warning, defaultWarning(th.Dark))
	succCol := pickColor(th.Success, defaultSuccess(th.Dark))
	outline := pickColor(cs.OutlineVariant, cs.Outline, muted.Alpha(0.35))

	lineNumAlpha := float32(0.68)
	guideAlpha := float32(0.55)
	wsAlpha := float32(0.42)
	activeLineAlpha := float32(0.06)
	if th.Dark {
		activeLineAlpha = 0.09
	}
	if highContrast {
		lineNumAlpha = 0.90
		guideAlpha = 0.78
		wsAlpha = 0.65
		activeLineAlpha = 0.12
	}

	selection := primary.Alpha(0.24)
	if cs.PrimaryContainer.A > 0 {
		selection = cs.PrimaryContainer.Alpha(0.78)
		if highContrast {
			selection = cs.PrimaryContainer
		}
	}
	inactiveSel := selection.Alpha(0.55)
	if cs.SecondaryContainer.A > 0 {
		inactiveSel = cs.SecondaryContainer.Alpha(0.55)
	}
	matchCol := tertiary.Alpha(0.26)
	if cs.TertiaryContainer.A > 0 {
		matchCol = cs.TertiaryContainer.Alpha(0.68)
	}

	ct := &editor.Theme{
		Background:        bg,
		Text:              text,
		LineNumber:        muted.Alpha(lineNumAlpha),
		ActiveLineNumber:  text,
		ActiveLine:        primary.Alpha(activeLineAlpha),
		Selection:         selection,
		InactiveSelection: inactiveSel,
		Cursor:            primary,
		MatchingBracket:   primary.Alpha(0.45),
		Match:             matchCol,
		CurrentMatch:      warnCol.Alpha(0.48),
		Occurrence:        secondary.Alpha(0.16),
		IndentGuide:       outline.Alpha(guideAlpha),
		ActiveIndentGuide: muted.Alpha(minF32(1, guideAlpha+0.15)),
		WrapGuide:         outline.Alpha(guideAlpha),
		Whitespace:        muted.Alpha(wsAlpha),
		Error:             errCol,
		Warning:           warnCol,
		Info:              primary,
		Hint:              muted,
		Added:             succCol,
		Deleted:           errCol,
		Modified:          primary,
	}

	minBodyRatio := 4.5
	minCommentRatio := 3.0
	if highContrast {
		minBodyRatio = 7.0
		minCommentRatio = 4.5
	} else if mediumContrast {
		minBodyRatio = 5.5
		minCommentRatio = 3.5
	}

	primaryHCT := tokens.HCTFromColor(primary)
	monochrome := primaryHCT.Chroma < 4.0

	if monochrome {
		applyMonochromeTokens(ct, th.Dark, bg, codeSurface, minBodyRatio, minCommentRatio)
		return ct
	}

	pHue := primaryHCT.Hue
	tHue := tokens.HCTFromColor(tertiary).Hue
	if hueDelta(pHue, tHue) < 24 {
		tHue = tokens.SanitizeDegrees(pHue + 60)
	}
	baseTone := 40.0
	baseChroma := 52.0
	if th.Dark {
		baseTone = 80.0
		baseChroma = 46.0
		if highContrast {
			baseTone = 88.0
			baseChroma = 54.0
		}
	} else if highContrast {
		baseTone = 30.0
		baseChroma = 58.0
	}

	fit := func(c ui.Color, ratio float64) ui.Color {
		return ensureDualContrast(c, bg, codeSurface, th.Dark, ratio)
	}
	fromHue := func(hue, chromaScale, toneShift float64) ui.Color {
		c := tokens.HCT{
			Hue:    tokens.SanitizeDegrees(hue),
			Chroma: baseChroma * chromaScale,
			Tone:   clampF64(baseTone+toneShift, 8, 96),
		}.ToColor()
		return fit(c, minBodyRatio)
	}

	keywordCol := fit(primary, minBodyRatio)
	typeCol := fromHue(28, 1.05, 0)         // warm amber/terracotta
	fnCol := fromHue(tHue, 1.0, 2)          // theme tertiary accent
	propCol := fromHue(225, 0.95, 2)        // crisp azure
	strCol := fit(succCol, minBodyRatio)    // harmonized emerald
	numCol := fromHue(255, 0.90, 0)         // indigo-blue
	metaCol := fromHue(295, 0.92, 0)        // violet
	commentCol := fit(muted, minCommentRatio)
	opCol := fit(secondary, minBodyRatio)
	invalidCol := fit(errCol, minBodyRatio)

	ct.Tokens[editor.TokenText] = editor.TokenStyle{Color: text}
	ct.Tokens[editor.TokenKeyword] = editor.TokenStyle{Color: keywordCol}
	ct.Tokens[editor.TokenOperator] = editor.TokenStyle{Color: opCol}
	ct.Tokens[editor.TokenPunctuation] = editor.TokenStyle{Color: muted}
	ct.Tokens[editor.TokenType] = editor.TokenStyle{Color: typeCol}
	ct.Tokens[editor.TokenFunction] = editor.TokenStyle{Color: fnCol}
	ct.Tokens[editor.TokenMeta] = editor.TokenStyle{Color: metaCol}
	ct.Tokens[editor.TokenVariable] = editor.TokenStyle{Color: text}
	ct.Tokens[editor.TokenProperty] = editor.TokenStyle{Color: propCol}
	ct.Tokens[editor.TokenConstant] = editor.TokenStyle{Color: numCol}
	ct.Tokens[editor.TokenNumber] = editor.TokenStyle{Color: numCol}
	ct.Tokens[editor.TokenAttribute] = editor.TokenStyle{Color: typeCol}
	ct.Tokens[editor.TokenString] = editor.TokenStyle{Color: strCol}
	ct.Tokens[editor.TokenEscape] = editor.TokenStyle{Color: fnCol}
	ct.Tokens[editor.TokenRegexp] = editor.TokenStyle{Color: strCol}
	ct.Tokens[editor.TokenTag] = editor.TokenStyle{Color: keywordCol}
	ct.Tokens[editor.TokenComment] = editor.TokenStyle{Color: commentCol, Italic: true}
	ct.Tokens[editor.TokenHeading] = editor.TokenStyle{Color: keywordCol, Bold: true}
	ct.Tokens[editor.TokenEmphasis] = editor.TokenStyle{Color: text, Italic: true}
	ct.Tokens[editor.TokenStrong] = editor.TokenStyle{Color: text, Bold: true}
	ct.Tokens[editor.TokenLink] = editor.TokenStyle{Color: propCol}
	ct.Tokens[editor.TokenCode] = editor.TokenStyle{Color: fnCol}
	ct.Tokens[editor.TokenInserted] = editor.TokenStyle{Color: strCol}
	ct.Tokens[editor.TokenDeleted] = editor.TokenStyle{Color: invalidCol}
	ct.Tokens[editor.TokenInvalid] = editor.TokenStyle{Color: invalidCol}

	return ct
}

// FontFromMD3E resolves the monospace ui.Font and line-height multiplier for
// ui/editor based on th.MonoFont, th.Density, and th.EffectiveTextScale.
// Passing nil uses md3e.DefaultTheme(nil).
func FontFromMD3E(th *md3e.Theme) (ui.Font, float32) {
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	family := th.MonoFont
	if family == "" {
		family = "monospace"
	}
	baseSize := float32(14)
	lineHeight := float32(1.5)
	switch th.Density.Normalize() {
	case tokens.DensityDesktop:
		baseSize = 13
		lineHeight = 1.45
	case tokens.DensityCompact:
		baseSize = 12
		lineHeight = 1.4
	}
	return ui.Font{
		Family: family,
		Size:   baseSize * th.EffectiveTextScale(),
	}, lineHeight
}

// ScrollbarFromMD3E returns an MD3E-styled ui.ScrollbarStyle for ui/editor.
// Passing nil uses md3e.DefaultTheme(nil).
func ScrollbarFromMD3E(th *md3e.Theme) ui.ScrollbarStyle {
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	cs := th.Colors
	ink := pickColor(cs.OnSurfaceVariant, cs.OnSurface, fallbackInk(th.Dark))
	restAlpha := float32(0.30)
	hoverAlpha := float32(0.52)
	activeAlpha := float32(0.72)
	if th.Contrast >= tokens.ContrastHigh {
		restAlpha = 0.55
		hoverAlpha = 0.78
		activeAlpha = 0.92
	}
	thumbW := float32(6)
	trackW := float32(10)
	if th.Density.Normalize() == tokens.DensityCompact {
		thumbW = 5
		trackW = 9
	}
	return ui.ScrollbarStyle{
		TrackWidth: trackW,
		ThumbWidth: thumbW,
		Thumb:      ink.Alpha(restAlpha),
		Hover:      ink.Alpha(hoverAlpha),
		Active:     ink.Alpha(activeAlpha),
	}
}

// ApplyToEditor configures st with MD3E colors, density-scaled monospace font,
// line height, and scrollbar styling derived from th.
func ApplyToEditor(st *editor.State, th *md3e.Theme) {
	if st == nil {
		return
	}
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	st.Theme = ThemeFromMD3E(th)
	st.Font, st.LineHeight = FontFromMD3E(th)
	st.Scrollbar = ScrollbarFromMD3E(th)
}

// ApplyToDiff configures d with MD3E colors and density-scaled monospace font
// derived from th.
func ApplyToDiff(d *editor.DiffState, th *md3e.Theme) {
	if d == nil {
		return
	}
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	d.Theme = ThemeFromMD3E(th)
	d.Font, d.LineHeight = FontFromMD3E(th)
}

// ConfigureEditor applies md3e.CurrentTheme(c) to st before rendering
// editor.View(c, st).
func ConfigureEditor(c *ui.Context, st *editor.State) {
	ApplyToEditor(st, md3e.CurrentTheme(c))
}

func applyMonochromeTokens(ct *editor.Theme, dark bool, bgA, bgB ui.Color, minBody, minComment float64) {
	var strongT, midT, softT float64
	if dark {
		strongT, midT, softT = 95, 82, 68
	} else {
		strongT, midT, softT = 10, 26, 44
	}
	fit := func(t float64, ratio float64) ui.Color {
		return ensureDualContrast(tokens.HCT{Hue: 0, Chroma: 0, Tone: t}.ToColor(), bgA, bgB, dark, ratio)
	}
	hi := fit(strongT, minBody)
	mid := fit(midT, minBody)
	soft := fit(softT, minComment)

	ct.Tokens[editor.TokenText] = editor.TokenStyle{Color: hi}
	ct.Tokens[editor.TokenKeyword] = editor.TokenStyle{Color: hi, Bold: true}
	ct.Tokens[editor.TokenOperator] = editor.TokenStyle{Color: mid}
	ct.Tokens[editor.TokenPunctuation] = editor.TokenStyle{Color: soft}
	ct.Tokens[editor.TokenType] = editor.TokenStyle{Color: mid, Bold: true}
	ct.Tokens[editor.TokenFunction] = editor.TokenStyle{Color: hi}
	ct.Tokens[editor.TokenMeta] = editor.TokenStyle{Color: mid, Italic: true}
	ct.Tokens[editor.TokenVariable] = editor.TokenStyle{Color: hi}
	ct.Tokens[editor.TokenProperty] = editor.TokenStyle{Color: mid}
	ct.Tokens[editor.TokenConstant] = editor.TokenStyle{Color: mid}
	ct.Tokens[editor.TokenNumber] = editor.TokenStyle{Color: mid}
	ct.Tokens[editor.TokenAttribute] = editor.TokenStyle{Color: mid}
	ct.Tokens[editor.TokenString] = editor.TokenStyle{Color: mid}
	ct.Tokens[editor.TokenEscape] = editor.TokenStyle{Color: hi}
	ct.Tokens[editor.TokenRegexp] = editor.TokenStyle{Color: mid}
	ct.Tokens[editor.TokenTag] = editor.TokenStyle{Color: hi, Bold: true}
	ct.Tokens[editor.TokenComment] = editor.TokenStyle{Color: soft, Italic: true}
	ct.Tokens[editor.TokenHeading] = editor.TokenStyle{Color: hi, Bold: true}
	ct.Tokens[editor.TokenEmphasis] = editor.TokenStyle{Color: hi, Italic: true}
	ct.Tokens[editor.TokenStrong] = editor.TokenStyle{Color: hi, Bold: true}
	ct.Tokens[editor.TokenLink] = editor.TokenStyle{Color: hi}
	ct.Tokens[editor.TokenCode] = editor.TokenStyle{Color: mid}
	ct.Tokens[editor.TokenInserted] = editor.TokenStyle{Color: hi}
	ct.Tokens[editor.TokenDeleted] = editor.TokenStyle{Color: soft}
	ct.Tokens[editor.TokenInvalid] = editor.TokenStyle{Color: hi, Bold: true}
}

func ensureDualContrast(col, bgA, bgB ui.Color, dark bool, minRatio float64) ui.Color {
	meets := func(c ui.Color) bool {
		return tokens.ContrastRatio(c, bgA) >= minRatio &&
			(bgB.A == 0 || tokens.ContrastRatio(c, bgB) >= minRatio)
	}
	if meets(col) {
		return col
	}
	h := tokens.HCTFromColor(col)
	step := -4.0
	if dark {
		step = 4.0
	}
	for range 24 {
		h.Tone = clampF64(h.Tone+step, 2, 98)
		cand := h.ToColor()
		if meets(cand) {
			return cand
		}
		if (dark && h.Tone >= 98) || (!dark && h.Tone <= 2) {
			return cand
		}
	}
	return h.ToColor()
}

func pickColor(candidates ...ui.Color) ui.Color {
	for _, c := range candidates {
		if c.A > 0 {
			return c
		}
	}
	return ui.Color{}
}

func defaultCanvas(dark bool) ui.Color {
	if dark {
		return ui.Hex("#141218")
	}
	return ui.Hex("#ffffff")
}

func fallbackInk(dark bool) ui.Color {
	if dark {
		return ui.Hex("#e6e0e9")
	}
	return ui.Hex("#1d1b20")
}

func defaultError(dark bool) ui.Color {
	if dark {
		return ui.Hex("#ffb4ab")
	}
	return ui.Hex("#ba1a1a")
}

func defaultWarning(dark bool) ui.Color {
	if dark {
		return ui.Hex("#ffb951")
	}
	return ui.Hex("#855400")
}

func defaultSuccess(dark bool) ui.Color {
	if dark {
		return ui.Hex("#7adc77")
	}
	return ui.Hex("#146c2e")
}

func hueDelta(a, b float64) float64 {
	d := tokens.SanitizeDegrees(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func clampF64(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minF32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
