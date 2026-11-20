package md3e

import (
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
)

// chipCanvas builds a 640x360 white canvas holding build's chips.
func chipCanvas(th *Theme, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(640, 360).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				build(c)
			})
		})
	}, 640, 360)
}

// chipBox returns the access box of a chip by role and name.
func chipBox(t *testing.T, tt *ui.Tester, role ui.Role, name string) ui.AccessInfo {
	t.Helper()
	info := buttonInfo(t, tt, role, name)
	if info.Bounds.H != chipHeight {
		t.Fatalf("%q chip height = %v, want %d", name, info.Bounds.H, chipHeight)
	}
	return info
}

// TestChipLooksAndElevation checks the four chips' 32 DIP geometry, the outlined
// outline, the elevated SurfaceContainerLow fill and shadow, and their
// accessible roles and names.
func TestChipLooksAndElevation(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	tt := chipCanvas(th, func(c *ui.Context) {
		ui.Column(c).Gap(24).Children(func() {
			AssistChip(c, "Add", ChipOptions{LeadingIcon: mdicon.Add.Draw})
			AssistChip(c, "Starred", ChipOptions{Elevated: true, LeadingIcon: mdicon.Star.Draw})
			FilterChip(c, new(bool), "Unread", FilterChipOptions{LeadingIcon: mdicon.Star.Draw})
			InputChip(c, "Alice", InputChipOptions{LeadingIcon: mdicon.Star.Draw, Avatar: true})
			SuggestionChip(c, "Try a sample")
		})
	})

	flat := chipBox(t, tt, ui.RoleButton, "Add")
	elevated := chipBox(t, tt, ui.RoleButton, "Starred")
	filter := chipBox(t, tt, ui.RoleCheckBox, "Unread")
	body := chipBox(t, tt, ui.RoleButton, "Alice")
	closeInfo := buttonInfo(t, tt, ui.RoleButton, "Remove Alice")
	suggestion := chipBox(t, tt, ui.RoleButton, "Try a sample")

	// Outlined: a transparent container over the canvas, with an
	// OutlineVariant border along the top edge.
	img := tt.Image()
	probe := func(x, y float32) color.RGBA { return img.RGBAAt(int(x), int(y)) }
	if px := probe(flat.Bounds.X+flat.Bounds.W/2, flat.Bounds.Y+6); !colorMatches(px, canvasRGB, 4) {
		t.Fatalf("outlined chip fill %+v is not the canvas", px)
	}
	borderHit := false
	for dy := 0; dy <= 1; dy++ {
		if colorMatches(probe(flat.Bounds.X+flat.Bounds.W/2, flat.Bounds.Y+float32(dy)), cs.OutlineVariant, 40) {
			borderHit = true
		}
	}
	if !borderHit {
		t.Fatalf("outlined chip top edge is not an OutlineVariant border")
	}

	// Elevated: SurfaceContainerLow inside, no border, and a Level1 shadow
	// below the container.
	wantLow := cs.SurfaceContainerLow
	if wantLow.A == 0 {
		wantLow = SurfaceColorForElevation(th, 1)
	}
	if px := probe(elevated.Bounds.X+elevated.Bounds.W/2, elevated.Bounds.Y+6); !colorMatches(px, wantLow, 5) {
		t.Fatalf("elevated chip fill %+v is not SurfaceContainerLow %+v", px, rgbOf(wantLow))
	}
	shadow := probe(elevated.Bounds.X+elevated.Bounds.W/2, elevated.Bounds.Y+chipHeight+2)
	s := int(shadow.R) + int(shadow.G) + int(shadow.B)
	if s >= 3*255-4 {
		t.Fatalf("elevated chip casts no Level1 shadow below it: %+v", shadow)
	}

	// The input chip's close button follows its body, inside the same 32 DIP
	// row, and is separate from it.
	if closeInfo.Bounds.X < body.Bounds.X {
		t.Fatalf("close button at %v is before the body %v", closeInfo.Bounds, body.Bounds)
	}
	if suggestion.Bounds.Y <= filter.Bounds.Y {
		t.Fatal("chips did not stack in the column")
	}
}

