package md3e

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/tokens"
)

// fieldCanvas builds a 360x420 white canvas holding build's fields.
func fieldCanvas(th *Theme, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(360, 420).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				build(c)
			})
		})
	}, 360, 420)
}

// fieldRect returns the field container of a TextField of width 280 placed at
// the canvas' 20 DIP padding, from its input's access box. Right to left, the
// canvas' start is its right edge, so the field sits at 60.
func fieldRect(t *testing.T, tt *ui.Tester, name string, height float32) (ui.Rect, ui.AccessInfo) {
	t.Helper()
	info := buttonInfo(t, tt, ui.RoleTextField, name)
	x := float32(20)
	if info.Bounds.X > 40 {
		// Right to left: the field's right edge is the input's plus the
		// content padding.
		x = info.Bounds.X + info.Bounds.W + textFieldPad - 280
	}
	return ui.Rect{
		X: x,
		Y: info.Bounds.Y + info.Bounds.H/2 - height/2,
		W: 280,
		H: height,
	}, info
}

// inkIn counts pixels of col inside r, within tol, and returns their vertical
// extent.
func inkIn(tt *ui.Tester, r ui.Rect, col ui.Color, tol int) (count, minY, maxY int) {
	img := tt.Image()
	minY, maxY = -1, -1
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if colorMatches(img.RGBAAt(x, y), col, tol) {
				count++
				if minY < 0 {
					minY = y
				}
				maxY = y
			}
		}
	}
	return count, minY, maxY
}

