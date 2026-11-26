package docview_test

import (
	"fmt"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"

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
	items = docview.ExtractOutline("# API\n\n# API\n\n# API-1\n\n# API\n")
	for i, want := range []string{"api", "api-1", "api-1-1", "api-2"} {
		if got := items[i].Slug; got != want {
			t.Errorf("heading %d: slug = %q, want %q", i, got, want)
		}
	}
}

func TestMarkdownOutlineLayoutSettles(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		stream    bool
		custom    bool
	}{
		{name: "no headings", src: "A paragraph."},
		{name: "open heading", src: "A paragraph.\n\n# Growing heading", stream: true},
		{name: "custom heading", src: "# Custom\n\n## Measured\n", custom: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var state docview.State
			src, builds := tc.src, 0
			tt := ui.NewTester(func(c *ui.Context) {
				builds++
				opts := docview.MarkdownOptions{State: &state}
				if tc.custom {
					opts.Block = func(c *ui.Context, _ *parser.Document, i int) bool {
						if i != 1 {
							return false
						}
						ui.Text(c, "Custom heading")
						return true
					}
				}
				if tc.stream {
					docview.MarkdownStream(c, src, opts)
				} else {
					docview.Markdown(c, src, opts)
				}
			}, 360, 240)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			tt.Advance(200 * time.Millisecond)
			if tt.FramePending() {
				t.Errorf("layout did not settle after %d builds", builds)
			}
			before := builds
			tt.Advance(200 * time.Millisecond)
			if builds != before {
				t.Errorf("idle view rebuilt %d times", builds-before)
			}
			if tc.custom {
				items := state.Items()
				if len(items) != 2 || items[0].HasOffset || !items[1].HasOffset {
					t.Errorf("unexpected custom and default heading offsets: %+v", items)
				}
			}
			if tc.stream {
				src += "\n\nDone.\n"
				tt.Frame()
				items := state.Items()
				if tt.FramePending() || len(items) != 1 || !items[0].HasOffset {
					t.Errorf("completed stream did not settle and measure its heading: %+v", items)
				}
			}
		})
	}
}

func TestMarkdownOutlineRemeasuresAfterResize(t *testing.T) {
	var state docview.State
	src := "# Start\n\n" + strings.Repeat("Wrapping words in a long paragraph. ", 20) + "\n\n## Next\n"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			docview.Markdown(c, src, docview.MarkdownOptions{State: &state})
		})
	}, 640, 360)
	before := state.Items()[1]
	tt.SetSize(250, 360)
	tt.Advance(200 * time.Millisecond)
	after := state.Items()[1]
	if !before.HasOffset || !after.HasOffset || after.OffsetY <= before.OffsetY || tt.FramePending() {
		t.Fatalf("outline did not remeasure and settle: before=%+v after=%+v", before, after)
	}
}

func TestMarkdownAnchorUsesDisambiguatedOutlineSlug(t *testing.T) {
	var state docview.State
	src := "# API\n\n" + strings.Repeat("First section.\n\n", 16) +
		"# API\n\n" + strings.Repeat("Repeated section.\n\n", 16) +
		"# API-1\n\nLiteral anchor paragraph.\n\n" + strings.Repeat("Final section.\n\n", 16)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			docview.Markdown(c, src, docview.MarkdownOptions{State: &state, Anchor: "api-1-1"})
		})
	}, 360, 180)
	tt.Advance(500 * time.Millisecond)
	if !tt.HasText("Literal anchor paragraph.") {
		t.Fatalf("disambiguated anchor did not reach its section, visible text: %v", tt.Texts())
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

func TestOutlineCrossDocumentSwitchClearsStaleJump(t *testing.T) {
	docA := "# Alpha One\n\n" + strings.Repeat("Intro A paragraph.\n\n", 10) +
		"## Alpha Two\n\n" + strings.Repeat("Body A2 paragraph.\n\n", 10) +
		"## Alpha Three\n\n" + strings.Repeat("Body A3 paragraph.\n\n", 10)
	docB := "# Beta One\n\n" + strings.Repeat("Intro B paragraph.\n\n", 10) +
		"## Beta Two\n\n" + strings.Repeat("Body B2 paragraph.\n\n", 10) +
		"## Beta Three\n\n" + strings.Repeat("Body B3 paragraph.\n\n", 10)

	// 1. With ScrollState attached: jumping to Alpha Three then switching to Doc B resets active heading to Beta One.
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{ReduceMotion: true})
	var docState docview.State
	var scroll ui.ScrollState
	activeDoc := docA

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Row(c).Fill().Children(func() {
				ui.Scroll(c.Key("scroll")).Grow(1).FillHeight().TrackScroll(&scroll).Children(func() {
					docview.Markdown(c.Key("doc"), activeDoc, docview.MarkdownOptions{
						State:  &docState,
						Scroll: &scroll,
					})
				})
				ui.Box(c).Width(220).FillHeight().Children(func() {
					docview.Outline(c.Key("toc"), &docState, docview.OutlineOptions{
						Scroll: &scroll,
					})
				})
			})
		}).Fill()
	}, 720, 320)

	tt.Advance(200 * time.Millisecond)
	if err := tt.Click("Alpha Three"); err != nil {
		t.Fatalf("click Alpha Three: %v", err)
	}
	tt.Advance(100 * time.Millisecond)
	if got := docState.ActiveSlug(); got != "alpha-three" {
		t.Fatalf("ActiveSlug after jump in Doc A = %q, want alpha-three", got)
	}

	// Switch to Doc B (which also has 3 headings) and reset scroll to top.
	activeDoc = docB
	scroll = ui.ScrollState{}
	tt.Frame()
	if got := docState.ActiveIndex(); got != 0 {
		t.Fatalf("first frame after switching to Doc B had stale ActiveIndex=%d (%q), want 0", got, docState.ActiveSlug())
	}
	tt.Advance(200 * time.Millisecond)
	if got := docState.ActiveSlug(); got != "beta-one" {
		t.Fatalf("ActiveSlug in Doc B = %q, want beta-one", got)
	}

	// 2. Without ScrollState (anchor fallback): jumping to Alpha Three then switching to Doc B clears pending jump.
	var noScrollState docview.State
	activeDoc2 := docA
	tt2 := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Row(c).Fill().Children(func() {
				ui.Scroll(c.Key("scroll2")).Grow(1).FillHeight().Children(func() {
					docview.Markdown(c.Key("doc2"), activeDoc2, docview.MarkdownOptions{
						State: &noScrollState,
					})
				})
				ui.Box(c.Key("toc2")).Width(220).FillHeight().Children(func() {
					docview.Outline(c.Key("toc2"), &noScrollState)
				})
			})
		}).Fill()
	}, 720, 320)

	tt2.Advance(100 * time.Millisecond)
	if err := tt2.Click("Alpha Three"); err != nil {
		t.Fatalf("click Alpha Three (anchor mode): %v", err)
	}
	tt2.Frame()
	if got := noScrollState.ActiveSlug(); got != "alpha-three" {
		t.Fatalf("anchor-mode ActiveSlug after jump = %q, want alpha-three", got)
	}
	activeDoc2 = docB
	tt2.Frame()
	if got := noScrollState.ActiveSlug(); got != "beta-one" {
		t.Fatalf("anchor-mode ActiveSlug after switching to Doc B = %q, want beta-one", got)
	}
}
