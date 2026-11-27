package md3e

import (
	"fmt"
	"time"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
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

type textFieldInputGroupKey struct{}

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
	dm := textFieldMetricsForDensity(th.Density)

	outer := ui.Column(c).Shrink(0).DimDisabled(false)
	if outlined && o.Label != "" {
		outer.Padding(textFieldOuterTop*scaleUp, 0, 0, 0)
	}
	disabled := o.Disabled || outer.IsDisabled()

	contentInk := cs.OnSurface
	variationInk := cs.OnSurfaceVariant
	if disabled {
		contentInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		variationInk = contentInk
	}

	fieldH := dm.height * scaleUp
	if multiline {
		fieldH = o.Height
		if fieldH <= 0 {
			fieldH = textAreaHeight * scaleUp
		}
		fieldH = max(fieldH, dm.height*scaleUp)
	}

	var (
		field       ui.Element
		input       ui.Element
		trailing    ui.Element
		hasTrailing bool
	)
	var labelStyle TypeStyleMetrics
	if o.Label != "" {
		labelStyle = typeStyleOfWithTheme(th, dm.textRole)
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
			topPad := dm.iconSize * scaleUp
			if outlined || o.Label == "" {
				topPad = dm.pad * scaleUp
			}
			field.Padding(topPad, 0, 8*scaleUp, 0)
		}
		padEnd := dm.pad
		if o.TrailingIcon != nil && o.OnTrailing != nil {
			padEnd = max(dm.pad-(dm.trailingBtn-dm.iconSize)/2, 4)
		}
		field.PaddingStart(dm.pad).PaddingEnd(padEnd)
		// Affixes show while the field is focused or holds text. Skip FocusWithin
		// completely when neither affix is configured.
		showAffix := (o.Prefix != "" || o.Suffix != "") && (*value != "" || field.FocusWithin())
		caretInk := cs.Primary
		trailingInk := variationInk
		if o.Error && !disabled {
			caretInk = cs.Error
			trailingInk = cs.Error
		}
		buildInputGroup := func() {
			if showAffix && o.Prefix != "" {
				applyTypeStyleWithTheme(th, ui.Text(c, o.Prefix).NoWrap().Shrink(0).TextColor(variationInk), dm.textRole)
			}
			c.Key(animChan3)
			if multiline {
				input = ui.TextAreaBase(c, value)
			} else {
				input = ui.TextInputBase(c, value)
			}
			input.DimDisabled(false).
				FocusRing(false).
				Disabled(disabled).
				TextColor(contentInk).
				CaretColor(caretInk).
				PlaceholderColor(variationInk).
				Grow(1).
				MinWidth(0)
			applyTypeStyleWithTheme(th, input, dm.textRole)
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
				applyTypeStyleWithTheme(th, ui.Text(c, o.Suffix).NoWrap().Shrink(0).TextColor(variationInk), dm.textRole)
			}
		}
		field.Children(func() {
			if o.LeadingIcon != nil {
				drawIconBox(c, o.LeadingIcon, dm.iconSize, variationInk)
			}
			inputGroup := ui.Row(c.Key(textFieldInputGroupKey{})).
				Grow(1).MinWidth(0).Gap(textFieldGap)
			if multiline {
				inputGroup.AlignSelf(ui.Stretch).AlignItems(ui.Stretch)
			} else {
				inputGroup.AlignItems(ui.Center)
				if !outlined && o.Label != "" {
					// Shift single-line Filled text downward so it clears the floated
					// label cleanly while leading/trailing icons remain centered.
					inputGroup.Padding(dm.filledTopPad*scaleUp, 0, 0, 0)
				}
			}
			inputGroup.Children(buildInputGroup)
			if o.TrailingIcon != nil {
				if o.OnTrailing != nil {
					hasTrailing = true
					trailing = ui.ButtonBase(c.Key(animChan4)).
						DimDisabled(false).
						FocusRing(false).
						Size(dm.trailingBtn, dm.trailingBtn).
						Shrink(0).
						Disabled(disabled).
						Label(labelOr(o.TrailingLabel, "Action")).
						TextColor(trailingInk)
					attachRippleWithContext(c, th, trailing, StateLayerOptions{
						Color:    trailingInk,
						Radius:   dm.trailingBtn / 2,
						Disabled: disabled,
					})
					trailing.Children(func() {
						drawIconBox(c, o.TrailingIcon, dm.iconSize, trailingInk)
					})
				} else {
					drawIconBox(c, o.TrailingIcon, dm.iconSize, trailingInk)
				}
			}
		})
	})

	focused := input.Focused()
	floated := false
	if o.Label != "" {
		floated = focused || *value != "" || input.Composing()
	}

	var labelT float32
	if o.Label != "" {
		labelT = clampF32(animateValue(field, animChan1, boolFloat(floated), spatial, th.ReduceMotion), 0, 1.1)
	}
	if o.Label == "" {
		// Without a label the placeholder is the only cue and always shows.
		input.Placeholder(o.Placeholder)
		input.PlaceholderColor(variationInk)
	} else if floated {
		input.Placeholder(o.Placeholder)
		input.PlaceholderColor(variationInk.Alpha(clampF32(labelT, 0, 1)))
	} else {
		// Explicitly clear upstream's persisted editor.placeholder while the
		// label rests in the content area.
		input.Placeholder("")
		input.PlaceholderColor(variationInk.Alpha(0))
	}
	indicatorT := clampF32(animateValue(field, animChan2, boolFloat(focused && !disabled), effects, th.ReduceMotion), 0, 1)

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
	labelInset := dm.pad
	hasLeadIcon := o.LeadingIcon != nil
	if hasLeadIcon {
		labelInset += dm.iconSize + textFieldGap
	}
	bg := cs.SurfaceContainerHighest
	if bg.A == 0 {
		bg = SurfaceColorForElevation(th, tokens.Level3)
	}
	if disabled && !outlined {
		bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
	}
	restBorderInk := cs.Outline
	hoverBorderInk := cs.OnSurface
	activeBorderInk := cs.Primary
	restLabelInk := cs.OnSurfaceVariant
	activeLabelInk := cs.Primary
	if disabled {
		restBorderInk = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		hoverBorderInk = restBorderInk
		activeBorderInk = restBorderInk
		restLabelInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		activeLabelInk = restLabelInk
	} else if o.Error {
		restBorderInk = cs.Error
		hoverBorderInk = cs.OnErrorContainer
		if hoverBorderInk.A == 0 {
			hoverBorderInk = cs.Error
		}
		activeBorderInk = cs.Error
		restLabelInk = cs.Error
		activeLabelInk = cs.Error
	}
	label := o.Label
	style := labelStyle
	hasEndIcon := o.TrailingIcon != nil
	isErr := o.Error
	restCenterOffset := dm.height * scaleUp / 2
	floatOffset := dm.floatOffset * scaleUp
	var outline *ui.Path
	if outlined && label != "" {
		outline = localDrawPath(field)
	}
	hoverSt := localHaloFade(field, animChan0)
	thReduce := th.ReduceMotion
	field.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		live := field.Valid()
		hovered := false
		if live && !disabled && !focused {
			hovered = field.Hovered()
		}
		targetHover := float32(0)
		if hovered {
			targetHover = 1
		}
		reduceMotion := thReduce || p.Preferences().ReduceMotion
		var hoverT float32
		var hoverMore bool
		if live {
			hoverT, hoverMore = hoverSt.step(targetHover, p.Now(), reduceMotion)
		} else {
			hoverT = hoverSt.alpha
		}

		// Smoothly interpolate border, stroke weight, and label color with
		// focus indicator progress and pointer hover.
		curBorderInk := restBorderInk
		if hoverT > 0.001 && !disabled {
			curBorderInk = restBorderInk.MixOklab(hoverBorderInk, hoverT)
		}
		if indicatorT > 0.001 && !disabled {
			curBorderInk = curBorderInk.MixOklab(activeBorderInk, indicatorT)
		}
		curLabelInk := restLabelInk
		if indicatorT > 0.001 && !disabled {
			curLabelInk = restLabelInk.MixOklab(activeLabelInk, indicatorT)
		}

		if !outlined {
			radii := [4]float32{textFieldRadius, textFieldRadius, 0, 0}
			p.FillRadii(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}, radii, bg)
			if hoverT > 0.001 && !disabled {
				p.FillRadii(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}, radii, cs.OnSurface.Alpha(float32(tokens.StateHoverOpacity)*hoverT))
			}
		}

		clampedLabelT := clampF32(labelT, 0, 1)
		sc := 1 - (1-textFieldFloatScale)*labelT
		// In Outlined fields with a leading icon, the floating label glides
		// horizontally back to the container start padding as it rises to the
		// top border; Filled labels stay aligned with the input text.
		curInset := labelInset
		floatInset := labelInset
		if outlined && hasLeadIcon {
			floatInset = dm.pad
			curInset = labelInset + (dm.pad-labelInset)*clampedLabelT
		}
		anchor := r.X + curInset
		floatAnchor := r.X + floatInset
		if rtl {
			anchor = r.X + r.W - curInset
			floatAnchor = r.X + r.W - floatInset
		}
		floatCenter := r.Y + floatOffset
		if outlined {
			floatCenter = r.Y
		}

		var drawLabel string
		var labelW float32
		var clipR ui.Rect
		if label != "" {
			restEndInset := dm.pad
			if hasEndIcon {
				restEndInset += dm.iconSize + textFieldGap
			}
			endInset := restEndInset + (dm.pad-restEndInset)*clampedLabelT
			maxLabelW := max(0, r.W-curInset-endInset)
			drawLabel, labelW = fitTextFieldLabel(p, label, curLabelInk, style, maxLabelW, sc)
			clipX := anchor
			if rtl {
				clipX = r.X + endInset
			}
			clipY := min(r.Y, floatCenter) - style.LineHeight - 4
			clipH := max(r.H, restCenterOffset+style.LineHeight) + style.LineHeight*2 + 8
			clipR = ui.Rect{X: clipX, Y: clipY, W: maxLabelW + 1, H: clipH}
		}

		if outlined {
			baseStrokeW := float32(1)
			if isErr && !disabled {
				baseStrokeW = 2
			}
			strokeW := baseStrokeW + (2-baseStrokeW)*indicatorT
			notchOpen := textFieldSmoothstep(0.22, 0.85, clampedLabelT)
			if drawLabel != "" && notchOpen > 0.001 {
				finalNotchW := labelW*textFieldFloatScale + 2*textFieldNotchPad
				var targetStart, targetEnd float32
				if rtl {
					targetEnd = floatAnchor + textFieldNotchPad
					targetStart = targetEnd - finalNotchW
				} else {
					targetStart = floatAnchor - textFieldNotchPad
					targetEnd = targetStart + finalNotchW
				}
				targetStart = max(targetStart, r.X+textFieldRadius)
				targetEnd = min(targetEnd, r.X+r.W-textFieldRadius)
				notchCenter := (targetStart + targetEnd) * 0.5
				halfW := (targetEnd - targetStart) * 0.5 * notchOpen
				notchStart := notchCenter - halfW
				notchEnd := notchCenter + halfW
				if notchEnd > notchStart {
					if path := outlinedFieldPath(outline, r, textFieldRadius, strokeW, notchStart, notchEnd); path != nil {
						p.StrokePath(path, strokeW, curBorderInk)
					} else {
						p.Stroke(r, curBorderInk, textFieldRadius, strokeW)
					}
				} else {
					p.Stroke(r, curBorderInk, textFieldRadius, strokeW)
				}
			} else {
				p.Stroke(r, curBorderInk, textFieldRadius, strokeW)
			}
		} else {
			indicator := cs.OnSurfaceVariant
			if hoverT > 0.001 && !disabled {
				indicator = indicator.MixOklab(cs.OnSurface, hoverT)
			}
			if disabled {
				indicator = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
			}
			activeInk := cs.Primary
			if isErr {
				indicator = disabledContent(disabled, cs.Error)
				activeInk = indicator
			}
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, indicator, 0)
			if indicatorT > 0.001 {
				activeW := r.W * indicatorT
				p.Fill(ui.Rect{
					X: r.X + (r.W-activeW)/2,
					Y: r.Y + r.H - 2,
					W: activeW,
					H: 2,
				}, disabledContent(disabled, activeInk), 0)
			}
		}
		if drawLabel != "" {
			drawTextFieldLabel(p, r, drawLabel, labelW, curLabelInk, style, labelT, anchor, floatCenter, restCenterOffset, clipR, rtl)
		}
		if live && hoverMore {
			p.After(16 * time.Millisecond)
		}
	})

	supportText := o.SupportingText
	textColor := cs.OnSurfaceVariant
	if disabled {
		textColor = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}
	if o.Error && o.ErrorMessage != "" {
		supportText = o.ErrorMessage
		textColor = cs.Error
		if disabled {
			textColor = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		}
	}
	counter := ""
	if o.CharacterCounter && o.MaxLength > 0 {
		counter = fmt.Sprintf("%d/%d", utf16Len(*value), o.MaxLength)
	}
	if supportText != "" || counter != "" {
		outer.Children(func() {
			row := ui.Row(c).
				Height(textFieldSupportH*scaleUp).
				PaddingX(dm.pad).
				Gap(textFieldGap).
				Margin(textFieldSupportGap, 0, 0, 0).
				AlignItems(ui.Center)
			row.Children(func() {
				if supportText != "" {
					applyTypeStyleWithTheme(th, ui.Text(c, supportText).NoWrap().Grow(1).TextColor(textColor).AccessHidden(true), tokens.BodySmall)
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

// textFieldSmoothstep maps x smoothly from 0 at edge0 to 1 at edge1.
func textFieldSmoothstep(edge0, edge1, x float32) float32 {
	if edge1 <= edge0 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := clampF32((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
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
// leaves no drawable border. The centerline is inset by half the stroke width
// so p.StrokePath aligns flush inside r like p.Stroke.
func outlinedFieldPath(p *ui.Path, r ui.Rect, radius, strokeW, notchStart, notchEnd float32) *ui.Path {
	hw := strokeW * 0.5
	x0, x1 := r.X+hw, r.X+r.W-hw
	y0, y1 := r.Y+hw, r.Y+r.H-hw
	innerR := max(float32(0), radius-hw)
	if x1-x0 <= 2*innerR || y1-y0 <= 2*innerR || notchEnd <= notchStart {
		return nil
	}
	lo, hi := x0+innerR, x1-innerR
	if notchStart < lo {
		notchStart = lo
	}
	if notchEnd > hi {
		notchEnd = hi
	}
	if notchEnd <= notchStart {
		return nil
	}
	k := innerR * 0.5522847498
	p.Reset()
	p.MoveTo(notchEnd, y0)
	p.LineTo(x1-innerR, y0)
	p.CubeTo(x1-innerR+k, y0, x1, y0+innerR-k, x1, y0+innerR)
	p.LineTo(x1, y1-innerR)
	p.CubeTo(x1, y1-innerR+k, x1-innerR+k, y1, x1-innerR, y1)
	p.LineTo(x0+innerR, y1)
	p.CubeTo(x0+innerR-k, y1, x0, y1-innerR+k, x0, y1-innerR)
	p.LineTo(x0, y0+innerR)
	p.CubeTo(x0, y0+innerR-k, x0+innerR-k, y0, x0+innerR, y0)
	p.LineTo(notchStart, y0)
	p.Cap(ui.ButtCap)
	return p
}

type textFieldDensityMetrics struct {
	height       float32
	pad          float32
	iconSize     float32
	trailingBtn  float32
	floatOffset  float32
	filledTopPad float32
	textRole     tokens.TypeRole
}

func textFieldMetricsForDensity(d tokens.Density) textFieldDensityMetrics {
	h := float32(d.TextFieldHeight())
	switch d.Normalize() {
	case tokens.DensityDesktop:
		return textFieldDensityMetrics{
			height:       h,
			pad:          12,
			iconSize:     20,
			trailingBtn:  32,
			floatOffset:  9,
			filledTopPad: 10,
			textRole:     tokens.BodyMedium,
		}
	case tokens.DensityCompact:
		return textFieldDensityMetrics{
			height:       h,
			pad:          10,
			iconSize:     18,
			trailingBtn:  28,
			floatOffset:  8,
			filledTopPad: 8,
			textRole:     tokens.BodyMedium,
		}
	default:
		return textFieldDensityMetrics{
			height:       h,
			pad:          textFieldPad,
			iconSize:     textFieldIconSize,
			trailingBtn:  40,
			floatOffset:  12,
			filledTopPad: 14,
			textRole:     tokens.BodyLarge,
		}
	}
}