// TestTextFieldOutlinedFloatBlurAndNotch checks the outlined field's floating
// label, its notch while floated, input, OnChanged, OnSubmitted, the notch
// remaining with text after blur, and its closing when the text is cleared.
func TestTextFieldOutlinedFloatBlurAndNotch(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	email := ""
	var changed, submitted []string
	tt := fieldCanvas(th, func(c *ui.Context) {
		ui.Column(c).Gap(16).Children(func() {
			TextField(c, &email, TextFieldOptions{
				Variant:     TextFieldOutlined,
				Label:       "Email",
				Placeholder: "you@example.com",
				OnChanged:   func(s string) { changed = append(changed, s) },
				OnSubmitted: func(s string) { submitted = append(submitted, s) },
			}).Width(280)
			Button(c, "Blur")
		})
	})
	field, info := fieldRect(t, tt, "Email", textFieldHeight)
	if info.Bounds.H != 24 {
		t.Fatalf("input line height = %v, want 24", info.Bounds.H)
	}
	// At rest the label is LabelLarge inside the field, drawn OnSurfaceVariant.
	labelArea := ui.Rect{X: field.X, Y: field.Y + 8, W: field.W, H: 40}
	n, minY, _ := inkIn(tt, labelArea, cs.OnSurfaceVariant, 70)
	if n < 5 || minY < int(field.Y+14) {
		t.Fatalf("resting label inked %d pixels from y=%d, want inside the field", n, minY)
	}

	// Clicking the field's padding, not the input, focuses the input, floats
	// the label with the spring and reveals the placeholder.
	tt.ClickAt(field.X+6, field.Y+field.H/2)
	if !tt.FramePending() {
		t.Fatal("focusing should start the label spring")
	}
	info, _ = tt.AccessByRole(ui.RoleTextField, "Email")
	if !info.Focused {
		t.Fatal("clicking the field did not focus its input")
	}
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("label spring did not settle after focusing")
	}
	info, _ = tt.AccessByRole(ui.RoleTextField, "Email")
	if info.Placeholder != "you@example.com" {
		t.Fatalf("focused placeholder = %q, want the placeholder to show", info.Placeholder)
	}
	n, minY, _ = inkIn(tt, ui.Rect{X: field.X + textFieldPad, Y: field.Y - 10, W: 34, H: 22}, cs.Primary, 70)
	if n < 5 || minY >= int(field.Y) {
		t.Fatalf("floated label inked %d pixels from y=%d, want it above the top border", n, minY)
	}
	// The notch leaves the canvas at the label's start, while the border far
	// to its end paints Primary.
	if !topEdgeCanvas(tt, field.X+14, field.Y) {
		t.Fatal("focused notch is not open to the canvas")
	}
	if !topEdgeHas(tt, field.X+field.W-20, field.Y, cs.Primary, 70) {
		t.Fatal("focused top border is not Primary")
	}

	// Typing reports changes, and Enter submits.
	tt.Type("hi")
	if email != "hi" {
		t.Fatalf("typed value = %q, want hi", email)
	}
	if len(changed) == 0 || changed[len(changed)-1] != "hi" {
		t.Fatalf("OnChanged got %v, want hi", changed)
	}
	tt.Advance(600 * time.Millisecond)
	tt.Key(0, ui.KeyEnter)
	if len(submitted) != 1 || submitted[0] != "hi" {
		t.Fatalf("OnSubmitted got %v, want [hi]", submitted)
	}

	// Blurred with text, the label stays floated and the notch stays open.
	blur := buttonInfo(t, tt, ui.RoleButton, "Blur")
	tt.ClickAt(center(blur.Bounds))
	tt.Advance(900 * time.Millisecond)
	if !topEdgeCanvas(tt, field.X+14, field.Y) {
		t.Fatal("blurred with text, the notch closed")
	}
	if !topEdgeHas(tt, field.X+field.W-20, field.Y, cs.Outline, 170) {
		img := tt.Image()
		var edge []color.RGBA
		for dy := -2; dy <= 1; dy++ {
			edge = append(edge, img.RGBAAt(int(field.X+field.W-20), int(field.Y)+dy))
		}
		t.Fatalf("blurred top border is not Outline (%+v, outline %+v, focused=%v)", edge, rgbOf(cs.Outline), info.Focused)
	}
	// The label stays in the top band, over the notch, in OnSurfaceVariant.
	if n, _, _ := inkIn(tt, ui.Rect{X: field.X, Y: field.Y - 9, W: 120, H: 18}, cs.OnSurfaceVariant, 70); n < 5 {
		t.Fatalf("label fell back after blur with text: %d pixels at the top", n)
	}

	// Clearing and blurring drops the label back and closes the notch.
	email = ""
	tt.Frame()
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("label spring did not settle after clearing")
	}
	if !topEdgeHas(tt, field.X+14, field.Y, cs.Outline, 170) {
		t.Fatal("the cleared field's notch did not close into an Outline border")
	}
}

