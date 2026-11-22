package codeview_test

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/codeview"
	"github.com/imicola/mygo-md3e/tokens"
)

func TestThemeFromMD3EVariantsAndContrast(t *testing.T) {
	cases := []struct {
		name       string
		th         *md3e.Theme
		minBody    float64
		minMuted   float64
		monochrome bool
	}{
		{
			name:    "nil",
			th:      nil,
			minBody: 4.5, minMuted: 3.0,
		},
		{
			name:    "light-expressive",
			th:      md3e.ThemeFromSeed(ui.Hex("#6750A4"), md3e.ThemeOptions{Variant: tokens.Expressive}),
			minBody: 4.5, minMuted: 3.0,
		},
		{
			name:    "dark-tonalspot",
			th:      md3e.ThemeFromSeed(ui.Hex("#0061A4"), md3e.ThemeOptions{Dark: true, Variant: tokens.TonalSpot}),
			minBody: 4.5, minMuted: 3.0,
		},
		{
			name:    "light-high-contrast",
			th:      md3e.ThemeFromSeed(ui.Hex("#2E6B3E"), md3e.ThemeOptions{Contrast: tokens.ContrastHigh}),
			minBody: 7.0, minMuted: 4.5,
		},
		{
			name:    "dark-high-contrast",
			th:      md3e.ThemeFromSeed(ui.Hex("#BA1A1A"), md3e.ThemeOptions{Dark: true, Contrast: tokens.ContrastHigh}),
			minBody: 7.0, minMuted: 4.5,
		},
		{
			name:       "dark-monochrome",
			th:         md3e.ThemeFromSeed(ui.Hex("#6750A4"), md3e.ThemeOptions{Dark: true, Variant: tokens.Monochrome}),
			minBody:    4.5,
			minMuted:   3.0,
			monochrome: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			et := codeview.ThemeFromMD3E(tc.th)
			if et == nil {
				t.Fatal("ThemeFromMD3E returned nil")
			}
			if et.Background.A == 0 || et.Text.A == 0 || et.Cursor.A == 0 {
				t.Fatalf("missing structural colors: bg=%v text=%v cursor=%v", et.Background, et.Text, et.Cursor)
			}
			if et.Added.A == 0 || et.Deleted.A == 0 || et.Modified.A == 0 {
				t.Fatalf("missing diff colors: added=%v deleted=%v modified=%v", et.Added, et.Deleted, et.Modified)
			}

			resolved := tc.th
			if resolved == nil {
				resolved = md3e.DefaultTheme(nil)
			}
			codeHigh := resolved.Colors.SurfaceContainerHigh

			for _, k := range []editor.TokenKind{
				editor.TokenText,
				editor.TokenKeyword,
				editor.TokenType,
				editor.TokenFunction,
				editor.TokenString,
				editor.TokenNumber,
				editor.TokenOperator,
				editor.TokenProperty,
				editor.TokenComment,
				editor.TokenInvalid,
			} {
				col := et.TokenColor(k)
				if col.A == 0 {
					t.Fatalf("token kind %d has transparent color", k)
				}
				minRatio := tc.minBody
				if k == editor.TokenComment {
					minRatio = tc.minMuted
				}
				if got := tokens.ContrastRatio(col, et.Background); got < minRatio {
					t.Errorf("token %d contrast on editor bg = %.2f < %.2f (col=%v bg=%v)", k, got, minRatio, col, et.Background)
				}
				if codeHigh.A > 0 {
					if got := tokens.ContrastRatio(col, codeHigh); got < minRatio {
						t.Errorf("token %d contrast on SurfaceContainerHigh = %.2f < %.2f (col=%v bg=%v)", k, got, minRatio, col, codeHigh)
					}
				}
			}

			if !et.Tokens[editor.TokenComment].Italic {
				t.Errorf("expected comment token italic")
			}
			if !et.Tokens[editor.TokenHeading].Bold || !et.Tokens[editor.TokenStrong].Bold {
				t.Errorf("expected heading/strong tokens bold")
			}

			if tc.monochrome {
				kw := tokens.HCTFromColor(et.TokenColor(editor.TokenKeyword))
				str := tokens.HCTFromColor(et.TokenColor(editor.TokenString))
				if kw.Chroma > 4.0 || str.Chroma > 4.0 {
					t.Errorf("monochrome theme injected chroma: kw=%.2f str=%.2f", kw.Chroma, str.Chroma)
				}
			}
		})
	}
}

func TestFontAndApplyHelpers(t *testing.T) {
	comfort := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{
		MonoFont:  "JetBrains Mono",
		TextScale: 1.25,
	})
	desktop := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{
		Density: tokens.DensityDesktop,
	})
	compact := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{
		Density: tokens.DensityCompact,
	})

	fComfort, lhComfort := codeview.FontFromMD3E(comfort)
	fDesktop, lhDesktop := codeview.FontFromMD3E(desktop)
	fCompact, lhCompact := codeview.FontFromMD3E(compact)

	if fComfort.Family != "JetBrains Mono" || fComfort.Size != 14*1.25 {
		t.Fatalf("unexpected comfort font: %+v", fComfort)
	}
	if fDesktop.Family != "monospace" || fDesktop.Size != 13 {
		t.Fatalf("unexpected desktop font: %+v", fDesktop)
	}
	if fCompact.Size != 12 {
		t.Fatalf("unexpected compact font: %+v", fCompact)
	}
	if !(lhComfort >= lhDesktop && lhDesktop >= lhCompact) {
		t.Fatalf("expected non-increasing line heights across density: %v %v %v", lhComfort, lhDesktop, lhCompact)
	}

	// Render real editor.View and editor.DiffView headlessly inside ThemeScope.
	buf := editor.NewBuffer("package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n")
	buf.SetLanguage(editor.LanguageFor("main.go"))
	edState := &editor.State{Buffer: buf, ReadOnly: true}
	codeview.ApplyToEditor(edState, desktop)

	oldBuf := editor.NewBuffer("package main\nfunc a() {}\n")
	newBuf := editor.NewBuffer("package main\nfunc b() {}\n")
	diffState := &editor.DiffState{Old: oldBuf, New: newBuf}
	codeview.ApplyToDiff(diffState, desktop)

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, desktop, func() {
			ui.Column(c).Fill().Children(func() {
				codeview.ConfigureEditor(c, edState)
				editor.View(c, edState).Grow(1).FillWidth()
				editor.DiffView(c, diffState).Grow(1).FillWidth()
			})
		}).Fill()
	}, 640, 480)
	tt.Frame()
	if !tt.HasText("Code editor") || !tt.HasText("Old text") || !tt.HasText("New text") {
		t.Fatalf("expected editor and diff accessibility regions; texts=%q", tt.Texts())
	}
	if edState.Theme == nil || diffState.Theme == nil {
		t.Fatal("expected Theme applied on editor and diff states")
	}
	img := tt.Image()
	bg := edState.Theme.Background
	bgCount, fgCount := 0, 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.R == bg.R && c.G == bg.G && c.B == bg.B {
				bgCount++
			} else {
				fgCount++
			}
		}
	}
	if bgCount == 0 || fgCount == 0 {
		t.Fatalf("expected painted editor pixels on MD3E background; bgCount=%d fgCount=%d", bgCount, fgCount)
	}

	// Nil-safety checks.
	codeview.ApplyToEditor(nil, desktop)
	codeview.ApplyToDiff(nil, desktop)
	sb := codeview.ScrollbarFromMD3E(nil)
	if sb.ThumbWidth <= 0 || sb.Thumb.A == 0 {
		t.Fatalf("expected valid scrollbar style, got %+v", sb)
	}
}
