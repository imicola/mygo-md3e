package md3e

import (
	"fmt"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// TextFieldVariant selects the Material Design 3 Expressive text field look.
type TextFieldVariant int

const (
	// TextFieldFilled fills the container with SurfaceContainerHighest, rounds
	// its top corners and draws the bottom indicator line.
	TextFieldFilled TextFieldVariant = iota
	// TextFieldOutlined draws a full border around a transparent container,
	// with a notch in the top edge while the label floats.
	TextFieldOutlined
)

// Text field metrics in DIP, as MD3E specifies them: a 56 DIP field, 4 DIP
// corners, 16 DIP content padding, a 24 DIP icon, a 24 DIP floating label line
// and 12 DIP floated label.
const (
	textFieldHeight     = 56
	textFieldRadius     = 4
	textFieldPad        = 16
	textFieldIconSize   = 24
	textFieldGap        = 8
	textFieldFloatScale = 0.75
	textFieldNotchPad   = 4
	textFieldSupportGap = 4
	textFieldSupportH   = 20
	textAreaHeight      = 120
	textFieldOuterTop   = 10
)

// TextFieldOptions configures TextField and TextArea. The zero value is an
// enabled, filled, single-line field with no label or icons.
type TextFieldOptions struct {
	// Variant selects the filled or outlined look.
	Variant TextFieldVariant
	// Label floats from the content area to the top of the field when the
	// field is focused, holds text, or an input method composes, and names the
	// field for assistive technology.
	Label string
	// Placeholder shows in the input while the label floats and the value is
	// empty. Without a Label it also names the field.
	Placeholder string
	// LeadingIcon draws a 24 DIP decorative icon before the text.
	LeadingIcon IconDraw
	// TrailingIcon draws a 24 DIP icon after the text. With OnTrailing set it
	// becomes an interactive button, as a password visibility toggle or a
	// clear action.
	TrailingIcon IconDraw
	// TrailingLabel names the interactive trailing button for assistive
	// technology; "Action" by default.
	TrailingLabel string
	// OnTrailing runs when the interactive trailing button is clicked.
	OnTrailing func()
	// Prefix shows before the text while the field is focused or holds text.
	Prefix string
	// Suffix shows after the text while the field is focused or holds text.
	Suffix string
	// SupportingText shows below the field and describes it to assistive
	// technology.
	SupportingText string
	// MaxLength limits the text, in UTF-16 code units as ui's inputs count it.
	MaxLength int
	// CharacterCounter shows the length below the field, as "12/50" when
	// MaxLength is set.
	CharacterCounter bool
	// Error marks the value invalid, paints the error palette and sets
	// aria-invalid on the input.
	Error bool
	// ErrorMessage tells assistive technology why the value is invalid, as
	// aria-errormessage, and shows in place of the supporting text.
	ErrorMessage string
	// ReadOnly lets the text be read and selected but not changed.
	ReadOnly bool
	// Password hides the text of a single-line field; a text area shows it.
	Password bool
	// Disabled disables the field with MD3E's disabled colors, without ui's
	// automatic dimming.
	Disabled bool
	// Height overrides the text area's 120 DIP height. TextField ignores it.
	Height float32
	// OnChanged runs with the value whenever the user edits it.
	OnChanged func(string)
	// OnSubmitted runs with the value when Enter submits a single-line field.
	OnSubmitted func(string)
}

// textFieldInputKey keys the input inside a field's row, so that a caller's
// c.Key on the TextField or TextArea stays on the outer element while the
// stateful input base keeps its own stable identity in that row. Sibling
// fields may share the key: each input sits under its own field row.
type (
	textFieldInputKey     struct{}
	textFieldTrailingKey  struct{}
	textFieldLabelKey     struct{}
	textFieldIndicatorKey struct{}
)

// TextField builds a Material Design 3 Expressive single-line text field over
// *value, filled or outlined, around a ui.TextInputBase keyed internally in the
// field's row. The floating label rests in the content area while the field is
// empty, blurred and no input method composes, and springs to the top with the
// Fast Spatial spring when the field is focused, holds text, or an IME is
// composing, so marked text never overlaps the label. An outlined field breaks
// its top border around the floated label; a filled field grows its bottom
// indicator from the middle in Primary, or shows Error.
//
// A caller's c.Key lands on the outer column, the documented identity of the
// field, while the input keeps its own key in the row:
//
//	md3e.TextField(c.Key("email"), &app.email, md3e.TextFieldOptions{
//		Label: "Email", SupportingText: "We never share it",
//	})
func TextField(c *ui.Context, value *string, opts ...TextFieldOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: TextField takes at most one TextFieldOptions")
	}
	if value == nil {
		panic("md3e: TextField requires a non-nil *string")
	}
	var o TextFieldOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	return buildTextField(c, value, false, o)
}