// TestTextFieldFilledIndicator checks the filled field's SurfaceContainerHighest
// container, the label resting in the middle, and the Primary focus indicator.
func TestTextFieldFilledIndicator(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	email := ""
	var changed, submitted []string
	tt := fieldCanvas(th, func(c *ui.Context) {
		TextField(c, &email, TextFieldOptions{
			Label:       "Email",
			Prefix:      "$",
			Suffix:      "USD",
			OnChanged:   func(s string) { changed = append(changed, s) },
			OnSubmitted: func(s string) { submitted = append(submitted, s) },
		}).Width(280)
	})
	field, info := fieldRect(t, tt, "Email", textFieldHeight)
	img := tt.Image()
	if px := img.RGBAAt(int(field.X+field.W/2), int(field.Y+8)); !colorMatches(px, cs.SurfaceContainerHighest, 5) {
		t.Fatalf("filled container %+v is not SurfaceContainerHighest", px)
	}
	if px := img.RGBAAt(int(field.X+field.W/2), int(field.Y+field.H-1)); !colorMatches(px, cs.OnSurfaceVariant, 40) {
		t.Fatalf("inactive indicator %+v is not OnSurfaceVariant", px)
	}
	if n, minY, _ := inkIn(tt, ui.Rect{X: field.X + 8, Y: field.Y + 10, W: 200, H: 36}, cs.OnSurfaceVariant, 70); n < 5 || minY < int(field.Y+16) {
		t.Fatalf("filled resting label inked %d pixels from y=%d", n, minY)
	}
	// The affixes stay hidden until the field is focused or holds text.
	if _, ok := tt.Find("USD"); ok {
		t.Fatal("suffix shows while the empty field is blurred")
	}

	// Tab focuses the field, which floats its label and shows the affixes.
	tt.Key(0, ui.KeyTab)
	tt.Advance(900 * time.Millisecond)
	if !info.Focused {
		info, _ = tt.AccessByRole(ui.RoleTextField, "Email")
	}
	if !info.Focused {
		t.Fatal("Tab did not focus the filled field")
	}
	img = tt.Image()
	if px := img.RGBAAt(int(field.X+field.W/2), int(field.Y+field.H-1)); !colorMatches(px, cs.Primary, 30) {
		t.Fatalf("focused indicator %+v is not Primary", px)
	}
	if n, minY, _ := inkIn(tt, ui.Rect{X: field.X + textFieldPad, Y: field.Y + 2, W: 80, H: 20}, cs.Primary, 70); n < 5 || minY > int(field.Y+8) {
		t.Fatalf("filled label did not float: %d pixels from y=%d", n, minY)
	}
	if _, ok := tt.Find("$"); !ok {
		t.Fatal("prefix does not show when focused")
	}
	if _, ok := tt.Find("USD"); !ok {
		t.Fatal("suffix does not show when focused")
	}

	// Editing and Enter report through the field.
	tt.Type("12")
	if email != "12" || len(changed) == 0 || changed[len(changed)-1] != "12" {
		t.Fatalf("filled field value = %q, changes = %v", email, changed)
	}
	tt.Advance(600 * time.Millisecond)
	tt.Key(0, ui.KeyEnter)
	if len(submitted) != 1 || submitted[0] != "12" {
		t.Fatalf("filled field submissions = %v", submitted)
	}
}

// TestTextFieldErrorAccessibility checks the error palette, aria-invalid and the
// error message as the field's description.
func TestTextFieldErrorAccessibility(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	email := ""
	tt := fieldCanvas(th, func(c *ui.Context) {
		TextField(c, &email, TextFieldOptions{
			Variant:      TextFieldOutlined,
			Label:        "Email",
			Error:        true,
			ErrorMessage: "Enter a valid email",
		}).Width(280)
	})
	field, info := fieldRect(t, tt, "Email", textFieldHeight)
	if !info.Invalid || !strings.Contains(info.Description, "Enter a valid email") {
		t.Fatalf("error field invalid=%v description=%q", info.Invalid, info.Description)
	}
	if !topEdgeHas(tt, field.X+20, field.Y, cs.Error, 70) {
		t.Fatal("error border is not Error")
	}
	if n, _, _ := inkIn(tt, ui.Rect{X: field.X + 8, Y: field.Y + 8, W: 200, H: 40}, cs.Error, 70); n < 5 {
		t.Fatalf("error label inked %d pixels, want an Error label", n)
	}
	if _, ok := tt.Find("Enter a valid email"); !ok {
		t.Fatal("error message is not shown below the field")
	}
}

