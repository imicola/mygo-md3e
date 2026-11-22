package docview_test

import (
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/docview"
	"github.com/imicola/mygo-md3e/tokens"
)

func TestMarkdownThemeMappingAndDensity(t *testing.T) {
	comfort := md3e.ThemeFromSeed(ui.Hex("#6750A4"), md3e.ThemeOptions{
		MonoFont: "JetBrains Mono",
	})
	desktop := md3e.ThemeFromSeed(ui.Hex("#0061A4"), md3e.ThemeOptions{
		Dark:    true,
		Density: tokens.DensityDesktop,
	})
	compact := md3e.ThemeFromSeed(ui.Hex("#2E6B3E"), md3e.ThemeOptions{
		Density: tokens.DensityCompact,
	})

	mc := docview.MarkdownThemeFor(comfort)
	md := docview.MarkdownThemeFor(desktop)
	mk := docview.MarkdownThemeFor(compact)

	if mc.FontSize != 16 || md.FontSize != 14 || mk.FontSize != 13 {
		t.Fatalf("unexpected body sizes across densities: %v %v %v", mc.FontSize, md.FontSize, mk.FontSize)
	}
	if mc.CodeFont != "JetBrains Mono" || md.CodeFont != "monospace" {
		t.Fatalf("unexpected code fonts: %q %q", mc.CodeFont, md.CodeFont)
	}
	if mc.Text != comfort.Colors.OnSurface || mc.Link != comfort.Colors.Primary {
		t.Fatalf("expected MD3E OnSurface/Primary colors, got text=%v link=%v", mc.Text, mc.Link)
	}
	if md.TableHeader != desktop.Colors.SurfaceContainerHigh || md.TableStripe != desktop.Colors.SurfaceContainerLow {
		t.Fatalf("expected SurfaceContainerHigh/Low table colors")
	}
	if mc.CodeColors == nil || md.CodeColors == nil {
		t.Fatal("expected non-nil CodeColors")
	}
	// Heading hierarchy monotonically decreases from H1 to H6.
	for i := 0; i < 5; i++ {
		if mc.Headings[i].Size < mc.Headings[i+1].Size {
			t.Fatalf("H%d size %.3f < H%d size %.3f", i+1, mc.Headings[i].Size, i+2, mc.Headings[i+1].Size)
		}
	}
}

func TestMarkdownBlocksCodeBlockCalloutAndCopy(t *testing.T) {
	const src = `# Documentation

Read the [guide](https://go.dev) and inline ` + "`config.json`" + `.

> A standard quote.

> [!NOTE]
> Useful context with **bold** text.

> [!WARNING]
> Check the threshold before deploying.

| Name | Status |
| :--- | :--- |
| API | Ready |

` + "```go\nfunc hello() {}\n```"

	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	snack := &md3e.SnackbarState{}
	var copiedCode, copiedLang string
	var opened []string

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Scroll(c).Fill().Children(func() {
				ui.Column(c).FillWidth().Padding(16).Children(func() {
					docview.Markdown(c.Key("doc"), src, docview.MarkdownOptions{
						OnLink:   func(u string) { opened = append(opened, u) },
						Snackbar: snack,
						OnCopyCode: func(code, lang string) {
							copiedCode = code
							copiedLang = lang
						},
					})
				})
			})
			md3e.SnackbarHost(c, snack)
		}).Fill()
	}, 640, 720)

	for _, want := range []string{
		"Documentation",
		"Note",
		"Useful context with bold text.",
		"Warning",
		"Check the threshold before deploying.",
		"API",
		"Ready",
		"func hello() {}",
		"go",
	} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q in %q", want, tt.Texts())
		}
	}

	if _, ok := tt.AccessByRole(ui.RoleHeading, "Documentation"); !ok {
		t.Fatal("missing heading accessibility role")
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Note"); !ok {
		t.Fatal("missing Note callout RoleRegion")
	}
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Warning"); !ok {
		t.Fatal("missing Warning callout RoleRegion")
	}

	if err := tt.Click("guide"); err != nil {
		t.Fatalf("click link: %v", err)
	}
	if !slices.Equal(opened, []string{"https://go.dev"}) {
		t.Fatalf("OnLink got %q", opened)
	}

	if err := tt.Click("Copy"); err != nil {
		t.Fatalf("click code Copy: %v", err)
	}
	tt.Frame()
	if copiedLang != "go" || copiedCode != "func hello() {}\n" {
		t.Fatalf("OnCopyCode got (%q, %q)", copiedLang, copiedCode)
	}
	if tt.Clipboard() != "func hello() {}\n" {
		t.Fatalf("clipboard = %q", tt.Clipboard())
	}
	if !tt.HasText("Copied go code") {
		t.Fatalf("expected Snackbar confirmation, texts=%q", tt.Texts())
	}

	tt.Advance(6 * time.Second)
	if tt.FramePending() {
		t.Fatal("expected settled frames after copy & snackbar")
	}
}

