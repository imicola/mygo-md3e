package docview_test

import (
	"fmt"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/docview"
)

func buildLongGuideMarkdown() string {
	var b strings.Builder
	b.WriteString("# Overview\n\n")
	b.WriteString(strings.Repeat("Introduction paragraph with clear prose.\n\n", 14))
	b.WriteString("## Architecture\n\n")
	b.WriteString(strings.Repeat("Architecture details and package boundaries.\n\n", 16))
	b.WriteString("### Examples\n\n")
	b.WriteString(strings.Repeat("First examples block for testing duplicate slugs.\n\n", 14))
	b.WriteString("### Examples\n\n")
	b.WriteString(strings.Repeat("Second examples block with unique slug disambiguation.\n\n", 16))
	b.WriteString("## Deployment\n\n")
	b.WriteString(strings.Repeat("Final deployment notes and verification steps.\n\n", 14))
	return b.String()
}

func TestExtractOutlineAndUniqueSlugs(t *testing.T) {
	items := docview.ExtractOutline(buildLongGuideMarkdown())
	if len(items) != 5 {
		t.Fatalf("ExtractOutline returned %d items, want 5: %+v", len(items), items)
	}
	wantSlugs := []string{"overview", "architecture", "examples", "examples-1", "deployment"}
	wantLevels := []int{1, 2, 3, 3, 2}
	for i, it := range items {
		if it.Slug != wantSlugs[i] || it.Level != wantLevels[i] {
			t.Fatalf("item %d: got slug=%q level=%d, want slug=%q level=%d", i, it.Slug, it.Level, wantSlugs[i], wantLevels[i])
		}
	}
}

func TestOutlineWithSupportingPaneScrollSync(t *testing.T) {
	src := buildLongGuideMarkdown()
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	var docState docview.State
	var scroll ui.ScrollState
	showSupporting := true

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			md3e.SupportingPane(c.Key("pane"),
				func() {
					ui.Scroll(c.Key("doc-scroll")).Fill().TrackScroll(&scroll).Children(func() {
						ui.Column(c).FillWidth().Padding(16).Children(func() {
							docview.Markdown(c.Key("doc"), src, docview.MarkdownOptions{
								State:  &docState,
								Scroll: &scroll,
							})
						})
					})
				},
				func() {
					ui.Column(c).Fill().Padding(12).Children(func() {
						docview.Outline(c.Key("toc"), &docState, docview.OutlineOptions{
							ShowHeader: true,
							Scroll:     &scroll,
						})
					})
				},
				md3e.SupportingPaneOptions{
					ShowSupporting: &showSupporting,
				},
			)
		}).Fill()
	}, 960, 420)

	// Settle initial layout measurement.
	tt.Advance(200 * time.Millisecond)

	items := docState.Items()
	if len(items) != 5 {
		t.Fatalf("State.Items() count = %d, want 5", len(items))
	}
	for i, it := range items {
		if !it.HasOffset {
			t.Fatalf("heading %d (%q) missing measured OffsetY", i, it.Title)
		}
		if i > 0 && it.OffsetY <= items[i-1].OffsetY {
			t.Fatalf("heading %d OffsetY=%.1f did not increase past %.1f", i, it.OffsetY, items[i-1].OffsetY)
		}
	}

	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Outline"); !ok {
		t.Fatal("missing RoleNavigation Outline landmark")
	}
	if !tt.HasText("On this page") {
		t.Fatal("missing Outline header")
	}
	ovBtn, ok := tt.AccessByRole(ui.RoleButton, "Overview")
	if !ok || ovBtn.Current != ui.CurrentLocation {
		t.Fatalf("expected Overview active initially (CurrentLocation), got ok=%v current=%v", ok, ovBtn.Current)
	}

	// Click "Deployment" in Outline -> smoothly scrolls doc and settles with Deployment active.
	if err := tt.Click("Deployment"); err != nil {
		t.Fatalf("click Deployment in Outline: %v", err)
	}
	for range 40 {
		tt.Advance(30 * time.Millisecond)
		if !tt.FramePending() {
			break
		}
	}
	if tt.FramePending() {
		t.Fatal("Outline scroll/indicator animation did not settle")
	}
	if scroll.Y <= 0 {
		t.Fatalf("expected scroll.Y > 0 after clicking Deployment, got %.1f", scroll.Y)
	}
	if got := docState.ActiveSlug(); got != "deployment" {
		t.Fatalf("ActiveSlug after jump = %q, want deployment", got)
	}
	depBtn, ok := tt.AccessByRole(ui.RoleButton, "Deployment")
	if !ok || depBtn.Current != ui.CurrentLocation {
		t.Fatalf("expected Deployment button CurrentLocation after jump, got ok=%v current=%v", ok, depBtn.Current)
	}

	// Manual scroll to Architecture offset updates Outline active item automatically.
	archY := items[1].OffsetY
	scroll.Y = archY + 8
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("Outline indicator spring did not settle after scroll")
	}
	if got := docState.ActiveSlug(); got != "architecture" {
		t.Fatalf("ActiveSlug after manual scroll to %.1f = %q, want architecture", scroll.Y, got)
	}
	archBtn, ok := tt.AccessByRole(ui.RoleButton, "Architecture")
	if !ok || archBtn.Current != ui.CurrentLocation {
		t.Fatalf("expected Architecture CurrentLocation after manual scroll, got ok=%v current=%v", ok, archBtn.Current)
	}
}