// TestTextFieldPasswordReadOnlyAndCounter checks the password and read-only
// modes, the character counter, and the trailing action slot.
func TestTextFieldPasswordReadOnlyAndCounter(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	password := ""
	trailing := 0
	tt := fieldCanvas(th, func(c *ui.Context) {
		TextField(c, &password, TextFieldOptions{
			Label:            "Password",
			Password:         true,
			MaxLength:        50,
			CharacterCounter: true,
			TrailingIcon:     mdicon.Visibility.Draw,
			TrailingLabel:    "Show password",
			OnTrailing:       func() { trailing++ },
		}).Width(280)
	})
	info := buttonInfo(t, tt, ui.RoleTextField, "Password")
	tt.ClickAt(info.Bounds.X+20, info.Bounds.Y+12)
	tt.Type("secret")
	tt.Advance(300 * time.Millisecond)
	if password != "secret" {
		t.Fatalf("password value = %q", password)
	}
	info, _ = tt.AccessByRole(ui.RoleTextField, "Password")
	if !info.Password {
		t.Fatal("assistive technology does not see the password field")
	}
	if _, ok := tt.Find("6/50"); !ok {
		t.Fatal("character counter does not show 6/50")
	}
	toggle := buttonInfo(t, tt, ui.RoleButton, "Show password")
	tt.ClickAt(center(toggle.Bounds))
	if trailing != 1 {
		t.Fatalf("trailing action ran %d times, want 1", trailing)
	}
	if password != "secret" {
		t.Fatal("trailing action changed the value")
	}

	fixed := "read only"
	ro := fieldCanvas(th, func(c *ui.Context) {
		TextField(c, &fixed, TextFieldOptions{Label: "Fixed", ReadOnly: true}).Width(280)
	})
	roInfo := buttonInfo(t, ro, ui.RoleTextField, "Fixed")
	if !roInfo.ReadOnly {
		t.Fatal("read-only field does not report itself read-only")
	}
	ro.ClickAt(roInfo.Bounds.X+20, roInfo.Bounds.Y+12)
	ro.Type("x")
	ro.Advance(300 * time.Millisecond)
	if fixed != "read only" {
		t.Fatalf("read-only field changed to %q", fixed)
	}
}

// TestTextFieldIMECompositionFloats checks that an input method's composition
// floats the label and opens the notch while the value is still empty.
func TestTextFieldIMECompositionFloats(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	text := ""
	tt := fieldCanvas(th, func(c *ui.Context) {
		TextField(c, &text, TextFieldOptions{
			Variant:     TextFieldOutlined,
			Label:       "Name",
			Placeholder: "Your name",
		}).Width(280)
	})
	field, info := fieldRect(t, tt, "Name", textFieldHeight)
	tt.ClickAt(info.Bounds.X+20, info.Bounds.Y+12)
	tt.Advance(900 * time.Millisecond)
	tt.Compose("ni", 2)
	if text != "" {
		t.Fatalf("value changed during composition: %q", text)
	}
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("label spring did not settle during composition")
	}
	if n, minY, _ := inkIn(tt, ui.Rect{X: field.X + textFieldPad, Y: field.Y - 10, W: 34, H: 22}, cs.Primary, 70); n < 5 || minY >= int(field.Y) {
		t.Fatalf("label did not float during composition: %d pixels from y=%d", n, minY)
	}
	if !topEdgeCanvas(tt, field.X+14, field.Y) {
		t.Fatal("the notch is not open during composition")
	}
	tt.Type("你")
	tt.Advance(600 * time.Millisecond)
	if text != "你" {
		t.Fatalf("committed composition = %q, want 你", text)
	}
}