// TextArea builds a multi-line Material Design 3 Expressive text area over
// *value, filled or outlined, around a ui.TextAreaBase keyed internally in the
// field's row. It shares the floating label, slots, supporting text, counter
// and error behavior of TextField, wraps at its width, keeps the 120 DIP height
// by default and scrolls its text beyond it.
//
//	md3e.TextArea(c, &app.notes, md3e.TextFieldOptions{
//		Variant: md3e.TextFieldOutlined, Label: "Notes", Height: 160,
//	})
func TextArea(c *ui.Context, value *string, opts ...TextFieldOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: TextArea takes at most one TextFieldOptions")
	}
	if value == nil {
		panic("md3e: TextArea requires a non-nil *string")
	}
	var o TextFieldOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	return buildTextField(c, value, true, o)
}

// buildTextField builds a single-line or multi-line field.
func buildTextField(c *ui.Context, value *string, multiline bool, o TextFieldOptions) ui.Element {
	th := CurrentTheme(c)
	cs := th.Colors
	rtl := c.Direction() == ui.RightToLeft
	outlined := o.Variant == TextFieldOutlined
	scaleUp := max(float32(1), th.EffectiveTextScale())

	outer := ui.Column(c).Shrink(0).DimDisabled(false)
	if outlined {
		outer.Padding(textFieldOuterTop*scaleUp, 0, 0, 0)
	}
	disabled := o.Disabled || outer.IsDisabled()

	contentInk := cs.OnSurface
	variationInk := cs.OnSurfaceVariant
	if disabled {
		contentInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		variationInk = contentInk
	}

	fieldH := float32(textFieldHeight) * scaleUp
	if multiline {
		fieldH = o.Height
		if fieldH <= 0 {
			fieldH = textAreaHeight * scaleUp
		}
		fieldH = max(fieldH, float32(textFieldHeight)*scaleUp)
	}

	var (
		field       ui.Element
		input       ui.Element
		trailing    ui.Element
		hasTrailing bool
	)
	var labelStyle TypeStyleMetrics
	if o.Label != "" {
		labelStyle = typeStyleOfWithTheme(th, tokens.BodyLarge)
	}
	spatial := th.Animation(motion.FastSpatial)
	effects := th.Animation(motion.FastEffects)

	outer.Children(func() {
		field = ui.Row(c).
			Height(fieldH).
			AlignItems(ui.Center).
			Gap(textFieldGap).
			DimDisabled(false)
		if outlined {
			field.Radius(textFieldRadius)
		} else {
			field.Radius(textFieldRadius, textFieldRadius, 0, 0)
		}
		if multiline {
			field.Padding(textFieldIconSize*scaleUp, 0, 8*scaleUp, 0)
		}
		field.PaddingStart(textFieldPad).PaddingEnd(textFieldPad)
		// Affixes show while the field is focused or holds text. Skip FocusWithin
		// completely when neither affix is configured.
		showAffix := (o.Prefix != "" || o.Suffix != "") && (*value != "" || field.FocusWithin())
		field.Children(func() {
			if o.LeadingIcon != nil {
				drawIconBox(c, o.LeadingIcon, textFieldIconSize, variationInk)
			}
			if showAffix && o.Prefix != "" {
				applyTypeStyleWithTheme(th, ui.Text(c, o.Prefix).NoWrap().Shrink(0).TextColor(variationInk), tokens.BodyLarge)
			}
			c.Key(textFieldInputKey{})
			if multiline {
				input = ui.TextAreaBase(c, value)
			} else {
				input = ui.TextInputBase(c, value)
			}
			input.DimDisabled(false).
				FocusRing(false).
				Disabled(disabled).
				TextColor(contentInk).
				CaretColor(cs.Primary).
				PlaceholderColor(variationInk).
				Grow(1)
			applyTypeStyleWithTheme(th, input, tokens.BodyLarge)
			if multiline {
				input.AlignSelf(ui.Stretch)
			}
			if o.MaxLength > 0 {
				input.MaxLength(o.MaxLength)
			}
			if o.ReadOnly {
				input.ReadOnly(true)
			}
			if o.Password && !multiline {
				input.Password()
			}
			name := o.Label
			if name == "" {
				name = o.Placeholder
			}
			if name != "" {
				input.Label(name)
			}
			if o.Error {
				// aria-invalid first, and the error or fallback supporting description
				// relation so assistive technology always receives context.
				input.Invalid(true)
				if o.ErrorMessage != "" {
					input.Error(o.ErrorMessage)
				} else if o.SupportingText != "" {
					input.Description(o.SupportingText)
				}
			} else if o.SupportingText != "" {
				input.Description(o.SupportingText)
			}

			if showAffix && o.Suffix != "" {
				applyTypeStyleWithTheme(th, ui.Text(c, o.Suffix).NoWrap().Shrink(0).TextColor(variationInk), tokens.BodyLarge)
			}
			if o.TrailingIcon != nil {
				if o.OnTrailing != nil {
					hasTrailing = true
					trailing = ui.ButtonBase(c.Key(textFieldTrailingKey{})).
						DimDisabled(false).
						FocusRing(false).
						Size(40, 40).
						Shrink(0).
						Disabled(disabled).
						Label(labelOr(o.TrailingLabel, "Action")).
						TextColor(variationInk)
					attachRippleWithTheme(th, trailing, StateLayerOptions{
						Color:    variationInk,
						Radius:   20,
						Disabled: disabled,
					})
					trailing.Children(func() {
						drawIconBox(c, o.TrailingIcon, textFieldIconSize, variationInk)
					})
				} else {
					drawIconBox(c, o.TrailingIcon, textFieldIconSize, variationInk)
				}
			}
		})
	})

	focused := input.Focused()
	floated := false
	if o.Label == "" {
		// Without a label the placeholder is the only cue and always shows.
		input.Placeholder(o.Placeholder)
	} else {
		floated = focused || *value != "" || input.Composing()
		if floated {
			input.Placeholder(o.Placeholder)
		}
	}

	var labelT float32
	if o.Label != "" {
		labelT = clampF32(animateValue(field, textFieldLabelKey{}, boolFloat(floated), spatial, th.ReduceMotion), 0, 1.1)
	}
	indicatorT := clampF32(animateValue(field, textFieldIndicatorKey{}, boolFloat(focused && !disabled), effects, th.ReduceMotion), 0, 1)

	if field.Clicked() && !disabled {
		// Any part of the field, its padding and its floating label too,
		// focuses the input.
		input.Focus()
	}
	if hasTrailing && trailing.Clicked() && o.OnTrailing != nil {
		o.OnTrailing()
	}

	changed := input.Changed()
	submitted := input.Submitted()

	// The label anchor is the content start, shifted past a leading icon.
	labelInset := float32(textFieldPad)
	if o.LeadingIcon != nil {
		labelInset += textFieldIconSize + textFieldGap
	}
	bg := cs.SurfaceContainerHighest
	if bg.A == 0 {
		bg = SurfaceColorForElevation(th, tokens.Level3)
	}
	if disabled && !outlined {
		bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
	}
	borderInk := cs.Outline
	switch {
	case disabled:
		borderInk = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
	case o.Error:
		borderInk = cs.Error
	case focused:
		borderInk = cs.Primary
	}
	// The label is OnSurfaceVariant at rest, Primary over a focused field, and
	// Error while the value is invalid; disabled fields use 38% content.
	labelColor := cs.OnSurfaceVariant
	switch {
	case disabled:
		labelColor = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	case o.Error:
		labelColor = cs.Error
	case focused:
		labelColor = cs.Primary
	}
	label := o.Label
	style := labelStyle
	hasEndIcon := o.TrailingIcon != nil
	isErr := o.Error
	restCenterOffset := float32(textFieldHeight) * scaleUp / 2
	floatOffset := 12 * scaleUp
	field.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		if !outlined {
			radii := [4]float32{textFieldRadius, textFieldRadius, 0, 0}
			p.FillRadii(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}, radii, bg)
		}
		sc := 1 - (1-textFieldFloatScale)*labelT
		anchor := r.X + labelInset
		if rtl {
			anchor = r.X + r.W - labelInset
		}
		floatCenter := r.Y + floatOffset
		if outlined {
			floatCenter = r.Y
		}

		var drawLabel string
		var labelW float32
		var clipR ui.Rect
		if label != "" {
			restEndInset := float32(textFieldPad)
			if hasEndIcon {
				restEndInset += textFieldIconSize + textFieldGap
			}
			endInset := restEndInset + (float32(textFieldPad)-restEndInset)*clampF32(labelT, 0, 1)
			maxLabelW := max(0, r.W-labelInset-endInset)
			drawLabel, labelW = fitTextFieldLabel(p, label, labelColor, style, maxLabelW, sc)
			clipX := anchor
			if rtl {
				clipX = r.X + endInset
			}
			clipY := min(r.Y, floatCenter) - style.LineHeight - 4
			clipH := max(r.H, restCenterOffset+style.LineHeight) + style.LineHeight*2 + 8
			clipR = ui.Rect{X: clipX, Y: clipY, W: maxLabelW + 1, H: clipH}
		}

		if outlined {
			if drawLabel != "" && labelT > 0.001 {
				notchW := labelW*sc + 2*textFieldNotchPad*labelT
				var notchStart, notchEnd float32
				if rtl {
					notchEnd = anchor + textFieldNotchPad*labelT
					notchStart = notchEnd - notchW
				} else {
					notchStart = anchor - textFieldNotchPad*labelT
					notchEnd = notchStart + notchW
				}
				notchStart = max(notchStart, r.X+textFieldRadius)
				notchEnd = min(notchEnd, r.X+r.W-textFieldRadius)
				if notchEnd > notchStart {
					if path := outlinedFieldPath(r, textFieldRadius, notchStart, notchEnd); path != nil {
						p.StrokePath(path, outlinedWidth(isErr, focused), borderInk)
					} else {
						p.Stroke(r, borderInk, textFieldRadius, outlinedWidth(isErr, focused))
					}
				} else {
					p.Stroke(r, borderInk, textFieldRadius, outlinedWidth(isErr, focused))
				}
			} else {
				p.Stroke(r, borderInk, textFieldRadius, outlinedWidth(isErr, focused))
			}
		} else {
			indicator := cs.OnSurfaceVariant
			if disabled {
				indicator = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
			}
			h := float32(1)
			if !isErr {
				p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - h, W: r.W, H: h}, indicator, 0)
				if indicatorT > 0.001 {
					activeW := r.W * indicatorT
					p.Fill(ui.Rect{
						X: r.X + (r.W-activeW)/2,
						Y: r.Y + r.H - 2,
						W: activeW,
						H: 2,
					}, disabledContent(disabled, cs.Primary), 0)
				}
			} else {
				h = 1
				if focused {
					h = 2
				}
				p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - h, W: r.W, H: h}, disabledContent(disabled, cs.Error), 0)
			}
		}
		if drawLabel != "" {
			drawTextFieldLabel(p, r, drawLabel, labelW, labelColor, style, labelT, anchor, floatCenter, restCenterOffset, clipR, rtl)
		}
	})

	if o.SupportingText != "" || o.ErrorMessage != "" || (o.CharacterCounter && o.MaxLength > 0) {
		text := o.SupportingText
		textColor := cs.OnSurfaceVariant
		if disabled {
			textColor = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		}
		counter := ""
		if o.CharacterCounter && o.MaxLength > 0 {
			counter = fmt.Sprintf("%d/%d", utf16Len(*value), o.MaxLength)
		}
		if o.Error && o.ErrorMessage != "" {
			text = o.ErrorMessage
			textColor = cs.Error
			if disabled {
				textColor = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
			}
		}
		outer.Children(func() {
			row := ui.Row(c).
				Height(textFieldSupportH*scaleUp).
				PaddingX(textFieldPad).
				Gap(textFieldGap).
				Margin(textFieldSupportGap, 0, 0, 0).
				AlignItems(ui.Center)
			row.Children(func() {
				if text != "" {
					applyTypeStyleWithTheme(th, ui.Text(c, text).NoWrap().Grow(1).TextColor(textColor).AccessHidden(true), tokens.BodySmall)
				} else {
					ui.Box(c).Grow(1)
				}
				if counter != "" {
					applyTypeStyleWithTheme(th, ui.Text(c, counter).NoWrap().Shrink(0).TextColor(textColor).AccessHidden(true), tokens.BodySmall)
				}
			})
		})
	}

	if changed && o.OnChanged != nil {
		o.OnChanged(*value)
	}
	if submitted && o.OnSubmitted != nil {
		o.OnSubmitted(*value)
	}
	return outer
}