// TestFilterChipToggleAndCheck checks that a click and Space toggle *selected,
// swap the container to SecondaryContainer without a border, draw the check
// mark in the leading slot, report the checked state, and settle.
func TestFilterChipToggleAndCheck(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	selected := false
	var changes []bool
	tt := chipCanvas(th, func(c *ui.Context) {
		FilterChip(c, &selected, "Unread", FilterChipOptions{
			OnChanged: func(on bool) { changes = append(changes, on) },
		})
	})

	info := chipBox(t, tt, ui.RoleCheckBox, "Unread")
	if info.Checked {
		t.Fatal("filter chip starts selected")
	}
	cx, cy := center(info.Bounds)
	tt.ClickAt(cx, cy)
	if !selected {
		t.Fatal("click did not select the filter chip")
	}
	if len(changes) != 1 || !changes[0] {
		t.Fatalf("OnChanged got %v, want one true", changes)
	}
	if !tt.FramePending() {
		t.Fatal("selecting a filter chip should start its springs")
	}
	tt.ClickAt(620, 340)
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("filter chip springs did not settle after selecting")
	}
	info = chipBox(t, tt, ui.RoleCheckBox, "Unread")
	if !info.Checked {
		t.Fatal("assistive technology does not see the filter chip selected")
	}

	img := tt.Image()
	// SecondaryContainer fill away from the label and the check.
	px := img.RGBAAt(int(info.Bounds.X+info.Bounds.W-10), int(info.Bounds.Y+info.Bounds.H/2))
	if !colorMatches(px, cs.SecondaryContainer, 6) {
		t.Fatalf("selected filter chip fill %+v is not SecondaryContainer", px)
	}
	// The check mark inks the leading slot in OnSecondaryContainer.
	ink := 0
	for y := int(info.Bounds.Y + 7); y < int(info.Bounds.Y+info.Bounds.H-7); y++ {
		for x := int(info.Bounds.X + 16); x < int(info.Bounds.X+16+chipIconSize); x++ {
			if colorMatches(img.RGBAAt(x, y), cs.OnSecondaryContainer, 24) {
				ink++
			}
		}
	}
	if ink < 6 {
		t.Fatalf("selected filter chip shows %d check pixels, want a visible check", ink)
	}

	// Space toggles it back while it has the keyboard focus.
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	if selected {
		t.Fatal("Space did not deselect the filter chip")
	}
	tt.ClickAt(620, 340)
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("filter chip springs did not settle after deselecting")
	}
	info = chipBox(t, tt, ui.RoleCheckBox, "Unread")
	px = tt.Image().RGBAAt(int(info.Bounds.X+info.Bounds.W-10), int(info.Bounds.Y+info.Bounds.H/2))
	if !colorMatches(px, canvasRGB, 4) {
		t.Fatalf("deselected filter chip fill %+v is not the outlined canvas", px)
	}
}

// TestInputChipIndependentHitRegions checks that the body and the close button
// report only their own clicks, with the close button accessible by name.
func TestInputChipIndependentHitRegions(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	bodyClicks, removeClicks := 0, 0
	tt := chipCanvas(th, func(c *ui.Context) {
		InputChip(c, "Alice", InputChipOptions{
			LeadingIcon: mdicon.Star.Draw,
			Avatar:      true,
			OnClick:     func() { bodyClicks++ },
			OnRemove:    func() { removeClicks++ },
		})
	})

	body := chipBox(t, tt, ui.RoleButton, "Alice")
	closeInfo := buttonInfo(t, tt, ui.RoleButton, "Remove Alice")
	if closeInfo.Bounds.W != chipCloseSize || closeInfo.Bounds.H != chipCloseSize {
		t.Fatalf("close button = %v, want %d×%d", closeInfo.Bounds, chipCloseSize, chipCloseSize)
	}
	bx, by := center(body.Bounds)
	tt.ClickAt(bx, by)
	if bodyClicks != 1 || removeClicks != 0 {
		t.Fatalf("body click reported body=%d remove=%d", bodyClicks, removeClicks)
	}
	cx, cy := center(closeInfo.Bounds)
	tt.ClickAt(cx, cy)
	if bodyClicks != 1 || removeClicks != 1 {
		t.Fatalf("close click reported body=%d remove=%d", bodyClicks, removeClicks)
	}
	// Clicking the body's right end, next to the close button, stays the body.
	tt.ClickAt(body.Bounds.X+body.Bounds.W-4, by)
	if bodyClicks != 2 || removeClicks != 1 {
		t.Fatalf("body edge click reported body=%d remove=%d", bodyClicks, removeClicks)
	}
}