// TestTextAreaMultilineAndScroll checks the text area's multiple lines, its
// fixed height, and that the wheel scrolls its text.
func TestTextAreaMultilineAndScroll(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	notes := ""
	tt := fieldCanvas(th, func(c *ui.Context) {
		TextArea(c, &notes, TextFieldOptions{
			Variant:        TextFieldOutlined,
			Label:          "Notes",
			Height:         120,
			SupportingText: "Markdown is fine",
		}).Width(280)
	})
	field, info := fieldRect(t, tt, "Notes", 120)
	if !info.Multiline {
		t.Fatal("text area does not report itself multiline")
	}
	tt.ClickAt(info.Bounds.X+20, info.Bounds.Y+12)
	tt.Type("first\nsecond\nthird")
	tt.Advance(600 * time.Millisecond)
	if !strings.Contains(notes, "first\nsecond\nthird") {
		t.Fatalf("text area value = %q", notes)
	}
	if _, ok := tt.Find("Markdown is fine"); !ok {
		t.Fatal("supporting text is not shown")
	}

	// Many lines overflow the 120 DIP field, and the wheel scrolls them.
	var many strings.Builder
	for i := 0; i < 30; i++ {
		if i > 0 {
			many.WriteByte('\n')
		}
		many.WriteString("line")
	}
	tt.Type("\n" + many.String())
	tt.Advance(600 * time.Millisecond)

	// The overflowing text area shows its vertical scroll bar inside its
	// right edge, in the theme's Scrollbar color.
	barInk := th.Colors.OnSurfaceVariant.Alpha(0.32).Over(canvasRGB)
	img := tt.Image()
	bars := 0
	for y := int(info.Bounds.Y); y < int(info.Bounds.Y+info.Bounds.H); y++ {
		for x := int(info.Bounds.X+info.Bounds.W) - 10; x < int(info.Bounds.X+info.Bounds.W)-2; x++ {
			if absColorDiff(img.RGBAAt(x, y), rgbOf(barInk)) <= 90 {
				bars++
			}
		}
	}
	if bars < 6 {
		t.Fatalf("text area shows no scroll bar: %d ink pixels near its right edge", bars)
	}

	before := inkSum(tt, field)
	tt.Wheel(field.X+field.W/2, field.Y+field.H/2, 0, 120)
	tt.Advance(600 * time.Millisecond)
	after := inkSum(tt, field)
	if after == before {
		t.Fatalf("wheel did not scroll the text area (ink sum %d)", before)
	}
	if tt.FramePending() {
		t.Fatal("text area left frames pending")
	}
}

// topEdgeHas reports whether the top edge of a field at x paints col within a
// few pixels around y, as strokes center on the boundary.
func topEdgeHas(tt *ui.Tester, x, y float32, col ui.Color, tol int) bool {
	img := tt.Image()
	for dy := -2; dy <= 1; dy++ {
		if colorMatches(img.RGBAAt(int(x), int(y)+dy), col, tol) {
			return true
		}
	}
	return false
}

// topEdgeCanvas reports whether the top edge of a field at x shows only the
// canvas, as it does in a label's notch.
func topEdgeCanvas(tt *ui.Tester, x, y float32) bool {
	img := tt.Image()
	for dy := -2; dy <= 1; dy++ {
		if !colorMatches(img.RGBAAt(int(x), int(y)+dy), canvasRGB, 8) {
			return false
		}
	}
	return true
}

// inkSum returns the sum of the pixels' channels inside r, a cheap fingerprint
// of what it shows.
func inkSum(tt *ui.Tester, r ui.Rect) int {
	img := tt.Image()
	sum := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			px := img.RGBAAt(x, y)
			sum += int(px.R) + int(px.G) + int(px.B)
		}
	}
	return sum
}