func TestMarkdownStreamAndReduceMotion(t *testing.T) {
	const full = "## Streaming\n\nFirst sentence. Second sentence!\n\n> [!TIP]\n> Fast path active.\n\n```go\nvar x = 1\n```\n"

	// 1. ReduceMotion settles immediately with zero pending frames.
	rmTheme := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{ReduceMotion: true})
	var streamText string
	var revealed bool
	ttRM := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, rmTheme, func() {
			el := docview.MarkdownStream(c.Key("stream"), streamText)
			revealed = docview.Revealed(el)
		})
	}, 560, 480)
	streamText = full
	ttRM.Frame()
	if ttRM.FramePending() {
		t.Fatal("ReduceMotion stream left pending frames")
	}
	if !revealed {
		t.Fatal("ReduceMotion stream should be revealed immediately")
	}
	if !ttRM.HasText("Fast path active.") || !ttRM.HasText("var x = 1") {
		t.Fatalf("ReduceMotion stream missing rendered content: %q", ttRM.Texts())
	}

	// 2. Normal incremental streaming converges to the exact static Markdown text.
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	var cur string
	done := false
	var elRevealed bool
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			var el ui.Element
			if done {
				el = docview.Markdown(c.Key("reply"), cur)
			} else {
				el = docview.MarkdownStream(c.Key("reply"), cur, docview.MarkdownOptions{
					Reveal: &markdown.Reveal{Unit: markdown.ByWord, CatchUp: 120 * time.Millisecond, Fade: 150 * time.Millisecond},
				})
			}
			elRevealed = docview.Revealed(el)
		})
	}, 560, 600)

	for i := 0; i < len(full); {
		_, w := utf8.DecodeRuneInString(full[i:])
		i += w
		cur = full[:i]
		if i%9 == 0 {
			tt.Advance(20 * time.Millisecond)
		}
	}
	done = true
	tt.Frame()
	for range 40 {
		tt.Advance(30 * time.Millisecond)
		if elRevealed && !tt.FramePending() {
			break
		}
	}
	if !elRevealed || tt.FramePending() {
		t.Fatalf("stream did not settle: revealed=%v pending=%v", elRevealed, tt.FramePending())
	}

	whole := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			docview.Markdown(c.Key("reply"), full)
		})
	}, 560, 600)

	gotJoined := strings.Join(tt.Texts(), "|")
	wantJoined := strings.Join(whole.Texts(), "|")
	if gotJoined != wantJoined {
		t.Fatalf("streamed texts:\n%s\nwant static:\n%s", gotJoined, wantJoined)
	}
}

func TestMarkdownCopyAsMarkdownAcrossCallout(t *testing.T) {
	const src = "Intro text.\n\n> [!IMPORTANT]\n> Check [the docs](https://example.org).\n\nOutro text."
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			docview.Markdown(c.Key("doc"), src)
		})
	}, 560, 400)

	r, ok := tt.Find("Intro text.")
	if !ok {
		t.Fatal("missing Intro text.")
	}
	tt.ClickAt(r.X+2, r.Y+r.H/2)
	tt.Command("selectAll")
	tt.Key(ui.Cmd|ui.Shift, ui.KeyC)
	got := tt.Clipboard()
	if !strings.Contains(got, "Intro text.") || !strings.Contains(got, "[the docs](https://example.org)") || !strings.Contains(got, "Outro text.") {
		t.Fatalf("Copy as Markdown across Callout lost content: %q", got)
	}
}