// TestChipGroupWrapDisabledAndReduceMotion checks that a narrow group flows its
// chips onto a second line, that a disabled chip ignores clicks, and that
// ReduceMotion settles the filter chip without frames.
func TestChipGroupWrapDisabledAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	disabled := true
	tt := chipCanvas(th, func(c *ui.Context) {
		ChipGroup(c, func() {
			FilterChip(c, new(bool), "One")
			FilterChip(c, new(bool), "Two")
			FilterChip(c, &disabled, "Three", FilterChipOptions{Disabled: true})
			FilterChip(c, new(bool), "Four")
		}, ChipGroupOptions{Label: "Tags", RowGap: 10, ColumnGap: 10}).Width(200)
	})
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Tags"); !ok {
		t.Fatal("chip group not in the access tree")
	}
	one := chipBox(t, tt, ui.RoleCheckBox, "One")
	two := chipBox(t, tt, ui.RoleCheckBox, "Two")
	three := chipBox(t, tt, ui.RoleCheckBox, "Three")
	four := chipBox(t, tt, ui.RoleCheckBox, "Four")
	if two.Bounds.Y != one.Bounds.Y {
		t.Fatalf("One and Two wrapped separately: %v and %v", one.Bounds, two.Bounds)
	}
	line2 := three.Bounds.Y
	if four.Bounds.Y != line2 {
		line2 = four.Bounds.Y
	}
	if line2 <= one.Bounds.Y {
		t.Fatalf("chips did not wrap onto a second line: %v", []ui.Rect{one.Bounds, two.Bounds, three.Bounds, four.Bounds})
	}
	// The second line starts at the group's start again.
	if absF32(three.Bounds.X-one.Bounds.X) > 1 && absF32(four.Bounds.X-one.Bounds.X) > 1 {
		t.Fatalf("the second line does not start at the group's start: %v", []ui.Rect{three.Bounds, four.Bounds})
	}
	cx, cy := center(three.Bounds)
	tt.ClickAt(cx, cy)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	tt.Advance(600 * time.Millisecond)
	if !disabled {
		t.Fatal("disabled filter chip changed its value")
	}
	if tt.FramePending() {
		t.Fatal("disabled chip left frames pending")
	}

	sel := false
	rm := chipCanvas(th, func(c *ui.Context) {
		FilterChip(c, &sel, "Unread")
	})
	rm.SetPreferences(ui.Preferences{ReduceMotion: true})
	info := chipBox(t, rm, ui.RoleCheckBox, "Unread")
	rm.ClickAt(center(info.Bounds))
	if !sel {
		t.Fatal("ReduceMotion click did not select the filter chip")
	}
	rm.ClickAt(620, 340)
	rm.Frame()
	if rm.FramePending() {
		t.Fatal("ReduceMotion filter chip should not ask for frames")
	}
	info = chipBox(t, rm, ui.RoleCheckBox, "Unread")
	px := rm.Image().RGBAAt(int(info.Bounds.X+info.Bounds.W-10), int(info.Bounds.Y+info.Bounds.H/2))
	if !colorMatches(px, th.Colors.SecondaryContainer, 6) {
		t.Fatalf("ReduceMotion selected fill %+v is not SecondaryContainer", px)
	}
}

// TestChipRTLMirrorsIcons checks that a leading icon paints at the leading edge
// of the chip, mirrored right to left, before the label.
func TestChipRTLMirrorsIcons(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := chipCanvas(th, func(c *ui.Context) {
		AssistChip(c, "Add", ChipOptions{LeadingIcon: mdicon.Add.Draw, Label: "Add event"})
	})
	info := chipBox(t, tt, ui.RoleButton, "Add event")
	label, ok := tt.Find("Add")
	if !ok {
		t.Fatal("chip label not found")
	}
	iconCenter := iconInkCenter(t, tt, info.Bounds, th.Colors.Primary)
	if iconCenter <= 0 {
		t.Fatal("leading icon not painted in Primary")
	}
	if iconCenter >= label.X+label.W/2 {
		t.Fatalf("LTR icon center %v is not before the label center %v", iconCenter, label.X+label.W/2)
	}

	tt.SetDirection(ui.RightToLeft)
	info = chipBox(t, tt, ui.RoleButton, "Add event")
	label, ok = tt.Find("Add")
	if !ok {
		t.Fatal("chip label not found right to left")
	}
	iconCenter = iconInkCenter(t, tt, info.Bounds, th.Colors.Primary)
	if iconCenter <= 0 {
		t.Fatal("RTL leading icon not painted in Primary")
	}
	if iconCenter <= label.X+label.W/2 {
		t.Fatalf("RTL icon center %v is not after the label center %v", iconCenter, label.X+label.W/2)
	}
}

// iconInkCenter returns the mean x of the pixels of col inside box, or -1 when
// the icon paints none.
func iconInkCenter(t *testing.T, tt *ui.Tester, box ui.Rect, col ui.Color) float32 {
	t.Helper()
	img := tt.Image()
	sum, n := 0, 0
	for y := int(box.Y + 4); y < int(box.Y+box.H-4); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if colorMatches(img.RGBAAt(x, y), col, 60) {
				sum += x
				n++
			}
		}
	}
	if n == 0 {
		return -1
	}
	return float32(sum) / float32(n)
}