// TestTextFieldDisabledAndReduceMotion checks the disabled palette and that a
// ReduceMotion field floats its label and opens its notch without frames.
func TestTextFieldDisabledAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors

	value := ""
	disabled := fieldCanvas(th, func(c *ui.Context) {
		TextField(c, &value, TextFieldOptions{Label: "Email", Disabled: true}).Width(280)
	})
	info := buttonInfo(t, disabled, ui.RoleTextField, "Email")
	if !info.Disabled {
		t.Fatal("disabled field does not report itself disabled")
	}
	field, _ := fieldRect(t, disabled, "Email", textFieldHeight)
	want := cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)).Over(canvasRGB)
	if px := disabled.Image().RGBAAt(int(field.X+field.W/2), int(field.Y+8)); !colorMatches(px, want, 8) {
		t.Fatalf("disabled filled container %+v, want OnSurface 12%% over the canvas %+v", px, rgbOf(want))
	}
	disabled.ClickAt(info.Bounds.X+20, info.Bounds.Y+12)
	disabled.Key(0, ui.KeyTab)
	disabled.Type("x")
	disabled.Advance(600 * time.Millisecond)
	if value != "" {
		t.Fatalf("disabled field changed to %q", value)
	}
	if disabled.FramePending() {
		t.Fatal("disabled field left frames pending")
	}

	rm := fieldCanvas(th, func(c *ui.Context) {
		TextField(c, &value, TextFieldOptions{Variant: TextFieldOutlined, Label: "Email"}).Width(280)
	})
	rm.SetPreferences(ui.Preferences{ReduceMotion: true})
	rmInfo := buttonInfo(t, rm, ui.RoleTextField, "Email")
	rm.ClickAt(rmInfo.Bounds.X+20, rmInfo.Bounds.Y+12)
	if rm.FramePending() {
		t.Fatal("ReduceMotion field should converge on the first frame without pending frames")
	}
	rm.Advance(100 * time.Millisecond)
	if rm.FramePending() {
		t.Fatal("ReduceMotion field should not ask for animation frames")
	}
	rmField, _ := fieldRect(t, rm, "Email", textFieldHeight)
	if !topEdgeCanvas(rm, rmField.X+14, rmField.Y) {
		t.Fatal("ReduceMotion notch is not open")
	}
	if n, minY, _ := inkIn(rm, ui.Rect{X: rmField.X + textFieldPad, Y: rmField.Y - 10, W: 34, H: 22}, cs.Primary, 70); n < 5 || minY >= int(rmField.Y) {
		t.Fatalf("ReduceMotion label did not float: %d pixels from y=%d", n, minY)
	}
}

// TestTextFieldRTLNotch checks that the floated label and its notch anchor on
// the right when the window reads right to left.
func TestTextFieldRTLNotch(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	email := ""
	tt := fieldCanvas(th, func(c *ui.Context) {
		TextField(c, &email, TextFieldOptions{Variant: TextFieldOutlined, Label: "Email"}).Width(280)
	})
	tt.SetDirection(ui.RightToLeft)
	field, info := fieldRect(t, tt, "Email", textFieldHeight)
	tt.ClickAt(info.Bounds.X+info.Bounds.W-20, info.Bounds.Y+12)
	tt.Advance(900 * time.Millisecond)
	if !topEdgeCanvas(tt, field.X+field.W-15, field.Y) {
		t.Fatal("RTL notch is not open to the canvas")
	}
	if !topEdgeHas(tt, field.X+20, field.Y, cs.Primary, 70) {
		t.Fatal("RTL left border is not Primary")
	}
	if n, minY, _ := inkIn(tt, ui.Rect{X: field.X + field.W - textFieldPad - 34, Y: field.Y - 10, W: 34, H: 22}, cs.Primary, 70); n < 5 || minY >= int(field.Y) {
		t.Fatalf("RTL label did not float on the right: %d pixels from y=%d", n, minY)
	}
}