// labelSpan returns the painter span of a field label.
func labelSpan(text string, ink ui.Color, style TypeStyleMetrics) ui.Span {
	return ui.Span{
		Text:          text,
		Font:          style.Font,
		Size:          style.Size,
		Weight:        style.Weight,
		Color:         ink,
		LetterSpacing: style.Tracking,
	}
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if rl := utf16.RuneLen(r); rl > 0 {
			n += rl
		} else {
			n++
		}
	}
	return n
}

func fitTextFieldLabel(p *ui.Painter, text string, ink ui.Color, style TypeStyleMetrics, maxScaledW, sc float32) (string, float32) {
	if text == "" || !(maxScaledW > 0) || sc <= 0 {
		return "", 0
	}
	w, _ := p.MeasureText(0, labelSpan(text, ink, style))
	if w*sc <= maxScaledW {
		return text, w
	}
	ellW, _ := p.MeasureText(0, labelSpan("…", ink, style))
	if ellW*sc > maxScaledW {
		return "", 0
	}
	runes := []rune(text)
	lo, hi := 0, len(runes)
	best := "…"
	bestW := ellW
	for lo <= hi {
		mid := (lo + hi) / 2
		cand := string(runes[:mid]) + "…"
		cw, _ := p.MeasureText(0, labelSpan(cand, ink, style))
		if cw*sc <= maxScaledW {
			best = cand
			bestW = cw
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return best, bestW
}

// drawTextFieldLabel paints a field's floating label at progress t, anchored at
// anchor (the content start, or its right edge right to left). floatCenter is
// where the label's middle rests when floated: inside the top of a filled
// field, or on the border of an outlined one.
func drawTextFieldLabel(p *ui.Painter, r ui.Rect, text string, w float32, ink ui.Color, style TypeStyleMetrics, t, anchor, floatCenter, restCenterOffset float32, clipR ui.Rect, rtl bool) {
	if style.LineHeight <= 0 || text == "" {
		return
	}
	t = clampF32(t, 0, 1.1)
	sc := 1 - (1-textFieldFloatScale)*t
	restCenter := r.Y + restCenterOffset
	y0 := restCenter - style.LineHeight/2
	y1 := floatCenter - style.LineHeight*sc/2
	y := y0 + (y1-y0)*t
	x := anchor
	if rtl {
		x = anchor - w*sc
	}
	paintLabel := func() {
		p.Transform(ui.Scaling(sc, sc), x, y, func() {
			p.RichText(x, y, 0, labelSpan(text, ink, style))
		})
	}
	if clipR.W > 0 && clipR.H > 0 {
		p.Clip(clipR, 0, paintLabel)
		return
	}
	paintLabel()
}

// outlinedWidth returns a text field border's stroke width: 2 DIP while
// focused or in error, 1 DIP otherwise.
func outlinedWidth(err, focused bool) float32 {
	if err || focused {
		return 2
	}
	return 1
}

// disabledContent returns c, faded to MD3E's disabled content opacity when
// disabled.
func disabledContent(disabled bool, c ui.Color) ui.Color {
	if disabled {
		return c.Alpha(float32(tokens.DisabledContentOpacity))
	}
	return c
}

// outlinedFieldPath returns the open path of an outlined field's border with a
// gap in the top edge from notchStart to notchEnd, or nil when the notch
// leaves no drawable border.
func outlinedFieldPath(r ui.Rect, radius, notchStart, notchEnd float32) *ui.Path {
	if r.W <= 2*radius || r.H <= 2*radius || notchEnd <= notchStart {
		return nil
	}
	x0, x1 := r.X, r.X+r.W
	y0, y1 := r.Y, r.Y+r.H
	lo, hi := x0+radius, x1-radius
	if notchStart < lo {
		notchStart = lo
	}
	if notchEnd > hi {
		notchEnd = hi
	}
	if notchEnd <= notchStart {
		return nil
	}
	k := radius * 0.5522847498
	var p ui.Path
	p.MoveTo(notchEnd, y0)
	p.LineTo(x1-radius, y0)
	p.CubeTo(x1-radius+k, y0, x1, y0+radius-k, x1, y0+radius)
	p.LineTo(x1, y1-radius)
	p.CubeTo(x1, y1-radius+k, x1-radius+k, y1, x1-radius, y1)
	p.LineTo(x0+radius, y1)
	p.CubeTo(x0+radius-k, y1, x0, y1-radius+k, x0, y1-radius)
	p.LineTo(x0, y0+radius)
	p.CubeTo(x0, y0+radius-k, x0+radius-k, y0, x0+radius, y0)
	p.LineTo(notchStart, y0)
	return &p
}