func TestOutlineAnchorFallbackWithoutScrollStateAndRTL(t *testing.T) {
	src := buildLongGuideMarkdown()
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{ReduceMotion: true})
	var docState docview.State

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Row(c).Fill().Direction(ui.RightToLeft).Children(func() {
				ui.Scroll(c.Key("scroll")).Grow(1).FillHeight().Children(func() {
					docview.Markdown(c.Key("doc"), src, docview.MarkdownOptions{
						State: &docState,
					})
				})
				ui.Box(c).Width(220).FillHeight().Padding(8).Children(func() {
					docview.Outline(c.Key("toc"), &docState, docview.OutlineOptions{
						MaxDepth: 3,
					})
				})
			})
		}).Fill()
	}, 800, 320)

	// Jump to the second duplicate "### Examples" heading ("examples-1") via State.JumpTo without ScrollState.
	docState.JumpTo("examples-1")
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("ReduceMotion outline anchor jump left pending frames")
	}
	if !tt.HasText("Second examples block") {
		t.Fatalf("expected second Examples section scrolled into view, texts=%q", tt.Texts())
	}

	// Verify RTL indentation pushes H2 text glyphs inward (left from right edge)
	// relative to H1, and active indicator bar sits on the right edge in RTL.
	// Query RoleButton so Markdown's off-screen heading nodes aren't matched first.
	h1Btn, ok1 := tt.AccessByRole(ui.RoleButton, "Overview")
	h2Btn, ok2 := tt.AccessByRole(ui.RoleButton, "Architecture")
	if !ok1 || !ok2 {
		t.Fatalf("missing outline RoleButton bounds: ok1=%v ok2=%v", ok1, ok2)
	}
	h1Rect, h2Rect := h1Btn.Bounds, h2Btn.Bounds
	rgbaClose := func(a, b color.RGBA, tol int) bool {
		diff := func(x, y uint8) int {
			d := int(x) - int(y)
			if d < 0 {
				return -d
			}
			return d
		}
		return diff(a.R, b.R) <= tol && diff(a.G, b.G) <= tol && diff(a.B, b.B) <= tol
	}
	img := tt.Image()
	rightmostInkX := func(rect ui.Rect) int {
		bg := img.RGBAAt(int(rect.X+rect.W-6), int(rect.Y+2))
		for x := int(rect.X + rect.W - 6); x >= int(rect.X+4); x-- {
			for y := int(rect.Y + 4); y <= int(rect.Y+rect.H-4); y++ {
				if !rgbaClose(img.RGBAAt(x, y), bg, 6) {
					return x
				}
			}
		}
		return -1
	}
	h1InkX := rightmostInkX(h1Rect)
	h2InkX := rightmostInkX(h2Rect)
	if h1InkX < 0 || h2InkX < 0 || !(h2InkX < h1InkX-6) {
		t.Fatalf("expected RTL indentation to shift H2 ink left from right edge (h2InkX=%d, h1InkX=%d, h1=%+v, h2=%+v)",
			h2InkX, h1InkX, h1Rect, h2Rect)
	}

	// Verify active indicator bar is mirrored to the right (Start) edge in RTL.
	depBtn, okDep := tt.AccessByRole(ui.RoleButton, "Deployment")
	if !okDep {
		t.Fatal("missing Deployment outline RoleButton")
	}
	depRect := depBtn.Bounds
	docState.JumpTo("deployment")
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("ReduceMotion JumpTo(deployment) left pending frames")
	}
	rtlImg := tt.Image()
	barPx := rtlImg.RGBAAt(int(depRect.X+depRect.W-2), int(depRect.Y+depRect.H/2))
	pri := color.RGBA{R: th.Colors.Primary.R, G: th.Colors.Primary.G, B: th.Colors.Primary.B, A: 255}
	if !rgbaClose(barPx, pri, 10) {
		t.Fatalf("expected RTL active bar on right edge (%+v) to match Primary (%+v) at dep=%+v", barPx, pri, depRect)
	}
}

func TestOutlineStandaloneItemsAndEmpty(t *testing.T) {
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	active := "sec-1"
	var clicked []string
	items := []docview.OutlineItem{
		{Title: "First", Slug: "sec-1", Level: 1},
		{Title: "Second", Slug: "sec-2", Level: 2},
	}

	showEmpty := false
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			if showEmpty {
				docview.Outline(c.Key("empty"), nil, docview.OutlineOptions{
					EmptyText: "No headings",
				})
				return
			}
			docview.Outline(c.Key("custom"), nil, docview.OutlineOptions{
				Items:    items,
				Active:   &active,
				OnSelect: func(it docview.OutlineItem) { clicked = append(clicked, it.Slug) },
			})
		})
	}, 300, 200)

	if err := tt.Click("Second"); err != nil {
		t.Fatalf("click Second: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if active != "sec-2" || fmt.Sprint(clicked) != "[sec-2]" {
		t.Fatalf("got active=%q clicked=%v, want sec-2", active, clicked)
	}
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	if active != "sec-1" || fmt.Sprint(clicked) != "[sec-2 sec-1]" {
		t.Fatalf("keyboard Up+Enter got active=%q clicked=%v, want sec-1", active, clicked)
	}

	showEmpty = true
	tt.Frame()
	if !tt.HasText("No headings") {
		t.Fatal("expected EmptyText placeholder when outline is empty")
	}
}