// TestTextFieldSiblingAndDynamicKeys checks that two sibling fields keep
// separate inputs with c.Key, and that a keyed list of text areas keeps its
// focus and editing identity when its items are reordered.
func TestTextFieldSiblingAndDynamicKeys(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	first, second := "", ""
	tt := fieldCanvas(th, func(c *ui.Context) {
		ui.Row(c).Gap(12).Children(func() {
			TextField(c.Key("first"), &first, TextFieldOptions{Label: "First"}).Width(160)
			TextField(c.Key("second"), &second, TextFieldOptions{Label: "Second"}).Width(160)
		})
	})
	a := buttonInfo(t, tt, ui.RoleTextField, "First")
	b := buttonInfo(t, tt, ui.RoleTextField, "Second")
	tt.ClickAt(a.Bounds.X+10, a.Bounds.Y+10)
	tt.Type("one")
	tt.ClickAt(b.Bounds.X+10, b.Bounds.Y+10)
	tt.Type("two")
	if first != "one" || second != "two" {
		t.Fatalf("sibling fields: first=%q second=%q", first, second)
	}

	type item struct {
		id, text string
	}
	items := []*item{{id: "alpha"}, {id: "beta"}}
	order := []int{0, 1}
	list := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(360, 420).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Gap(8).Children(func() {
					for _, idx := range order {
						it := items[idx]
						TextArea(c.Key(it.id), &it.text, TextFieldOptions{Label: it.id}).Width(280)
					}
				})
			})
		})
	}, 360, 420)
	alpha := buttonInfo(t, list, ui.RoleTextField, "alpha")
	list.ClickAt(alpha.Bounds.X+20, alpha.Bounds.Y+12)
	list.Type("hello")
	alpha, _ = list.AccessByRole(ui.RoleTextField, "alpha")
	if !alpha.Focused {
		t.Fatal("alpha text area did not take the focus")
	}

	order = []int{1, 0}
	list.Frame()
	list.Advance(100 * time.Millisecond)
	alpha, _ = list.AccessByRole(ui.RoleTextField, "alpha")
	if !alpha.Focused {
		t.Fatal("reordering the keyed text areas moved the focus off alpha")
	}
	if items[0].text != "hello" {
		t.Fatalf("alpha text = %q after reordering", items[0].text)
	}
	beta, _ := list.AccessByRole(ui.RoleTextField, "beta")
	list.ClickAt(beta.Bounds.X+20, beta.Bounds.Y+12)
	list.Type("world")
	if items[1].text != "world" || items[0].text != "hello" {
		t.Fatalf("reordered text areas edited the wrong items: %q %q", items[0].text, items[1].text)
	}
}

func TestTextFieldUTF16CounterErrorSupportAndTrailingKey(t *testing.T) {
	// 1. Non-BMP emoji counts as 2 UTF-16 code units, matching MaxLength.
	emojiVal := "😀"
	tt := fieldCanvas(nil, func(c *ui.Context) {
		TextField(c, &emojiVal, TextFieldOptions{
			Label:            "Emoji",
			MaxLength:        2,
			CharacterCounter: true,
			Error:            true,
			SupportingText:   "Use work email",
			Prefix:           "$",
			TrailingIcon:     mdicon.Clear.Draw,
			TrailingLabel:    "Clear",
			OnTrailing:       func() { emojiVal = "" },
		})
	})
	if _, ok := tt.Find("2/2"); !ok {
		t.Fatal("expected UTF-16 character counter \"2/2\" for astral emoji")
	}

	// 2. Error=true with empty ErrorMessage keeps SupportingText as accessible description.
	info := buttonInfo(t, tt, ui.RoleTextField, "Emoji")
	if !info.Invalid || !strings.Contains(info.Description, "Use work email") {
		t.Fatalf("expected invalid=true and supporting Description, got invalid=%v desc=%q", info.Invalid, info.Description)
	}

	// 3. Focusing Trailing button then activating OnTrailing clears value and removes Prefix/Suffix:
	// Trailing must retain focus because it is keyed explicitly.
	tt.Key(0, ui.KeyTab)
	for !tt.Focused("Clear") {
		tt.Key(0, ui.KeyTab)
	}
	emojiVal = ""
	tt.Frame()
	if !tt.Focused("Clear") {
		t.Fatal("TextField trailing button lost focus when affix disappeared")
	}
}

func TestTextFieldTextScaleAndNarrowLabelTruncation(t *testing.T) {
	thBig := ThemeFromSeed(ui.Hex("#6750A4"), ThemeOptions{TextScale: 1.5, ReduceMotion: true})
	val := ""
	ttBig := fieldCanvas(thBig, func(c *ui.Context) {
		ui.Column(c).Width(240).Children(func() {
			TextField(c, &val, TextFieldOptions{
				Label:          "Email",
				SupportingText: "Helper",
				Variant:        TextFieldOutlined,
			})
		})
	})
	scaledField, bigInfo := fieldRect(t, ttBig, "Email", textFieldHeight*1.5)
	if bigInfo.Bounds.H <= 24 {
		t.Fatalf("scaled TextField input did not expand for TextScale 1.5: h=%v", bigInfo.Bounds.H)
	}
	if !topEdgeHas(ttBig, scaledField.X+60, scaledField.Y, thBig.Colors.Outline, 70) {
		t.Fatalf("expected scaled outlined border at height %v (y=%v)", textFieldHeight*1.5, scaledField.Y)
	}
	if !topEdgeHas(ttBig, scaledField.X+60, scaledField.Y+scaledField.H-1, thBig.Colors.Outline, 70) {
		t.Fatalf("expected scaled outlined bottom border at height %v", scaledField.H)
	}

	// Narrow outlined TextField truncates long label with ellipsis cleanly inside bounds.
	ttNarrow := fieldCanvas(nil, func(c *ui.Context) {
		ui.Column(c).Width(100).Children(func() {
			TextField(c, &val, TextFieldOptions{
				Label:        "Very Long Descriptive Field Label",
				Variant:      TextFieldOutlined,
				TrailingIcon: mdicon.Search.Draw,
			})
		})
	})
	field, _ := fieldRect(t, ttNarrow, "Very Long Descriptive Field Label", textFieldHeight)
	th := ThemeFromSeed(ui.Hex("#6750A4"))
	// Zero label ink outside the right boundary of the narrow field box.
	outside := ui.Rect{X: field.X + field.W + 2, Y: field.Y - 12, W: 140, H: field.H + 20}
	if n, _, _ := inkIn(ttNarrow, outside, th.Colors.OnSurfaceVariant, 70); n > 0 {
		t.Fatalf("narrow TextField label spilled outside bounds: %d ink pixels to the right", n)
	}
}

func TestTextFieldEmptyLabelAndEmptyAffixSettles(t *testing.T) {
	val := ""
	tt := fieldCanvas(nil, func(c *ui.Context) {
		TextField(c, &val, TextFieldOptions{
			Placeholder: "Quick filter",
		})
	})
	info := buttonInfo(t, tt, ui.RoleTextField, "Quick filter")
	tt.ClickAt(info.Bounds.X+20, info.Bounds.Y+12)
	tt.Type("ok")
	if val != "ok" {
		t.Fatalf("empty-label TextField typed %q, want %q", val, "ok")
	}
	tt.Advance(400 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("empty-label TextField indicator did not settle cleanly")
	}
}

func TestTextFieldDensityRTLAndReduceMotion(t *testing.T) {
	for _, d := range []tokens.Density{tokens.DensityComfortable, tokens.DensityDesktop, tokens.DensityCompact} {
		th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{
			Density:      d,
			ReduceMotion: true,
		})
		val := ""
		var boxR ui.Rect
		tt := fieldCanvas(th, func(c *ui.Context) {
			TextField(c, &val, TextFieldOptions{
				Variant:      TextFieldOutlined,
				Label:        "Host",
				TrailingIcon: mdicon.Clear.Draw,
				OnTrailing:   func() { val = "" },
			}).Width(260).DrawOver(func(_ *ui.Painter, r ui.Rect) { boxR = r })
		})
		tt.SetDirection(ui.RightToLeft)
		tt.Frame()

		wantFieldH := float32(d.TextFieldHeight())
		// Outlined outer adds textFieldOuterTop (10 DIP) top padding.
		wantOuterH := wantFieldH + textFieldOuterTop
		if absF32(boxR.H-wantOuterH) > 0.5 {
			t.Fatalf("density %v: TextField outer height = %.1f, want %.1f", d, boxR.H, wantOuterH)
		}
		info := buttonInfo(t, tt, ui.RoleTextField, "Host")
		tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
		tt.Type("127.0.0.1")
		if val != "127.0.0.1" {
			t.Fatalf("density %v: typed value = %q, want 127.0.0.1", d, val)
		}
		if tt.FramePending() {
			t.Fatalf("density %v: ReduceMotion TextField left pending frames", d)
		}
	}
}
